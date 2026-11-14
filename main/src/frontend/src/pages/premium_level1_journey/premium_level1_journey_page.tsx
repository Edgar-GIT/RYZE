import {
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  Dumbbell,
  Flame,
  Minus,
  Plus,
  RefreshCw,
  Sparkles,
  Trophy,
  Zap,
  type LucideIcon
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";

import { CheckoutPanel } from "@/components/checkout_panel/checkout_panel";
import j from "@/components/journey/journey.module.css";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { type Copy, useI18n } from "@/i18n/i18n";
import { rememberAfterLogin } from "@/services/after_login";
import { fetchAllPremiumLevel1Programs } from "@/services/marketplace_api";
import { fetchAccountPlanPreferences, readLocalPlanPreferences, type Sex } from "@/services/plan_preferences_api";
import { matchPremiumLevel1, type PremiumMatch, type PremiumObjective } from "@/services/premium_journeys_api";
import { ApiError, publicFileUrl } from "@utils/http_client";

type Step = "intro" | "goal" | "details" | "plan";

const OBJECTIVES: { id: PremiumObjective; icon: LucideIcon }[] = [
  { id: "gain_muscle", icon: Dumbbell },
  { id: "strength_conditioning", icon: Zap },
  { id: "lose_fat", icon: Flame },
  { id: "sport_specific", icon: Trophy }
];

interface Body {
  weight: string;
  age: string;
  height: string;
}

const LIMITS = {
  weight: { min: 30, max: 300, step: 1 },
  age: { min: 14, max: 100, step: 1 },
  height: { min: 120, max: 230, step: 1 }
} as const;

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
  objectives: Record<PremiumObjective, { title: string; text: string }>;
  soon: string;
  continue: string;
  detailsTitle: string;
  detailsLead: string;
  weight: string;
  age: string;
  height: string;
  kg: string;
  years: string;
  cm: string;
  range: (min: number, max: number) => string;
  invalid: string;
  sex: string;
  sexes: Record<Sex, string>;
  sexHint: string;
  sexMissing: string;
  find: string;
  finding: string;
  planKicker: string;
  planTitle: string;
  planLead: string;
  weeks: (n: number) => string;
  perWeek: (n: number) => string;
  nutrition: string;
  another: string;
  signInTitle: string;
  signInText: string;
  signIn: string;
  register: string;
  noPlan: string;
  noPlanText: string;
  allOwned: string;
  allOwnedText: string;
  error: string;
  retry: string;
  checkoutIncludes: (weeks: number | null) => string[];
  category: string;
  open: string;
  successNote: string;
}> = {
  en: {
    back: "Back",
    backServices: "Services",
    steps: ["Service", "Goal", "About you", "Your plan"],
    kicker: "RYZE Premium",
    title: "Training and nutrition ",
    titleAccent: "for your goal.",
    lead: "Tell us what you want to achieve and a little about you. RYZE picks a complete plan for that goal from our catalogue — training and nutrition — and it is yours right after the payment.",
    panelTitle: "What you get",
    price: "€14.99",
    priceNote: "one-time",
    includes: ["A training plan for your goal", "A nutrition plan to go with it", "Available right after payment", "Never a plan you already have"],
    how: [
      { title: "Choose your goal", text: "Muscle, strength, fat loss or training for a sport." },
      { title: "Tell us about you", text: "Your sex, weight, age and height." },
      { title: "RYZE picks your plan", text: "One of our plans for that goal, different from the ones you own." },
      { title: "Pay once and train", text: "Training and nutrition appear in your account." }
    ],
    learnMore: "Learn more",
    goalTitle: "What do you want to achieve?",
    goalLead: "Choose one goal. You can change it later.",
    objectives: {
      gain_muscle: { title: "Build muscle", text: "Size and strength with progressive training." },
      strength_conditioning: { title: "Strength", text: "Get stronger and fitter with structured work." },
      lose_fat: { title: "Lose weight", text: "Lose fat while keeping your training quality." },
      sport_specific: { title: "Sport-specific", text: "Train for your sport, e.g. football." }
    },
    soon: "Coming soon",
    continue: "Continue",
    detailsTitle: "A little about you",
    detailsLead: "We use this to choose your plan and, later, to adapt your nutrition.",
    weight: "Weight",
    age: "Age",
    height: "Height",
    kg: "kg",
    years: "years",
    cm: "cm",
    range: (min, max) => `Between ${min} and ${max}`,
    invalid: "Check the highlighted values.",
    sex: "Sex",
    sexes: { female: "Woman", male: "Man" },
    sexHint: "RYZE picks among the plans for everyone and the ones made for you.",
    sexMissing: "Choose woman or man.",
    find: "Find my plan",
    finding: "Choosing your plan…",
    planKicker: "Your plan",
    planTitle: "We found your plan.",
    planLead: "Chosen for your goal from the RYZE catalogue. Pay once and it is in your account straight away.",
    weeks: (n) => `${n} ${n === 1 ? "week" : "weeks"}`,
    perWeek: (n) => `${n}×/week`,
    nutrition: "Nutrition included",
    another: "Choose another goal",
    signInTitle: "Sign in to get your plan",
    signInText: "Your plan is tied to your RYZE account. You come straight back here after signing in.",
    signIn: "Sign in",
    register: "Create a free account",
    noPlan: "No plan for this goal yet",
    noPlanText: "We are still preparing plans for this goal. Choose another goal or come back soon.",
    allOwned: "You already have every plan for this goal",
    allOwnedText: "RYZE never gives you the same plan twice. Choose another goal.",
    error: "We could not choose your plan right now.",
    retry: "Try again",
    checkoutIncludes: (weeks) => [weeks ? `Training plan · ${weeks} ${weeks === 1 ? "week" : "weeks"}` : "Training plan", "Nutrition plan", "Available right after payment"],
    category: "Premium · training + nutrition",
    open: "Open my plan",
    successNote: "Your training plan and its nutrition are now in Train."
  },
  pt: {
    back: "Voltar",
    backServices: "Serviços",
    steps: ["Serviço", "Objetivo", "Sobre ti", "O teu plano"],
    kicker: "RYZE Premium",
    title: "Treino e nutrição ",
    titleAccent: "para o teu objetivo.",
    lead: "Diz-nos o que queres alcançar e um pouco sobre ti. A RYZE escolhe um plano completo para esse objetivo do nosso catálogo — treino e nutrição — e fica teu logo após o pagamento.",
    panelTitle: "O que recebes",
    price: "14,99 €",
    priceNote: "pagamento único",
    includes: ["Plano de treino para o teu objetivo", "Plano de nutrição a acompanhar", "Disponível logo após o pagamento", "Nunca um plano que já tenhas"],
    how: [
      { title: "Escolhe o objetivo", text: "Músculo, força, perder peso ou treino para um desporto." },
      { title: "Fala-nos de ti", text: "O teu sexo, peso, idade e altura." },
      { title: "A RYZE escolhe o plano", text: "Um dos nossos planos para esse objetivo, diferente dos que já tens." },
      { title: "Pagas uma vez e treinas", text: "Treino e nutrição aparecem na tua conta." }
    ],
    learnMore: "Saber mais",
    goalTitle: "O que queres alcançar?",
    goalLead: "Escolhe um objetivo. Podes mudar depois.",
    objectives: {
      gain_muscle: { title: "Ganhar músculo", text: "Volume e força com treino progressivo." },
      strength_conditioning: { title: "Força", text: "Fica mais forte e em forma com treino estruturado." },
      lose_fat: { title: "Perder peso", text: "Perde gordura sem perder qualidade no treino." },
      sport_specific: { title: "Treino específico", text: "Treina para o teu desporto, ex.: futebol." }
    },
    soon: "Em breve",
    continue: "Continuar",
    detailsTitle: "Um pouco sobre ti",
    detailsLead: "Usamos isto para escolher o teu plano e, mais tarde, para adaptar a tua nutrição.",
    weight: "Peso",
    age: "Idade",
    height: "Altura",
    kg: "kg",
    years: "anos",
    cm: "cm",
    range: (min, max) => `Entre ${min} e ${max}`,
    invalid: "Verifica os valores assinalados.",
    sex: "Sexo",
    sexes: { female: "Mulher", male: "Homem" },
    sexHint: "A RYZE escolhe entre os planos para todos e os pensados para ti.",
    sexMissing: "Escolhe mulher ou homem.",
    find: "Encontrar o meu plano",
    finding: "A escolher o teu plano…",
    planKicker: "O teu plano",
    planTitle: "Encontrámos o teu plano.",
    planLead: "Escolhido para o teu objetivo do catálogo RYZE. Pagas uma vez e fica logo na tua conta.",
    weeks: (n) => `${n} ${n === 1 ? "semana" : "semanas"}`,
    perWeek: (n) => `${n}×/semana`,
    nutrition: "Nutrição incluída",
    another: "Escolher outro objetivo",
    signInTitle: "Inicia sessão para receberes o teu plano",
    signInText: "O plano fica associado à tua conta RYZE. Voltas logo a esta página depois de iniciares sessão.",
    signIn: "Iniciar sessão",
    register: "Criar conta grátis",
    noPlan: "Ainda não há planos para este objetivo",
    noPlanText: "Ainda estamos a preparar planos para este objetivo. Escolhe outro objetivo ou volta em breve.",
    allOwned: "Já tens todos os planos deste objetivo",
    allOwnedText: "A RYZE nunca te dá o mesmo plano duas vezes. Escolhe outro objetivo.",
    error: "Não foi possível escolher o teu plano agora.",
    retry: "Tentar de novo",
    checkoutIncludes: (weeks) => [weeks ? `Plano de treino · ${weeks} ${weeks === 1 ? "semana" : "semanas"}` : "Plano de treino", "Plano de nutrição", "Disponível logo após o pagamento"],
    category: "Premium · treino + nutrição",
    open: "Abrir o meu plano",
    successNote: "O teu plano de treino e a nutrição já estão em Treinar."
  }
};

const isObjective = (value: string | null): value is PremiumObjective =>
  OBJECTIVES.some((objective) => objective.id === value);

type Problem = "guest" | "noPlan" | "allOwned" | "error" | null;

// PremiumLevel1JourneyPage is the Premium Level 1 journey (owner request,
// 2026-11-14): what the service is → the goal → sex, weight, age and height → the
// plan RYZE picked at random among the matching ones (never one the client
// owns) → payment. The pick is made and recorded by the backend.
export const PremiumLevel1JourneyPage = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const history = useHistory();
  const location = useLocation();
  const initialGoal = new URLSearchParams(location.search).get("goal");
  const [step, setStep] = useState<Step>(isObjective(initialGoal) ? "details" : "intro");
  const [goal, setGoal] = useState<PremiumObjective | null>(isObjective(initialGoal) ? initialGoal : null);
  const [body, setBody] = useState<Body>({ weight: "75", age: "30", height: "175" });
  // Pre-filled from the account (or this browser) when already answered.
  const [sex, setSex] = useState<Sex | null>(() => readLocalPlanPreferences()?.sex ?? null);
  const [submitted, setSubmitted] = useState(false);
  const [matching, setMatching] = useState(false);
  const [problem, setProblem] = useState<Problem>(null);
  const [match, setMatch] = useState<PremiumMatch | null>(null);
  const [available, setAvailable] = useState<Set<string> | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetchAllPremiumLevel1Programs()
      .then((programs) => {
        if (!cancelled) setAvailable(new Set(programs.programs.map((program) => program.objective ?? "").filter(Boolean)));
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    fetchAccountPlanPreferences()
      .then((stored) => {
        if (!cancelled && stored?.sex) setSex((current) => current ?? stored.sex);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    window.scrollTo({ top: 0, behavior: "smooth" });
  }, [step]);

  const values = useMemo(
    () => ({ weight: Number(body.weight), age: Number(body.age), height: Number(body.height) }),
    [body]
  );
  const invalid = (key: keyof Body) => {
    const value = values[key];
    const limit = LIMITS[key];
    return !Number.isFinite(value) || body[key].trim() === "" || value < limit.min || value > limit.max;
  };
  const anyInvalid = invalid("weight") || invalid("age") || invalid("height");

  const nudge = (key: keyof Body, direction: 1 | -1) => {
    const limit = LIMITS[key];
    const current = Number(body[key]) || limit.min;
    const next = Math.min(limit.max, Math.max(limit.min, Math.round(current + direction * limit.step)));
    setBody((state) => ({ ...state, [key]: String(next) }));
  };

  const find = async () => {
    setSubmitted(true);
    if (!goal || !sex || anyInvalid) return;
    setMatching(true);
    setProblem(null);
    try {
      const result = await matchPremiumLevel1({
        objective: goal,
        sex,
        weight_kg: values.weight,
        height_cm: values.height,
        age: Math.round(values.age)
      });
      setMatch(result);
      setStep("plan");
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setProblem("guest");
      } else if (error instanceof ApiError && error.code === "NO_MATCHING_PLAN") {
        setProblem("noPlan");
      } else if (error instanceof ApiError && error.code === "ALL_PLANS_OWNED") {
        setProblem("allOwned");
      } else {
        setProblem("error");
      }
    } finally {
      setMatching(false);
    }
  };

  const stepIndex = ["intro", "goal", "details", "plan"].indexOf(step);
  const goBack = () => {
    if (step === "goal") setStep("intro");
    else if (step === "details") setStep("goal");
    else if (step === "plan") setStep("details");
    else history.push("/services");
  };

  const program = match?.program;
  const cover = publicFileUrl(program?.cover_image_url);

  return (
    <PageWrapper>
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
                <div className={j.actions}>
                  <button type="button" className={j.primary} onClick={() => setStep("goal")}>
                    {text.learnMore} <ArrowRight aria-hidden="true" />
                  </button>
                </div>
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
            <div className={j.choices}>
              {OBJECTIVES.map(({ id, icon: Icon }) => {
                const soon = available !== null && !available.has(id);
                return (
                  <button
                    key={id}
                    type="button"
                    className={j.choice}
                    aria-pressed={goal === id}
                    onClick={() => {
                      setGoal(id);
                      setProblem(null);
                      setStep("details");
                    }}
                  >
                    {soon ? <em className={j.soon}>{text.soon}</em> : null}
                    <span className={j.choiceIcon} aria-hidden="true">
                      <Icon />
                    </span>
                    <span>
                      <strong>{text.objectives[id].title}</strong>
                      <span>{text.objectives[id].text}</span>
                    </span>
                  </button>
                );
              })}
            </div>
          </section>
        ) : null}

        {step === "details" && goal ? (
          <section aria-labelledby="details-title">
            <p className={j.kicker}>{text.objectives[goal].title}</p>
            <h1 id="details-title" className={j.title}>
              {text.detailsTitle}
            </h1>
            <p className={j.lead}>{text.detailsLead}</p>
            <div className={j.blocks}>
              <div className={j.block} role="group" aria-labelledby="body-sex" aria-describedby="body-sex-hint">
                <span id="body-sex" className={j.blockLabel}>
                  {text.sex}
                </span>
                <div className={j.segment}>
                  {(["female", "male"] as const).map((option) => (
                    <button key={option} type="button" aria-pressed={sex === option} onClick={() => setSex(option)}>
                      {text.sexes[option]}
                    </button>
                  ))}
                </div>
                <p id="body-sex-hint" className={submitted && !sex ? j.blockError : j.blockHint}>
                  {submitted && !sex ? text.sexMissing : text.sexHint}
                </p>
              </div>
              {(["weight", "age", "height"] as const).map((key) => {
                const unit = key === "weight" ? text.kg : key === "age" ? text.years : text.cm;
                const label = key === "weight" ? text.weight : key === "age" ? text.age : text.height;
                const bad = submitted && invalid(key);
                return (
                  <div key={key} className={j.block}>
                    <label htmlFor={`body-${key}`}>{label}</label>
                    <div className={j.blockRow}>
                      <input
                        id={`body-${key}`}
                        type="number"
                        inputMode={key === "weight" ? "decimal" : "numeric"}
                        min={LIMITS[key].min}
                        max={LIMITS[key].max}
                        value={body[key]}
                        aria-invalid={bad}
                        aria-describedby={`body-${key}-hint`}
                        onChange={(event) => setBody((state) => ({ ...state, [key]: event.target.value }))}
                      />
                      <em>{unit}</em>
                      <span className={j.stepButtons}>
                        <button type="button" aria-label={`− ${label}`} onClick={() => nudge(key, -1)}>
                          <Minus aria-hidden="true" />
                        </button>
                        <button type="button" aria-label={`+ ${label}`} onClick={() => nudge(key, 1)}>
                          <Plus aria-hidden="true" />
                        </button>
                      </span>
                    </div>
                    <p id={`body-${key}-hint`} className={bad ? j.blockError : j.blockHint}>
                      {text.range(LIMITS[key].min, LIMITS[key].max)}
                    </p>
                  </div>
                );
              })}
            </div>

            {submitted && anyInvalid ? <p className={j.blockError}>{text.invalid}</p> : null}

            {problem ? (
              <div className={`${j.notice} ${problem === "error" ? j.noticeError : ""}`} role="alert">
                <strong>
                  {problem === "guest"
                    ? text.signInTitle
                    : problem === "noPlan"
                      ? text.noPlan
                      : problem === "allOwned"
                        ? text.allOwned
                        : text.error}
                </strong>
                <p>
                  {problem === "guest"
                    ? text.signInText
                    : problem === "noPlan"
                      ? text.noPlanText
                      : problem === "allOwned"
                        ? text.allOwnedText
                        : ""}
                </p>
                <div className={j.actions}>
                  {problem === "guest" ? (
                    <>
                      <Link className={j.primary} to="/login" onClick={() => rememberAfterLogin(`/services/premium-level-1?goal=${goal}`)}>
                        {text.signIn}
                      </Link>
                      <Link className={j.secondary} to="/register" onClick={() => rememberAfterLogin(`/services/premium-level-1?goal=${goal}`)}>
                        {text.register}
                      </Link>
                    </>
                  ) : problem === "error" ? (
                    <button type="button" className={j.secondary} onClick={() => void find()}>
                      <RefreshCw aria-hidden="true" /> {text.retry}
                    </button>
                  ) : (
                    <button type="button" className={j.secondary} onClick={() => setStep("goal")}>
                      {text.another}
                    </button>
                  )}
                </div>
              </div>
            ) : null}

            <div className={j.actions}>
              <button type="button" className={j.primary} disabled={matching} onClick={() => void find()}>
                {matching ? text.finding : text.find} <ArrowRight aria-hidden="true" />
              </button>
            </div>
          </section>
        ) : null}

        {step === "plan" && program ? (
          <section aria-labelledby="plan-title">
            <p className={j.kicker}>
              <Sparkles aria-hidden="true" /> {text.planKicker}
            </p>
            <h1 id="plan-title" className={j.title}>
              {text.planTitle}
            </h1>
            <p className={j.lead}>{text.planLead}</p>
            <div className={j.result}>
              <article className={j.planCard}>
                <div className={j.planCover}>
                  {cover ? <img src={cover} alt="" /> : <Dumbbell aria-hidden="true" />}
                  {goal ? <span className={j.planTag}>{text.objectives[goal].title}</span> : null}
                </div>
                <div className={j.planBody}>
                  <h2>{program.name}</h2>
                  {program.description ? <p>{program.description}</p> : null}
                  <ul className={j.chips}>
                    {program.duration_weeks ? <li>{text.weeks(program.duration_weeks)}</li> : null}
                    {program.frequency_per_week ? <li>{text.perWeek(program.frequency_per_week)}</li> : null}
                    {program.level ? <li>{program.level}</li> : null}
                    <li>{text.nutrition}</li>
                  </ul>
                  <button type="button" className={j.back} onClick={() => setStep("goal")}>
                    <ArrowLeft aria-hidden="true" /> {text.another}
                  </button>
                </div>
              </article>
              <CheckoutPanel
                programId={program.id}
                itemTitle={program.name}
                itemCategory={text.category}
                priceMinor={program.price_minor_units}
                includes={text.checkoutIncludes(program.duration_weeks)}
                done={{ to: `/services/my-programs/${program.id}`, label: text.open }}
                successNote={<p>{text.successNote}</p>}
                loginReturn={`/services/premium-level-1?goal=${goal ?? ""}`}
              />
            </div>
          </section>
        ) : null}
      </div>
    </PageWrapper>
  );
};

export default PremiumLevel1JourneyPage;
