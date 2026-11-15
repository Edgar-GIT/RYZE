import {
  Activity,
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  ClipboardList,
  Dumbbell,
  Flame,
  HeartPulse,
  RefreshCw,
  Repeat,
  Sparkles,
  Trophy,
  Zap,
  type LucideIcon
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";

import { CheckoutPanel } from "@/components/checkout_panel/checkout_panel";
import j from "@/components/journey/journey.module.css";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import ui from "@/components/public_ui/public_ui.module.css";
import { QuestionField } from "@/components/premium_level2_intake/question_field";
import { PREMIUM_LEVEL2_PT } from "@/i18n/catalogs/premium_level2.pt";
import { QUESTIONNAIRE_PT } from "@/i18n/catalogs/questionnaire.pt";
import { type Copy, useGettext, useI18n } from "@/i18n/i18n";
import { rememberAfterLogin } from "@/services/after_login";
import type { MarketplaceProgram } from "@/services/marketplace_api";
import {
  fetchPremiumLevel2Draft,
  savePremiumLevel2Draft,
  type PremiumLevel2Answers
} from "@/services/premium_journeys_api";
import { fetchPremiumLevel2Catalog, fetchPremiumLevel2Programs, type PremiumLevel2Catalog } from "@/services/premium_level_2_api";
import { ApiError } from "@utils/http_client";

type Step = "intro" | "goal" | "questions" | "pay";

const GOAL_ICONS: Record<string, LucideIcon> = {
  muscle_gain: Dumbbell,
  fat_loss: Flame,
  strength: Zap,
  recomposition: Repeat,
  athletic_performance: Trophy,
  general_health: HeartPulse
};

const COPY: Copy<{
  back: string;
  backServices: string;
  steps: string[];
  kicker: string;
  title: string;
  titleAccent: string;
  lead: string;
  panelTitle: string;
  price: string;
  priceNote: string;
  includes: string[];
  how: { title: string; text: string }[];
  learnMore: string;
  goalTitle: string;
  goalLead: string;
  goalText: Record<string, string>;
  unavailable: string;
  unavailableText: string;
  signInTitle: string;
  signInText: string;
  signIn: string;
  register: string;
  loading: string;
  error: string;
  retry: string;
  section: (current: number, total: number) => string;
  saved: string;
  saving: string;
  next: string;
  previous: string;
  finish: string;
  fix: string;
  required: string;
  payTitle: string;
  payLead: string;
  edit: string;
  category: string;
  checkoutIncludes: string[];
  done: string;
  successNote: string;
}> = {
  en: {
    back: "Back",
    backServices: "Services",
    steps: ["Service", "Goal", "Questionnaire", "Payment"],
    kicker: "RYZE Personalised",
    title: "Training and nutrition ",
    titleAccent: "made only for you.",
    lead: "The most complete RYZE service. You answer a full questionnaire about your goals, routine, experience, nutrition and health; we analyse it and a RYZE trainer prepares your training plan and your nutrition plan, fully adapted to you.",
    panelTitle: "What you get",
    price: "€19.49",
    priceNote: "one-time",
    includes: ["A full questionnaire about you", "Training plan 100% personalised", "Nutrition plan adapted to your routine", "Prepared and reviewed by a RYZE trainer"],
    how: [
      { title: "Choose your main goal", text: "What matters most to you right now." },
      { title: "Answer the questionnaire", text: "Your progress is saved; you can stop and come back." },
      { title: "Pay once", text: "Payment comes only after the questionnaire." },
      { title: "We prepare your plan", text: "You are notified when training and nutrition are ready." }
    ],
    learnMore: "Learn more",
    goalTitle: "What is your main goal?",
    goalLead: "This opens your questionnaire. Everything else is asked next.",
    goalText: {
      muscle_gain: "Gain muscle and size.",
      fat_loss: "Lose fat sustainably.",
      strength: "Get stronger in the main lifts.",
      recomposition: "Lose fat and gain muscle together.",
      athletic_performance: "Perform better in your sport.",
      general_health: "Feel better and move more."
    },
    unavailable: "Personalised is not open yet",
    unavailableText: "No Personalised offer is available right now. Please come back soon.",
    signInTitle: "Sign in to start your questionnaire",
    signInText: "Your answers are saved to your RYZE account so you can stop and continue later.",
    signIn: "Sign in",
    register: "Create a free account",
    loading: "Loading your questionnaire…",
    error: "The questionnaire could not be loaded.",
    retry: "Try again",
    section: (current, total) => `Section ${current} of ${total}`,
    saved: "Your answers are saved.",
    saving: "Saving…",
    next: "Save and continue",
    previous: "Previous",
    finish: "Finish questionnaire",
    fix: "Complete the highlighted answers to continue.",
    required: "This field is required.",
    payTitle: "Questionnaire complete.",
    payLead: "Pay once to send it. A RYZE trainer then prepares your training and nutrition; you are notified when they are ready.",
    edit: "Review my answers",
    category: "Personalised · training + nutrition",
    checkoutIncludes: ["Full questionnaire", "Personalised training plan", "Personalised nutrition plan"],
    done: "See my request",
    successNote: "We received your answers and your plan is being prepared."
  },
  pt: {
    back: "Voltar",
    backServices: "Serviços",
    steps: ["Serviço", "Objetivo", "Questionário", "Pagamento"],
    kicker: "RYZE Personalizado",
    title: "Treino e nutrição ",
    titleAccent: "feitos só para ti.",
    lead: "O serviço mais completo da RYZE. Respondes a um questionário completo sobre os teus objetivos, rotina, experiência, alimentação e saúde; analisamos tudo e um treinador RYZE prepara o teu plano de treino e o teu plano de nutrição, totalmente adaptados a ti.",
    panelTitle: "O que recebes",
    price: "19,49 €",
    priceNote: "pagamento único",
    includes: ["Questionário completo sobre ti", "Plano de treino 100% personalizado", "Plano de nutrição adaptado à tua rotina", "Preparado e revisto por um treinador RYZE"],
    how: [
      { title: "Escolhe o objetivo principal", text: "O que mais importa para ti agora." },
      { title: "Responde ao questionário", text: "O progresso fica guardado; podes parar e voltar." },
      { title: "Pagas uma vez", text: "O pagamento só aparece no fim do questionário." },
      { title: "Preparamos o teu plano", text: "És avisado quando o treino e a nutrição estiverem prontos." }
    ],
    learnMore: "Saber mais",
    goalTitle: "Qual é o teu objetivo principal?",
    goalLead: "Isto abre o teu questionário. O resto é perguntado a seguir.",
    goalText: {
      muscle_gain: "Ganhar músculo e volume.",
      fat_loss: "Perder gordura de forma sustentável.",
      strength: "Ficar mais forte nos exercícios principais.",
      recomposition: "Perder gordura e ganhar músculo ao mesmo tempo.",
      athletic_performance: "Render mais no teu desporto.",
      general_health: "Sentir-te melhor e mexer-te mais."
    },
    unavailable: "O Personalizado ainda não está disponível",
    unavailableText: "De momento não há nenhuma oferta Personalizado disponível. Volta em breve.",
    signInTitle: "Inicia sessão para começares o questionário",
    signInText: "As respostas ficam guardadas na tua conta RYZE para poderes parar e continuar depois.",
    signIn: "Iniciar sessão",
    register: "Criar conta grátis",
    loading: "A carregar o teu questionário…",
    error: "Não foi possível carregar o questionário.",
    retry: "Tentar de novo",
    section: (current, total) => `Secção ${current} de ${total}`,
    saved: "As tuas respostas estão guardadas.",
    saving: "A guardar…",
    next: "Guardar e continuar",
    previous: "Anterior",
    finish: "Terminar questionário",
    fix: "Completa as respostas assinaladas para continuar.",
    required: "Esta resposta é obrigatória.",
    payTitle: "Questionário completo.",
    payLead: "Paga uma vez para o enviares. Um treinador RYZE prepara depois o teu treino e a tua nutrição; avisamos-te quando estiverem prontos.",
    edit: "Rever respostas",
    category: "Personalizado · treino + nutrição",
    checkoutIncludes: ["Questionário completo", "Plano de treino personalizado", "Plano de nutrição personalizado"],
    done: "Ver o meu pedido",
    successNote: "Recebemos as tuas respostas e o teu plano está a ser preparado."
  }
};

type Load = { status: "idle" } | { status: "loading" } | { status: "guest" } | { status: "error" } | { status: "ready" };

// PremiumLevel2JourneyPage is the Premium Level 2 journey (owner request,
// 2026-11-14): what the service is → the main goal → the long questionnaire,
// section by section and saved on the server → payment only at the end. The
// completed purchase turns the saved answers into the request's intake.
export const PremiumLevel2JourneyPage = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const tr = useGettext(PREMIUM_LEVEL2_PT, QUESTIONNAIRE_PT);
  const history = useHistory();
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  const initialGoal = params.get("goal");
  // A specific offer (e.g. from a Home card); otherwise the first published one.
  const offerParam = params.get("offer");
  const [step, setStep] = useState<Step>(initialGoal ? "questions" : "intro");
  const [offer, setOffer] = useState<MarketplaceProgram | null | undefined>(undefined);
  const [catalog, setCatalog] = useState<PremiumLevel2Catalog | null>(null);
  const [answers, setAnswers] = useState<PremiumLevel2Answers>(initialGoal ? { primary_goal: initialGoal } : {});
  const [load, setLoad] = useState<Load>({ status: "idle" });
  const [section, setSection] = useState(0);
  const [saving, setSaving] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [missing, setMissing] = useState<string[]>([]);
  const [showErrors, setShowErrors] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetchPremiumLevel2Programs()
      .then((programs) => {
        if (!cancelled) setOffer(programs.find((program) => program.id === offerParam) ?? programs[0] ?? null);
      })
      .catch(() => {
        if (!cancelled) setOffer(null);
      });
    return () => {
      cancelled = true;
    };
  }, [offerParam]);

  // Where to come back to after signing in: the same offer (and goal).
  const returnPath = (goal?: unknown) => {
    const query = new URLSearchParams();
    if (typeof goal === "string" && goal) query.set("goal", goal);
    if (offer) query.set("offer", offer.id);
    const search = query.toString();
    return search ? `/services/premium-level-2?${search}` : "/services/premium-level-2";
  };

  useEffect(() => {
    window.scrollTo({ top: 0, behavior: "smooth" });
  }, [step, section]);

  // The public catalogue question for the main goal drives the goal cards.
  const goalQuestion = useMemo(
    () => catalog?.sections.flatMap((entry) => entry.questions).find((question) => question.key === "primary_goal") ?? null,
    [catalog]
  );

  const openQuestionnaire = useCallback(
    async (goal?: string) => {
      if (!offer) return;
      setLoad({ status: "loading" });
      try {
        const [loadedCatalog, draft] = await Promise.all([fetchPremiumLevel2Catalog(), fetchPremiumLevel2Draft(offer.id)]);
        setCatalog(loadedCatalog);
        setAnswers((current) => ({ ...draft.answers, ...current, ...(goal ? { primary_goal: goal } : {}) }));
        setErrors(draft.invalid);
        setMissing(draft.missing);
        setLoad({ status: "ready" });
      } catch (error) {
        setLoad({ status: error instanceof ApiError && error.status === 401 ? "guest" : "error" });
      }
    },
    [offer]
  );

  // Arriving from the sign-in (?goal=) or opening the goal step loads the
  // catalogue (it needs the session).
  useEffect(() => {
    if (offer && (step === "questions" || step === "goal") && load.status === "idle") {
      void openQuestionnaire(initialGoal ?? undefined);
    }
  }, [offer, step, load.status, openQuestionnaire, initialGoal]);

  const sections = catalog?.sections ?? [];
  const current = sections[section];
  const keysOf = (index: number) => sections[index]?.questions.map((question) => question.key) ?? [];
  const sectionDone = (index: number) => keysOf(index).every((key) => !missing.includes(key) && !errors[key]);
  const progress = sections.length > 0 ? Math.round((sections.filter((_, index) => sectionDone(index)).length / sections.length) * 100) : 0;

  const save = async (advance: boolean) => {
    if (!offer) return;
    setSaving(true);
    try {
      const draft = await savePremiumLevel2Draft(offer.id, answers);
      setErrors(draft.invalid);
      setMissing(draft.missing);
      const keys = keysOf(section);
      const blocked = keys.some((key) => draft.missing.includes(key) || draft.invalid[key]);
      if (!advance) return;
      if (blocked) {
        setShowErrors(true);
        return;
      }
      setShowErrors(false);
      if (section < sections.length - 1) {
        setSection(section + 1);
      } else if (draft.complete) {
        setStep("pay");
      } else {
        const firstOpen = sections.findIndex((_, index) => keysOf(index).some((key) => draft.missing.includes(key) || draft.invalid[key]));
        setSection(Math.max(0, firstOpen));
        setShowErrors(true);
      }
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) setLoad({ status: "guest" });
      else setLoad({ status: "error" });
    } finally {
      setSaving(false);
    }
  };

  const stepIndex = ["intro", "goal", "questions", "pay"].indexOf(step);
  const goBack = () => {
    if (step === "intro") history.push("/services");
    else if (step === "goal") setStep("intro");
    else if (step === "questions") {
      if (section > 0) setSection(section - 1);
      else setStep("goal");
    } else setStep("questions");
  };

  const fieldError = (key: string) => {
    if (!showErrors) return "";
    if (errors[key]) return errors[key];
    return missing.includes(key) ? text.required : "";
  };

  const guestNotice = (
    <div className={j.notice} role="alert">
      <strong>{text.signInTitle}</strong>
      <p>{text.signInText}</p>
      <div className={j.actions}>
        <Link className={j.primary} to="/login" onClick={() => rememberAfterLogin(returnPath(answers.primary_goal))}>
          {text.signIn}
        </Link>
        <Link className={j.secondary} to="/register" onClick={() => rememberAfterLogin(returnPath(answers.primary_goal))}>
          {text.register}
        </Link>
      </div>
    </div>
  );

  return (
    <PageWrapper className={ui.page}>
      <div className={j.page}>
        <div className={j.top}>
          <button type="button" className={j.back} onClick={goBack}>
            <ArrowLeft aria-hidden="true" /> {step === "intro" ? text.backServices : text.back}
          </button>
          <ol className={j.progress} aria-label={text.steps[stepIndex]}>
            {text.steps.map((label, index) => (
              <li key={label} data-state={index < stepIndex ? "done" : index === stepIndex ? "current" : "next"}>
                <span className={j.progressLabel}>{label}</span>
              </li>
            ))}
          </ol>
        </div>

        {step === "intro" ? (
          <>
            <div className={j.hero}>
              <div>
                <p className={j.kicker}>
                  <Sparkles aria-hidden="true" /> {text.kicker}
                </p>
                <h1 className={j.title}>
                  {text.title}
                  <span>{text.titleAccent}</span>
                </h1>
                <p className={j.lead}>{text.lead}</p>
                {offer === null ? (
                  <div className={j.notice}>
                    <strong>{text.unavailable}</strong>
                    <p>{text.unavailableText}</p>
                  </div>
                ) : (
                  <div className={j.actions}>
                    <button type="button" className={j.primary} disabled={!offer} onClick={() => setStep("goal")}>
                      {text.learnMore} <ArrowRight aria-hidden="true" />
                    </button>
                  </div>
                )}
              </div>
              <aside className={j.heroPanel} aria-label={text.panelTitle}>
                <h2>{text.panelTitle}</h2>
                <p className={j.heroPrice}>
                  {text.price}
                  <span>{text.priceNote}</span>
                </p>
                <ul className={j.heroList}>
                  {text.includes.map((line) => (
                    <li key={line}>
                      <CheckCircle2 aria-hidden="true" />
                      {line}
                    </li>
                  ))}
                </ul>
              </aside>
            </div>
            <ol className={j.steps}>
              {text.how.map((item) => (
                <li key={item.title}>
                  <div>
                    <strong>{item.title}</strong>
                    <span>{item.text}</span>
                  </div>
                </li>
              ))}
            </ol>
          </>
        ) : null}

        {step === "goal" ? (
          <section aria-labelledby="goal-title">
            <h1 id="goal-title" className={j.title}>
              {text.goalTitle}
            </h1>
            <p className={j.lead}>{text.goalLead}</p>
            {load.status === "guest" ? guestNotice : null}
            {load.status === "error" ? (
              <div className={`${j.notice} ${j.noticeError}`} role="alert">
                <strong>{text.error}</strong>
                <div className={j.actions}>
                  <button type="button" className={j.secondary} onClick={() => void openQuestionnaire()}>
                    <RefreshCw aria-hidden="true" /> {text.retry}
                  </button>
                </div>
              </div>
            ) : null}
            <div className={`${j.choices} ${j.three}`}>
              {(goalQuestion?.options ?? Object.keys(GOAL_ICONS).map((value) => ({ value, label: value }))).map((option) => {
                const Icon = GOAL_ICONS[option.value] ?? Activity;
                return (
                  <button
                    key={option.value}
                    type="button"
                    className={j.choice}
                    aria-pressed={answers.primary_goal === option.value}
                    disabled={load.status !== "ready"}
                    onClick={() => {
                      setAnswers((state) => ({ ...state, primary_goal: option.value }));
                      setSection(0);
                      setStep("questions");
                    }}
                  >
                    <span className={j.choiceIcon} aria-hidden="true">
                      <Icon />
                    </span>
                    <span>
                      <strong>{tr(option.label)}</strong>
                      <span>{text.goalText[option.value] ?? ""}</span>
                    </span>
                  </button>
                );
              })}
            </div>
          </section>
        ) : null}

        {step === "questions" ? (
          <section aria-labelledby="questions-title">
            <p className={j.kicker}>
              <ClipboardList aria-hidden="true" /> {current ? text.section(section + 1, sections.length) : text.kicker}
            </p>
            <h1 id="questions-title" className={j.title}>
              {current ? tr(current.title) : text.loading}
            </h1>
            {load.status === "guest" ? guestNotice : null}
            {load.status === "error" ? (
              <div className={`${j.notice} ${j.noticeError}`} role="alert">
                <strong>{text.error}</strong>
                <div className={j.actions}>
                  <button type="button" className={j.secondary} onClick={() => void openQuestionnaire()}>
                    <RefreshCw aria-hidden="true" /> {text.retry}
                  </button>
                </div>
              </div>
            ) : null}
            {load.status === "ready" && current ? (
              <div className={j.questionnaire}>
                <div className={j.bar} aria-hidden="true">
                  <span style={{ width: `${progress}%` }} />
                </div>
                <ol className={j.sectionNav}>
                  {sections.map((entry, index) => (
                    <li key={entry.key}>
                      <button
                        type="button"
                        aria-current={index === section ? "step" : undefined}
                        data-done={sectionDone(index)}
                        onClick={() => {
                          void save(false);
                          setSection(index);
                        }}
                      >
                        {index + 1}. {tr(entry.title)}
                      </button>
                    </li>
                  ))}
                </ol>
                <div className={j.sectionCard}>
                  {current.description ? <p className={j.sectionIntro}>{tr(current.description)}</p> : null}
                  <div className={j.fields}>
                    {current.questions.map((question) => (
                      <QuestionField
                        key={question.key}
                        question={question}
                        value={answers[question.key]}
                        error={fieldError(question.key)}
                        disabled={saving}
                        onChange={(value) => setAnswers((state) => ({ ...state, [question.key]: value }))}
                      />
                    ))}
                  </div>
                  {showErrors && current.questions.some((question) => fieldError(question.key)) ? (
                    <p className={j.blockError} role="alert">
                      {text.fix}
                    </p>
                  ) : null}
                  <p className={j.saved}>{saving ? text.saving : text.saved}</p>
                </div>
                <div className={j.actions}>
                  {section > 0 ? (
                    <button type="button" className={j.secondary} onClick={() => setSection(section - 1)}>
                      <ArrowLeft aria-hidden="true" /> {text.previous}
                    </button>
                  ) : null}
                  <button type="button" className={j.primary} disabled={saving} onClick={() => void save(true)}>
                    {section === sections.length - 1 ? text.finish : text.next} <ArrowRight aria-hidden="true" />
                  </button>
                </div>
              </div>
            ) : load.status === "loading" ? (
              <p className={j.lead}>{text.loading}</p>
            ) : null}
          </section>
        ) : null}

        {step === "pay" && offer ? (
          <section aria-labelledby="pay-title">
            <p className={j.kicker}>
              <CheckCircle2 aria-hidden="true" /> {text.kicker}
            </p>
            <h1 id="pay-title" className={j.title}>
              {text.payTitle}
            </h1>
            <p className={j.lead}>{text.payLead}</p>
            <div className={j.result}>
              <article className={j.planCard}>
                <div className={j.planBody}>
                  <h2>{offer.name}</h2>
                  {offer.description ? <p>{offer.description}</p> : null}
                  <ul className={j.chips}>
                    {text.includes.map((line) => (
                      <li key={line}>{line}</li>
                    ))}
                  </ul>
                  <button type="button" className={j.back} onClick={() => setStep("questions")}>
                    <ArrowLeft aria-hidden="true" /> {text.edit}
                  </button>
                </div>
              </article>
              <CheckoutPanel
                programId={offer.id}
                itemTitle={offer.name}
                itemCategory={text.category}
                priceMinor={offer.price_minor_units}
                includes={text.checkoutIncludes}
                done={{ to: "/services/my-programs", label: text.done }}
                successNote={<p>{text.successNote}</p>}
                loginReturn={returnPath()}
              />
            </div>
          </section>
        ) : null}
      </div>
    </PageWrapper>
  );
};

export default PremiumLevel2JourneyPage;
