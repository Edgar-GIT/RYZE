import {
  Activity,
  ArrowRight,
  ArrowRightLeft,
  BookOpen,
  Check,
  CalendarCheck,
  ChevronRight,
  Clock3,
  Dumbbell,
  Flame,
  ListChecks,
  MailWarning,
  Play,
  ReceiptText,
  Ruler,
  ShoppingBag,
  Sparkles,
  Timer,
  Trophy,
  Weight
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import { useMember } from "@/components/app_shell/member_shell";
import { LifterCard } from "@/components/lifter/lifter";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { Waves } from "@/components/public_ui/public_ui";
import { BarChart } from "@/components/training/charts";
import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchAchievements, type Achievements } from "@/services/achievements_api";
import {
  fetchPlanFocus,
  hasTrained,
  loadLibraryPlans,
  nextWorkoutOf,
  pickFocus,
  savePlanFocus,
  sortByLastAction,
  type LibraryPlan,
  type PlanFocusChoice
} from "@/services/plan_focus";
import { fetchProgramAccess, type ProgramAccessDetail, type ProgramAccessWorkout } from "@/services/purchases_api";
import { fetchClientStatistics, type ClientStatistics } from "@/services/statistics_api";
import { ProfileAvatar } from "@/pages/account_settings_page/public_profile_card";
import { hasNickname, publicNameOf } from "@/services/profile_api";
import { profilePath } from "@/services/public_profiles_api";
import {
  currentStreakWeeks,
  estimatedOneRepMax,
  exerciseSeries,
  formatWeight,
  summarizeSession,
  trainedExercises,
  weeklyBuckets
} from "@/services/training_insights";
import {
  fetchActiveWorkoutSession,
  fetchWorkoutHistory,
  startWorkoutSession,
  type WorkoutSession
} from "@/services/workout_sessions_api";
import { joinClassNames } from "@utils/class_names";
import { apiPost } from "@utils/http_client";

import styles from "./member_dashboard_page.module.css";

interface DashboardText {
  hello: string;
  title: string;
  streak: (weeks: number) => string;
  noStreak: string;
  comebackTitle: (days: number) => string;
  comebackText: string;
  comebackAction: string;
  unverified: string;
  resend: string;
  resent: string;
  nextWorkout: string;
  inProgress: string;
  weekDay: (week: number, day: number) => string;
  exercises: (count: number) => string;
  sets: (count: number) => string;
  minutes: (count: number) => string;
  start: string;
  resume: string;
  starting: string;
  openPlan: string;
  planDone: string;
  planDoneText: string;
  noPlanTitle: string;
  noPlanText: string;
  browsePlans: string;
  firstWorkout: string;
  startFirst: string;
  switchPlan: string;
  switchTitle: string;
  inFocus: string;
  otherPlans: string;
  otherPlansLead: string;
  continueWith: (name: string) => string;
  focusThis: string;
  finished: string;
  seePremium: string;
  browseFree: string;
  startError: string;
  thisWeek: string;
  workoutsOf: (done: number, target: number | null) => string;
  minutesShort: string;
  volume: string;
  days: string[];
  totals: string;
  workouts: string;
  trainingTime: string;
  totalVolume: string;
  completedSets: string;
  activity: string;
  activityHint: string;
  workoutsPerWeek: (label: string, count: number) => string;
  plan: string;
  planEmpty: string;
  weekShort: (week: number) => string;
  completed: string;
  recent: string;
  recentEmpty: string;
  seeAll: string;
  records: string;
  recordsEmpty: string;
  bestSet: (weight: string, reps: number) => string;
  e1rm: (weight: string) => string;
  shortcuts: string;
  plans: string;
  plansText: string;
  library: string;
  libraryText: string;
  purchases: string;
  purchasesText: string;
  publicProfile: string;
  muscles: string;
  musclesText: string;
  measurements: string;
  measurementsText: string;
  loadError: string;
}

const COPY: Copy<DashboardText> = {
  en: {
    hello: "Hello",
    title: "Your training, at a glance.",
    streak: (weeks) => (weeks === 1 ? "1 week in a row" : `${weeks} weeks in a row`),
    noStreak: "Start your streak this week",
    comebackTitle: (days) => `${days} days since your last workout`,
    comebackText: "Your next session is ready. Even a short one keeps your progress going.",
    comebackAction: "Train now",
    unverified: "Your email is not confirmed yet.",
    resend: "Send confirmation email",
    resent: "Confirmation email sent.",
    nextWorkout: "Next workout",
    inProgress: "Workout in progress",
    weekDay: (week, day) => `Week ${week} · Day ${day}`,
    exercises: (count) => (count === 1 ? "1 exercise" : `${count} exercises`),
    sets: (count) => (count === 1 ? "1 set" : `${count} sets`),
    minutes: (count) => `~${count} min`,
    start: "Start workout",
    resume: "Resume workout",
    starting: "Starting…",
    openPlan: "Open plan",
    planDone: "Plan complete",
    planDoneText: "You finished every day of this plan. Repeat a day or pick your next plan.",
    noPlanTitle: "You have no plan yet.",
    noPlanText: "Pick a free plan to start today, or go Premium for a complete plan with nutrition. Your workouts, history and progress appear here.",
    browsePlans: "Browse training plans",
    firstWorkout: "Your first workout",
    startFirst: "Start your first workout",
    switchPlan: "Switch plan",
    switchTitle: "Your active plans",
    inFocus: "On your home",
    otherPlans: "Your other plans",
    otherPlansLead: "Each one keeps its own progress.",
    continueWith: (name) => `Continue with ${name}`,
    focusThis: "Show on home",
    finished: "Plan complete",
    seePremium: "Discover Premium",
    browseFree: "See free plans",
    startError: "Another workout is in progress. Finish or resume it first.",
    thisWeek: "This week",
    workoutsOf: (done, target) => (target ? `${done} of ${target} workouts` : done === 1 ? "1 workout" : `${done} workouts`),
    minutesShort: "min trained",
    volume: "kg lifted",
    days: ["M", "T", "W", "T", "F", "S", "S"],
    totals: "All time",
    workouts: "Workouts",
    trainingTime: "Training time",
    totalVolume: "Volume lifted",
    completedSets: "Sets completed",
    activity: "Activity",
    activityHint: "Workouts per week, last 8 weeks",
    workoutsPerWeek: (label, count) => `Week of ${label}: ${count} workouts`,
    plan: "Current plan",
    planEmpty: "No plan yet.",
    weekShort: (week) => `W${week}`,
    completed: "complete",
    recent: "Recent workouts",
    recentEmpty: "Your finished workouts appear here.",
    seeAll: "See all",
    records: "Personal bests",
    recordsEmpty: "Log weights in a workout to see your best lifts.",
    bestSet: (weight, reps) => `${weight} kg × ${reps}`,
    e1rm: (weight) => `est. 1RM ${weight} kg`,
    shortcuts: "Shortcuts",
    plans: "Training plans",
    plansText: "Find your next plan",
    library: "Exercise library",
    libraryText: "Every movement, explained",
    purchases: "Purchases",
    purchasesText: "Orders and receipts",
    publicProfile: "View my public profile",
    muscles: "Muscle recovery",
    musclesText: "See which muscles are ready to train",
    measurements: "Body measurements",
    measurementsText: "Log your weight and measurements",
    loadError: "Some of your data could not be loaded."
  },
  pt: {
    hello: "Olá",
    title: "O teu treino, num relance.",
    streak: (weeks) => (weeks === 1 ? "1 semana seguida" : `${weeks} semanas seguidas`),
    noStreak: "Começa a tua sequência esta semana",
    comebackTitle: (days) => `Há ${days} dias que não treinas`,
    comebackText: "O teu próximo treino está pronto. Mesmo um treino curto mantém o teu progresso.",
    comebackAction: "Treinar agora",
    unverified: "O teu email ainda não está confirmado.",
    resend: "Enviar email de confirmação",
    resent: "Email de confirmação enviado.",
    nextWorkout: "Próximo treino",
    inProgress: "Treino em curso",
    weekDay: (week, day) => `Semana ${week} · Dia ${day}`,
    exercises: (count) => (count === 1 ? "1 exercício" : `${count} exercícios`),
    sets: (count) => (count === 1 ? "1 série" : `${count} séries`),
    minutes: (count) => `~${count} min`,
    start: "Começar treino",
    resume: "Continuar treino",
    starting: "A começar…",
    openPlan: "Abrir plano",
    planDone: "Plano concluído",
    planDoneText: "Concluíste todos os dias deste plano. Repete um dia ou escolhe o teu próximo plano.",
    noPlanTitle: "Ainda não tens um plano.",
    noPlanText: "Escolhe um plano grátis para começares hoje, ou passa a Premium para um plano completo com nutrição. Os teus treinos, histórico e progresso aparecem aqui.",
    browsePlans: "Ver planos de treino",
    firstWorkout: "O teu primeiro treino",
    startFirst: "Começar primeiro treino",
    switchPlan: "Trocar plano",
    switchTitle: "Os teus planos ativos",
    inFocus: "Na tua home",
    otherPlans: "Os teus outros planos",
    otherPlansLead: "Cada um guarda o seu progresso.",
    continueWith: (name) => `Continuar com ${name}`,
    focusThis: "Pôr na home",
    finished: "Plano concluído",
    seePremium: "Conhecer o Premium",
    browseFree: "Ver planos grátis",
    startError: "Já tens outro treino em curso. Termina-o ou continua-o primeiro.",
    thisWeek: "Esta semana",
    workoutsOf: (done, target) => (target ? `${done} de ${target} treinos` : done === 1 ? "1 treino" : `${done} treinos`),
    minutesShort: "min de treino",
    volume: "kg levantados",
    days: ["S", "T", "Q", "Q", "S", "S", "D"],
    totals: "Desde sempre",
    workouts: "Treinos",
    trainingTime: "Tempo de treino",
    totalVolume: "Volume levantado",
    completedSets: "Séries concluídas",
    activity: "Atividade",
    activityHint: "Treinos por semana, últimas 8 semanas",
    workoutsPerWeek: (label, count) => `Semana de ${label}: ${count} treinos`,
    plan: "Plano atual",
    planEmpty: "Ainda sem plano.",
    weekShort: (week) => `S${week}`,
    completed: "concluído",
    recent: "Treinos recentes",
    recentEmpty: "Os teus treinos concluídos aparecem aqui.",
    seeAll: "Ver todos",
    records: "Recordes pessoais",
    recordsEmpty: "Regista pesos num treino para veres os teus melhores levantamentos.",
    bestSet: (weight, reps) => `${weight} kg × ${reps}`,
    e1rm: (weight) => `1RM est. ${weight} kg`,
    shortcuts: "Atalhos",
    plans: "Planos de treino",
    plansText: "Encontra o teu próximo plano",
    library: "Biblioteca de exercícios",
    libraryText: "Cada movimento, explicado",
    purchases: "Compras",
    purchasesText: "Encomendas e recibos",
    publicProfile: "Ver o meu perfil público",
    muscles: "Recuperação muscular",
    musclesText: "Vê que músculos estão prontos a treinar",
    measurements: "Medições corporais",
    measurementsText: "Regista o teu peso e as tuas medidas",
    loadError: "Não foi possível carregar alguns dos teus dados."
  }
};

// Rough duration: ~45 s per set plus its rest, rounded to 5 minutes (the same
// estimate as the plan screen).
const estimateMinutes = (workout: ProgramAccessWorkout): number => {
  const seconds = workout.exercises.reduce(
    (sum, exercise) => sum + exercise.sets.reduce((setSum, set) => setSum + 45 + (set.rest_seconds ?? 60), 0),
    0
  );
  return Math.max(5, Math.round(seconds / 60 / 5) * 5);
};

const EmailNotice = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const [state, setState] = useState<"idle" | "sending" | "sent">("idle");
  return (
    <div className={styles.notice} role="status">
      <MailWarning aria-hidden="true" />
      <span>{text.unverified}</span>
      {state === "sent" ? (
        <strong>{text.resent}</strong>
      ) : (
        <button
          type="button"
          disabled={state === "sending"}
          onClick={() => {
            setState("sending");
            apiPost("/me/email/verification", {})
              .then(() => setState("sent"))
              .catch(() => setState("sent"));
          }}
        >
          {text.resend}
        </button>
      )}
    </div>
  );
};

// MemberDashboardPage is the member's home: the lifter level, EXP and medals
// first, then the next workout, this week, all-time totals, activity, the
// current plan, recent workouts and personal bests — everything about the
// member's training in one look.
export const MemberDashboardPage = () => {
  const { locale, intlLocale, formatDate, formatNumber } = useI18n();
  const text = COPY[locale];
  const { user } = useMember();
  const history = useHistory();
  const [sessions, setSessions] = useState<WorkoutSession[] | null>(null);
  const [stats, setStats] = useState<ClientStatistics | null>(null);
  const [programs, setPrograms] = useState<LibraryPlan[] | null>(null);
  // The plan chosen with "Trocar plano" (undefined while loading).
  const [choice, setChoice] = useState<PlanFocusChoice | null | undefined>(undefined);
  const [others, setOthers] = useState<Record<string, ProgramAccessDetail | null>>({});
  const [switching, setSwitching] = useState(false);
  const [active, setActive] = useState<WorkoutSession | null>(null);
  const [achievements, setAchievements] = useState<Achievements | null>(null);
  const [achievementsFailed, setAchievementsFailed] = useState(false);
  const [plan, setPlan] = useState<ProgramAccessDetail | null>(null);
  const [failed, setFailed] = useState(false);
  const [starting, setStarting] = useState(false);
  const [startError, setStartError] = useState("");

  useEffect(() => {
    let cancelled = false;
    const guard = <T,>(promise: Promise<T>, apply: (value: T) => void, fallback?: () => void) =>
      promise
        .then((value) => {
          if (!cancelled) apply(value);
        })
        .catch(() => {
          if (!cancelled) {
            fallback?.();
          }
        });

    void guard(fetchWorkoutHistory(60), setSessions, () => {
      setSessions([]);
      setFailed(true);
    });
    void guard(fetchClientStatistics(), setStats);
    void guard(fetchActiveWorkoutSession(), setActive);
    void guard(fetchAchievements(), setAchievements, () => setAchievementsFailed(true));
    void guard(loadLibraryPlans(), setPrograms, () => setPrograms([]));
    void guard(fetchPlanFocus(), setChoice, () => setChoice(null));
    return () => {
      cancelled = true;
    };
  }, []);

  const completed = useMemo(() => (sessions ?? []).filter((session) => session.status === "completed"), [sessions]);
  // Days since the last completed workout: the in-app training reminder
  // (owner request, 2026-11-24) shows from 4 days.
  const idleDays = useMemo(() => {
    const last = completed.reduce((latest, session) => {
      const at = session.completed_at ? Date.parse(session.completed_at) : 0;
      return at > latest ? at : latest;
    }, 0);
    return last ? Math.floor((Date.now() - last) / 86_400_000) : 0;
  }, [completed]);

  // The plan in focus (owner request, 2026-11-22): the workout in progress,
  // else the plan of the latest action — added, trained or chosen.
  const currentProgramId = useMemo(
    () => pickFocus(programs ?? [], sessions ?? [], active, choice ?? null),
    [active, choice, programs, sessions]
  );
  const otherPlans = useMemo(
    () => sortByLastAction((programs ?? []).filter((program) => program.id !== currentProgramId), sessions ?? [], choice ?? null),
    [choice, currentProgramId, programs, sessions]
  );
  const otherIds = otherPlans
    .slice(0, 4)
    .map((program) => program.id)
    .join(",");

  // The other plans' next workouts (at most four).
  useEffect(() => {
    if (!otherIds) return;
    let cancelled = false;
    for (const id of otherIds.split(",")) {
      fetchProgramAccess(id)
        .then((detail) => {
          if (!cancelled) setOthers((current) => ({ ...current, [id]: detail }));
        })
        .catch(() => {
          if (!cancelled) setOthers((current) => ({ ...current, [id]: null }));
        });
    }
    return () => {
      cancelled = true;
    };
  }, [otherIds]);

  // Choosing a plan puts it in focus now (saved on the account).
  const choosePlan = useCallback((programId: string) => {
    setSwitching(false);
    setChoice({ program_id: programId, chosen_at: new Date().toISOString() });
    savePlanFocus(programId)
      .then(setChoice)
      .catch(() => undefined);
  }, []);

  useEffect(() => {
    if (!currentProgramId) {
      setPlan(null);
      return;
    }
    let cancelled = false;
    fetchProgramAccess(currentProgramId)
      .then((detail) => {
        if (!cancelled) setPlan(detail);
      })
      .catch(() => {
        if (!cancelled) setPlan(null);
      });
    return () => {
      cancelled = true;
    };
  }, [currentProgramId]);

  const next = useMemo(() => (plan ? nextWorkoutOf(plan, sessions ?? []) : null), [plan, sessions]);
  const currentProgram = programs?.find((program) => program.id === currentProgramId) ?? null;

  const weeks = useMemo(() => weeklyBuckets(completed, 8), [completed]);
  const streak = useMemo(() => currentStreakWeeks(completed), [completed]);
  const thisWeek = weeks[weeks.length - 1];
  const trainedDays = useMemo(() => {
    const days = new Set<number>();
    if (!thisWeek) return days;
    const start = thisWeek.weekStart.getTime();
    for (const session of completed) {
      const at = new Date(session.started_at).getTime();
      if (at >= start && at < start + 7 * 86400000) days.add(Math.floor((at - start) / 86400000));
    }
    return days;
  }, [completed, thisWeek]);
  const todayIndex = (new Date().getDay() + 6) % 7;

  const totals = useMemo(() => {
    let volume = 0;
    let seconds = 0;
    let sets = 0;
    for (const session of completed) {
      const summary = summarizeSession(session);
      volume += summary.volumeKg;
      seconds += summary.durationSeconds;
      sets += summary.completedSets;
    }
    return {
      workouts: stats?.completed_sessions ?? completed.length,
      seconds: stats?.total_active_seconds ?? seconds,
      sets: stats?.completed_sets ?? sets,
      volume
    };
  }, [completed, stats]);

  const records = useMemo(
    () =>
      trainedExercises(completed)
        .map((exercise) => {
          const series = exerciseSeries(completed, exercise.exerciseId);
          let best = { weight: 0, reps: 0 };
          let e1rm = 0;
          for (const point of series) {
            if (point.bestWeight > best.weight || (point.bestWeight === best.weight && point.bestReps > best.reps)) {
              best = { weight: point.bestWeight, reps: point.bestReps };
            }
            e1rm = Math.max(e1rm, estimatedOneRepMax(point.bestWeight, point.bestReps), point.e1rm);
          }
          return { ...exercise, best, e1rm };
        })
        .filter((record) => record.best.weight > 0)
        .sort((a, b) => b.e1rm - a.e1rm)
        .slice(0, 5),
    [completed]
  );

  const planProgress = useMemo(() => {
    if (!plan) return null;
    const done = new Set(
      completed.filter((session) => session.program_id === plan.id).map((session) => session.program_workout_id)
    );
    const perWeek = [...plan.weeks]
      .sort((a, b) => a.week_number - b.week_number)
      .map((week) => {
        const days = week.workouts.filter((workout) => workout.exercises.length > 0);
        return { week: week.week_number, total: days.length, done: days.filter((workout) => done.has(workout.id)).length };
      });
    const total = perWeek.reduce((sum, week) => sum + week.total, 0);
    const finished = perWeek.reduce((sum, week) => sum + week.done, 0);
    return { perWeek, percent: total > 0 ? Math.round((finished / total) * 100) : 0, finished, total };
  }, [plan, completed]);

  const handleStart = useCallback(async () => {
    if (active) {
      history.push(`/services/workout-sessions/${active.id}`);
      return;
    }
    if (!plan || !next) return;
    setStarting(true);
    setStartError("");
    try {
      const session = await startWorkoutSession(plan.id, next.workout.id);
      history.push(`/services/workout-sessions/${session.id}`);
    } catch {
      setStartError(text.startError);
      setStarting(false);
    }
  }, [active, plan, next, history, text.startError]);

  // The display name when set (owner request, 2026-11-20), else the first name.
  const firstName = user?.display_name?.trim() || user?.first_name || "";
  const loading = sessions === null || programs === null || choice === undefined;
  const firstTime = plan ? !hasTrained(plan.id, sessions ?? []) : false;
  const nextLabel = (detail: ProgramAccessDetail | null | undefined) => {
    if (!detail) return "";
    const upcoming = nextWorkoutOf(detail, sessions ?? []);
    if (!upcoming) return text.finished;
    return `${text.weekDay(upcoming.week, upcoming.day)}${upcoming.workout.name?.trim() ? ` · ${upcoming.workout.name.trim()}` : ""}`;
  };
  const hasPlan = (programs?.length ?? 0) > 0;

  return (
    <PageWrapper className={styles.page}>
      <header className={styles.header}>
        <div>
          <p className={styles.date}>{formatDate(new Date().toISOString(), { weekday: "long", day: "numeric", month: "long" })}</p>
          {user ? (
            <div className={styles.identity}>
              <ProfileAvatar user={user} className={styles.identityAvatar} />
              <span className={styles.identityText}>
                <strong>{publicNameOf(user)}</strong>
                {hasNickname(user) ? <span>@{user.username}</span> : null}
                <Link className={styles.identityLink} to={profilePath(user.username)}>
                  {text.publicProfile}
                </Link>
              </span>
            </div>
          ) : null}
          <h1 className={styles.title}>
            {text.hello}
            {firstName ? (
              <>
                , <span className={styles.highlight}>{firstName}</span>
              </>
            ) : null}
            .
          </h1>
          <p className={styles.lead}>{text.title}</p>
        </div>
        <span className={joinClassNames(styles.streak, streak > 0 && styles.streakOn)}>
          <Flame aria-hidden="true" />
          {streak > 0 ? text.streak(streak) : text.noStreak}
        </span>
      </header>

      {user && !user.email_verified ? <EmailNotice /> : null}
      {!loading && !active && idleDays >= 4 ? (
        <section className={styles.comeback} aria-live="polite">
          <Flame aria-hidden="true" />
          <div>
            <strong>{text.comebackTitle(idleDays)}</strong>
            <p>{text.comebackText}</p>
          </div>
          <Link className={styles.comebackAction} to="/services/next-workout">
            {text.comebackAction}
          </Link>
        </section>
      ) : null}
      {failed ? <p className={styles.error}>{text.loadError}</p> : null}

      {/* The lifter level, EXP and medals come first (owner request, 2026-11-14). */}
      {achievementsFailed ? null : <LifterCard data={achievements} totalLevels={achievements?.levels.length ?? 8} />}

      <div className={styles.grid}>
        {/* The one thing to do next: the login page's light-blue wave panel. */}
        <section className={joinClassNames(styles.hero, loading && styles.skeleton)} aria-labelledby="dashboard-next">
          {!loading && !active && (programs?.length ?? 0) > 1 ? (
            <div className={styles.switcher}>
              <button type="button" className={styles.switchButton} aria-expanded={switching} onClick={() => setSwitching((open) => !open)}>
                <ArrowRightLeft aria-hidden="true" /> {text.switchPlan}
              </button>
              {switching ? (
                <div className={styles.switchMenu} role="menu" aria-label={text.switchTitle}>
                  <p className={styles.switchTitle}>{text.switchTitle}</p>
                  {sortByLastAction(programs ?? [], sessions ?? [], choice ?? null).map((program) => {
                    const current = program.id === currentProgramId;
                    return (
                      <button
                        key={program.id}
                        type="button"
                        role="menuitemradio"
                        aria-checked={current}
                        className={joinClassNames(styles.switchItem, current && styles.switchItemCurrent)}
                        onClick={() => choosePlan(program.id)}
                      >
                        <span className={styles.switchName}>{program.name}</span>
                        <span className={styles.switchMeta}>{current ? text.inFocus : nextLabel(others[program.id])}</span>
                        {current ? <Check aria-hidden="true" /> : null}
                      </button>
                    );
                  })}
                </div>
              ) : null}
            </div>
          ) : null}
          {loading ? null : active ? (
            <>
              <p className={styles.heroKicker}>
                <Timer aria-hidden="true" /> {text.inProgress}
              </p>
              <h2 id="dashboard-next" className={styles.heroTitle}>
                {active.program_name}
              </h2>
              <p className={styles.heroMeta}>
                {text.weekDay(active.week_number, active.workout_position)} · {active.completed_sets}/{active.total_sets}
              </p>
              <div className={styles.heroActions}>
                <button type="button" className={styles.inkButton} onClick={() => void handleStart()}>
                  <Play aria-hidden="true" /> {text.resume}
                </button>
              </div>
            </>
          ) : hasPlan && plan && next ? (
            <>
              <p className={styles.heroKicker}>
                {firstTime ? <Sparkles aria-hidden="true" /> : <CalendarCheck aria-hidden="true" />} {firstTime ? text.firstWorkout : text.nextWorkout}
              </p>
              <h2 id="dashboard-next" className={styles.heroTitle}>
                {next.workout.name?.trim() || text.weekDay(next.week, next.day)}
              </h2>
              <p className={styles.heroMeta}>
                {next.workout.name?.trim() ? `${text.weekDay(next.week, next.day)} · ` : ""}
                {plan.name}
              </p>
              <ul className={styles.heroChips}>
                <li>
                  <Dumbbell aria-hidden="true" />
                  {text.exercises(next.workout.exercises.length)}
                </li>
                <li>
                  <ListChecks aria-hidden="true" />
                  {text.sets(next.workout.exercises.reduce((sum, exercise) => sum + exercise.sets.length, 0))}
                </li>
                <li>
                  <Clock3 aria-hidden="true" />
                  {text.minutes(estimateMinutes(next.workout))}
                </li>
              </ul>
              <ol className={styles.heroExercises}>
                {[...next.workout.exercises]
                  .sort((a, b) => a.position - b.position)
                  .slice(0, 4)
                  .map((exercise) => (
                    <li key={exercise.id}>
                      <span>{exercise.exercise.name}</span>
                      <span>{text.sets(exercise.sets.length)}</span>
                    </li>
                  ))}
              </ol>
              <div className={styles.heroActions}>
                <button type="button" className={styles.inkButton} onClick={() => void handleStart()} disabled={starting}>
                  <Play aria-hidden="true" /> {starting ? text.starting : firstTime ? text.startFirst : text.start}
                </button>
                <Link className={styles.ghostInk} to={`/services/my-programs/${encodeURIComponent(plan.id)}`}>
                  {text.openPlan} <ArrowRight aria-hidden="true" />
                </Link>
              </div>
              {startError ? (
                <p className={styles.heroError} role="alert">
                  {startError}
                </p>
              ) : null}
            </>
          ) : hasPlan && plan ? (
            <>
              <p className={styles.heroKicker}>
                <Trophy aria-hidden="true" /> {text.planDone}
              </p>
              <h2 id="dashboard-next" className={styles.heroTitle}>
                {plan.name}
              </h2>
              <p className={styles.heroText}>{text.planDoneText}</p>
              <div className={styles.heroActions}>
                {otherPlans[0] ? (
                  <button type="button" className={styles.inkButton} onClick={() => choosePlan(otherPlans[0].id)}>
                    {text.continueWith(otherPlans[0].name)} <ArrowRight aria-hidden="true" />
                  </button>
                ) : null}
                <Link className={otherPlans[0] ? styles.ghostInk : styles.inkButton} to={`/services/my-programs/${encodeURIComponent(plan.id)}`}>
                  {text.openPlan}
                </Link>
                {otherPlans[0] ? null : (
                  <Link className={styles.ghostInk} to="/services/generic-program">
                    {text.browsePlans} <ArrowRight aria-hidden="true" />
                  </Link>
                )}
              </div>
            </>
          ) : hasPlan && currentProgram ? (
            <>
              <p className={styles.heroKicker}>
                <CalendarCheck aria-hidden="true" /> {text.nextWorkout}
              </p>
              <h2 id="dashboard-next" className={styles.heroTitle}>
                {currentProgram.name}
              </h2>
              <div className={styles.heroActions}>
                <Link className={styles.inkButton} to={`/services/my-programs/${encodeURIComponent(currentProgram.id)}`}>
                  {text.openPlan} <ArrowRight aria-hidden="true" />
                </Link>
              </div>
            </>
          ) : (
            <>
              <p className={styles.heroKicker}>
                <ShoppingBag aria-hidden="true" /> {text.plans}
              </p>
              <h2 id="dashboard-next" className={styles.heroTitle}>
                {text.noPlanTitle}
              </h2>
              <p className={styles.heroText}>{text.noPlanText}</p>
              <div className={styles.heroActions}>
                <Link className={styles.inkButton} to="/services/generic-program">
                  {text.browseFree} <ArrowRight aria-hidden="true" />
                </Link>
                <Link className={styles.ghostInk} to="/services/premium-level-1">
                  {text.seePremium}
                </Link>
              </div>
            </>
          )}
          <Waves className={styles.heroWaves} />
        </section>

        {/* This week: a day strip and the week's numbers. */}
        <section className={joinClassNames(styles.card, styles.week)} aria-labelledby="dashboard-week">
          <h2 id="dashboard-week" className={styles.cardTitle}>
            {text.thisWeek}
          </h2>
          <p className={styles.weekCount}>
            <strong>{thisWeek?.workouts ?? 0}</strong>
            <span>{text.workoutsOf(thisWeek?.workouts ?? 0, currentProgram?.frequency ?? null)}</span>
          </p>
          <ol className={styles.days}>
            {text.days.map((day, index) => (
              <li
                key={index}
                className={joinClassNames(
                  styles.day,
                  trainedDays.has(index) && styles.dayDone,
                  index === todayIndex && styles.dayToday
                )}
              >
                <span className={styles.dayDot} aria-hidden="true" />
                <span>{day}</span>
              </li>
            ))}
          </ol>
          <dl className={styles.weekStats}>
            <div>
              <dt>{text.minutesShort}</dt>
              <dd>{formatNumber(Math.round((thisWeek?.seconds ?? 0) / 60))}</dd>
            </div>
            <div>
              <dt>{text.volume}</dt>
              <dd>{formatWeight(thisWeek?.volumeKg ?? 0, intlLocale)}</dd>
            </div>
          </dl>
        </section>

        {!loading && otherPlans.length > 0 ? (
          <section className={joinClassNames(styles.card, styles.otherPlans)} aria-labelledby="dashboard-other-plans">
            <div className={styles.otherHead}>
              <h2 id="dashboard-other-plans" className={styles.cardTitle}>
                {text.otherPlans}
              </h2>
              <p className={styles.otherLead}>{text.otherPlansLead}</p>
            </div>
            <ul className={styles.otherList}>
              {otherPlans.slice(0, 4).map((program) => {
                const detail = others[program.id];
                const upcoming = detail ? nextWorkoutOf(detail, sessions ?? []) : null;
                return (
                  <li key={program.id} className={styles.otherItem}>
                    <Link className={styles.otherName} to={`/services/my-programs/${encodeURIComponent(program.id)}`}>
                      <strong>{program.name}</strong>
                      <span>{detail === undefined ? "…" : nextLabel(detail) || text.openPlan}</span>
                    </Link>
                    {upcoming && !active ? (
                      <button
                        type="button"
                        className={styles.otherStart}
                        onClick={() => {
                          startWorkoutSession(program.id, upcoming.workout.id)
                            .then((session) => history.push(`/services/workout-sessions/${session.id}`))
                            .catch(() => setStartError(text.startError));
                        }}
                        aria-label={`${text.start}: ${program.name}`}
                      >
                        <Play aria-hidden="true" />
                      </button>
                    ) : null}
                    <button type="button" className={styles.otherFocus} onClick={() => choosePlan(program.id)}>
                      {text.focusThis}
                    </button>
                  </li>
                );
              })}
            </ul>
          </section>
        ) : null}

        {/* All-time totals. */}
        <section className={styles.totals} aria-label={text.totals}>
          <article className={styles.tile}>
            <span className={styles.tileIcon} aria-hidden="true">
              <CalendarCheck />
            </span>
            <span className={styles.tileLabel}>{text.workouts}</span>
            <strong className={styles.tileValue}>{formatNumber(totals.workouts)}</strong>
          </article>
          <article className={styles.tile}>
            <span className={styles.tileIcon} aria-hidden="true">
              <Clock3 />
            </span>
            <span className={styles.tileLabel}>{text.trainingTime}</span>
            <strong className={styles.tileValue}>
              {formatNumber(Math.floor(totals.seconds / 3600))}
              <small>h</small> {formatNumber(Math.round((totals.seconds % 3600) / 60))}
              <small>min</small>
            </strong>
          </article>
          <article className={styles.tile}>
            <span className={styles.tileIcon} aria-hidden="true">
              <Weight />
            </span>
            <span className={styles.tileLabel}>{text.totalVolume}</span>
            <strong className={styles.tileValue}>
              {formatWeight(totals.volume, intlLocale)}
              <small>kg</small>
            </strong>
          </article>
          <article className={styles.tile}>
            <span className={styles.tileIcon} aria-hidden="true">
              <ListChecks />
            </span>
            <span className={styles.tileLabel}>{text.completedSets}</span>
            <strong className={styles.tileValue}>{formatNumber(totals.sets)}</strong>
          </article>
        </section>

        {/* Activity over the last eight weeks. */}
        <section className={joinClassNames(styles.card, styles.activity)} aria-labelledby="dashboard-activity">
          <div className={styles.cardHead}>
            <h2 id="dashboard-activity" className={styles.cardTitle}>
              {text.activity}
            </h2>
            <span className={styles.cardHint}>{text.activityHint}</span>
          </div>
          <BarChart
            title={text.activity}
            points={weeks.map((bucket) => {
              const label = formatDate(bucket.weekStart.toISOString(), { day: "numeric", month: "short" });
              return { label, value: bucket.workouts, description: text.workoutsPerWeek(label, bucket.workouts) };
            })}
          />
        </section>

        {/* The current plan and how far along it is. */}
        <section className={joinClassNames(styles.card, styles.plan)} aria-labelledby="dashboard-plan">
          <div className={styles.cardHead}>
            <h2 id="dashboard-plan" className={styles.cardTitle}>
              {text.plan}
            </h2>
            {plan ? (
              <Link className={styles.cardLink} to={`/services/my-programs/${encodeURIComponent(plan.id)}`}>
                {text.openPlan} <ChevronRight aria-hidden="true" />
              </Link>
            ) : null}
          </div>
          {plan && planProgress ? (
            <>
              <div className={styles.planHead}>
                <div
                  className={styles.ring}
                  style={{ "--p": planProgress.percent } as React.CSSProperties}
                  role="img"
                  aria-label={`${planProgress.percent}% ${text.completed}`}
                >
                  <span>{planProgress.percent}%</span>
                </div>
                <div className={styles.planText}>
                  <strong>{plan.name}</strong>
                  <span>
                    {planProgress.finished}/{planProgress.total} · {text.completed}
                  </span>
                </div>
              </div>
              <ol className={styles.weeksBar}>
                {planProgress.perWeek.map((week) => (
                  <li key={week.week} title={`${text.weekShort(week.week)}: ${week.done}/${week.total}`}>
                    <span className={styles.weekTrack}>
                      <span style={{ width: `${week.total ? (week.done / week.total) * 100 : 0}%` }} />
                    </span>
                    <span className={styles.weekLabel}>{text.weekShort(week.week)}</span>
                  </li>
                ))}
              </ol>
            </>
          ) : (
            <p className={styles.empty}>{text.planEmpty}</p>
          )}
        </section>

        {/* Recent workouts. */}
        <section className={joinClassNames(styles.card, styles.recent)} aria-labelledby="dashboard-recent">
          <div className={styles.cardHead}>
            <h2 id="dashboard-recent" className={styles.cardTitle}>
              {text.recent}
            </h2>
            <Link className={styles.cardLink} to="/services/workout-history">
              {text.seeAll} <ChevronRight aria-hidden="true" />
            </Link>
          </div>
          {completed.length === 0 ? (
            <p className={styles.empty}>{text.recentEmpty}</p>
          ) : (
            <ul className={styles.sessions}>
              {completed.slice(0, 5).map((session) => {
                const summary = summarizeSession(session);
                const date = new Date(session.started_at);
                return (
                  <li key={session.id}>
                    <Link className={styles.session} to={`/services/workout-sessions/${session.id}`}>
                      <span className={styles.sessionDate}>
                        <strong>{date.getDate()}</strong>
                        <span>{formatDate(session.started_at, { month: "short" })}</span>
                      </span>
                      <span className={styles.sessionBody}>
                        <strong>{text.weekDay(session.week_number, session.workout_position)}</strong>
                        <span>{session.program_name}</span>
                      </span>
                      <span className={styles.sessionStats}>
                        <span>
                          <Clock3 aria-hidden="true" />
                          {Math.max(1, Math.round(summary.durationSeconds / 60))} min
                        </span>
                        <span>
                          <Weight aria-hidden="true" />
                          {formatWeight(summary.volumeKg, intlLocale)} kg
                        </span>
                      </span>
                      <ChevronRight className={styles.chevron} aria-hidden="true" />
                    </Link>
                  </li>
                );
              })}
            </ul>
          )}
        </section>

        {/* Personal bests. */}
        <section className={joinClassNames(styles.card, styles.records)} aria-labelledby="dashboard-records">
          <div className={styles.cardHead}>
            <h2 id="dashboard-records" className={styles.cardTitle}>
              <Trophy aria-hidden="true" /> {text.records}
            </h2>
          </div>
          {records.length === 0 ? (
            <p className={styles.empty}>{text.recordsEmpty}</p>
          ) : (
            <ol className={styles.recordList}>
              {records.map((record, index) => (
                <li key={record.exerciseId}>
                  <span className={styles.recordRank}>{index + 1}</span>
                  <span className={styles.recordName}>{record.name}</span>
                  <span className={styles.recordValue}>
                    <strong>{text.bestSet(formatWeight(record.best.weight, intlLocale), record.best.reps)}</strong>
                    <span>{text.e1rm(formatWeight(record.e1rm, intlLocale))}</span>
                  </span>
                </li>
              ))}
            </ol>
          )}
        </section>

        {/* Shortcuts. */}
        <nav className={styles.shortcuts} aria-label={text.shortcuts}>
          {[
            { to: "/services/recovery", icon: Activity, title: text.muscles, body: text.musclesText },
            { to: "/services/progress?tab=measurements", icon: Ruler, title: text.measurements, body: text.measurementsText },
            { to: "/services/generic-program", icon: ShoppingBag, title: text.plans, body: text.plansText },
            { to: "/exercises", icon: BookOpen, title: text.library, body: text.libraryText },
            { to: "/account/purchases", icon: ReceiptText, title: text.purchases, body: text.purchasesText }
          ].map(({ to, icon: Icon, title, body }) => (
            <Link key={to} className={styles.shortcut} to={to}>
              <span className={styles.tileIcon} aria-hidden="true">
                <Icon />
              </span>
              <span className={styles.shortcutText}>
                <strong>{title}</strong>
                <span>{body}</span>
              </span>
              <ArrowRight className={styles.shortcutArrow} aria-hidden="true" />
            </Link>
          ))}
        </nav>
      </div>
    </PageWrapper>
  );
};
