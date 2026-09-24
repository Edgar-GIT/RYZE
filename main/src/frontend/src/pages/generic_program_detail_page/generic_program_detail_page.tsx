import { ArrowLeft, CreditCard, LogIn, RefreshCw, CheckCircle2, XCircle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ProgramStructure } from "@/components/program_structure/program_structure";
import { ApiError } from "@utils/http_client";
import {
  fetchMarketplaceProgram,
  formatMarketplacePrice,
  FREE_PROGRAM_TYPE,
  type MarketplaceProgramDetail
} from "@/services/marketplace_api";
import {
  capturePayment,
  createPurchaseIntent,
  fetchEntitlements,
  fetchMyPurchases,
  initiatePayment,
  type Purchase
} from "@/services/purchases_api";

import styles from "./generic_program_detail_page.module.css";

const PURCHASE_METHOD = "paypal";
const CANCELLED_STATUS = "cancelled";
const DUPLICATE_PURCHASE = "DUPLICATE_PURCHASE";

// The checkout lifecycle is purely client convenience: the backend is fully
// authoritative. Browser redirects never complete a purchase — only the
// verified server-side capture can do that.
type PurchaseState =
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

interface PurchasePanelProps {
  isFree: boolean;
  minorUnits: number;
  currency: string;
  state: PurchaseState;
  onBuy: () => void;
  onRetryPurchase: (purchaseId: string) => void;
}

const PurchasePanel = ({ isFree, minorUnits, currency, state, onBuy, onRetryPurchase }: PurchasePanelProps) => {
  if (isFree || state.status === "owned") {
    return (
      <aside className={styles.purchase}>
        <div className={styles.purchaseCard}>
          <p className={styles.purchaseTitle}>
            <CheckCircle2 size={16} aria-hidden="true" />
            You have access
          </p>
          <p className={styles.purchaseText}>
            {isFree
              ? "This training plan is available at no cost."
              : "You already own this training plan."}
          </p>
          <div className={styles.purchaseActions}>
            <Button to="/services/generic-program" variant="secondary" size="small">
              Back to Training plans
            </Button>
          </div>
        </div>
      </aside>
    );
  }

  switch (state.status) {
    case "checking":
      return (
        <aside className={styles.purchase}>
          <div className={styles.purchaseCard}>
            <p className={styles.purchaseText}>Checking access…</p>
          </div>
        </aside>
      );

    case "guest":
      return (
        <aside className={styles.purchase}>
          <div className={styles.purchaseCard}>
            <p className={styles.purchaseTitle}>
              {formatMarketplacePrice(minorUnits, currency, "premium")}
            </p>
            <p className={styles.purchaseText}>Sign in to purchase this training plan.</p>
            <div className={styles.purchaseActions}>
              <Button
                to="/login"
                variant="primary"
                size="small"
                icon={<LogIn size={15} />}
                iconPosition="left"
              >
                Sign in
              </Button>
            </div>
          </div>
        </aside>
      );

    case "creating":
    case "negotiating":
      return (
        <aside className={styles.purchase}>
          <div className={styles.purchaseCard}>
            <p className={styles.purchaseText}>Preparing secure checkout…</p>
          </div>
        </aside>
      );

    case "redirecting":
      return (
        <aside className={styles.purchase}>
          <div className={styles.purchaseCard}>
            <p className={styles.purchaseText}>
              Redirecting to the secure payment page. Do not close this window…
            </p>
          </div>
        </aside>
      );

    case "processing":
      return (
        <aside className={styles.purchase}>
          <div className={styles.purchaseCard}>
            <p className={styles.purchaseText}>
              Confirming your payment. This may take a few seconds…
            </p>
          </div>
        </aside>
      );

    case "success":
      return (
        <aside className={styles.purchase}>
          <div className={`${styles.purchaseCard} ${styles.purchaseSuccess}`}>
            <p className={styles.purchaseTitle}>
              <CheckCircle2 size={16} aria-hidden="true" />
              Purchase complete
            </p>
            <p className={styles.purchaseText}>You now have access to this training plan.</p>
            <div className={styles.purchaseActions}>
              <Button
                to="/services/generic-program"
                variant="secondary"
                size="small"
              >
                Back to Training plans
              </Button>
            </div>
          </div>
        </aside>
      );

    case "cancelled":
      return (
        <aside className={styles.purchase}>
          <div className={`${styles.purchaseCard} ${styles.purchaseCancelled}`}>
            <p className={styles.purchaseTitle}>
              <XCircle size={16} aria-hidden="true" />
              Payment cancelled
            </p>
            <p className={styles.purchaseText}>
              No payment was completed. You can try again whenever you are ready.
            </p>
            <div className={styles.purchaseActions}>
              <Button
                variant="primary"
                size="small"
                onClick={() => onRetryPurchase(state.purchaseId)}
              >
                Try again
              </Button>
              <Button to="/services/generic-program" variant="ghost" size="small">
                Back to Training plans
              </Button>
            </div>
          </div>
        </aside>
      );

    case "error":
      return (
        <aside className={styles.purchase}>
          <div className={`${styles.purchaseCard} ${styles.purchaseError}`}>
            <p className={styles.purchaseTitle}>
              <XCircle size={16} aria-hidden="true" />
              Purchase unavailable
            </p>
            <p className={styles.purchaseText}>{state.message}</p>
            <div className={styles.purchaseActions}>
              <Button
                variant="primary"
                size="small"
                onClick={onBuy}
                icon={<RefreshCw size={15} />}
              >
                Try again
              </Button>
              <Button to="/services/generic-program" variant="ghost" size="small">
                Back to Training plans
              </Button>
            </div>
          </div>
        </aside>
      );

    case "ready":
    default:
      return (
        <aside className={styles.purchase}>
          <div className={styles.purchaseCard}>
            <p className={styles.purchaseTitle}>
              {formatMarketplacePrice(minorUnits, currency, "premium")}
            </p>
            <p className={styles.purchaseText}>One-time purchase. Secure checkout with PayPal.</p>
            <div className={styles.purchaseActions}>
              <Button
                variant="primary"
                size="small"
                onClick={onBuy}
                icon={<CreditCard size={15} />}
                iconPosition="left"
              >
                Buy now
              </Button>
            </div>
          </div>
        </aside>
      );
  }
};

export const GenericProgramDetailPage = () => {
  const { programId } = useParams<{ programId: string }>();
  const [detail, setDetail] = useState<MarketplaceProgramDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [purchaseState, setPurchaseState] = useState<PurchaseState>({ status: "checking" });

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");
    try {
      setDetail(await fetchMarketplaceProgram(programId));
    } catch {
      setDetail(null);
      setErrorMessage("This training plan is not available through the marketplace.");
    } finally {
      setLoading(false);
    }
  }, [programId]);

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
    async (purchaseId: string, orderId: string) => {
      setPurchaseState({ status: "processing" });
      try {
        const purchase = await capturePayment(purchaseId, orderId);
        setPurchaseState({ status: "success", purchase });
      } catch (error) {
        // 409 (PURCHASE_NOT_PENDING) can mean the purchase was already
        // completed by the server-side webhook between approval and capture.
        // Falling back to the ownership check reflects the actual state.
        if (error instanceof ApiError && error.status === 409) {
          try {
            if ((await resolveOwnership()) === "owned") {
              setPurchaseState({ status: "success", purchase: { id: purchaseId } as Purchase });
              return;
            }
          } catch {
            // fall through to the error state
          }
        }
        setPurchaseState({ status: "error", message: "We could not confirm your payment. No money was taken at this point." });
      }
    },
    [resolveOwnership]
  );

  const negotiatePayment = useCallback(async (purchaseId: string) => {
    setPurchaseState({ status: "negotiating" });
    try {
      const initiation = await initiatePayment(purchaseId, PURCHASE_METHOD);
      if (!initiation.checkout_url) {
        setPurchaseState({ status: "error", message: "Checkout is temporarily unavailable. Please try again." });
        return;
      }
      setPurchaseState({ status: "redirecting" });
      window.location.assign(initiation.checkout_url);
    } catch {
      setPurchaseState({ status: "error", message: "We could not start the checkout. No money was taken at this point." });
    }
  }, []);

  const handleBuy = useCallback(async () => {
    setPurchaseState({ status: "creating" });
    try {
      const purchase = await createPurchaseIntent(programId);
      await negotiatePayment(purchase.id);
    } catch (error) {
      // A pending purchase already exists for this program (e.g. the user
      // navigated away mid-checkout): recover by resuming that purchase.
      if (error instanceof ApiError && error.code === DUPLICATE_PURCHASE) {
        try {
          const purchases = await fetchMyPurchases();
          const pending = purchases.find((p) => p.program_id === programId && p.status === "pending");
          if (pending) {
            await negotiatePayment(pending.id);
            return;
          }
        } catch {
          // fall through to the error state
        }
      }
      setPurchaseState({ status: "error", message: "We could not start your purchase. No money was taken at this point." });
    }
  }, [programId, negotiatePayment]);

  const handleRetryPurchase = useCallback(
    async (purchaseId: string) => {
      await negotiatePayment(purchaseId);
    },
    [negotiatePayment]
  );

  // Resolve the initial purchase post-a state once the program is known.
  // This intentionally runs alongside the content load: the entitlements
  // fetch doubles as the authentication gate.
  useEffect(() => {
    if (!detail || detail.type === FREE_PROGRAM_TYPE || loading) {
      return;
    }

    let cancelled = false;
    const params = new URLSearchParams(window.location.search);
    const purchaseId = params.get("purchase_id");
    const orderId = params.get("token");

    const resolveInitial = async () => {
      if (params.get("status") === CANCELLED_STATUS) {
        if (purchaseId) {
          setPurchaseState({ status: "cancelled", purchaseId });
        } else {
          setPurchaseState({ status: "ready" });
        }
        return;
      }

      // Browser return from the payment page: attempt the server-verified
      // capture. The order token accompanies every return URL.
      if (purchaseId && orderId) {
        await runCapture(purchaseId, orderId);
        return;
      }

      try {
        setPurchaseState({ status: await resolveOwnership() });
      } catch {
        if (!cancelled) {
          setPurchaseState({ status: "error", message: "We could not verify your access. Please try again." });
        }
      }
    };

    void resolveInitial();

    return () => {
      cancelled = true;
    };
  }, [detail, loading, programId, runCapture, resolveOwnership]);

  const isFreeProgram = detail ? detail.type === FREE_PROGRAM_TYPE : false;

  return (
    <PageWrapper className={styles.page}>
      <section className={styles.detail}>
        <Container size="wide" className={styles.container}>
          <div className={styles.back}>
            <Button
              to="/services/generic-program"
              variant="ghost"
              size="small"
              icon={<ArrowLeft size={15} />}
              iconPosition="left"
            >
              Back to Training plans
            </Button>
          </div>

          {loading ? (
            <div className={styles.state}>
              <p>Loading training plan…</p>
            </div>
          ) : errorMessage || !detail ? (
            <div className={styles.state}>
              <p>{errorMessage}</p>
              <div className={styles.retry}>
                <Button
                  variant="secondary"
                  size="small"
                  onClick={() => void load()}
                  icon={<RefreshCw size={15} />}
                >
                  Retry
                </Button>
              </div>
            </div>
          ) : (
            <>
              <header className={styles.heading}>
                <h1 className={styles.title}>{detail.name}</h1>
                {detail.description ? (
                  <p className={styles.subtitle}>{detail.description}</p>
                ) : null}

                <div className={styles.chips}>
                  <span className={detail.type === FREE_PROGRAM_TYPE ? styles.chipFree : styles.chipPrice}>
                    {formatMarketplacePrice(detail.price_minor_units, detail.currency, detail.type)}
                  </span>
                  {detail.training_type ? (
                    <span className={styles.chip}>{detail.training_type}</span>
                  ) : null}
                  {detail.level ? <span className={styles.chip}>{detail.level}</span> : null}
                  {detail.duration_weeks !== null ? (
                    <span className={styles.chip}>
                      {detail.duration_weeks} week{detail.duration_weeks === 1 ? "" : "s"}
                    </span>
                  ) : null}
                  {detail.frequency_per_week !== null ? (
                    <span className={styles.chip}>
                      {detail.frequency_per_week} day{detail.frequency_per_week === 1 ? "" : "s"}/week
                    </span>
                  ) : null}
                </div>

                <PurchasePanel
                  isFree={isFreeProgram}
                  minorUnits={detail.price_minor_units}
                  currency={detail.currency}
                  state={purchaseState}
                  onBuy={() => void handleBuy()}
                  onRetryPurchase={(purchaseId) => void handleRetryPurchase(purchaseId)}
                />
              </header>

              <ProgramStructure weeks={detail.weeks} />
            </>
          )}
        </Container>
      </section>
    </PageWrapper>
  );
};