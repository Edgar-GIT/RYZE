import { ArrowLeft, Info, Pause, Play, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useHistory, useParams } from "react-router-dom";

import { WorkoutReport } from "@/components/training/workout_report";
import { useI18n } from "@/i18n/i18n";
import { formatClock, previousSets } from "@/services/training_insights";
import {
  abandonWorkoutSession,
  completeWorkoutSession,
  fetchWorkoutHistory,
  fetchWorkoutSession,
  pauseWorkoutSession,
  resumeWorkoutSession,
  submitWorkoutSet,
  type WorkoutSession,
  type WorkoutSessionExercise,
  type WorkoutSessionSet,
  type WorkoutSetInput
} from "@/services/workout_sessions_api";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";

import { ExerciseInfoSheet } from "./exercise_info_sheet";
import { RestBar } from "./rest_bar";
import { SetRow } from "./set_row";
import styles from "./workout_session_page.module.css";

type LoadState = "loading" | "ready" | "error" | "missing";

// A short line describing what the trainer prescribed for an exercise.
const usePrescription = () => {
  const { t } = useI18n();
  return (exercise: WorkoutSessionExercise): string => {
    const sets = exercise.sets;
    const reps = sets.map((set) => set.prescribed_reps).filter((value): value is number => value != null);
    const min = reps.length ? Math.min(...reps) : null;
    const max = reps.length ? Math.max(...reps) : null;
    const repsText = min == null ? null : min === max ? String(min) : `${min}–${max}`;
    const first = sets[0];
    return [
      repsText ? t("training.plan.setsReps", { sets: sets.length, reps: repsText }) : t("training.plan.sets1", { sets: sets.length }),
      first?.prescribed_rest_seconds != null ? t("training.player.rest", { seconds: first.prescribed_rest_seconds }) : null,
      first?.prescribed_rir != null ? t("training.player.rir", { value: first.prescribed_rir }) : null,
      first?.prescribed_rpe != null ? t("training.player.rpe", { value: first.prescribed_rpe }) : null,
      first?.prescribed_tempo ? t("training.player.tempo", { value: first.prescribed_tempo }) : null
    ]
      .filter(Boolean)
      .join(" · ");
  };
};

// The set whose rest timer should show: the latest completed set whose rest
// has not ended more than a few seconds ago.
const activeRestSet = (session: WorkoutSession, now: number, dismissed: string | null): WorkoutSessionSet | null => {
  let latest: WorkoutSessionSet | null = null;
  for (const set of session.exercises.flatMap((exercise) => exercise.sets)) {
    if (set.status !== "completed" || !set.rest_ends_at || !set.completed_at || set.id === dismissed) continue;
    if (new Date(set.rest_ends_at).getTime() < now - 5000) continue;
    if (!latest || (latest.completed_at && set.completed_at > latest.completed_at)) latest = set;
  }
  return latest;
};

// WorkoutSessionPage is the workout player: a full-screen view of today's
// exercises where every set is logged as it is done, with a running clock,
// rest timers, the previous performance and an end-of-workout summary.
export const WorkoutSessionPage = () => {
  const { t } = useI18n();
  const prescription = usePrescription();
  const { sessionId } = useParams<{ sessionId: string }>();
  const history = useHistory();
  const [session, setSession] = useState<WorkoutSession | null>(null);
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [past, setPast] = useState<WorkoutSession[]>([]);
  const [view, setView] = useState<"workout" | "review">("workout");
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState("");
  const [clockBase, setClockBase] = useState({ elapsed: 0, at: Date.now() });
  const [now, setNow] = useState(() => Date.now());
  const [dismissedRest, setDismissedRest] = useState<string | null>(null);
  const [infoFor, setInfoFor] = useState<WorkoutSessionExercise | null>(null);
  const topRef = useRef<HTMLDivElement>(null);

  const apply = useCallback((next: WorkoutSession) => {
    setSession(next);
    setClockBase({ elapsed: next.elapsed_seconds, at: Date.now() });
  }, []);

  const load = useCallback(async () => {
    setLoadState("loading");
    try {
      apply(await fetchWorkoutSession(sessionId));
      setLoadState("ready");
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setLoadState(error instanceof ApiError && error.status === 404 ? "missing" : "error");
    }
  }, [apply, history, sessionId]);

  useEffect(() => {
    void load();
  }, [load]);

  // Earlier sessions give the "previous" column and the personal records.
  useEffect(() => {
    fetchWorkoutHistory(100)
      .then(setPast)
      .catch(() => setPast([]));
  }, []);

  const running = session?.status === "in_progress";
  const finished = session?.status === "completed" || session?.status === "abandoned";

  useEffect(() => {
    if (!running && !session) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [running, session]);

  // Keep the screen on while training, where the browser allows it.
  useEffect(() => {
    if (!running || !("wakeLock" in navigator)) return;
    let lock: { release: () => Promise<void> } | null = null;
    const nav = navigator as Navigator & { wakeLock: { request: (type: "screen") => Promise<{ release: () => Promise<void> }> } };
    nav.wakeLock
      .request("screen")
      .then((sentinel) => {
        lock = sentinel;
      })
      .catch(() => undefined);
    return () => {
      void lock?.release().catch(() => undefined);
    };
  }, [running]);

  const clock = clockBase.elapsed + (running ? Math.max(0, (now - clockBase.at) / 1000) : 0);
  const previous = useMemo(() => (session ? previousSets(past, session) : new Map<string, WorkoutSessionSet[]>()), [past, session]);
  const restSet = session && running ? activeRestSet(session, now, dismissedRest) : null;

  const saveSet = useCallback(
    async (setId: string, input: WorkoutSetInput) => {
      const next = await submitWorkoutSet(sessionId, setId, input);
      apply(next);
      if (input.status === "completed") setDismissedRest(null);
    },
    [apply, sessionId]
  );

  const runAction = async (action: () => Promise<WorkoutSession>) => {
    setBusy(true);
    setActionError("");
    try {
      apply(await action());
    } catch {
      setActionError(t("training.player.actionError"));
    } finally {
      setBusy(false);
    }
  };

  const openReview = () => {
    setView("review");
    window.scrollTo({ top: 0 });
  };

  if (loadState === "loading" && !session) {
    return (
      <main className={styles.page}>
        <p className={styles.muted}>{t("common.loading")}</p>
      </main>
    );
  }

  if (!session) {
    return (
      <main className={styles.page}>
        <div className={styles.stateCard}>
          <p>{loadState === "missing" ? t("training.player.unavailable") : t("training.player.loadError")}</p>
          <div className={styles.stateActions}>
            <button type="button" className={styles.secondaryButton} onClick={() => void load()}>
              <RefreshCw aria-hidden="true" /> {t("common.retry")}
            </button>
            <Link className={styles.secondaryButton} to="/services/my-programs">
              {t("account.training.myPrograms")}
            </Link>
          </div>
        </div>
      </main>
    );
  }

  if (finished) {
    return (
      <main className={styles.page}>
        <WorkoutReport session={session} history={past} mode="saved" />
      </main>
    );
  }

  if (view === "review") {
    return (
      <main className={styles.page}>
        {actionError ? (
          <p className={styles.actionError} role="alert">
            {actionError}
          </p>
        ) : null}
        <WorkoutReport
          session={session}
          history={past}
          mode="review"
          busy={busy}
          onSave={() => void runAction(() => completeWorkoutSession(session.id))}
          onKeepGoing={() => setView("workout")}
          onAbandon={() => void runAction(() => abandonWorkoutSession(session.id))}
        />
      </main>
    );
  }

  const done = session.completed_sets + session.skipped_sets;
  const progress = session.total_sets > 0 ? done / session.total_sets : 0;

  return (
    <div className={styles.player} ref={topRef}>
      <header className={styles.topBar}>
        <div className={styles.topRow}>
          <Link
            className={styles.iconButton}
            to={`/services/my-programs/${encodeURIComponent(session.program_id)}`}
            aria-label={t("training.player.back")}
          >
            <ArrowLeft aria-hidden="true" />
          </Link>
          <div className={styles.titleBlock}>
            <p className={styles.kicker}>{session.program_name}</p>
            <h1 className={styles.title}>
              {t("account.training.weekDay", { week: session.week_number, day: session.workout_position })}
            </h1>
          </div>
          <button
            type="button"
            className={joinClassNames(styles.clock, !running && styles.clockPaused)}
            onClick={() => void runAction(() => (running ? pauseWorkoutSession(session.id) : resumeWorkoutSession(session.id)))}
            disabled={busy}
            aria-label={running ? t("training.player.pause") : t("training.player.resume")}
          >
            {running ? <Pause aria-hidden="true" /> : <Play aria-hidden="true" />}
            <span>{formatClock(clock)}</span>
          </button>
          <button type="button" className={styles.finishButton} onClick={openReview} disabled={busy}>
            {t("training.player.finish")}
          </button>
        </div>
        <div className={styles.progressRow}>
          <div
            className={styles.progressTrack}
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={session.total_sets}
            aria-valuenow={done}
            aria-label={t("training.player.progress", { done, total: session.total_sets })}
          >
            <span style={{ transform: `scaleX(${progress})` }} />
          </div>
          <span className={styles.progressText}>{t("training.player.progress", { done, total: session.total_sets })}</span>
        </div>
      </header>

      <main className={styles.workout}>
        {!running ? (
          <div className={styles.pausedBanner}>
            <span>{t("training.player.paused")}</span>
            <button type="button" onClick={() => void runAction(() => resumeWorkoutSession(session.id))} disabled={busy}>
              <Play aria-hidden="true" /> {t("training.player.resume")}
            </button>
          </div>
        ) : null}

        {actionError ? (
          <p className={styles.actionError} role="alert">
            {actionError}
          </p>
        ) : null}

        {session.exercises.map((exercise) => {
          const last = previous.get(exercise.exercise_id) ?? [];
          return (
            <article key={exercise.id} className={styles.exerciseCard}>
              <header className={styles.exerciseHead}>
                <span className={styles.exerciseIndex}>{exercise.position}</span>
                <div className={styles.exerciseTitle}>
                  <h2>
                    <button type="button" onClick={() => setInfoFor(exercise)}>
                      {exercise.name}
                    </button>
                  </h2>
                  <p>{prescription(exercise)}</p>
                </div>
                <button
                  type="button"
                  className={styles.iconButton}
                  onClick={() => setInfoFor(exercise)}
                  aria-label={t("training.player.info", { name: exercise.name })}
                >
                  <Info aria-hidden="true" />
                </button>
              </header>
              {exercise.notes ? <p className={styles.exerciseNote}>{exercise.notes}</p> : null}

              <div className={styles.setTable}>
                <div className={styles.setHeader} aria-hidden="true">
                  <span>{t("training.player.setCol")}</span>
                  <span>{t("training.player.previousCol")}</span>
                  <span>{t("training.player.kgCol")}</span>
                  <span>{t("training.player.repsCol")}</span>
                  <span>{t("training.player.doneCol")}</span>
                </div>
                {exercise.sets.map((set, index) => (
                  <SetRow
                    key={set.id}
                    set={set}
                    previous={last[index]}
                    disabled={busy}
                    onSave={(input) => saveSet(set.id, input)}
                  />
                ))}
              </div>
            </article>
          );
        })}

        <button type="button" className={styles.finishWide} onClick={openReview} disabled={busy}>
          {t("training.player.finish")}
        </button>
      </main>

      {restSet?.rest_ends_at ? (
        <RestBar
          key={restSet.id}
          endsAt={restSet.rest_ends_at}
          plannedSeconds={restSet.prescribed_rest_seconds ?? 60}
          onDismiss={() => setDismissedRest(restSet.id)}
        />
      ) : null}

      {infoFor ? (
        <ExerciseInfoSheet
          exerciseId={infoFor.exercise_id}
          name={infoFor.name}
          instructions={infoFor.instructions}
          notes={infoFor.notes}
          onClose={() => setInfoFor(null)}
        />
      ) : null}
    </div>
  );
};
