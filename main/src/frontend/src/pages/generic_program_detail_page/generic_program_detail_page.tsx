import { ArrowLeft, ArrowRight, BarChart3, CheckCircle2, CreditCard, Dumbbell, FileText, FlaskConical, Gift, Lock, LogIn, RefreshCw, XCircle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useHistory, useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { useTestMode } from "@/components/test_mode/test_mode_context";
import { CheckoutTaxIdField } from "@/components/checkout_tax_id/checkout_tax_id_field";
import { isValidOptionalTaxId } from "@/components/checkout_tax_id/tax_id";
import { isValidTestEmail } from "@/components/test_mode/test_mode_email";
import { receiptPath } from "@/services/invoices_api";
import { TestModeEmailField } from "@/components/test_mode/test_mode_email_field";
import { useProgramCheckout, type PurchaseState } from "@/hooks/use_program_checkout";
import {
  fetchMarketplaceProgram,
  formatMarketplacePrice,
  FREE_PROGRAM_TYPE,
  type MarketplaceProgramDetail
} from "@/services/marketplace_api";
import type { PaymentMethodInfo } from "@/services/purchases_api";
import { CoachAvatar } from "@/components/coach_avatar/coach_avatar";
import { recordProgramView } from "@/services/coaches_api";
import { rememberAfterLogin } from "@/services/after_login";
import {
  fetchFreeActivations,
  type FreeProgramActivation
} from "@/services/free_programs_api";
import { COMMERCE_PT } from "@/i18n/catalogs/commerce.pt";
import { type Catalog, useGettext, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";
import { ApiError, publicFileUrl } from "@utils/http_client";

import { FreePlanOutline } from "./free_plan_outline";
import styles from "./generic_program_detail_page.module.css";
import { useBackTarget } from "@/hooks/use_back_target";

const PAGE_PT: Catalog = {
  "Created by": "Criado por",
  "Premium Level 1 package": "Pacote Premium Nível 1",
  "This package is matched to you by the Level 1 questionnaire (your goal, height, weight and age), so it is not bought from here.":
    "Este pacote é-te recomendado pelo questionário do Nível 1 (objetivo, altura, peso e idade), por isso não se compra aqui.",
  "Answer the Level 1 questionnaire": "Responder ao questionário do Nível 1",
  "See {name}'s coach profile": "Ver o perfil de treinador de {name}",
  "Free plan": "Plano gratuito",
  "Off the market": "Fora do mercado",
  "Its creator took this plan off the market for a while. People who already have it keep using it.":
    "Quem o criou tirou este plano do mercado por uns tempos. Quem já o tem continua a usá-lo.",
  "Sign in to activate {name} and unlock the full training structure.":
    "Entra na tua conta para ativar {name} e desbloquear a estrutura de treino completa.",
  "You have activated {name}.": "Já ativaste {name}.",
  "Activate {name} to unlock the full training structure and workout player.":
    "Ativa {name} para desbloquear a estrutura de treino completa e o modo de treino.",
  "We could not activate this plan. Please try again.": "Não foi possível ativar este plano. Tenta novamente.",
  "Activating…": "A ativar…",
  "Activate free plan": "Ativar plano gratuito",
  "Get free plan": "Obter plano grátis",
  "This training plan is not available through the marketplace.": "Este plano de treino não está disponível no catálogo.",
  "Loading training plan…": "A carregar o plano de treino…",
  "Full training structure": "Estrutura de treino completa",
  "Weeks, training days, exercises and prescriptions unlock after activation or purchase. You will then be able to log every set and track progress.":
    "Semanas, dias de treino, exercícios e prescrições ficam disponíveis depois da ativação ou da compra. Depois podes registar cada série e acompanhar o progresso.",
  "Guided workout sessions": "Treinos guiados",
  "Real progress statistics": "Estatísticas reais de progresso",
  Length: "Duração",
  "Training days": "Dias de treino",
  Level: "Nível",
  Where: "Onde"
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
  testEmail: string;
  onTestEmailChange: (email: string) => void;
  taxId: string;
  onTaxIdChange: (taxId: string) => void;
  archived?: boolean;
}

// ArchivedPanel replaces buying or getting a plan that is off the market.
const ArchivedPanel = () => {
  const tr = useGettext(PAGE_PT, COMMERCE_PT);
  return (
    <aside className={styles.purchase}>
      <div className={styles.purchaseCard}>
        <p className={styles.purchaseTitle}>{tr("Off the market")}</p>
        <p className={styles.purchaseText}>
          {tr("Its creator took this plan off the market for a while. People who already have it keep using it.")}
        </p>
        <div className={styles.purchaseActions}>
          <Button to="/services/generic-program" variant="secondary" size="small">
            {tr("Back to Training plans")}
          </Button>
        </div>
      </div>
    </aside>
  );
};

const PurchasePanel = ({ isFree, minorUnits, currency, programId, state, paymentMethods, selectedMethod, onSelectMethod, onBuy, onRetryPurchase, testMode = false, testEmail, onTestEmailChange, taxId, onTaxIdChange, archived = false }: PurchasePanelProps) => {
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

  if (archived && state.status !== "checking") {
    return <ArchivedPanel />;
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
              <Link
                className={joinClassNames(ui.action, ui.primary)}
                to="/login"
                onClick={() => rememberAfterLogin(`/services/generic-program/${encodeURIComponent(programId)}`)}
              >
                <span>{tr("Sign in")}</span>
                <LogIn aria-hidden="true" />
              </Link>
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
              <Button to={receiptPath(state.purchase.id)} variant="ghost" size="small" icon={<FileText size={15} />} iconPosition="left">
                {tr("Purchase receipt")}
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
            {testMode ? (
              <TestModeEmailField value={testEmail} onChange={onTestEmailChange} />
            ) : null}
            <CheckoutTaxIdField value={taxId} onChange={onTaxIdChange} />
            <div className={styles.purchaseActions}>
              <Button
                variant="primary"
                size="small"
                disabled={!isValidOptionalTaxId(taxId) || (testMode && !isValidTestEmail(testEmail))}
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
              <TestModeEmailField value={testEmail} onChange={onTestEmailChange} />
              <CheckoutTaxIdField value={taxId} onChange={onTaxIdChange} />
              <div className={styles.purchaseActions}>
                <Button
                  variant="primary"
                  size="small"
                  disabled={!isValidTestEmail(testEmail) || !isValidOptionalTaxId(taxId)}
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
            <CheckoutTaxIdField value={taxId} onChange={onTaxIdChange} />
            <div className={styles.purchaseActions}>
              <Button
                variant="primary"
                size="small"
                disabled={!isValidOptionalTaxId(taxId)}
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
  archived?: boolean;
}

// Free plans still require an explicit activation. Until that happens the
// public catalog only supplies metadata, never the programme structure.
const FreeAccessPanel = ({ programId, programName, archived = false }: FreeAccessPanelProps) => {
  const tr = useGettext(PAGE_PT, COMMERCE_PT);
  const [activation, setActivation] = useState<FreeProgramActivation | null>(null);
  const [checking, setChecking] = useState(true);
  const [guest, setGuest] = useState(false);

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

  if (archived && !checking && !activation) {
    return <ArchivedPanel />;
  }

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
        <div className={styles.purchaseActions}>
          {guest ? (
            // After signing in the visitor comes back to this plan (owner
            // request, 2026-11-28), not to their home.
            <Link
              className={joinClassNames(ui.action, ui.primary)}
              to="/login"
              onClick={() => rememberAfterLogin(`/services/generic-program/${encodeURIComponent(programId)}`)}
            >
              <span>{tr("Sign in")}</span>
              <LogIn aria-hidden="true" />
            </Link>
          ) : activation ? (
            <PublicAction to={`/services/my-programs/${programId}`} icon={<ArrowRight />}>
              {tr("Open program")}
            </PublicAction>
          ) : checking ? (
            <PublicAction disabled icon={<Gift />}>
              {tr("Get free plan")}
            </PublicAction>
          ) : (
            // The next step is the order summary of the free plan (owner
            // request, 2026-11-20), not an immediate activation.
            <PublicAction to={`/services/generic-program/${encodeURIComponent(programId)}/get`} icon={<Gift />}>
              {tr("Get free plan")}
            </PublicAction>
          )}
        </div>
      </div>
    </aside>
  );
};

export const GenericProgramDetailPage = () => {
  const back = useBackTarget();
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
  const isLevel1Package = detail?.product_type === "premium_level_1";
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
  // After buying, straight to the plan in the member's area (owner request,
  // 2026-11-28).
  const history = useHistory();
  const purchased = checkout.state.status === "success";
  useEffect(() => {
    if (purchased) history.replace(`/services/my-programs/${encodeURIComponent(programId)}`);
  }, [purchased, history, programId]);

  const priceLabel = detail
    ? isTestModePurchase
      ? formatMarketplacePrice(0, detail.currency, detail.type, locale)
      : formatMarketplacePrice(detail.price_minor_units, detail.currency, detail.type, locale)
    : "";

  return (
    <PageWrapper className={joinClassNames(ui.page, styles.page)}>
      <section className={styles.detail}>
        <Container size="wide" className={styles.container}>
          <div className={styles.back}>
            <Button to={back.to} onClick={back.onClick} variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
              {back.label}
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
              {/* Redesign (owner request, 2026-11-28): the login page's split
                  panel — the cover on the light-blue wave half, the plan and
                  how to get it on the navy half — then the plan week by week. */}
              <article className={joinClassNames(ui.split, styles.hero)} aria-labelledby="plan-title">
                <div className={joinClassNames(ui.splitPane, ui.toneLight, styles.showcase)}>
                  <figure className={styles.cover}>
                    {detail.cover_image_url ? (
                      <img src={publicFileUrl(detail.cover_image_url)} alt="" />
                    ) : (
                      <Dumbbell aria-hidden="true" />
                    )}
                    <figcaption className={joinClassNames(styles.priceTag, isFreeProgram && styles.priceTagFree)}>{priceLabel}</figcaption>
                  </figure>
                </div>

                <div className={joinClassNames(ui.splitPane, styles.info)}>
                  <h1 id="plan-title" className={styles.title}>
                    {detail.name}
                  </h1>
                  {detail.coach ? (
                    <Link
                      to={`/coaches/${encodeURIComponent(detail.coach.id)}`}
                      className={styles.coach}
                      aria-label={tr("See {name}'s coach profile", { name: detail.coach.name })}
                    >
                      <CoachAvatar className={styles.coachAvatar} name={detail.coach.name} avatarUrl={detail.coach.avatar_url} />
                      <span>
                        <span className={styles.coachLabel}>{tr("Created by")}</span>
                        <span className={styles.coachName}>{detail.coach.name}</span>
                      </span>
                    </Link>
                  ) : null}
                  {detail.description ? <p className={styles.subtitle}>{detail.description}</p> : null}
                  {isTestModePurchase ? (
                    <span className={styles.chipTestMode}>
                      <FlaskConical size={12} aria-hidden="true" />
                      {tr("Test Mode")}
                    </span>
                  ) : null}

                  <dl className={styles.specs}>
                    {detail.duration_weeks !== null ? (
                      <div>
                        <dt>{tr("Length")}</dt>
                        <dd>{tr(detail.duration_weeks === 1 ? "{count} week" : "{count} weeks", { count: detail.duration_weeks })}</dd>
                      </div>
                    ) : null}
                    {detail.frequency_per_week !== null ? (
                      <div>
                        <dt>{tr("Training days")}</dt>
                        <dd>{tr(detail.frequency_per_week === 1 ? "{count} day/week" : "{count} days/week", { count: detail.frequency_per_week })}</dd>
                      </div>
                    ) : null}
                    {detail.level ? (
                      <div>
                        <dt>{tr("Level")}</dt>
                        <dd>{tr(detail.level)}</dd>
                      </div>
                    ) : null}
                    {detail.training_type ? (
                      <div>
                        <dt>{tr("Where")}</dt>
                        <dd>{tr(detail.training_type)}</dd>
                      </div>
                    ) : null}
                  </dl>

                  {isFreeProgram ? (
                    <FreeAccessPanel programId={programId} programName={detail.name} archived={detail.archived} />
                  ) : isLevel1Package ? (
                    // A Premium Level 1 package is recommended by the Level 1
                    // questionnaire, never bought straight from here (owner,
                    // 2026-11-28).
                    <aside className={styles.purchase}>
                      <div className={styles.purchaseCard}>
                        <p className={styles.purchaseTitle}>{tr("Premium Level 1 package")}</p>
                        <p className={styles.purchaseText}>
                          {tr("This package is matched to you by the Level 1 questionnaire (your goal, height, weight and age), so it is not bought from here.")}
                        </p>
                        <div className={styles.purchaseActions}>
                          <PublicAction to="/services/premium-level-1" icon={<ArrowRight />}>
                            {tr("Answer the Level 1 questionnaire")}
                          </PublicAction>
                        </div>
                      </div>
                    </aside>
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
                      testEmail={checkout.testEmail}
                      onTestEmailChange={checkout.setTestEmail}
                      taxId={checkout.taxId}
                      onTaxIdChange={checkout.setTaxId}
                      archived={detail.archived}
                    />
                  )}
                </div>
              </article>

              {isFreeProgram ? (
                <FreePlanOutline programId={programId} />
              ) : (
                <section className={styles.locked} aria-labelledby="plan-contents">
                  <div className={styles.lockedIcon}>
                    <Lock size={20} aria-hidden="true" />
                  </div>
                  <div>
                    <h2 id="plan-contents" className={styles.lockedTitle}>
                      {tr("Full training structure")}
                    </h2>
                    <p className={styles.lockedText}>
                      {tr("Weeks, training days, exercises and prescriptions unlock after activation or purchase. You will then be able to log every set and track progress.")}
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
              )}
            </>
          )}
        </Container>
      </section>
    </PageWrapper>
  );
};