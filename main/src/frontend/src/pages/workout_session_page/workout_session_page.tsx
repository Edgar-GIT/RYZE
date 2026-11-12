import { ArrowLeft, CheckCircle2, Pause, Play, RefreshCw, Square, Timer } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useHistory, useParams } from "react-router-dom";

import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";
import {
  abandonWorkoutSession,
  completeWorkoutSession,
  fetchWorkoutSession,
  pauseWorkoutSession,
  resumeWorkoutSession,
  submitWorkoutSet,
  type WorkoutSession,
  type WorkoutSessionSet,
  type WorkoutSetStatus
} from "@/services/workout_sessions_api";

import styles from "./workout_session_page.module.css";

const formatClock = (totalSeconds: number): string => {
  const safe = Math.max(0, Math.floor(totalSeconds));
  const minutes = Math.floor(safe / 60);
  const seconds = safe % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
};

const SET_STATUS_KEY = {
  pending: "account.training.setPending",
  completed: "account.training.setCompleted",
  skipped: "account.training.setSkipped"
} as const satisfies Record<WorkoutSetStatus, string>;

interface SetRowProps {
  set: WorkoutSessionSet;
  disabled: boolean;
  onSave: (input: {
    status: WorkoutSetStatus;
    actual_reps: number | null;
    actual_weight_kg: number | null;
  }) => Promise<void>;
}

const SetRow = ({ set, disabled, onSave }: SetRowProps) => {
  const { t } = useI18n();
  const [reps, setReps] = useState(set.actual_reps == null ? "" : String(set.actual_reps));
  const [weight, setWeight] = useState(
    set.actual_weight_kg == null ? "" : String(set.actual_weight_kg)
  );
  const [saving, setSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    setReps(set.actual_reps == null ? "" : String(set.actual_reps));
    setWeight(set.actual_weight_kg == null ? "" : String(set.actual_weight_kg));
  }, [set.actual_reps, set.actual_weight_kg]);

  const save = async (status: WorkoutSetStatus) => {
    if (status === "completed" && reps.trim() === "") {
      setErrorMessage(t("account.training.enterReps"));
      return;
    }
    setSaving(true);
    setErrorMessage("");
    try {
      await onSave({
        status,
        actual_reps: reps.trim() === "" ? null : Number(reps),
        actual_weight_kg: weight.trim() === "" ? null : Number(weight)
      });
    } catch {
      setErrorMessage(t("account.training.saveSetError"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      className={joinClassNames(
        styles.setRow,
        set.status === "completed" && styles.setCompleted,
        set.status === "skipped" && styles.setSkipped
      )}
    >
      <div className={styles.setIdentity}>
        <span className={styles.setNumber}>{t("account.training.setNumber", { set: set.set_number })}</span>
        <span className={styles.setType}>{set.set_type}</span>
        <span className={styles.setStatus}>{t(SET_STATUS_KEY[set.status])}</span>
      </div>

      <div className={styles.prescription}>
        <span>{t("account.training.target", { reps: set.prescribed_reps ?? "—" })}</span>
        <span>
          <strong>{set.prescribed_weight_kg ?? "—"}</strong> kg
        </span>
        {set.prescribed_rest_seconds != null ? (
          <span>{t("account.training.restSeconds", { seconds: set.prescribed_rest_seconds })}</span>
        ) : null}
        {set.prescribed_rir != null ? <span>RIR {set.prescribed_rir}</span> : null}
        {set.prescribed_rpe != null ? <span>RPE {set.prescribed_rpe}</span> : null}
        {set.prescribed_tempo ? (
          <span>
            {t("account.training.colTempo")} {set.prescribed_tempo}
          </span>
        ) : null}
      </div>

      {!disabled ? (
        <div className={styles.setInputs}>
          <label className={styles.setField}>
            <span>{t("account.training.reps")}</span>
            <input
              type="number"
              min="0"
              inputMode="numeric"
              value={reps}
              disabled={saving}
              onChange={(event) => setReps(event.target.value)}
              aria-label={t("account.training.actualReps", { set: set.set_number })}
            />
          </label>
          <label className={styles.setField}>
            <span>{t("account.training.weightKg")}</span>
            <input
              type="number"
              min="0"
              step="0.5"
              inputMode="decimal"
              value={weight}
              disabled={saving}
              onChange={(event) => setWeight(event.target.value)}
              aria-label={t("account.training.actualWeight", { set: set.set_number })}
            />
          </label>
        </div>
      ) : null}

      {errorMessage ? (
        <p className={styles.setError} role="alert">
          {errorMessage}
        </p>
      ) : null}

      {!disabled ? (
        <div className={styles.setActions}>
          {set.status === "pending" ? (
            <>
              <Button size="small" variant="primary" disabled={saving} onClick={() => void save("completed")}>
                {t("account.training.completeSet")}
              </Button>
              <Button size="small" variant="ghost" disabled={saving} onClick={() => void save("skipped")}>
                {t("account.training.skip")}
              </Button>
            </>
          ) : (
            <Button size="small" variant="secondary" disabled={saving} onClick={() => void save("pending")}>
              {t("account.training.undo")}
            </Button>
          )}
        </div>
      ) : null}
    </div>
  );
};

export const WorkoutSessionPage = () => {
  const { t } = useI18n();
  const { sessionId } = useParams<{ sessionId: string }>();
  const history = useHistory();
  const [session, setSession] = useState<WorkoutSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [actionError, setActionError] = useState("");
  const [busy, setBusy] = useState(false);
  const [clock, setClock] = useState(0);
  const [now, setNow] = useState(() => Date.now());

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");
    try {
      const loaded = await fetchWorkoutSession(sessionId);
      setSession(loaded);
      setClock(loaded.elapsed_seconds);
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setSession(null);
      setErrorMessage(
        error instanceof ApiError && error.status === 404
          ? t("account.training.sessionUnavailable")
          : t("account.training.sessionError")
      );
    } finally {
      setLoading(false);
    }
  }, [sessionId, history]);

  useEffect(() => {
    void load();
  }, [load]);

  const running = session?.status === "in_progress";

  useEffect(() => {
    if (!session || session.status === "completed" || session.status === "abandoned") return;
    const interval = window.setInterval(() => {
      setNow(Date.now());
      if (running) {
        setClock((value) => value + 1);
      }
    }, 1000);
    return () => window.clearInterval(interval);
  }, [running, session]);

  const rest = useMemo(() => {
    if (!session) return null;
    let latest: { setId: string; setNumber: number; endsAt: number } | null = null;
    for (const exercise of session.exercises) {
      for (const set of exercise.sets) {
        if (set.status !== "completed" || !set.rest_ends_at) continue;
        const endsAt = new Date(set.rest_ends_at).getTime();
        if (!latest || endsAt > latest.endsAt) {
          latest = { setId: set.id, setNumber: set.set_number, endsAt };
        }
      }
    }
    if (!latest || latest.endsAt <= now) return null;
    return {
      setId: latest.setId,
      setNumber: latest.setNumber,
      remaining: Math.ceil((latest.endsAt - now) / 1000)
    };
  }, [session, now]);

  const replaceSession = (updated: WorkoutSession) => {
    setSession(updated);
    setClock(updated.elapsed_seconds);
    setNow(Date.now());
  };

  const runAction = async (action: () => Promise<WorkoutSession>) => {
    setBusy(true);
    setActionError("");
    try {
      replaceSession(await action());
    } catch {
      setActionError(t("account.training.actionError"));
    } finally {
      setBusy(false);
    }
  };

  const handleSetSave = async (
    setId: string,
    input: { status: WorkoutSetStatus; actual_reps: number | null; actual_weight_kg: number | null }
  ) => {
    const updated = await submitWorkoutSet(sessionId, setId, input);
    replaceSession(updated);
  };

  const handleComplete = () => {
    if (!session) return;
    void runAction(() => completeWorkoutSession(session.id));
  };

  const handleAbandon = () => {
    if (!session) return;
    if (!window.confirm(t("account.training.abandonConfirm"))) return;
    void runAction(() => abandonWorkoutSession(session.id));
  };

  const finished = session?.status === "completed" || session?.status === "abandoned";
  const editable = session?.status === "in_progress" || session?.status === "paused";

  return (
    <PageWrapper className={styles.page}>
      <AnimatedBackground />
      <AccountNav />

      <main className={styles.main}>
        {loading ? <LoadingScreen /> : null}

        {!loading && (errorMessage || !session) ? (
          <section className={styles.stateCard}>
            <p>{errorMessage}</p>
            <div className={styles.stateActions}>
              <Button variant="secondary" size="small" onClick={() => void load()} icon={<RefreshCw size={15} />}>
                {t("common.retry")}
              </Button>
              <Button to="/services/my-programs" variant="ghost" size="small">
                {t("account.training.myPrograms")}
              </Button>
            </div>
          </section>
        ) : null}

        {!loading && session ? (
          <section className={styles.content}>
            <div className={styles.topBar}>
              <Button
                to={`/services/my-programs/${session.program_id}`}
                variant="ghost"
                size="small"
                icon={<ArrowLeft size={15} />}
                iconPosition="left"
              >
                {t("account.training.backToProgram")}
              </Button>
              <span className={joinClassNames(styles.statusPill, finished && styles.statusFinished)}>
                {session.status === "in_progress"
                  ? t("account.training.inProgress")
                  : session.status === "paused"
                    ? t("account.training.paused")
                    : session.status === "completed"
                      ? t("account.training.completed")
                      : t("account.training.abandoned")}
              </span>
            </div>

            <header className={styles.heading}>
              <p className={styles.eyebrow}>
                {t("account.training.weekDay", { week: session.week_number, day: session.workout_position })}
              </p>
              <h1 className={styles.title}>{session.program_name}</h1>
              <div className={styles.meta}>
                <span className={styles.clock}>
                  <Timer size={16} aria-hidden="true" />
                  {formatClock(clock)}
                </span>
                <span>{t("account.training.setsCompleted", { done: session.completed_sets, total: session.total_sets })}</span>
                {session.skipped_sets > 0 ? <span>{t("account.training.skipped", { count: session.skipped_sets })}</span> : null}
              </div>
            </header>

            {rest ? (
              <aside className={styles.restBanner} aria-live="polite">
                <span>
                  {t("account.training.restAfter", { set: rest.setNumber })} <strong>{formatClock(rest.remaining)}</strong>
                </span>
              </aside>
            ) : null}

            {actionError ? (
              <p className={styles.actionError} role="alert">
                {actionError}
              </p>
            ) : null}

            <div className={styles.exerciseList}>
              {session.exercises.map((exercise) => (
                <article key={exercise.id} className={styles.exercise}>
                  <header className={styles.exerciseHead}>
                    <span className={styles.exercisePos}>{exercise.position}</span>
                    <div>
                      <h2 className={styles.exerciseName}>{exercise.name}</h2>
                      {exercise.instructions ? (
                        <p className={styles.exerciseNote}>{exercise.instructions}</p>
                      ) : null}
                      {exercise.notes ? <p className={styles.exerciseNote}>{exercise.notes}</p> : null}
                    </div>
                  </header>

                  <div className={styles.sets}>
                    {exercise.sets.map((set) => (
                      <SetRow
                        key={set.id}
                        set={set}
                        disabled={!editable || busy}
                        onSave={(input) => handleSetSave(set.id, input)}
                      />
                    ))}
                  </div>
                </article>
              ))}
            </div>

            {finished ? (
              <section className={styles.finishCard}>
                <CheckCircle2 size={22} aria-hidden="true" />
                <div>
                  <h2>
                    {session.status === "completed"
                      ? t("account.training.finishedComplete")
                      : t("account.training.finishedAbandoned")}
                  </h2>
                  <p>
                    {t("account.training.finishedSummary", {
                      done: session.completed_sets,
                      total: session.total_sets,
                      time: formatClock(session.elapsed_seconds)
                    })}
                  </p>
                  <div className={styles.stateActions}>
                    <Button to="/services/progress" variant="primary" size="small">
                      {t("account.training.viewProgress")}
                    </Button>
                    <Button to="/services/workout-history" variant="secondary" size="small">
                      {t("account.training.history")}
                    </Button>
                    <Button to={`/services/my-programs/${session.program_id}`} variant="ghost" size="small">
                      {t("account.training.backToProgram")}
                    </Button>
                  </div>
                </div>
              </section>
            ) : (
              <footer className={styles.controls}>
                {session.status === "in_progress" ? (
                  <Button
                    variant="secondary"
                    size="small"
                    disabled={busy}
                    onClick={() => void runAction(() => pauseWorkoutSession(session.id))}
                    icon={<Pause size={15} />}
                    iconPosition="left"
                  >
                    {t("account.training.pause")}
                  </Button>
                ) : (
                  <Button
                    variant="secondary"
                    size="small"
                    disabled={busy}
                    onClick={() => void runAction(() => resumeWorkoutSession(session.id))}
                    icon={<Play size={15} />}
                    iconPosition="left"
                  >
                    {t("account.training.resume")}
                  </Button>
                )}
                <Button variant="primary" size="small" disabled={busy} onClick={handleComplete} icon={<CheckCircle2 size={15} />} iconPosition="left">
                  {t("account.training.completeWorkout")}
                </Button>
                <Button variant="ghost" size="small" disabled={busy} onClick={handleAbandon} icon={<Square size={14} />} iconPosition="left">
                  {t("account.training.abandon")}
                </Button>
              </footer>
            )}
          </section>
        ) : null}
      </main>
    </PageWrapper>
  );
};
