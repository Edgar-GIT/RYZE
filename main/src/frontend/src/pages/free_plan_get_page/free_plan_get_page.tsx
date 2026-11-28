import { ArrowLeft, ArrowRight, CalendarDays, CheckCircle2, CreditCard, Dumbbell, Gift, ListChecks, Loader2, LogIn, ShieldCheck, Zap } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useHistory, useParams } from "react-router-dom";

import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import ui from "@/components/public_ui/public_ui.module.css";
import { joinClassNames } from "@utils/class_names";
import { type Catalog, useGettext, useI18n } from "@/i18n/i18n";
import { rememberAfterLogin } from "@/services/after_login";
import { activateFreeProgram, fetchFreeActivations } from "@/services/free_programs_api";
import {
  fetchMarketplaceProgram,
  fetchProgramOutline,
  FREE_PROGRAM_TYPE,
  type MarketplaceProgramDetail,
  type ProgramOutline
} from "@/services/marketplace_api";
import { ApiError, publicFileUrl } from "@utils/http_client";

import styles from "./free_plan_get_page.module.css";

const PAGE_PT: Catalog = {
  "Back to the plan": "Voltar ao plano",
  "Get your free plan": "Obter o teu plano grátis",
  "One step and it is yours: no card, no payment.": "Um passo e é teu: sem cartão, sem pagamento.",
  "Order summary": "Resumo do pedido",
  "Free training plan": "Plano de treino grátis",
  Subtotal: "Subtotal",
  "What you get": "O que recebes",
  "No card and no payment": "Sem cartão e sem pagamento",
  "Saved to your RYZE account": "Fica guardado na tua conta RYZE",
  "Start training right away": "Começas a treinar logo a seguir",
  "{count} weeks": "{count} semanas",
  "{count} training days": "{count} dias de treino",
  "{count} exercises": "{count} exercícios",
  "{count} sessions per week": "{count} treinos por semana",
  Total: "Total",
  Free: "Grátis",
  "Free plan: no payment and no purchase receipt.": "Plano grátis: sem pagamento e sem comprovativo de compra.",
  "Every week and training day of the plan": "Todas as semanas e dias de treino do plano",
  "The workout player with sets and history": "O player de treino com séries e histórico",
  "Your progress statistics": "As tuas estatísticas de progresso",
  "Get free plan": "Obter plano grátis",
  "Getting your plan…": "A obter o teu plano…",
  "Sign in to get this plan": "Inicia sessão para obter este plano",
  "The plan is tied to your RYZE account.": "O plano fica associado à tua conta RYZE.",
  "Sign in": "Iniciar sessão",
  "Your plan is ready": "O teu plano está pronto",
  "{name} is now in My programs.": "{name} já está nos teus programas.",
  "You already have this plan": "Já tens este plano",
  "Open the plan": "Abrir o plano",
  "My programs": "Os meus programas",
  "Loading…": "A carregar…",
  "This plan is not available.": "Este plano não está disponível.",
  "We could not get this plan. Please try again.": "Não foi possível obter este plano. Tenta de novo."
};

type Step = "loading" | "unavailable" | "guest" | "ready" | "getting" | "owned" | "done";

// FreePlanGetPage is the step after a free plan's overview (owner request,
// 2026-11-20; redesigned 2026-11-28 as a real checkout): the plan, the order
// summary adapted to a free plan — no payment, no purchase receipt — and the
// button that adds the plan to the account (a free activation, not a purchase).
export const FreePlanGetPage = () => {
  const tr = useGettext(PAGE_PT);
  const { locale } = useI18n();
  const { programId } = useParams<{ programId: string }>();
  const [plan, setPlan] = useState<MarketplaceProgramDetail | null>(null);
  const [outline, setOutline] = useState<ProgramOutline | null>(null);
  const [step, setStep] = useState<Step>("loading");
  const [errorMessage, setErrorMessage] = useState("");
  const planPath = `/services/generic-program/${encodeURIComponent(programId)}`;
  const history = useHistory();

  const load = useCallback(async () => {
    setStep("loading");
    try {
      const detail = await fetchMarketplaceProgram(programId);
      if (detail.type !== FREE_PROGRAM_TYPE) {
        setStep("unavailable");
        return;
      }
      setPlan(detail);
      fetchProgramOutline(programId)
        .then(setOutline)
        .catch(() => setOutline(null));
    } catch {
      setStep("unavailable");
      return;
    }
    try {
      const activations = await fetchFreeActivations();
      setStep(activations.some((entry) => entry.program.id === programId) ? "owned" : "ready");
    } catch (error) {
      setStep(error instanceof ApiError && error.status === 401 ? "guest" : "ready");
    }
  }, [programId]);

  useEffect(() => {
    void load();
  }, [load]);

  const getPlan = async () => {
    setStep("getting");
    setErrorMessage("");
    try {
      await activateFreeProgram(programId);
      setStep("done");
      // Straight to the plan in the member's area (owner request, 2026-11-28).
      history.replace(`/services/my-programs/${encodeURIComponent(programId)}`);
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setStep("guest");
        return;
      }
      setErrorMessage(tr("We could not get this plan. Please try again."));
      setStep("ready");
    }
  };

  const free = new Intl.NumberFormat(locale === "pt" ? "pt-PT" : "en-IE", { style: "currency", currency: plan?.currency || "EUR" }).format(0);

  return (
    <PageWrapper className={ui.page}>
      <section className={styles.section}>
        <Container size="wide" className={styles.container}>
          <Link className={styles.back} to={planPath}>
            <ArrowLeft aria-hidden="true" /> {tr("Back to the plan")}
          </Link>

          {step === "loading" ? (
            <p className={styles.state}>
              <Loader2 className={styles.spin} aria-hidden="true" /> {tr("Loading…")}
            </p>
          ) : step === "unavailable" || !plan ? (
            <p className={styles.state}>{tr("This plan is not available.")}</p>
          ) : (
            <div className={styles.layout}>
              <header className={styles.heading}>
                <h1 className={styles.title}>{tr("Get your free plan")}</h1>
                <p className={styles.lead}>{tr("One step and it is yours: no card, no payment.")}</p>
              </header>

              {/* Redesign (owner request, 2026-11-28: "it does not even look
                  like a sales page"): a checkout in the login page's split
                  panel — the plan on the bright half, the order and the one
                  action on the navy half. */}
              <div className={joinClassNames(ui.split, styles.checkout)}>
                <div className={joinClassNames(ui.splitPane, ui.toneLight, styles.product)}>
                  <div className={styles.cover}>
                    {plan.cover_image_url ? <img src={publicFileUrl(plan.cover_image_url)} alt="" /> : <Dumbbell aria-hidden="true" />}
                  </div>
                  <div className={styles.productText}>
                    <p className={styles.productKind}>{tr("Free training plan")}</p>
                    <p className={styles.productName}>{plan.name}</p>
                    <ul className={styles.facts}>
                      {outline && outline.weeks.length > 0 ? (
                        <li>
                          <CalendarDays aria-hidden="true" /> {tr("{count} weeks", { count: outline.weeks.length })}
                        </li>
                      ) : plan.duration_weeks ? (
                        <li>
                          <CalendarDays aria-hidden="true" /> {tr("{count} weeks", { count: plan.duration_weeks })}
                        </li>
                      ) : null}
                      {plan.frequency_per_week ? (
                        <li>
                          <ListChecks aria-hidden="true" /> {tr("{count} sessions per week", { count: plan.frequency_per_week })}
                        </li>
                      ) : null}
                      {outline && outline.total_exercises > 0 ? (
                        <li>
                          <Dumbbell aria-hidden="true" /> {tr("{count} exercises", { count: outline.total_exercises })}
                        </li>
                      ) : null}
                    </ul>
                  </div>
                </div>

                <div className={joinClassNames(ui.splitPane, styles.order)}>
                  <h2 className={styles.orderTitle}>{tr("Order summary")}</h2>
                  <dl className={styles.lines}>
                    <div>
                      <dt>{plan.name}</dt>
                      <dd>{free}</dd>
                    </div>
                    <div>
                      <dt>{tr("Subtotal")}</dt>
                      <dd>{free}</dd>
                    </div>
                    <div className={styles.totalLine}>
                      <dt>{tr("Total")}</dt>
                      <dd>
                        {free}
                        <span className={styles.freeBadge}>{tr("Free")}</span>
                      </dd>
                    </div>
                  </dl>

                  <div className={styles.included}>
                    <p className={styles.includedTitle}>{tr("What you get")}</p>
                    <ul>
                      {["Every week and training day of the plan", "The workout player with sets and history", "Your progress statistics"].map((line) => (
                        <li key={line}>
                          <CheckCircle2 aria-hidden="true" /> {tr(line)}
                        </li>
                      ))}
                    </ul>
                  </div>

                  <div className={styles.action} aria-live="polite">
                    {step === "guest" ? (
                      <>
                        <p className={styles.actionTitle}>{tr("Sign in to get this plan")}</p>
                        <p className={styles.actionText}>{tr("The plan is tied to your RYZE account.")}</p>
                        <Link className={joinClassNames(ui.action, ui.primary, styles.cta)} to="/login" onClick={() => rememberAfterLogin(`${planPath}/get`)}>
                          <LogIn aria-hidden="true" /> {tr("Sign in")}
                        </Link>
                      </>
                    ) : step === "done" || step === "owned" ? (
                      <>
                        <p className={styles.actionTitle}>
                          <CheckCircle2 aria-hidden="true" /> {step === "done" ? tr("Your plan is ready") : tr("You already have this plan")}
                        </p>
                        {step === "done" ? <p className={styles.actionText}>{tr("{name} is now in My programs.", { name: plan.name })}</p> : null}
                        <Link className={joinClassNames(ui.action, ui.primary, styles.cta)} to={`/services/my-programs/${encodeURIComponent(programId)}`}>
                          {tr("Open the plan")} <ArrowRight aria-hidden="true" />
                        </Link>
                        <Link className={joinClassNames(ui.action, ui.ghostOnDark, styles.cta)} to="/services/my-programs">
                          {tr("My programs")}
                        </Link>
                      </>
                    ) : (
                      <>
                        {errorMessage ? (
                          <p className={styles.error} role="alert">
                            {errorMessage}
                          </p>
                        ) : null}
                        <button
                          type="button"
                          className={joinClassNames(ui.action, ui.primary, styles.cta)}
                          disabled={step === "getting"}
                          onClick={() => void getPlan()}
                        >
                          {step === "getting" ? <Loader2 className={styles.spin} aria-hidden="true" /> : <Gift aria-hidden="true" />}
                          {step === "getting" ? tr("Getting your plan…") : tr("Get free plan")}
                        </button>
                      </>
                    )}
                  </div>

                  <ul className={styles.trust}>
                    <li>
                      <CreditCard aria-hidden="true" /> {tr("No card and no payment")}
                    </li>
                    <li>
                      <ShieldCheck aria-hidden="true" /> {tr("Saved to your RYZE account")}
                    </li>
                    <li>
                      <Zap aria-hidden="true" /> {tr("Start training right away")}
                    </li>
                  </ul>
                  <p className={styles.note}>{tr("Free plan: no payment and no purchase receipt.")}</p>
                </div>
              </div>
            </div>
          )}
        </Container>
      </section>
    </PageWrapper>
  );
};
