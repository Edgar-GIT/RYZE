import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { useHistory } from "react-router-dom";
import {
  AlertCircle,
  AlertTriangle,
  ArrowDown,
  ArrowLeft,
  ArrowUp,
  Check,
  CheckCircle2,
  ChevronRight,
  Info,
  Plus,
  Repeat,
  RotateCcw,
  Search,
  Trash2
} from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Button } from "@/components/button/button";
import { ExerciseLibraryDialog } from "@/components/exercise_library/exercise_library_dialog";
import { joinClassNames } from "@utils/class_names";
import { ProgramStatusEnum, ProgramType, type ProgramTypeValue } from "@/services/admin2_api";
import {
  exerciseMetaLine,
  fetchExerciseDetail,
  fetchExerciseLibrary,
  type ExerciseCatalogEntry,
  type ExerciseDetail
} from "@/services/exercise_library";
import { PlanPreview } from "./plan_preview";
import { PlanExerciseEditorModal } from "./plan_exercise_editor";
import {
  addDay,
  addExerciseToDay,
  addSetToExercise,
  addWeek,
  countDays,
  countExercises,
  countSets,
  countWeeks,
  createPlanDraft,
  moveExerciseInDay,
  removeDay,
  removeExerciseFromDay,
  removeSetFromExercise,
  removeWeek,
  replaceExerciseInDay,
  updateExerciseAssignment,
  updateSetInExercise,
  validatePlan,
  type PlanDraft,
  type PlanDraftExercise,
  type PlanDraftWorkout,
  type PlanDraftWeek
} from "./plan_builder_model";

import styles from "./admin2_plan_create_page.module.css";

const STEPS = ["Basic information", "Program structure", "Marketplace", "Review & publish"];

const NotSavedNotice = () => (
  <div className={styles.notice}>
    <Info size={15} className={styles.noticeIcon} aria-hidden="true" />
    <span>
      <strong>Workspace preview.</strong> The backend program-creation API is not implemented yet.
      This workspace is a complete client-side design; nothing is persisted until the endpoint
      ships.
    </span>
  </div>
);

interface AssignmentLocation {
  weekId: string;
  workoutId: string;
  assignment: PlanDraftExercise;
}

type PickerTarget =
  | { mode: "add"; weekId: string; workoutId: string }
  | { mode: "replace"; weekId: string; workoutId: string; assignmentId: string };

interface EditorTarget {
  weekId: string;
  workoutId: string;
  assignmentId: string;
}

const findAssignment = (draft: PlanDraft, workoutId: string, assignmentId: string): AssignmentLocation | null => {
  for (const week of draft.weeks) {
    const workout = week.workouts.find((entry) => entry.id === workoutId);
    const assignment = workout?.exercises.find((entry) => entry.id === assignmentId);
    if (workout && assignment) {
      return { weekId: week.id, workoutId: workout.id, assignment };
    }
  }
  return null;
};

const WeekEditor = ({
  week,
  exerciseMap,
  onRemoveWeek,
  onAddDay,
  onRemoveDay,
  onAddExercise,
  onEditAssignment,
  onMoveAssignment,
  onReplaceAssignment,
  onRemoveAssignment
}: {
  week: PlanDraftWeek;
  exerciseMap: Record<string, ExerciseCatalogEntry>;
  onRemoveWeek: () => void;
  onAddDay: () => void;
  onRemoveDay: (workoutId: string) => void;
  onAddExercise: (workoutId: string) => void;
  onEditAssignment: (workoutId: string, assignmentId: string) => void;
  onMoveAssignment: (workoutId: string, assignmentId: string, direction: -1 | 1) => void;
  onReplaceAssignment: (workoutId: string, assignmentId: string) => void;
  onRemoveAssignment: (workoutId: string, assignmentId: string) => void;
}) => (
  <div className={styles.weekCard}>
    <div className={styles.weekHead}>
      <p className={styles.weekTitle}>Week {week.week_number}</p>
      <div className={styles.weekActions}>
        <button
          type="button"
          className={joinClassNames(styles.iconButton, styles.iconButtonDanger)}
          onClick={onRemoveWeek}
          aria-label={`Remove week ${week.week_number}`}
          title="Remove week"
        >
          <Trash2 size={15} aria-hidden="true" />
        </button>
      </div>
    </div>

    <div className={styles.dayList}>
      {week.workouts.map((workout) => (
        <DayEditor
          key={workout.id}
          workout={workout}
          exerciseMap={exerciseMap}
          onRemoveDay={onRemoveDay}
          onAddExercise={onAddExercise}
          onEditAssignment={onEditAssignment}
          onMoveAssignment={onMoveAssignment}
          onReplaceAssignment={onReplaceAssignment}
          onRemoveAssignment={onRemoveAssignment}
        />
      ))}
    </div>

    <div className={styles.addRow}>
      <Button
        type="button"
        variant="ghost"
        size="small"
        icon={<Plus size={14} />}
        onClick={onAddDay}
      >
        Add training day
      </Button>
    </div>
  </div>
);

const DayEditor = ({
  workout,
  exerciseMap,
  onRemoveDay,
  onAddExercise,
  onEditAssignment,
  onMoveAssignment,
  onReplaceAssignment,
  onRemoveAssignment
}: {
  workout: PlanDraftWorkout;
  exerciseMap: Record<string, ExerciseCatalogEntry>;
  onRemoveDay: (workoutId: string) => void;
  onAddExercise: (workoutId: string) => void;
  onEditAssignment: (workoutId: string, assignmentId: string) => void;
  onMoveAssignment: (workoutId: string, assignmentId: string, direction: -1 | 1) => void;
  onReplaceAssignment: (workoutId: string, assignmentId: string) => void;
  onRemoveAssignment: (workoutId: string, assignmentId: string) => void;
}) => (
  <div className={styles.dayCard}>
    <div className={styles.dayHead}>
      <p className={styles.dayTitle}>Day {workout.position}</p>
      <button
        type="button"
        className={joinClassNames(styles.iconButton, styles.iconButtonDanger)}
        onClick={() => onRemoveDay(workout.id)}
        aria-label={`Remove day ${workout.position}`}
        title="Remove day"
      >
        <Trash2 size={14} aria-hidden="true" />
      </button>
    </div>

    {workout.exercises.length === 0 ? (
      <p className={styles.chipsEmpty}>No exercises assigned yet.</p>
    ) : (
      <div className={styles.exerciseList}>
        {workout.exercises.map((assignment) => {
          const exercise = exerciseMap[assignment.exercise_id];
          const setCount = assignment.sets.length;
          return (
            <div key={assignment.id} className={styles.exerciseRow}>
              <span className={styles.exercisePos}>{assignment.position}</span>
              <button
                type="button"
                className={styles.exerciseInfo}
                onClick={() => onEditAssignment(workout.id, assignment.id)}
                aria-label={`Edit ${exercise?.name ?? "exercise"} details`}
              >
                <span className={styles.exerciseName}>{exercise?.name ?? "Exercise"}</span>
                <span className={styles.exerciseMeta}>
                  {setCount} set{setCount === 1 ? "" : "s"}
                  {exercise ? ` · ${exerciseMetaLine(exercise)}` : ""}
                </span>
              </button>
              <div className={styles.exerciseActions}>
                <button
                  type="button"
                  className={styles.iconButton}
                  disabled={assignment.position === 1}
                  onClick={() => onMoveAssignment(workout.id, assignment.id, -1)}
                  aria-label={`Move ${exercise?.name ?? "exercise"} up`}
                  title="Move up"
                >
                  <ArrowUp size={14} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  className={styles.iconButton}
                  disabled={assignment.position === workout.exercises.length}
                  onClick={() => onMoveAssignment(workout.id, assignment.id, 1)}
                  aria-label={`Move ${exercise?.name ?? "exercise"} down`}
                  title="Move down"
                >
                  <ArrowDown size={14} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  className={styles.iconButton}
                  onClick={() => onReplaceAssignment(workout.id, assignment.id)}
                  aria-label={`Replace ${exercise?.name ?? "exercise"}`}
                  title="Replace exercise"
                >
                  <Repeat size={14} aria-hidden="true" />
                </button>
                <button
                  type="button"
                  className={joinClassNames(styles.iconButton, styles.iconButtonDanger)}
                  onClick={() => onRemoveAssignment(workout.id, assignment.id)}
                  aria-label={`Remove ${exercise?.name ?? "exercise"} from day`}
                  title="Remove from day"
                >
                  <Trash2 size={14} aria-hidden="true" />
                </button>
              </div>
            </div>
          );
        })}
      </div>
    )}

    <div className={styles.addRow}>
      <Button
        type="button"
        variant="ghost"
        size="small"
        icon={<Search size={14} />}
        onClick={() => onAddExercise(workout.id)}
      >
        Add exercises
      </Button>
    </div>
  </div>
);

export default function Admin2PlanCreatePage() {
  const history = useHistory();
  const [draft, setDraft] = useState<PlanDraft>(createPlanDraft);
  const [step, setStep] = useState(0);
  const [dirty, setDirty] = useState(false);
  const [exerciseMap, setExerciseMap] = useState<Record<string, ExerciseCatalogEntry>>({});
  const [pickerTarget, setPickerTarget] = useState<PickerTarget | null>(null);
  const [editorTarget, setEditorTarget] = useState<EditorTarget | null>(null);
  const [editorDetail, setEditorDetail] = useState<ExerciseDetail | null>(null);
  const [attemptMessage, setAttemptMessage] = useState("");
  const editorRef = useRef<EditorTarget | null>(null);

  const update = useCallback((producer: (current: PlanDraft) => PlanDraft) => {
    setDraft((current) => producer(current));
    setDirty(true);
    setAttemptMessage("");
  }, []);

  useEffect(() => {
    let cancelled = false;

    fetchExerciseLibrary({}, 1, 100)
      .then((result) => {
        if (cancelled) {
          return;
        }
        const map: Record<string, ExerciseCatalogEntry> = {};
        result.exercises.forEach((exercise) => {
          map[exercise.id] = exercise;
        });
        setExerciseMap(map);
      })
      .catch(() => {
        // The workspace remains usable; unknown references fall back to "Exercise".
      });

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!dirty) {
      return;
    }
    const unblock = history.block("You have unsaved changes. Leave anyway?");
    return unblock;
  }, [dirty, history]);

  useEffect(() => {
    if (!dirty) {
      return;
    }
    const preventLeave = (event: BeforeUnloadEvent) => {
      event.preventDefault();
    };
    window.addEventListener("beforeunload", preventLeave);
    return () => window.removeEventListener("beforeunload", preventLeave);
  }, [dirty]);

  const validation = useMemo(() => validatePlan(draft), [draft]);

  const completion: boolean[] = [
    Boolean(draft.name.trim()) && Boolean(draft.description.trim()),
    countExercises(draft) > 0 && countWeeks(draft) > 0,
    validation.messages.length === 0,
    true
  ];

  const pickerLocation = useMemo(() => {
    if (!pickerTarget) {
      return null;
    }
    for (const week of draft.weeks) {
      const workout = week.workouts.find((entry) => entry.id === pickerTarget.workoutId);
      if (workout) {
        return { weekId: week.id, workout };
      }
    }
    return null;
  }, [pickerTarget, draft.weeks]);

  const pickerAddedIds = useMemo(() => {
    if (!pickerTarget || !pickerLocation) {
      return [];
    }
    if (pickerTarget.mode === "replace") {
      const assignment = pickerLocation.workout.exercises.find(
        (entry) => entry.id === pickerTarget.assignmentId
      );
      return assignment ? [assignment.exercise_id] : [];
    }
    return pickerLocation.workout.exercises.map((assignment) => assignment.exercise_id);
  }, [pickerTarget, pickerLocation]);

  const editorLocation = useMemo(() => {
    if (!editorTarget) {
      return null;
    }
    return findAssignment(draft, editorTarget.workoutId, editorTarget.assignmentId);
  }, [editorTarget, draft]);

  const closeEditor = () => {
    editorRef.current = null;
    setEditorTarget(null);
    setEditorDetail(null);
  };

  const openEditor = (workoutId: string, assignmentId: string) => {
    const location = findAssignment(draft, workoutId, assignmentId);
    if (!location) {
      return;
    }
    const target: EditorTarget = {
      weekId: location.weekId,
      workoutId: location.workoutId,
      assignmentId: location.assignment.id
    };
    editorRef.current = target;
    setEditorTarget(target);
    setEditorDetail(null);
    fetchExerciseDetail(location.assignment.exercise_id)
      .then((detail) => {
        if (editorRef.current?.assignmentId === assignmentId) {
          setEditorDetail(detail);
        }
      })
      .catch(() => {
        // The set editor remains usable without catalogue details.
      });
  };

  const discard = () => {
    setDraft(createPlanDraft());
    setDirty(false);
    setStep(0);
    setAttemptMessage("");
  };

  const handlePublishAttempt = () => {
    if (!validation.valid) {
      setAttemptMessage("Fix the validation issues below before publishing.");
      setStep(3);
      return;
    }
    setAttemptMessage(
      "Publishing is not available yet: the backend program-creation API is not implemented. This workspace is a complete client-side design that will persist through the create endpoint in a later milestone."
    );
  };

  const renderStep = (): ReactNode => {
    if (step === 0) {
      return (
        <Admin2Section title="Basic information" subtitle="What the plan is called and promises">
          <div className={styles.sectionBody}>
            <div className={styles.formRow}>
              <label className={styles.formLabel} htmlFor="plan-name">
                Plan name
              </label>
              <input
                id="plan-name"
                className={styles.formInput}
                type="text"
                value={draft.name}
                maxLength={120}
                placeholder="e.g. Full Body Foundation"
                onChange={(event) => update((current) => ({ ...current, name: event.target.value }))}
              />
            </div>
            <div className={styles.formRow}>
              <label className={styles.formLabel} htmlFor="plan-description">
                Description
              </label>
              <textarea
                id="plan-description"
                className={joinClassNames(styles.formInput, styles.formTextarea)}
                value={draft.description}
                maxLength={600}
                placeholder="What members will achieve, how it is structured and who it fits."
                onChange={(event) =>
                  update((current) => ({ ...current, description: event.target.value }))
                }
              />
            </div>
          </div>
        </Admin2Section>
      );
    }

    if (step === 1) {
      return (
        <Admin2Section
          title="Program structure"
          subtitle="Weeks, training days and the exercises assigned to them"
        >
          <div className={styles.sectionBody}>
            {draft.weeks.length === 0 ? (
              <p className={styles.chipsEmpty}>No weeks yet. Add a week to start building.</p>
            ) : (
              draft.weeks.map((week) => (
                <WeekEditor
                  key={week.id}
                  week={week}
                  exerciseMap={exerciseMap}
                  onRemoveWeek={() => update((current) => removeWeek(current, week.id))}
                  onAddDay={() => update((current) => addDay(current, week.id))}
                  onRemoveDay={(workoutId) =>
                    update((current) => removeDay(current, week.id, workoutId))
                  }
                  onAddExercise={(workoutId) =>
                    setPickerTarget({ mode: "add", weekId: week.id, workoutId })
                  }
                  onEditAssignment={(workoutId, assignmentId) =>
                    openEditor(workoutId, assignmentId)
                  }
                  onMoveAssignment={(workoutId, assignmentId, direction) =>
                    update((current) =>
                      moveExerciseInDay(current, week.id, workoutId, assignmentId, direction)
                    )
                  }
                  onReplaceAssignment={(workoutId, assignmentId) =>
                    setPickerTarget({ mode: "replace", weekId: week.id, workoutId, assignmentId })
                  }
                  onRemoveAssignment={(workoutId, assignmentId) =>
                    update((current) =>
                      removeExerciseFromDay(current, week.id, workoutId, assignmentId)
                    )
                  }
                />
              ))
            )}

            {draft.weeks.length < 12 ? (
              <div className={styles.addRow}>
                <Button
                  type="button"
                  variant="secondary"
                  size="small"
                  icon={<Plus size={14} />}
                  onClick={() => update(addWeek)}
                >
                  Add week
                </Button>
              </div>
            ) : (
              <p className={styles.formHint}>RYZE plans are capped at 12 weeks.</p>
            )}
          </div>
        </Admin2Section>
      );
    }

    if (step === 2) {
      const types: Array<{ id: ProgramTypeValue; name: string; description: string }> = [
        {
          id: ProgramType.FREE,
          name: "Generic",
          description: "Ready-made training-only catalogue plans, free for members."
        },
        {
          id: ProgramType.PREMIUM,
          name: "Premium · Level 1",
          description: "Training plus AI-composed nutrition guidance."
        },
        {
          id: ProgramType.PERSONALIZED,
          name: "Premium · Level 2",
          description: "Trainer-created individualized plans."
        }
      ];

      return (
        <Admin2Section
          title="Marketplace & publishing"
          subtitle="How the plan is sold and surfaced"
        >
          <div className={styles.sectionBody}>
            <div className={styles.formRow}>
              <span className={styles.formLabel}>Product type</span>
              <div className={styles.typeGrid}>
                {types.map((typeOption) => (
                  <button
                    key={typeOption.id}
                    type="button"
                    className={joinClassNames(
                      styles.typeCard,
                      draft.type === typeOption.id && styles.typeCardActive
                    )}
                    onClick={() => update((current) => ({ ...current, type: typeOption.id }))}
                  >
                    <p className={styles.typeCardName}>{typeOption.name}</p>
                    <p className={styles.typeCardDesc}>{typeOption.description}</p>
                  </button>
                ))}
              </div>
            </div>

            <div className={styles.priceRow}>
              <div className={styles.formRow}>
                <label className={styles.formLabel} htmlFor="plan-price">
                  Price
                </label>
                <input
                  id="plan-price"
                  className={styles.formInput}
                  type="number"
                  min="0"
                  step="0.01"
                  value={draft.type === ProgramType.FREE ? "0.00" : draft.price}
                  disabled={draft.type === ProgramType.FREE}
                  onChange={(event) =>
                    update((current) => ({ ...current, price: event.target.value }))
                  }
                  aria-label="Plan price in euros"
                />
              </div>
              <div className={styles.currencyChip}>EUR</div>
            </div>

            <div className={styles.formRow}>
              <span className={styles.formLabel}>Publication status</span>
              <div className={styles.statusRow}>
                {[ProgramStatusEnum.DRAFT, ProgramStatusEnum.PUBLISHED].map((statusOption) => (
                  <span
                    key={statusOption}
                    className={joinClassNames(
                      styles.statusOption,
                      draft.status === statusOption && styles.statusOptionActive
                    )}
                    role="radio"
                    aria-checked={draft.status === statusOption}
                    tabIndex={0}
                    onClick={() => update((current) => ({ ...current, status: statusOption }))}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        update((current) => ({ ...current, status: statusOption }));
                      }
                    }}
                  >
                    <span className={styles.statusDot} aria-hidden="true" />
                    {statusOption === ProgramStatusEnum.PUBLISHED ? "Publish immediately" : "Save as draft"}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </Admin2Section>
      );
    }

    return (
      <Admin2Section title="Review & publish" subtitle="Confirm the plan before publishing">
        <div className={styles.sectionBody}>
          <div className={styles.reviewSummary}>
            <span className={styles.chip}>{countWeeks(draft)} week{countWeeks(draft) === 1 ? "" : "s"}</span>
            <span className={styles.chip}>{countDays(draft)} day{countDays(draft) === 1 ? "" : "s"}</span>
            <span className={styles.chip}>
              {countExercises(draft)} exercise{countExercises(draft) === 1 ? "" : "s"}
            </span>
            <span className={styles.chip}>
              {countSets(draft)} set{countSets(draft) === 1 ? "" : "s"}
            </span>
          </div>

          <div className={styles.reviewList}>
            {validation.messages.length === 0 ? (
              <p className={joinClassNames(styles.reviewItem, styles.reviewItemOk)}>
                <CheckCircle2 size={16} className={styles.reviewIcon} aria-hidden="true" />
                The plan is ready to be published.
              </p>
            ) : (
              validation.messages.map((message) => (
                <p key={message} className={joinClassNames(styles.reviewItem, styles.reviewItemError)}>
                  <AlertCircle size={16} className={styles.reviewIcon} aria-hidden="true" />
                  {message}
                </p>
              ))
            )}
          </div>

          {attemptMessage ? (
            <div className={joinClassNames(styles.notice, styles.noticeWarn)}>
              <AlertTriangle size={15} className={styles.noticeIcon} aria-hidden="true" />
              <span>{attemptMessage}</span>
            </div>
          ) : null}
        </div>
      </Admin2Section>
    );
  };

  return (
    <>
      <Admin2PageHeader
        eyebrow="Programs"
        title="Create a plan"
        description="Design a ready-made catalogue plan with the RYZE program model: weeks, training days and catalogue exercises."
        actions={
          <Button
            to="/admin2/plans"
            variant="ghost"
            size="small"
            icon={<ArrowLeft size={15} />}
            iconPosition="left"
          >
            Back to plans
          </Button>
        }
      />

      <NotSavedNotice />

      <div className={styles.topBar}>
        <div className={styles.topBarLeft}>
          {dirty ? (
            <span className={styles.unsavedPill}>
              <span aria-hidden="true">●</span> Unsaved changes
            </span>
          ) : null}
          <Button
            type="button"
            variant="ghost"
            size="small"
            icon={<RotateCcw size={14} />}
            onClick={discard}
            disabled={!dirty}
          >
            Discard
          </Button>
        </div>
      </div>

      <div className={styles.stepper} role="tablist" aria-label="Plan creation steps">
        {STEPS.map((label, index) => (
          <button
            key={label}
            type="button"
            role="tab"
            aria-selected={step === index}
            className={joinClassNames(
              styles.step,
              step === index && styles.stepActive,
              completion[index] && index !== step && styles.stepDone
            )}
            onClick={() => setStep(index)}
          >
            <span className={styles.stepIndex}>
              {completion[index] && index !== step ? <Check size={13} aria-hidden="true" /> : index + 1}
            </span>
            {label}
          </button>
        ))}
      </div>

      <div className={styles.workspace}>
        <div className={styles.formColumn}>
          {renderStep()}

          <div className={styles.submitBar}>
            {step > 0 ? (
              <Button
                type="button"
                variant="ghost"
                size="small"
                onClick={() => setStep(step - 1)}
              >
                Back
              </Button>
            ) : null}
            {step < STEPS.length - 1 ? (
              <Button
                type="button"
                size="small"
                icon={<ChevronRight size={15} />}
                onClick={() => setStep(step + 1)}
              >
                Next
              </Button>
            ) : (
              <Button
                type="button"
                size="small"
                onClick={handlePublishAttempt}
                disabled={!dirty}
              >
                {draft.status === ProgramStatusEnum.PUBLISHED ? "Publish plan" : "Save draft"}
              </Button>
            )}
          </div>
        </div>

        <div className={styles.previewColumn}>
          <p className={styles.previewCaption}>Live preview</p>
          <PlanPreview draft={draft} />
        </div>
      </div>

      {pickerTarget && pickerLocation ? (
        <ExerciseLibraryDialog
          title={pickerTarget.mode === "replace" ? "Replace exercise" : "Add exercises"}
          description={
            pickerTarget.mode === "replace"
              ? "Swap the assigned exercise while keeping its position and prescription in the day."
              : "Search the RYZE catalogue and assign exercises to this training day."
          }
          addedExerciseIds={pickerAddedIds}
          actionLabel={pickerTarget.mode === "replace" ? "Replace" : "Add to day"}
          addedLabel={pickerTarget.mode === "replace" ? "Already used" : "Already added"}
          onClose={() => setPickerTarget(null)}
          onAdd={(exercise) => {
            if (pickerTarget.mode === "replace") {
              update((current) =>
                replaceExerciseInDay(
                  current,
                  pickerLocation.weekId,
                  pickerLocation.workout.id,
                  pickerTarget.assignmentId,
                  exercise.id
                )
              );
              setPickerTarget(null);
            } else {
              update((current) =>
                addExerciseToDay(current, pickerLocation.weekId, pickerLocation.workout.id, exercise.id)
              );
            }
          }}
        />
      ) : null}

      {editorLocation ? (
        <PlanExerciseEditorModal
          assignment={editorLocation.assignment}
          exercise={editorDetail?.exercise ?? null}
          alternatives={editorDetail?.alternatives ?? []}
          onClose={closeEditor}
          onUpdateInstructions={(assignmentId, value) =>
            update((current) =>
              updateExerciseAssignment(current, editorLocation.weekId, editorLocation.workoutId, assignmentId, {
                instructions: value
              })
            )
          }
          onUpdateSet={(assignmentId, setId, patch) =>
            update((current) =>
              updateSetInExercise(current, editorLocation.weekId, editorLocation.workoutId, assignmentId, setId, patch)
            )
          }
          onRemoveSet={(assignmentId, setId) =>
            update((current) =>
              removeSetFromExercise(current, editorLocation.weekId, editorLocation.workoutId, assignmentId, setId)
            )
          }
          onAddSet={(assignmentId) =>
            update((current) =>
              addSetToExercise(current, editorLocation.weekId, editorLocation.workoutId, assignmentId)
            )
          }
          onReplace={(assignmentId) => {
            const target = editorRef.current;
            if (!target) {
              return;
            }
            setPickerTarget({
              mode: "replace",
              weekId: target.weekId,
              workoutId: target.workoutId,
              assignmentId
            });
            closeEditor();
          }}
          onRemove={(assignmentId) => {
            const target = editorRef.current;
            if (!target) {
              return;
            }
            update((current) =>
              removeExerciseFromDay(current, target.weekId, target.workoutId, assignmentId)
            );
            closeEditor();
          }}
        />
      ) : null}
    </>
  );
}