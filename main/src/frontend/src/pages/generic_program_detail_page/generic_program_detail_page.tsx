import { ArrowLeft, BarChart3, CreditCard, Dumbbell, FlaskConical, Lock, LogIn, RefreshCw, CheckCircle2, XCircle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useHistory, useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useTestMode } from "@/components/test_mode/test_mode_context";
import { useProgramCheckout, type PurchaseState } from "@/hooks/use_program_checkout";
import {
  fetchMarketplaceProgram,
  formatMarketplacePrice,
  FREE_PROGRAM_TYPE,
  type MarketplaceProgramDetail
} from "@/services/marketplace_api";
import type { PaymentMethodInfo } from "@/services/purchases_api";
import { coachInitials, recordProgramView } from "@/services/coaches_api";
import {
  activateFreeProgram,
  fetchFreeActivations,
  type FreeProgramActivation
} from "@/services/free_programs_api";
import { COMMERCE_PT } from "@/i18n/catalogs/commerce.pt";
import { type Catalog, useGettext, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";

import styles from "./generic_program_detail_page.module.css";

const PAGE_PT: Catalog = {
  "Created by": "Criado por",
  "See {name}'s coach profile": "Ver o perfil de treinador de {name}",
  "Free plan": "Plano gratuito",
  "Sign in to activate {name} and unlock the full training structure.":
    "Entra na tua conta para ativar {name} e desbloquear a estrutura de treino completa.",
  "You have activated {name}.": "Já ativaste {name}.",
  "Activate {name} to unlock the full training structure and workout player.":
    "Ativa {name} para desbloquear a estrutura de treino completa e o modo de treino.",
  "We could not activate this plan. Please try again.": "Não foi possível ativar este plano. Tenta novamente.",
  "Activating…": "A ativar…",
  "Activate free plan": "Ativar plano gratuito",
  "This training plan is not available through the marketplace.": "Este plano de treino não está disponível no catálogo.",
  "Loading training plan…": "A carregar o plano de treino…",
  "Full training structure": "Estrutura de treino completa",
  "Weeks, training days, exercises and prescriptions unlock after activation or purchase. You will then be able to log every set, follow rest timers and track progress.":
    "Semanas, dias de treino, exercícios e prescrições ficam disponíveis depois da ativação ou da compra. Depois podes registar cada série, seguir os tempos de descanso e acompanhar o progresso.",
  "Guided workout sessions": "Treinos guiados",
  "Real progress statistics": "Estatísticas reais de progresso"
};

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
  const tr = useGettext(COMMERCE_PT);
  const { locale } = useI18n();
  if (state.status === "owned") {
    return (
      <aside className={styles.purchase}>
        <div className={styles.purchaseCard}>
          <p className={styles.purchaseTitle}>
            <CheckCircle2 size={16} aria-hidden="true" />
            {tr("You have access")}
          </p>
          <p className={styles.purchaseText}>{tr("You already own this training plan.")}</p>
          <div className={styles.purchaseActions}>
            <Button
              to={`/services/my-programs/${programId}`}
              variant="primary"
              size="small"
            >
              {tr("Open program")}
            </Button>
            <Button to="/services/generic-program" variant="secondary" size="small">
              {tr("Back to Training plans")}
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
            <p className={styles.purchaseText}>{tr("Checking access…")}</p>
          </div>
        </aside>
      );

    case "guest":
      return (
        <aside className={styles.purchase}>
          <div className={styles.purchaseCard}>
            <p className={styles.purchaseTitle}>
              {formatMarketplacePrice(minorUnits, currency, "premium", locale)}
            </p>
            <p className={styles.purchaseText}>{tr("Sign in to purchase this training plan.")}</p>
            <div className={styles.purchaseActions}>
              <Button
                to="/login"
                variant="primary"
                size="small"
                icon={<LogIn size={15} />}
                iconPosition="left"
              >
                {tr("Sign in")}
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
            <p className={styles.purchaseText}>{tr("Preparing secure checkout…")}</p>
          </div>
        </aside>
      );

    case "redirecting":
      return (
        <aside className={styles.purchase}>
          <div className={styles.purchaseCard}>
            <p className={styles.purchaseText}>
              {tr("Redirecting to the secure payment page. Do not close this window…")}
            </p>
          </div>
        </aside>
      );

    case "processing":
      return (
        <aside className={styles.purchase}>
          <div className={styles.purchaseCard}>
            <p className={styles.purchaseText}>
              {tr("Confirming your payment. This may take a few seconds…")}
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
              {tr("Purchase complete")}
            </p>
            <p className={styles.purchaseText}>{tr("You now have access to this training plan.")}</p>
            <div className={styles.purchaseActions}>
              <Button
                to={`/services/my-programs/${programId}`}
                variant="primary"
                size="small"
              >
                {tr("Open program")}
              </Button>
              <Button to="/services/my-programs" variant="ghost" size="small">
                {tr("My Programs")}
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
              {tr("Payment cancelled")}
            </p>
            <p className={styles.purchaseText}>
              {tr("No payment was completed. You can try again whenever you are ready.")}
            </p>
            <div className={styles.purchaseActions}>
              <Button
                variant="primary"
                size="small"
                onClick={() => onRetryPurchase(state.purchaseId)}
              >
                {tr("Try again")}
              </Button>
              <Button to="/services/generic-program" variant="ghost" size="small">
                {tr("Back to Training plans")}
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
              {tr("Purchase unavailable")}
            </p>
            <p className={styles.purchaseText}>{tr(state.message)}</p>
            <div className={styles.purchaseActions}>
              <Button
                variant="primary"
                size="small"
                onClick={onBuy}
                icon={<RefreshCw size={15} />}
              >
                {tr("Try again")}
              </Button>
              <Button to="/services/generic-program" variant="ghost" size="small">
                {tr("Back to Training plans")}
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
                {formatMarketplacePrice(0, currency, "premium", locale)}
              </p>
              <p className={styles.purchaseText}>
                {tr("Test Mode purchase — this plan is granted instantly at no cost. No real payment is made.")}
              </p>
              <div className={styles.purchaseActions}>
                <Button
                  variant="primary"
                  size="small"
                  onClick={onBuy}
                  icon={<FlaskConical size={15} />}
                  iconPosition="left"
                >
                  {tr("Get plan in Test Mode")}
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
                {tr("Checkout unavailable")}
              </p>
              <p className={styles.purchaseText}>
                {tr("No payment provider is currently configured. Please try again later.")}
              </p>
              <div className={styles.purchaseActions}>
                <Button to="/services/generic-program" variant="ghost" size="small">
                  {tr("Back to Training plans")}
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
              {formatMarketplacePrice(minorUnits, currency, "premium", locale)}
            </p>
            <p className={styles.purchaseText}>{tr("One-time purchase. Secure checkout.")}</p>
            {paymentMethods.length > 1 ? (
              <fieldset className={styles.methodGroup}>
                <legend className={styles.methodLegend}>{tr("Payment method")}</legend>
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
                {tr("Buy now")}
              </Button>
            </div>
          </div>
        </aside>
      );
  }
};

interface FreeAccessPanelProps {
  programId: string;
  programName: string;
}

// Free plans still require an explicit activation. Until that happens the
// public catalog only supplies metadata, never the programme structure.
const FreeAccessPanel = ({ programId, programName }: FreeAccessPanelProps) => {
  const tr = useGettext(PAGE_PT, COMMERCE_PT);
  const history = useHistory();
  const [activation, setActivation] = useState<FreeProgramActivation | null>(null);
  const [checking, setChecking] = useState(true);
  const [activating, setActivating] = useState(false);
  const [guest, setGuest] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    let cancelled = false;
    fetchFreeActivations()
      .then((activations) => {
        if (!cancelled) {
          setActivation(activations.find((entry) => entry.program.id === programId) ?? null);
        }
      })
      .catch((error: unknown) => {
        if (!cancelled && error instanceof ApiError && error.status === 401) {
          setGuest(true);
        }
      })
      .finally(() => {
        if (!cancelled) setChecking(false);
      });
    return () => {
      cancelled = true;
    };
  }, [programId]);

  const handleActivate = async () => {
    setActivating(true);
    setErrorMessage("");
    try {
      await activateFreeProgram(programId);
      history.push(`/services/my-programs/${programId}`);
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.push("/login");
        return;
      }
      setErrorMessage(tr("We could not activate this plan. Please try again."));
      setActivating(false);
    }
  };

  return (
    <aside className={styles.purchase}>
      <div className={styles.purchaseCard}>
        <p className={styles.purchaseTitle}>
          <CheckCircle2 size={16} aria-hidden="true" />
          {tr("Free plan")}
        </p>
        <p className={styles.purchaseText}>
          {guest
            ? tr("Sign in to activate {name} and unlock the full training structure.", { name: programName })
            : activation
              ? tr("You have activated {name}.", { name: programName })
              : tr("Activate {name} to unlock the full training structure and workout player.", { name: programName })}
        </p>
        {errorMessage ? (
          <p className={styles.purchaseErrorText} role="alert">
            {errorMessage}
          </p>
        ) : null}
        <div className={styles.purchaseActions}>
          {guest ? (
            <Button to="/login" variant="primary" size="small" icon={<LogIn size={15} />} iconPosition="left">
              {tr("Sign in")}
            </Button>
          ) : activation ? (
            <Button to={`/services/my-programs/${programId}`} variant="primary" size="small">
              {tr("Open program")}
            </Button>
          ) : (
            <Button variant="primary" size="small" disabled={checking || activating} onClick={() => void handleActivate()}>
              {activating ? tr("Activating…") : tr("Activate free plan")}
            </Button>
          )}
          <Button to="/services/generic-program" variant="secondary" size="small">
            {tr("Back to Training plans")}
          </Button>
        </div>
      </div>
    </aside>
  );
};

export const GenericProgramDetailPage = () => {
  const tr = useGettext(PAGE_PT, COMMERCE_PT);
  const { locale } = useI18n();
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
      setErrorMessage(tr("This training plan is not available through the marketplace."));
    } finally {
      setLoading(false);
    }
  }, [programId]);

  const isFreeProgram = detail ? detail.type === FREE_PROGRAM_TYPE : false;
  const isTestModePurchase = testModeActive && !isFreeProgram;

  useEffect(() => {
    void load();
  }, [load]);

  // One view per opened program feeds the coach showcase ranking; the server
  // deduplicates repeat visits and ignores the coach's own.
  const loadedProgramId = detail?.id;
  useEffect(() => {
    if (loadedProgramId) void recordProgramView(loadedProgramId);
  }, [loadedProgramId]);

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
      ? formatMarketplacePrice(0, detail.currency, detail.type, locale)
      : formatMarketplacePrice(detail.price_minor_units, detail.currency, detail.type, locale)
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
              {tr("Back to Training plans")}
            </Button>
          </div>

          {loading ? (
            <div className={styles.state}>
              <p>{tr("Loading training plan…")}</p>
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
                  {tr("Retry")}
                </Button>
              </div>
            </div>
          ) : (
            <>
              <header className={styles.heading}>
                <h1 className={styles.title}>{detail.name}</h1>
                {detail.coach ? (
                  <Link
                    to={`/coaches/${encodeURIComponent(detail.coach.id)}`}
                    className={styles.coach}
                    aria-label={tr("See {name}'s coach profile", { name: detail.coach.name })}
                  >
                    <span className={styles.coachAvatar} aria-hidden="true">
                      {coachInitials(detail.coach.name)}
                    </span>
                    <span>
                      <span className={styles.coachLabel}>{tr("Created by")}</span>
                      <span className={styles.coachName}>{detail.coach.name}</span>
                    </span>
                  </Link>
                ) : null}
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
                      {tr("Test Mode")}
                    </span>
                  ) : null}
                  {detail.training_type ? (
                    <span className={styles.chip}>{tr(detail.training_type)}</span>
                  ) : null}
                  {detail.level ? <span className={styles.chip}>{tr(detail.level)}</span> : null}
                  {detail.duration_weeks !== null ? (
                    <span className={styles.chip}>
                      {tr(detail.duration_weeks === 1 ? "{count} week" : "{count} weeks", { count: detail.duration_weeks })}
                    </span>
                  ) : null}
                  {detail.frequency_per_week !== null ? (
                    <span className={styles.chip}>
                      {tr(detail.frequency_per_week === 1 ? "{count} day/week" : "{count} days/week", { count: detail.frequency_per_week })}
                    </span>
                  ) : null}
                </div>

                {isFreeProgram ? (
                  <FreeAccessPanel programId={programId} programName={detail.name} />
                ) : (
                  <PurchasePanel
                    isFree={false}
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
                )}
              </header>

              <section className={styles.locked} aria-labelledby="plan-contents">
                <div className={styles.lockedIcon}>
                  <Lock size={20} aria-hidden="true" />
                </div>
                <div>
                  <h2 id="plan-contents" className={styles.lockedTitle}>
                    {tr("Full training structure")}
                  </h2>
                  <p className={styles.lockedText}>
                    {tr("Weeks, training days, exercises and prescriptions unlock after activation or purchase. You will then be able to log every set, follow rest timers and track progress.")}
                  </p>
                  <ul className={styles.lockedList}>
                    <li>
                      <Dumbbell size={15} aria-hidden="true" /> {tr("Guided workout sessions")}
                    </li>
                    <li>
                      <BarChart3 size={15} aria-hidden="true" /> {tr("Real progress statistics")}
                    </li>
                  </ul>
                </div>
              </section>
            </>
          )}
        </Container>
      </section>
    </PageWrapper>
  );
};