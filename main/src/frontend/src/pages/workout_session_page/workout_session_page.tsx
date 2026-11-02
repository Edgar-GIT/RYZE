import { ArrowLeft, CheckCircle2, Pause, Play, RefreshCw, Square, Timer } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useHistory, useParams } from "react-router-dom";

import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
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

const statusLabel: Record<WorkoutSetStatus, string> = {
  pending: "Not logged",
  completed: "Completed",
  skipped: "Skipped"
};

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
      setErrorMessage("Enter the reps you completed.");
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
      setErrorMessage("Unable to save this set. Please try again.");
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
        <span className={styles.setNumber}>Set {set.set_number}</span>
        <span className={styles.setType}>{set.set_type}</span>
        <span className={styles.setStatus}>{statusLabel[set.status]}</span>
      </div>

      <div className={styles.prescription}>
        <span>
          Target <strong>{set.prescribed_reps ?? "—"}</strong> reps
        </span>
        <span>
          <strong>{set.prescribed_weight_kg ?? "—"}</strong> kg
        </span>
        {set.prescribed_rest_seconds != null ? (
          <span>
            Rest <strong>{set.prescribed_rest_seconds}</strong>s
          </span>
        ) : null}
        {set.prescribed_rir != null ? <span>RIR {set.prescribed_rir}</span> : null}
        {set.prescribed_rpe != null ? <span>RPE {set.prescribed_rpe}</span> : null}
        {set.prescribed_tempo ? <span>Tempo {set.prescribed_tempo}</span> : null}
      </div>

      {!disabled ? (
        <div className={styles.setInputs}>
          <label className={styles.setField}>
            <span>Reps</span>
            <input
              type="number"
              min="0"
              inputMode="numeric"
              value={reps}
              disabled={saving}
              onChange={(event) => setReps(event.target.value)}
              aria-label={`Actual reps for set ${set.set_number}`}
            />
          </label>
          <label className={styles.setField}>
            <span>Weight (kg)</span>
            <input
              type="number"
              min="0"
              step="0.5"
              inputMode="decimal"
              value={weight}
              disabled={saving}
              onChange={(event) => setWeight(event.target.value)}
              aria-label={`Actual weight for set ${set.set_number}`}
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
                Complete set
              </Button>
              <Button size="small" variant="ghost" disabled={saving} onClick={() => void save("skipped")}>
                Skip
              </Button>
            </>
          ) : (
            <Button size="small" variant="secondary" disabled={saving} onClick={() => void save("pending")}>
              Undo
            </Button>
          )}
        </div>
      ) : null}
    </div>
  );
};

export const WorkoutSessionPage = () => {
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
          ? "This workout session is no longer available."
          : "Unable to load this workout session."
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
      setActionError("That action could not be completed. Please refresh and try again.");
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
    if (!window.confirm("Abandon this workout? Logged sets stay in your history.")) return;
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
                Try again
              </Button>
              <Button to="/services/my-programs" variant="ghost" size="small">
                My Programs
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
                Back to program
              </Button>
              <span className={joinClassNames(styles.statusPill, finished && styles.statusFinished)}>
                {session.status === "in_progress"
                  ? "In progress"
                  : session.status === "paused"
                    ? "Paused"
                    : session.status === "completed"
                      ? "Completed"
                      : "Abandoned"}
              </span>
            </div>

            <header className={styles.heading}>
              <p className={styles.eyebrow}>
                Week {session.week_number} · Day {session.workout_position}
              </p>
              <h1 className={styles.title}>{session.program_name}</h1>
              <div className={styles.meta}>
                <span className={styles.clock}>
                  <Timer size={16} aria-hidden="true" />
                  {formatClock(clock)}
                </span>
                <span>
                  {session.completed_sets}/{session.total_sets} sets completed
                </span>
                {session.skipped_sets > 0 ? <span>{session.skipped_sets} skipped</span> : null}
              </div>
            </header>

            {rest ? (
              <aside className={styles.restBanner} aria-live="polite">
                <span>
                  Rest after set {rest.setNumber}: <strong>{formatClock(rest.remaining)}</strong>
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
                  <h2>Workout {session.status === "completed" ? "complete" : "abandoned"}</h2>
                  <p>
                    {session.completed_sets} of {session.total_sets} sets logged in{" "}
                    {formatClock(session.elapsed_seconds)}.
                  </p>
                  <div className={styles.stateActions}>
                    <Button to="/services/progress" variant="primary" size="small">
                      View progress
                    </Button>
                    <Button to="/services/workout-history" variant="secondary" size="small">
                      Workout history
                    </Button>
                    <Button to={`/services/my-programs/${session.program_id}`} variant="ghost" size="small">
                      Back to program
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
                    Pause
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
                    Resume
                  </Button>
                )}
                <Button variant="primary" size="small" disabled={busy} onClick={handleComplete} icon={<CheckCircle2 size={15} />} iconPosition="left">
                  Complete workout
                </Button>
                <Button variant="ghost" size="small" disabled={busy} onClick={handleAbandon} icon={<Square size={14} />} iconPosition="left">
                  Abandon
                </Button>
              </footer>
            )}
          </section>
        ) : null}
      </main>
    </PageWrapper>
  );
};
