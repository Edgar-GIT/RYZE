import { ArrowLeft, CreditCard, FlaskConical, LogIn, RefreshCw, CheckCircle2, XCircle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ProgramStructure } from "@/components/program_structure/program_structure";
import { useTestMode } from "@/components/test_mode/test_mode_context";
import { useProgramCheckout, type PurchaseState } from "@/hooks/use_program_checkout";
import {
  fetchMarketplaceProgram,
  formatMarketplacePrice,
  FREE_PROGRAM_TYPE,
  type MarketplaceProgramDetail
} from "@/services/marketplace_api";
import type { PaymentMethodInfo } from "@/services/purchases_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./generic_program_detail_page.module.css";

interface PurchasePanelProps {
  isFree: boolean;
  minorUnits: number;
  currency: string;
  programId: string;
  state: PurchaseState;
  paymentMethods: PaymentMethodInfo[];
  selectedMethod: string;
  onSelectMethod: (method: string) => void;
  onBuy: () => void;
  onRetryPurchase: (purchaseId: string) => void;
  testMode?: boolean;
}

const PurchasePanel = ({ isFree, minorUnits, currency, programId, state, paymentMethods, selectedMethod, onSelectMethod, onBuy, onRetryPurchase, testMode = false }: PurchasePanelProps) => {
  if (isFree || state.status === "owned") {
    // Free programs have no Program Access page: their panel only links back
    // to the marketplace. Owned (paid/test) programs get an "Open program"
    // action into the account area.
    const isOwned = state.status === "owned" && !isFree;
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
            {isOwned ? (
              <Button
                to={`/services/my-programs/${programId}`}
                variant="primary"
                size="small"
              >
                Open program
              </Button>
            ) : null}
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
                to={`/services/my-programs/${programId}`}
                variant="primary"
                size="small"
              >
                Open program
              </Button>
              <Button to="/services/my-programs" variant="ghost" size="small">
                My Programs
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
      if (testMode) {
        return (
          <aside className={styles.purchase}>
            <div className={`${styles.purchaseCard} ${styles.purchaseTestMode}`}>
              <p className={styles.purchaseTitle}>
                {formatMarketplacePrice(0, currency, "premium")}
              </p>
              <p className={styles.purchaseText}>
                Test Mode purchase — this plan is granted instantly at no cost.
                No real payment is made.
              </p>
              <div className={styles.purchaseActions}>
                <Button
                  variant="primary"
                  size="small"
                  onClick={onBuy}
                  icon={<FlaskConical size={15} />}
                  iconPosition="left"
                >
                  Get plan in Test Mode
                </Button>
              </div>
            </div>
          </aside>
        );
      }
      if (paymentMethods.length === 0) {
        return (
          <aside className={styles.purchase}>
            <div className={`${styles.purchaseCard} ${styles.purchaseError}`}>
              <p className={styles.purchaseTitle}>
                <XCircle size={16} aria-hidden="true" />
                Checkout unavailable
              </p>
              <p className={styles.purchaseText}>
                No payment provider is currently configured. Please try again later.
              </p>
              <div className={styles.purchaseActions}>
                <Button to="/services/generic-program" variant="ghost" size="small">
                  Back to Training plans
                </Button>
              </div>
            </div>
          </aside>
        );
      }
      return (
        <aside className={styles.purchase}>
          <div className={styles.purchaseCard}>
            <p className={styles.purchaseTitle}>
              {formatMarketplacePrice(minorUnits, currency, "premium")}
            </p>
            <p className={styles.purchaseText}>One-time purchase. Secure checkout.</p>
            {paymentMethods.length > 1 ? (
              <fieldset className={styles.methodGroup}>
                <legend className={styles.methodLegend}>Payment method</legend>
                {paymentMethods.map((method) => (
                  <label
                    key={method.method}
                    className={joinClassNames(
                      styles.methodOption,
                      selectedMethod === method.method && styles.methodOptionSelected
                    )}
                  >
                    <input
                      type="radio"
                      name="payment-method"
                      value={method.method}
                      checked={selectedMethod === method.method}
                      onChange={() => onSelectMethod(method.method)}
                    />
                    <span>{method.label}</span>
                  </label>
                ))}
              </fieldset>
            ) : null}
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
  const { isActive: testModeActive } = useTestMode();
  const [detail, setDetail] = useState<MarketplaceProgramDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
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

  const isFreeProgram = detail ? detail.type === FREE_PROGRAM_TYPE : false;
  const isTestModePurchase = testModeActive && !isFreeProgram;

  useEffect(() => {
    void load();
  }, [load]);

  // The whole purchase lifecycle now lives in a shared hook, so Generic and
  // Premium Level 1 checkout cannot drift apart.
  const checkout = useProgramCheckout({
    programId,
    free: isFreeProgram,
    testModeActive,
    enabled: Boolean(detail) && !loading
  });
  const priceLabel = detail
    ? isTestModePurchase
      ? formatMarketplacePrice(0, detail.currency, detail.type)
      : formatMarketplacePrice(detail.price_minor_units, detail.currency, detail.type)
    : "";

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
                    {priceLabel}
                  </span>
                  {isTestModePurchase ? (
                    <span className={styles.chipTestMode}>
                      <FlaskConical size={12} aria-hidden="true" />
                      Test Mode
                    </span>
                  ) : null}
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
                  programId={programId}
                  state={checkout.state}
                  paymentMethods={checkout.paymentMethods}
                  selectedMethod={checkout.selectedMethod}
                  onSelectMethod={checkout.selectMethod}
                  onBuy={() => void checkout.buy()}
                  onRetryPurchase={(purchaseId) => void checkout.retryPurchase(purchaseId)}
                  testMode={isTestModePurchase}
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