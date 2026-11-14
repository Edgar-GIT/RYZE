import {
  ArrowRight,
  BookOpen,
  CalendarCheck,
  ChevronRight,
  Clock3,
  Dumbbell,
  Flame,
  LifeBuoy,
  ListChecks,
  MailWarning,
  Play,
  ReceiptText,
  ShoppingBag,
  Timer,
  Trophy,
  Weight
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import { useMember } from "@/components/app_shell/member_shell";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { Waves } from "@/components/public_ui/public_ui";
import { BarChart } from "@/components/training/charts";
import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchFreeActivations } from "@/services/free_programs_api";
import { fetchEntitlements, fetchProgramAccess, type ProgramAccessDetail, type ProgramAccessWorkout } from "@/services/purchases_api";
import { fetchClientStatistics, type ClientStatistics } from "@/services/statistics_api";
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
  support: string;
  supportText: string;
  loadError: string;
}

const COPY: Copy<DashboardText> = {
  en: {
    hello: "Hello",
    title: "Your training, at a glance.",
    streak: (weeks) => (weeks === 1 ? "1 week in a row" : `${weeks} weeks in a row`),
    noStreak: "Start your streak this week",
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
    noPlanTitle: "Start with a free plan.",
    noPlanText: "Pick a training plan — free ones included — and your workouts, history and progress appear here.",
    browsePlans: "Browse training plans",
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
    support: "Support",
    supportText: "Talk to the RYZE team",
    loadError: "Some of your data could not be loaded."
  },
  pt: {
    hello: "Olá",
    title: "O teu treino, num relance.",
    streak: (weeks) => (weeks === 1 ? "1 semana seguida" : `${weeks} semanas seguidas`),
    noStreak: "Começa a tua sequência esta semana",
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
    noPlanTitle: "Começa com um plano gratuito.",
    noPlanText: "Escolhe um plano de treino — incluindo gratuitos — e os teus treinos, histórico e progresso aparecem aqui.",
    browsePlans: "Ver planos de treino",
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
    support: "Suporte",
    supportText: "Fala com a equipa RYZE",
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

interface NextWorkout {
  week: number;
  day: number;
  workout: ProgramAccessWorkout;
}

const nextWorkoutOf = (plan: ProgramAccessDetail, history: WorkoutSession[]): NextWorkout | null => {
  const done = new Set(
    history.filter((session) => session.status === "completed" && session.program_id === plan.id).map((session) => session.program_workout_id)
  );
  for (const week of [...plan.weeks].sort((a, b) => a.week_number - b.week_number)) {
    for (const workout of [...week.workouts].sort((a, b) => a.position - b.position)) {
      if (workout.exercises.length > 0 && !done.has(workout.id)) {
        return { week: week.week_number, day: workout.position, workout };
      }
    }
  }
  return null;
};

interface ProgramRef {
  id: string;
  name: string;
  frequency: number | null;
}

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

// MemberDashboardPage is the member's home: the next workout first, then this
// week, all-time totals, activity, the current plan, recent workouts and
// personal bests — everything about the member's training in one look.
export const MemberDashboardPage = () => {
  const { locale, intlLocale, formatDate, formatNumber } = useI18n();
  const text = COPY[locale];
  const { user } = useMember();
  const history = useHistory();
  const [sessions, setSessions] = useState<WorkoutSession[] | null>(null);
  const [stats, setStats] = useState<ClientStatistics | null>(null);
  const [programs, setPrograms] = useState<ProgramRef[] | null>(null);
  const [active, setActive] = useState<WorkoutSession | null>(null);
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
    void guard(
      Promise.all([fetchEntitlements().catch(() => []), fetchFreeActivations().catch(() => [])]),
      ([entitlements, free]) => {
        const refs = new Map<string, ProgramRef>();
        for (const entry of [...entitlements].sort((a, b) => b.created_at.localeCompare(a.created_at))) {
          refs.set(entry.program_id, {
            id: entry.program_id,
            name: entry.program.name,
            frequency: entry.program.frequency_per_week ?? null
          });
        }
        for (const entry of free) {
          if (!refs.has(entry.program.id)) {
            refs.set(entry.program.id, { id: entry.program.id, name: entry.program.name, frequency: entry.program.frequency_per_week });
          }
        }
        setPrograms([...refs.values()]);
      },
      () => setPrograms([])
    );
    return () => {
      cancelled = true;
    };
  }, []);

  const completed = useMemo(() => (sessions ?? []).filter((session) => session.status === "completed"), [sessions]);

  // The current plan: the workout in progress, else the last one trained, else
  // the most recent programme the member got.
  const currentProgramId = useMemo(() => {
    if (active) return active.program_id;
    const owned = new Set((programs ?? []).map((program) => program.id));
    const lastTrained = completed.find((session) => owned.has(session.program_id));
    return lastTrained?.program_id ?? programs?.[0]?.id ?? null;
  }, [active, completed, programs]);

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

  const firstName = user?.first_name || "";
  const loading = sessions === null || programs === null;
  const hasPlan = (programs?.length ?? 0) > 0;

  return (
    <PageWrapper className={styles.page}>
      <header className={styles.header}>
        <div>
          <p className={styles.date}>{formatDate(new Date().toISOString(), { weekday: "long", day: "numeric", month: "long" })}</p>
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
      {failed ? <p className={styles.error}>{text.loadError}</p> : null}

      <div className={styles.grid}>
        {/* The one thing to do next: the login page's light-blue wave panel. */}
        <section className={joinClassNames(styles.hero, loading && styles.skeleton)} aria-labelledby="dashboard-next">
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
                <CalendarCheck aria-hidden="true" /> {text.nextWorkout}
              </p>
              <h2 id="dashboard-next" className={styles.heroTitle}>
                {text.weekDay(next.week, next.day)}
              </h2>
              <p className={styles.heroMeta}>{plan.name}</p>
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
                  <Play aria-hidden="true" /> {starting ? text.starting : text.start}
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
                <Link className={styles.inkButton} to={`/services/my-programs/${encodeURIComponent(plan.id)}`}>
                  {text.openPlan}
                </Link>
                <Link className={styles.ghostInk} to="/services/generic-program">
                  {text.browsePlans} <ArrowRight aria-hidden="true" />
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
                  {text.browsePlans} <ArrowRight aria-hidden="true" />
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
            { to: "/services/generic-program", icon: ShoppingBag, title: text.plans, body: text.plansText },
            { to: "/exercises", icon: BookOpen, title: text.library, body: text.libraryText },
            { to: "/account/purchases", icon: ReceiptText, title: text.purchases, body: text.purchasesText },
            { to: "/support", icon: LifeBuoy, title: text.support, body: text.supportText }
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
