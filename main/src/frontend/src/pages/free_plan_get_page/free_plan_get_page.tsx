import { ArrowLeft, CalendarDays, CheckCircle2, Dumbbell, Gift, ListChecks, Loader2, LogIn } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import ui from "@/components/public_ui/public_ui.module.css";
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
import { ApiError } from "@utils/http_client";

import styles from "./free_plan_get_page.module.css";

const PAGE_PT: Catalog = {
  "Back to the plan": "Voltar ao plano",
  "Get your free plan": "Obter o teu plano grátis",
  "Your plan": "O teu plano",
  "{count} weeks": "{count} semanas",
  "{count} training days": "{count} dias de treino",
  "{count} exercises": "{count} exercícios",
  "{count} sessions per week": "{count} treinos por semana",
  Total: "Total",
  Free: "Grátis",
  "Free plan: no payment and no purchase receipt.": "Plano grátis: sem pagamento e sem comprovativo de compra.",
  "Included": "Incluído",
  "Every week and training day of the plan": "Todas as semanas e dias de treino do plano",
  "The workout player with sets, rest timer and history": "O player de treino com séries, cronómetro de descanso e histórico",
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
// 2026-11-20): an order summary like a checkout, adapted to a free plan — no
// payment, no purchase receipt — and the button that adds the plan to the
// account (a free activation, not a purchase).
export const FreePlanGetPage = () => {
  const tr = useGettext(PAGE_PT);
  const { locale } = useI18n();
  const { programId } = useParams<{ programId: string }>();
  const [plan, setPlan] = useState<MarketplaceProgramDetail | null>(null);
  const [outline, setOutline] = useState<ProgramOutline | null>(null);
  const [step, setStep] = useState<Step>("loading");
  const [errorMessage, setErrorMessage] = useState("");
  const planPath = `/services/generic-program/${encodeURIComponent(programId)}`;

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
              <div className={styles.summary}>
                <h1 className={styles.title}>{tr("Get your free plan")}</h1>
                <div className={styles.card}>
                  <p className={styles.label}>{tr("Your plan")}</p>
                  <div className={styles.item}>
                    <strong>{plan.name}</strong>
                    <b>{free}</b>
                  </div>
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
                  <p className={styles.label}>{tr("Included")}</p>
                  <ul className={styles.included}>
                    {["Every week and training day of the plan", "The workout player with sets, rest timer and history", "Your progress statistics"].map((line) => (
                      <li key={line}>
                        <CheckCircle2 aria-hidden="true" /> {tr(line)}
                      </li>
                    ))}
                  </ul>
                  <div className={styles.total}>
                    <span>{tr("Total")}</span>
                    <strong>{tr("Free")}</strong>
                  </div>
                  <p className={styles.note}>{tr("Free plan: no payment and no purchase receipt.")}</p>
                </div>
              </div>

              <div className={styles.action} aria-live="polite">
                {step === "guest" ? (
                  <>
                    <p className={styles.actionTitle}>{tr("Sign in to get this plan")}</p>
                    <p className={styles.actionText}>{tr("The plan is tied to your RYZE account.")}</p>
                    <Link className={styles.primary} to="/login" onClick={() => rememberAfterLogin(`${planPath}/get`)}>
                      <LogIn aria-hidden="true" /> {tr("Sign in")}
                    </Link>
                  </>
                ) : step === "done" || step === "owned" ? (
                  <>
                    <p className={styles.actionTitle}>
                      <CheckCircle2 aria-hidden="true" /> {step === "done" ? tr("Your plan is ready") : tr("You already have this plan")}
                    </p>
                    {step === "done" ? <p className={styles.actionText}>{tr("{name} is now in My programs.", { name: plan.name })}</p> : null}
                    <Link className={styles.primary} to={`/services/my-programs/${encodeURIComponent(programId)}`}>
                      {tr("Open the plan")}
                    </Link>
                    <Link className={styles.secondary} to="/services/my-programs">
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
                    <button type="button" className={styles.primary} disabled={step === "getting"} onClick={() => void getPlan()}>
                      {step === "getting" ? <Loader2 className={styles.spin} aria-hidden="true" /> : <Gift aria-hidden="true" />}
                      {step === "getting" ? tr("Getting your plan…") : tr("Get free plan")}
                    </button>
                  </>
                )}
              </div>
            </div>
          )}
        </Container>
      </section>
    </PageWrapper>
  );
};
