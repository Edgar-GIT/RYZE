import { useCallback, useEffect, useState } from "react";

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
// The backend refuses to rebind a pending purchase to a different payment
// method, so the checkout must always resume with the recorded one.
const PAYMENT_METHOD_MISMATCH = "PAYMENT_METHOD_MISMATCH";
// Query parameters each provider appends to the return URL: Stripe substitutes
// {CHECKOUT_SESSION_ID} into the configured success URL, while PayPal appends
// the order token to the configured return URL.
const STRIPE_SESSION_PARAM = "session_id";
const PAYPAL_TOKEN_PARAM = "token";

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

  const runCapture = useCallback(
    async (purchaseId: string, providerPaymentId: string) => {
      setState({ status: "processing" });
      try {
        const purchase = await capturePayment(purchaseId, providerPaymentId);
        setState({ status: "success", purchase });
      } catch (error) {
        // 409 (PURCHASE_NOT_PENDING) can mean the purchase was already
        // completed by the server-side webhook between approval and capture.
        // Falling back to the ownership check reflects the actual state.
        if (error instanceof ApiError && error.status === 409) {
          try {
            if ((await resolveOwnership()) === "owned") {
              setState({ status: "success", purchase: { id: purchaseId } as Purchase });
              return;
            }
          } catch {
            // fall through to the error state
          }
        }
        setState({ status: "error", message: "We could not confirm your payment. No money was taken at this point." });
      }
    },
    [resolveOwnership]
  );

  // A purchase is bound to its payment method once a provider has been
  // resolved, so resuming a pending purchase must reuse the recorded method
  // rather than the one currently highlighted in the selector.
  const resolveMethodForPurchase = useCallback(
    (purchase: { payment_method?: string | null }): string | null => {
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
      setState({ status: "error", message: "We could not start the checkout. No money was taken at this point." });
    }
  }, []);

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
        try {
          if ((await resolveOwnership()) === "owned") {
            setState({ status: "owned" });
            return;
          }
        } catch {
          // fall through to the error state
        }
      }
      setState({ status: "error", message: "We could not complete this Test Mode purchase. Please try again." });
    }
  }, [programId, resolveOwnership]);

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
      setState({ status: "error", message: "We could not start your purchase. No money was taken at this point." });
    }
  }, [programId, negotiatePayment, resolveMethodForPurchase, runTestModePurchase, testModeActive]);

  const retryPurchase = useCallback(
    async (purchaseId: string) => {
      await negotiatePayment(purchaseId, selectedMethod);
    },
    [negotiatePayment, selectedMethod]
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
    if (!enabled || free) {
      return;
    }

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
      // capture.
      if (purchaseId && providerPaymentId) {
        await runCapture(purchaseId, providerPaymentId);
        return;
      }

      try {
        setState({ status: await resolveOwnership() });
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