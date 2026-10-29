import { Fragment, useCallback, useEffect, useState } from "react";
import { ArrowDown, ArrowUp, ChevronDown, ChevronRight, Pencil, Plus, RefreshCw, Trash2, X } from "lucide-react";
import { useHistory, useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { ExerciseLibraryDialog } from "@/components/exercise_library/exercise_library_dialog";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PrescriptionEditor } from "@/components/trainer/prescription_editor";
import {
  addWorkoutExercise,
  createProgramWeek,
  createProgramWorkout,
  deleteProgramWeek,
  deleteProgramWorkout,
  deleteTrainerProgram,
  fetchProgramWeeks,
  fetchTrainerProgram,
  fetchWorkoutExercises,
  publishTrainerProgram,
  removeWorkoutExercise,
  reorderProgramWeeks,
  reorderProgramWorkouts,
  reorderWorkoutExercises,
  updateTrainerProgram,
  updateWorkoutExercisePrescription,
  TrainerProgramStatus,
  TrainerProgramType,
  type TrainerPrescriptionInput,
  type TrainerProgram,
  type TrainerProgramStatusValue,
  type TrainerProgramTypeValue,
  type TrainerWeek,
  type TrainerWorkout,
  type TrainerWorkoutExercise,
  type TrainerWorkoutSet
} from "@/services/trainer_api";
import { ApiError } from "@utils/http_client";

import styles from "@/components/trainer/trainer_page.module.css";
import builder from "./trainer_program_detail_page.module.css";

const MIN_PAID_PRICE_MINOR_UNITS = 100;

type ProgramState =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; program: TrainerProgram };

// The exercise lists are loaded lazily per workout and cached by workout id, so
// expanding a workout never re-fetches what is already known.
type ExerciseCache = {
  [workoutId: string]: TrainerWorkoutExercise[] | "loading";
};

const moveItem = <T,>(items: T[], from: number, to: number): T[] => {
  const next = [...items];
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return next;
};

const describeSet = (set: TrainerWorkoutSet): string =>
  [
    set.reps !== null ? `${set.reps} reps` : "",
    set.weight_kg !== null ? `${set.weight_kg} kg` : "",
    set.rir !== null ? `RIR ${set.rir}` : "",
    set.rpe !== null ? `RPE ${set.rpe}` : "",
    set.rest_seconds !== null ? `${set.rest_seconds}s rest` : "",
    set.tempo ? `tempo ${set.tempo}` : ""
  ]
    .filter(Boolean)
    .join(" · ");

const describePrescription = (sets: TrainerWorkoutSet[]): string => {
  if (sets.length === 0) {
    return "No sets prescribed";
  }
  if (sets.length === 1) {
    const detail = describeSet(sets[0]);
    return detail ? `1 set · ${detail}` : "1 set";
  }
  const first = describeSet(sets[0]);
  const same = first !== "" && sets.every((set) => describeSet(set) === first);
  return same ? `${sets.length} sets · ${first}` : `${sets.length} sets · varied`;
};

export const TrainerProgramDetailPage = () => {
  const { programId } = useParams<{ programId: string }>();
  const history = useHistory();

  const [state, setState] = useState<ProgramState>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [type, setType] = useState<TrainerProgramTypeValue>(TrainerProgramType.FREE);
  const [status, setStatus] = useState<TrainerProgramStatusValue>(TrainerProgramStatus.DRAFT);
  const [priceEuros, setPriceEuros] = useState("");

  const [weeks, setWeeks] = useState<TrainerWeek[]>([]);
  const [weeksLoaded, setWeeksLoaded] = useState(false);
  const [exercises, setExercises] = useState<ExerciseCache>({});
  const [expandedWorkoutId, setExpandedWorkoutId] = useState("");
  const [editingExerciseId, setEditingExerciseId] = useState("");
  const [dialogTarget, setDialogTarget] = useState<{ weekId: string; workoutId: string } | null>(null);

  const [busy, setBusy] = useState("");
  const [feedback, setFeedback] = useState<{ kind: "error" | "success"; text: string } | null>(null);
  const [confirmDeleteProgram, setConfirmDeleteProgram] = useState(false);
  const [confirmKey, setConfirmKey] = useState("");

  const applyProgram = (program: TrainerProgram) => {
    setState({ status: "ready", program });
    setName(program.name);
    setDescription(program.description);
    setType(program.type);
    setStatus(program.status);
    setPriceEuros(program.type === TrainerProgramType.FREE ? "" : (program.price_minor_units / 100).toFixed(2));
  };

  const load = useCallback(async () => {
    setState({ status: "loading" });
    setWeeks([]);
    setWeeksLoaded(false);
    setExercises({});
    setExpandedWorkoutId("");
    try {
      const program = await fetchTrainerProgram(programId);
      applyProgram(program);
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        history.replace("/trainer/programs");
        return;
      }
      setState({ status: "error" });
    }
  }, [programId, history]);

  const loadWeeks = useCallback(async () => {
    try {
      const result = await fetchProgramWeeks(programId);
      setWeeks(result);
    } catch {
      setWeeks([]);
    } finally {
      setWeeksLoaded(true);
    }
  }, [programId]);

  useEffect(() => {
    void load();
  }, [load, retryCount]);

  useEffect(() => {
    if (state.status === "ready") {
      void loadWeeks();
    }
  }, [state.status, loadWeeks]);

  const run = async (
    actionKey: string,
    action: () => Promise<void>,
    errorText = "That action could not be completed."
  ) => {
    setBusy(actionKey);
    setFeedback(null);
    try {
      await action();
    } catch (error) {
      if (error instanceof ApiError && error.code === "VALIDATION_ERROR") {
        setFeedback({ kind: "error", text: "The server rejected the change. Check the program details." });
      } else {
        setFeedback({ kind: "error", text: errorText });
      }
    } finally {
      setBusy("");
    }
  };

  const saveMetadata = () =>
    run("save", async () => {
      const trimmedName = name.trim();
      if (!trimmedName) {
        setFeedback({ kind: "error", text: "Give the program a name." });
        return;
      }
      let priceMinorUnits = 0;
      if (type !== TrainerProgramType.FREE) {
        const euros = Number.parseFloat(priceEuros.replace(",", "."));
        if (!Number.isFinite(euros) || euros < MIN_PAID_PRICE_MINOR_UNITS / 100) {
          setFeedback({ kind: "error", text: "Paid programs need a price of at least 1.00 EUR." });
          return;
        }
        priceMinorUnits = Math.round(euros * 100);
      }
      const program = await updateTrainerProgram(programId, {
        name: trimmedName,
        description: description.trim(),
        type,
        status,
        price_minor_units: priceMinorUnits
      });
      applyProgram(program);
      setFeedback({ kind: "success", text: "Program saved." });
    });

  const togglePublished = () =>
    run("publish", async () => {
      if (state.status !== "ready") {
        return;
      }
      if (state.program.status === TrainerProgramStatus.PUBLISHED) {
        const program = await updateTrainerProgram(programId, { status: TrainerProgramStatus.DRAFT });
        applyProgram(program);
        setFeedback({ kind: "success", text: "Program moved back to draft." });
      } else {
        const program = await publishTrainerProgram(programId);
        applyProgram(program);
        setFeedback({ kind: "success", text: "Program published." });
      }
    });

  const destroyProgram = () =>
    run("delete-program", async () => {
      await deleteTrainerProgram(programId);
      history.replace("/trainer/programs");
    });

  const addWeek = () =>
    run("add-week", async () => {
      await createProgramWeek(programId);
      await loadWeeks();
    });

  const removeWeek = (week: TrainerWeek) =>
    run(`delete-week-${week.id}`, async () => {
      await deleteProgramWeek(programId, week.id);
      setConfirmKey("");
      await loadWeeks();
    });

  const shiftWeek = (index: number, direction: -1 | 1) =>
    run(`week-order-${index}`, async () => {
      const target = index + direction;
      const ids = moveItem(weeks, index, target).map((week) => week.id);
      await reorderProgramWeeks(programId, ids);
      await loadWeeks();
    });

  const addWorkout = (week: TrainerWeek) =>
    run(`add-workout-${week.id}`, async () => {
      await createProgramWorkout(programId, week.id);
      await loadWeeks();
    });

  const removeWorkout = (week: TrainerWeek, workout: TrainerWorkout) =>
    run(`delete-workout-${workout.id}`, async () => {
      await deleteProgramWorkout(programId, week.id, workout.id);
      setConfirmKey("");
      await loadWeeks();
    });

  const shiftWorkout = (week: TrainerWeek, index: number, direction: -1 | 1) =>
    run(`workout-order-${week.id}-${index}`, async () => {
      const ids = moveItem(week.workouts, index, index + direction).map((workout) => workout.id);
      await reorderProgramWorkouts(programId, week.id, ids);
      await loadWeeks();
    });

  const toggleWorkout = async (workout: TrainerWorkout) => {
    if (expandedWorkoutId === workout.id) {
      setExpandedWorkoutId("");
      setEditingExerciseId("");
      return;
    }
    setExpandedWorkoutId(workout.id);
    setEditingExerciseId("");
    if (!exercises[workout.id]) {
      setExercises((current) => ({ ...current, [workout.id]: "loading" }));
      try {
        const week = weeks.find((entry) => entry.workouts.some((item) => item.id === workout.id));
        if (!week) {
          return;
        }
        const list = await fetchWorkoutExercises(programId, week.id, workout.id);
        setExercises((current) => ({ ...current, [workout.id]: list }));
      } catch {
        setExercises((current) => ({ ...current, [workout.id]: [] }));
      }
    }
  };

  const addExerciseFromDialog = async (exerciseId: string) => {
    if (!dialogTarget) {
      return;
    }
    const { weekId, workoutId } = dialogTarget;
    setBusy(`add-exercise-${workoutId}`);
    try {
      const created = await addWorkoutExercise(programId, weekId, workoutId, exerciseId);
      setExercises((current) => {
        const cached = current[workoutId];
        const list = Array.isArray(cached) ? cached : [];
        return { ...current, [workoutId]: [...list, created] };
      });
    } catch {
      setFeedback({ kind: "error", text: "Unable to add that exercise right now." });
    } finally {
      setBusy("");
    }
  };

  const savePrescription = (
    week: TrainerWeek,
    workoutId: string,
    entry: TrainerWorkoutExercise,
    input: TrainerPrescriptionInput
  ) =>
    run(`prescription-${entry.id}`, async () => {
      const updated = await updateWorkoutExercisePrescription(
        programId,
        week.id,
        workoutId,
        entry.id,
        input
      );
      setExercises((current) => {
        const cached = current[workoutId];
        const list = Array.isArray(cached) ? cached : [];
        return {
          ...current,
          [workoutId]: list.map((item) => (item.id === updated.id ? updated : item))
        };
      });
      setEditingExerciseId("");
      setFeedback({ kind: "success", text: "Prescription saved." });
    }, "Unable to save this prescription right now.");

  const removeExercise = (week: TrainerWeek, workoutId: string, entry: TrainerWorkoutExercise) =>
    run(`delete-exercise-${entry.id}`, async () => {
      await removeWorkoutExercise(programId, week.id, workoutId, entry.id);
      if (editingExerciseId === entry.id) {
        setEditingExerciseId("");
      }
      try {
        const list = await fetchWorkoutExercises(programId, week.id, workoutId);
        setExercises((current) => ({ ...current, [workoutId]: list }));
      } catch {
        setExercises((current) => ({ ...current, [workoutId]: [] }));
      }
    });

  const shiftExercise = (week: TrainerWeek, workoutId: string, index: number, direction: -1 | 1, list: TrainerWorkoutExercise[]) =>
    run(`exercise-order-${workoutId}-${index}`, async () => {
      const ids = moveItem(list, index, index + direction).map((entry) => entry.id);
      await reorderWorkoutExercises(programId, week.id, workoutId, ids);
      const refreshed = await fetchWorkoutExercises(programId, week.id, workoutId);
      setExercises((current) => ({ ...current, [workoutId]: refreshed }));
    });

  if (state.status === "loading") {
    return <LoadingScreen />;
  }

  if (state.status === "error") {
    return (
      <section className={styles.stateCard}>
        <h1 className={styles.stateTitle}>Program</h1>
        <p className={styles.stateText}>Unable to load this program right now.</p>
        <div className={styles.actions}>
          <Button
            variant="secondary"
            size="small"
            icon={<RefreshCw size={15} />}
            onClick={() => setRetryCount((count) => count + 1)}
          >
            Try again
          </Button>
        </div>
      </section>
    );
  }

  const { program } = state;
  const isPaid = type !== TrainerProgramType.FREE;

  return (
    <>
      <header className={styles.pageHeading}>
        <div>
          <p className={styles.eyebrow}>Trainer · Program</p>
          <h1 className={styles.title}>{program.name}</h1>
          <p className={styles.subtitle}>
            {program.status === TrainerProgramStatus.PUBLISHED
              ? "This program is published and visible in the catalogue."
              : "This program is a draft and only visible to you."}
          </p>
        </div>
        <div className={styles.headingActions}>
          <Button
            variant="secondary"
            size="small"
            disabled={busy === "publish"}
            onClick={() => void togglePublished()}
          >
            {program.status === TrainerProgramStatus.PUBLISHED
              ? busy === "publish"
                ? "Working…"
                : "Move to draft"
              : busy === "publish"
                ? "Publishing…"
                : "Publish"}
          </Button>
        </div>
      </header>

      {feedback ? (
        <p className={feedback.kind === "error" ? styles.error : styles.success}>{feedback.text}</p>
      ) : null}

      <section className={styles.card}>
        <h2 className={styles.cardTitle}>Details</h2>
        <div className={styles.form}>
          <div className={styles.formRow}>
            <label className={styles.field}>
              <span className={styles.label}>Name</span>
              <input className={styles.input} value={name} maxLength={255} onChange={(event) => setName(event.target.value)} />
            </label>
            <label className={styles.field}>
              <span className={styles.label}>Type</span>
              <select className={styles.select} value={type} onChange={(event) => setType(event.target.value as TrainerProgramTypeValue)}>
                <option value={TrainerProgramType.FREE}>Free</option>
                <option value={TrainerProgramType.PREMIUM}>Premium</option>
                <option value={TrainerProgramType.PERSONALIZED}>Personalized</option>
              </select>
            </label>
            <label className={styles.field}>
              <span className={styles.label}>Price (EUR)</span>
              <input
                className={styles.input}
                inputMode="decimal"
                value={isPaid ? priceEuros : "0"}
                disabled={!isPaid}
                onChange={(event) => setPriceEuros(event.target.value)}
              />
            </label>
            <label className={styles.field}>
              <span className={styles.label}>Status</span>
              <select className={styles.select} value={status} onChange={(event) => setStatus(event.target.value as TrainerProgramStatusValue)}>
                <option value={TrainerProgramStatus.DRAFT}>Draft</option>
                <option value={TrainerProgramStatus.PUBLISHED}>Published</option>
              </select>
            </label>
          </div>
          <label className={styles.field}>
            <span className={styles.label}>Description</span>
            <textarea
              className={styles.textarea}
              value={description}
              maxLength={5000}
              onChange={(event) => setDescription(event.target.value)}
            />
          </label>
          <div className={styles.actions}>
            <Button variant="primary" size="small" disabled={busy === "save"} onClick={() => void saveMetadata()}>
              {busy === "save" ? "Saving…" : "Save changes"}
            </Button>
          </div>
        </div>
      </section>

      <section className={styles.sectionGap}>
        <div className={builder.builderHeader}>
          <h2 className={styles.cardTitle}>Weeks and workouts</h2>
          <Button variant="secondary" size="small" icon={<Plus size={15} />} disabled={busy === "add-week"} onClick={() => void addWeek()}>
            {busy === "add-week" ? "Adding…" : "Add week"}
          </Button>
        </div>

        {!weeksLoaded ? (
          <div className={styles.stateCard}>
            <p className={styles.stateText}>Loading weeks…</p>
          </div>
        ) : weeks.length === 0 ? (
          <div className={styles.stateCard}>
            <p className={styles.stateText}>
              This program has no weeks yet. Add a week, then add workouts inside it.
            </p>
          </div>
        ) : (
          <div className={builder.weekList}>
            {weeks.map((week, weekIndex) => (
              <article key={week.id} className={builder.weekCard}>
                <header className={builder.weekHeader}>
                  <span className={builder.weekTitle}>Week {week.week_number}</span>
                  <div className={builder.iconRow}>
                    <button
                      type="button"
                      className={builder.iconButton}
                      aria-label={`Move week ${week.week_number} up`}
                      disabled={weekIndex === 0 || busy.startsWith("week-order")}
                      onClick={() => void shiftWeek(weekIndex, -1)}
                    >
                      <ArrowUp size={15} />
                    </button>
                    <button
                      type="button"
                      className={builder.iconButton}
                      aria-label={`Move week ${week.week_number} down`}
                      disabled={weekIndex === weeks.length - 1 || busy.startsWith("week-order")}
                      onClick={() => void shiftWeek(weekIndex, 1)}
                    >
                      <ArrowDown size={15} />
                    </button>
                    {confirmKey === `week-${week.id}` ? (
                      <>
                        <span className={styles.rowMeta}>Delete week?</span>
                        <Button variant="danger" size="small" disabled={busy === `delete-week-${week.id}`} onClick={() => void removeWeek(week)}>
                          Confirm
                        </Button>
                        <Button variant="ghost" size="small" onClick={() => setConfirmKey("")}>
                          Cancel
                        </Button>
                      </>
                    ) : (
                      <button
                        type="button"
                        className={builder.iconButton}
                        aria-label={`Delete week ${week.week_number}`}
                        onClick={() => setConfirmKey(`week-${week.id}`)}
                      >
                        <Trash2 size={15} />
                      </button>
                    )}
                  </div>
                </header>

                <div className={builder.workoutList}>
                  {week.workouts.length === 0 ? (
                    <p className={styles.hint}>No workouts in this week yet.</p>
                  ) : null}
                  {week.workouts.map((workout, workoutIndex) => {
                    const expanded = expandedWorkoutId === workout.id;
                    const cache = exercises[workout.id];
                    const list = Array.isArray(cache) ? cache : [];
                    return (
                      <div key={workout.id} className={builder.workoutCard}>
                        <header className={builder.workoutHeader}>
                          <button
                            type="button"
                            className={builder.expandButton}
                            aria-expanded={expanded}
                            onClick={() => void toggleWorkout(workout)}
                          >
                            {expanded ? <ChevronDown size={15} /> : <ChevronRight size={15} />}
                            Workout {workoutIndex + 1}
                          </button>
                          <div className={builder.iconRow}>
                            <button
                              type="button"
                              className={builder.iconButton}
                              aria-label={`Move workout ${workoutIndex + 1} up`}
                              disabled={workoutIndex === 0 || busy.startsWith("workout-order")}
                              onClick={() => void shiftWorkout(week, workoutIndex, -1)}
                            >
                              <ArrowUp size={15} />
                            </button>
                            <button
                              type="button"
                              className={builder.iconButton}
                              aria-label={`Move workout ${workoutIndex + 1} down`}
                              disabled={workoutIndex === week.workouts.length - 1 || busy.startsWith("workout-order")}
                              onClick={() => void shiftWorkout(week, workoutIndex, 1)}
                            >
                              <ArrowDown size={15} />
                            </button>
                            {confirmKey === `workout-${workout.id}` ? (
                              <>
                                <span className={styles.rowMeta}>Delete workout?</span>
                                <Button
                                  variant="danger"
                                  size="small"
                                  disabled={busy === `delete-workout-${workout.id}`}
                                  onClick={() => void removeWorkout(week, workout)}
                                >
                                  Confirm
                                </Button>
                                <Button variant="ghost" size="small" onClick={() => setConfirmKey("")}>
                                  Cancel
                                </Button>
                              </>
                            ) : (
                              <button
                                type="button"
                                className={builder.iconButton}
                                aria-label={`Delete workout ${workoutIndex + 1}`}
                                onClick={() => setConfirmKey(`workout-${workout.id}`)}
                              >
                                <Trash2 size={15} />
                              </button>
                            )}
                          </div>
                        </header>

                        {expanded ? (
                          <div className={builder.exerciseList}>
                            {cache === "loading" ? (
                              <p className={styles.hint}>Loading exercises…</p>
                            ) : (
                              <>
                                {list.length === 0 ? (
                                  <p className={styles.hint}>No exercises in this workout yet.</p>
                                ) : null}
                                {list.map((entry, exerciseIndex) => (
                                  <Fragment key={entry.id}>
                                    <div className={builder.exerciseRow}>
                                      <div className={styles.rowMain}>
                                        <span className={builder.exerciseName}>{entry.exercise.name}</span>
                                        <span className={builder.prescriptionSummary}>
                                          {describePrescription(entry.sets)}
                                        </span>
                                        <span className={styles.rowMeta}>
                                          {[entry.exercise.target_muscles, entry.exercise.equipment]
                                            .map((value) => value?.trim())
                                            .filter(Boolean)
                                            .join(" · ")}
                                        </span>
                                      </div>
                                      <div className={builder.iconRow}>
                                        <button
                                          type="button"
                                          className={builder.iconButton}
                                          aria-label={`Edit prescription for ${entry.exercise.name}`}
                                          aria-expanded={editingExerciseId === entry.id}
                                          onClick={() =>
                                            setEditingExerciseId((current) => (current === entry.id ? "" : entry.id))
                                          }
                                        >
                                          <Pencil size={15} />
                                        </button>
                                        <button
                                          type="button"
                                          className={builder.iconButton}
                                          aria-label={`Move ${entry.exercise.name} up`}
                                          disabled={exerciseIndex === 0 || busy.startsWith("exercise-order")}
                                          onClick={() => void shiftExercise(week, workout.id, exerciseIndex, -1, list)}
                                        >
                                          <ArrowUp size={15} />
                                        </button>
                                        <button
                                          type="button"
                                          className={builder.iconButton}
                                          aria-label={`Move ${entry.exercise.name} down`}
                                          disabled={exerciseIndex === list.length - 1 || busy.startsWith("exercise-order")}
                                          onClick={() => void shiftExercise(week, workout.id, exerciseIndex, 1, list)}
                                        >
                                          <ArrowDown size={15} />
                                        </button>
                                        <button
                                          type="button"
                                          className={builder.iconButton}
                                          aria-label={`Remove ${entry.exercise.name}`}
                                          disabled={busy === `delete-exercise-${entry.id}`}
                                          onClick={() => void removeExercise(week, workout.id, entry)}
                                        >
                                          <X size={15} />
                                        </button>
                                      </div>
                                    </div>
                                    {editingExerciseId === entry.id ? (
                                      <PrescriptionEditor
                                        entry={entry}
                                        busy={busy === `prescription-${entry.id}`}
                                        onSave={(input) => void savePrescription(week, workout.id, entry, input)}
                                        onCancel={() => setEditingExerciseId("")}
                                      />
                                    ) : null}
                                  </Fragment>
                                ))}
                                <div className={styles.actions}>
                                  <Button
                                    variant="secondary"
                                    size="small"
                                    icon={<Plus size={15} />}
                                    onClick={() => setDialogTarget({ weekId: week.id, workoutId: workout.id })}
                                  >
                                    Add exercise
                                  </Button>
                                </div>
                              </>
                            )}
                          </div>
                        ) : null}
                      </div>
                    );
                  })}
                  <div className={styles.actions}>
                    <Button
                      variant="ghost"
                      size="small"
                      icon={<Plus size={15} />}
                      disabled={busy === `add-workout-${week.id}`}
                      onClick={() => void addWorkout(week)}
                    >
                      {busy === `add-workout-${week.id}` ? "Adding…" : "Add workout"}
                    </Button>
                  </div>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>

      <section className={styles.sectionGap}>
        <div className={builder.dangerZone}>
          <div className={styles.rowMain}>
            <span className={styles.rowTitle}>Delete this program</span>
            <span className={styles.rowMeta}>
              The program is removed from your catalogue. This is permanent.
            </span>
          </div>
          {confirmDeleteProgram ? (
            <div className={styles.rowActions}>
              <Button variant="danger" size="small" disabled={busy === "delete-program"} onClick={() => void destroyProgram()}>
                {busy === "delete-program" ? "Deleting…" : "Confirm delete"}
              </Button>
              <Button variant="ghost" size="small" onClick={() => setConfirmDeleteProgram(false)}>
                Cancel
              </Button>
            </div>
          ) : (
            <Button variant="danger" size="small" icon={<Trash2 size={15} />} onClick={() => setConfirmDeleteProgram(true)}>
              Delete program
            </Button>
          )}
        </div>
      </section>

      {dialogTarget ? (
        <ExerciseLibraryDialog
          title="Add exercise to workout"
          description="Pick a catalogue exercise for this workout."
          addedExerciseIds={
            Array.isArray(exercises[dialogTarget.workoutId])
              ? (exercises[dialogTarget.workoutId] as TrainerWorkoutExercise[]).map((entry) => entry.exercise.id)
              : []
          }
          actionLabel="Add to workout"
          onAdd={(exercise) => void addExerciseFromDialog(exercise.id)}
          onClose={() => setDialogTarget(null)}
        />
      ) : null}
    </>
  );
};
