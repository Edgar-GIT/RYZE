import { useCallback, useEffect, useRef, useState } from "react";

import { ApiError } from "@utils/http_client";
import {
  capturePayment,
  createPurchaseIntent,
  fetchEntitlements,
  fetchMyPurchases,
  fetchPaymentMethods,
  initiatePayment,
  type PaymentMethodInfo,
  type Purchase
} from "@/services/purchases_api";
import { completeTestPurchase } from "@/services/test_mode_api";

const CANCELLED_STATUS = "cancelled";
const DUPLICATE_PURCHASE = "DUPLICATE_PURCHASE";
// The pair resolved another way (a sibling purchase completed first), so no
// provider session can ever grant access. The real access state is owned.
const DUPLICATE_ENTITLEMENT = "DUPLICATE_ENTITLEMENT";
// The backend refuses to rebind a pending purchase to a different payment
// method, so the checkout must always resume with the recorded one.
const PAYMENT_METHOD_MISMATCH = "PAYMENT_METHOD_MISMATCH";
// A pending purchase outlived its payment window and can no longer be paid.
// The backend supersedes it (marks it failed) on the next intent, so the
// checkout recovers by creating a fresh intent once and continuing.
const PURCHASE_EXPIRED = "PURCHASE_EXPIRED";
// Query parameters each provider appends to the return URL: Stripe substitutes
// {CHECKOUT_SESSION_ID} into the configured success URL, while PayPal appends
// the order token to the configured return URL.
const STRIPE_SESSION_PARAM = "session_id";
const PAYPAL_TOKEN_PARAM = "token";
// A 409 PURCHASE_NOT_PENDING means the capture raced the server-side webhook
// reconciliation, which may still be settling a few hundred milliseconds after
// the browser returns. Bounded polling reflects the real ownership state
// instead of declaring a failure the instant the capture loses the race.
const OWNERSHIP_POLL_ATTEMPTS = 5;
const OWNERSHIP_POLL_INTERVAL_MS = 1000;

// The checkout lifecycle is purely client convenience: the backend is fully
// authoritative. Browser redirects never complete a purchase — only the
// verified server-side capture can do that. Inside Test Mode the purchase is
// completed directly by the backend with no payment provider at all.
export type PurchaseState =
  | { status: "checking" }
  | { status: "guest" }
  | { status: "owned" }
  | { status: "ready" }
  | { status: "creating" }
  | { status: "negotiating" }
  | { status: "redirecting" }
  | { status: "processing" }
  | { status: "success"; purchase: Purchase }
  | { status: "cancelled"; purchaseId: string }
  | { status: "error"; message: string };

export interface ProgramCheckout {
  state: PurchaseState;
  paymentMethods: PaymentMethodInfo[];
  selectedMethod: string;
  selectMethod: (method: string) => void;
  buy: () => Promise<void>;
  retryPurchase: (purchaseId: string) => Promise<void>;
  /** Re-reads the ownership state after an external event (for example a
   *  completed Test Mode purchase) so the panel reflects real access. */
  refreshOwnership: () => Promise<"guest" | "owned" | "ready">;
}

interface UseProgramCheckoutOptions {
  programId: string;
  /** A free program has no checkout at all, so the post-a resolution is
   *  skipped entirely for it. */
  free: boolean;
  testModeActive: boolean;
  /** The caller signals that its content has loaded. Ownership resolution and
   *  the entitlements fetch double as the authentication gate, so it only runs
   *  once the product is actually known. */
  enabled: boolean;
}

// useProgramCheckout owns the whole purchase lifecycle for a single program:
// ownership resolution, purchase creation, payment negotiation, provider
// redirect and return, capture, cancellation, duplicate-purchase recovery,
// webhook race recovery and the Test Mode shortcut.
//
// It is deliberately provider agnostic. It never names PayPal, Stripe or MB
// WAY and never hard-codes a payment method: the backend advertises the
// available methods and the caller follows whatever checkout URL comes back.
export const useProgramCheckout = ({ programId, free, testModeActive, enabled }: UseProgramCheckoutOptions): ProgramCheckout => {
  const [state, setState] = useState<PurchaseState>({ status: "checking" });
  const [paymentMethods, setPaymentMethods] = useState<PaymentMethodInfo[]>([]);
  const [selectedMethod, setSelectedMethod] = useState("");
  // Guards the PURCHASE_EXPIRED recovery so a checkout freshens its intent at
  // most once, keeping the auto-recovery bounded instead of recursive.
  const freshenedIntent = useRef(false);

  const resolveOwnership = useCallback(async (): Promise<"guest" | "owned" | "ready"> => {
    try {
      const entitlements = await fetchEntitlements();
      if (entitlements.some((entitlement) => entitlement.program_id === programId)) {
        return "owned";
      }
      return "ready";
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        return "guest";
      }
      throw error;
    }
  }, [programId]);

  // A conflict (409) may simply mean the (user, program) pair is already
  // owned — a sibling purchase completed, or the persona bought it in an
  // earlier session. The checkout resolves the real access state so an owned
  // program is never presented as a failure; anything else falls back to the
  // caller's message.
  const recoverOwnershipOrFail = useCallback(
    async (fallbackMessage: string) => {
      try {
        if ((await resolveOwnership()) === "owned") {
          setState({ status: "owned" });
          return;
        }
      } catch {
        // fall through to the error state
      }
      setState({ status: "error", message: fallbackMessage });
    },
    [resolveOwnership]
  );

  const runCapture = useCallback(
    async (purchaseId: string, providerPaymentId: string) => {
      setState({ status: "processing" });
      try {
        const purchase = await capturePayment(purchaseId, providerPaymentId);
        setState({ status: "success", purchase });
      } catch (error) {
        // 409 (PURCHASE_NOT_PENDING) can mean the purchase was already
        // completed by the server-side webhook between approval and capture.
        // The payment may already have moved, so the capture outcome is
        // decided by the real access state, which the server is still settling.
        if (error instanceof ApiError && error.status === 409) {
          try {
            for (let attempt = 0; attempt < OWNERSHIP_POLL_ATTEMPTS; attempt += 1) {
              if ((await resolveOwnership()) === "owned") {
                setState({ status: "success", purchase: { id: purchaseId } as Purchase });
                return;
              }
              if (attempt < OWNERSHIP_POLL_ATTEMPTS - 1) {
                await new Promise((resolve) => setTimeout(resolve, OWNERSHIP_POLL_INTERVAL_MS));
              }
            }
          } catch {
            // fall through to the error state
          }
          setState({
            status: "error",
            message: "Your payment is being confirmed. If your access does not appear within a few minutes, please try again later."
          });
          return;
        }
        setState({ status: "error", message: "We could not confirm your payment." });
      }
    },
    [resolveOwnership]
  );

  // A purchase is bound to its payment method once a provider has been
  // resolved, so resuming a pending purchase must reuse the recorded method
  // rather than the one currently highlighted in the selector.
  const resolveMethodForPurchase = useCallback(
    (purchase: { id?: string; payment_method?: string | null }): string | null => {
      if (purchase.payment_method) return purchase.payment_method;
      if (!selectedMethod) return null;
      return selectedMethod;
    },
    [selectedMethod]
  );

  const negotiatePayment = useCallback(async (purchaseId: string, method: string | null) => {
    setState({ status: "negotiating" });
    if (!method) {
      setState({ status: "error", message: "Checkout is temporarily unavailable. Please try again later." });
      return;
    }
    try {
      const initiation = await initiatePayment(purchaseId, method);
      if (!initiation.checkout_url) {
        setState({ status: "error", message: "Checkout is temporarily unavailable. Please try again." });
        return;
      }
      setState({ status: "redirecting" });
      window.location.assign(initiation.checkout_url);
    } catch (error) {
      if (error instanceof ApiError && error.code === PAYMENT_METHOD_MISMATCH) {
        setState({
          status: "error",
          message: "This purchase was already started with another payment method. Please try again later."
        });
        return;
      }
      // 409 DUPLICATE_ENTITLEMENT: the provider refused because the pair is
      // already resolved — a sibling purchase completed while this one waited,
      // so no provider session can ever grant access. Resolving ownership
      // reflects the actual access state.
      if (error instanceof ApiError && error.code === DUPLICATE_ENTITLEMENT) {
        await recoverOwnershipOrFail("We could not start the checkout. No money was taken at this point.");
        return;
      }
      // The pending purchase outlived its window, so the provider session for
      // it can no longer be paid. Recover by starting a fresh intent, which
      // the backend uses to supersede (fail) the expired purchase.
      if (error instanceof ApiError && error.code === PURCHASE_EXPIRED && !freshenedIntent.current) {
        freshenedIntent.current = true;
        try {
          const fresh = await createPurchaseIntent(programId);
          await negotiatePayment(fresh.id, resolveMethodForPurchase(fresh));
          return;
        } catch {
          // fall through to the error state
        }
      }
      setState({ status: "error", message: "We could not start the checkout. No money was taken at this point." });
    }
  }, [programId, recoverOwnershipOrFail, resolveMethodForPurchase]);

  const runTestModePurchase = useCallback(async () => {
    setState({ status: "creating" });
    try {
      const purchase = await completeTestPurchase(programId);
      setState({ status: "success", purchase });
    } catch (error) {
      // 409 DUPLICATE_ENTITLEMENT: the persona already owns the program
      // (e.g. it was purchased in a previous Test Mode session). Resolving
      // ownership reflects the actual access state.
      if (error instanceof ApiError && error.status === 409) {
        await recoverOwnershipOrFail("We could not complete this Test Mode purchase. Please try again.");
        return;
      }
      setState({ status: "error", message: "We could not complete this Test Mode purchase. Please try again." });
    }
  }, [programId, recoverOwnershipOrFail]);

  const buy = useCallback(async () => {
    if (testModeActive) {
      await runTestModePurchase();
      return;
    }
    setState({ status: "creating" });
    try {
      const purchase = await createPurchaseIntent(programId);
      await negotiatePayment(purchase.id, resolveMethodForPurchase(purchase));
    } catch (error) {
      // A pending purchase already exists for this program (e.g. the user
      // navigated away mid-checkout): recover by resuming that purchase.
      if (error instanceof ApiError && error.code === DUPLICATE_PURCHASE) {
        try {
          const purchases = await fetchMyPurchases();
          const pending = purchases.find((p) => p.program_id === programId && p.status === "pending");
          if (pending) {
            await negotiatePayment(pending.id, resolveMethodForPurchase(pending));
            return;
          }
        } catch {
          // fall through to the error state
        }
      }
      // 409 DUPLICATE_ENTITLEMENT on intent creation: the pair resolved
      // between the ownership check and this click, so the user already owns
      // the program. Reflect the real access state.
      if (error instanceof ApiError && error.code === DUPLICATE_ENTITLEMENT) {
        await recoverOwnershipOrFail("We could not start your purchase. No money was taken at this point.");
        return;
      }
      setState({ status: "error", message: "We could not start your purchase. No money was taken at this point." });
    }
  }, [programId, negotiatePayment, recoverOwnershipOrFail, resolveMethodForPurchase, runTestModePurchase, testModeActive]);

  const retryPurchase = useCallback(
    async (purchaseId: string) => {
      setState({ status: "negotiating" });
      try {
        // A pending purchase is bound to the method recorded at its first
        // successful initiation, so retrying must reuse that method rather
        // than whichever one the selector happens to highlight. The purchase
        // list is the only client-side source of the recorded method.
        const purchases = await fetchMyPurchases();
        const purchase = purchases.find((entry) => entry.id === purchaseId);
        await negotiatePayment(purchaseId, resolveMethodForPurchase(purchase ?? { id: purchaseId }));
      } catch {
        setState({ status: "error", message: "We could not start the checkout. No money was taken at this point." });
      }
    },
    [negotiatePayment, resolveMethodForPurchase]
  );

  // Resolve the currently configured payment methods. The endpoint is public
  // and the frontend only renders what the backend advertises — it never
  // hard-codes a payment method. A failure simply leaves the checkout panel
  // in its unavailable state.
  useEffect(() => {
    let cancelled = false;
    void fetchPaymentMethods()
      .then((methods) => {
        if (cancelled) return;
        setPaymentMethods(methods);
        setSelectedMethod(methods.length > 0 ? methods[0].method : "");
      })
      .catch(() => {
        if (cancelled) return;
        setPaymentMethods([]);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Resolve the initial purchase post-a state once the product is known.
  // This intentionally runs alongside the content load: the entitlements
  // fetch doubles as the authentication gate.
  useEffect(() => {
    let cancelled = false;
    const params = new URLSearchParams(window.location.search);
    const purchaseId = params.get("purchase_id");
    // Provider return identifiers: PayPal appends the order token, Stripe
    // appends the Checkout Session ID. Whichever arrives is only a correlation
    // hint — the backend re-verifies the payment before completing anything.
    const providerPaymentId = params.get(STRIPE_SESSION_PARAM) ?? params.get(PAYPAL_TOKEN_PARAM);

    const resolveInitial = async () => {
      if (params.get("status") === CANCELLED_STATUS) {
        if (purchaseId) {
          setState({ status: "cancelled", purchaseId });
        } else {
          setState({ status: "ready" });
        }
        return;
      }

      // Browser return from the payment page: attempt the server-verified
      // capture. This runs as soon as the page mounts, before and independent
      // of the product details: a buyer who already paid at the provider must
      // never be left un-captured just because the program data failed to load.
      if (purchaseId && providerPaymentId) {
        await runCapture(purchaseId, providerPaymentId);
        return;
      }

      if (!enabled || free) {
        return;
      }

      try {
        const ownership = await resolveOwnership();
        if (!cancelled) {
          setState({ status: ownership });
        }
      } catch {
        if (!cancelled) {
          setState({ status: "error", message: "We could not verify your access. Please try again." });
        }
      }
    };

    void resolveInitial();

    return () => {
      cancelled = true;
    };
  }, [enabled, free, runCapture, resolveOwnership]);

  return {
    state,
    paymentMethods,
    selectedMethod,
    selectMethod: setSelectedMethod,
    buy,
    retryPurchase,
    refreshOwnership: resolveOwnership
  };
};