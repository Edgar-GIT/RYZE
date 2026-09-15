import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import { useHistory } from "react-router-dom";
import {
  AlertCircle,
  AlertTriangle,
  ArrowLeft,
  Check,
  CheckCircle2,
  ChevronRight,
  Info,
  Plus,
  RotateCcw,
  Search,
  Trash2,
  X
} from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2Modal } from "@/components/admin2/admin2_modal/admin2_modal";
import { Button } from "@/components/button/button";
import { joinClassNames } from "@utils/class_names";
import {
  Exercise,
  fetchExercises,
  ProgramStatusEnum,
  ProgramType,
  type ProgramTypeValue
} from "@/services/admin2_api";
import { PlanPreview } from "./plan_preview";
import {
  addDay,
  addExerciseToDay,
  addWeek,
  countDays,
  countExercises,
  countWeeks,
  createPlanDraft,
  removeDay,
  removeExerciseFromDay,
  removeWeek,
  validatePlan,
  type PlanDraft,
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

const ExercisePickerDialog = ({
  catalog,
  onClose,
  onPick
}: {
  catalog: Exercise[];
  onClose: () => void;
  onPick: (exerciseId: string) => void;
}) => {
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) {
      return catalog;
    }
    return catalog.filter((exercise) =>
      [exercise.name, exercise.target_muscles, exercise.equipment, exercise.difficulty]
        .filter(Boolean)
        .some((value) => value.toLowerCase().includes(q))
    );
  }, [catalog, query]);

  return (
    <Admin2Modal
      title="Add exercises"
      description="Pick exercises from the RYZE catalog. They are assigned to the selected training day."
      onClose={onClose}
    >
      <div className={styles.formRow}>
        <label className={styles.formLabel} htmlFor="exercise-search">
          Search the catalog
        </label>
        <div className={styles.pickerSearch}>
          <input
            id="exercise-search"
            className={styles.formInput}
            type="search"
            placeholder="Name, muscle group, equipment…"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            aria-label="Search exercises"
          />
        </div>
      </div>

      <div className={styles.pickerResults}>
        {filtered.length === 0 ? (
          <p className={styles.pickerEmpty}>No exercises match your search.</p>
        ) : (
          filtered.map((exercise) => (
            <div key={exercise.id} className={styles.pickerItem}>
              <div className={styles.pickerItemBody}>
                <p className={styles.pickerItemName}>{exercise.name}</p>
                <p className={styles.pickerItemMeta}>
                  {[exercise.target_muscles, exercise.equipment, exercise.difficulty]
                    .filter(Boolean)
                    .join(" · ") || "Generic exercise"}
                </p>
              </div>
              <Button variant="secondary" size="small" onClick={() => onPick(exercise.id)}>
                Add
              </Button>
            </div>
          ))
        )}
      </div>

      <div className={styles.submitBar}>
        <Button type="button" variant="ghost" size="small" onClick={onClose}>
          Done
        </Button>
      </div>
    </Admin2Modal>
  );
};

const WeekEditor = ({
  week,
  exerciseMap,
  onRemoveWeek,
  onAddDay,
  onRemoveDay,
  onOpenPicker,
  onRemoveExercise
}: {
  week: PlanDraftWeek;
  exerciseMap: Record<string, Exercise>;
  onRemoveWeek: () => void;
  onAddDay: () => void;
  onRemoveDay: (workoutId: string) => void;
  onOpenPicker: (workoutId: string) => void;
  onRemoveExercise: (workoutId: string, exerciseId: string) => void;
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
          onOpenPicker={onOpenPicker}
          onRemoveExercise={onRemoveExercise}
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
  onOpenPicker,
  onRemoveExercise
}: {
  workout: PlanDraftWorkout;
  exerciseMap: Record<string, Exercise>;
  onRemoveDay: (workoutId: string) => void;
  onOpenPicker: (workoutId: string) => void;
  onRemoveExercise: (workoutId: string, exerciseId: string) => void;
}) => {
  const removeExercise = (exerciseId: string) => {
    onRemoveExercise(workout.id, exerciseId);
  };

  return (
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

      <div className={styles.chips}>
        {workout.exercise_ids.length === 0 ? (
          <p className={styles.chipsEmpty}>No exercises assigned yet.</p>
        ) : (
          workout.exercise_ids.map((exerciseId) => (
            <span key={exerciseId} className={styles.chip}>
              <span className={styles.chipName}>
                {exerciseMap[exerciseId]?.name ?? "Exercise"}
              </span>
              <button
                type="button"
                className={styles.chipRemove}
                onClick={() => removeExercise(exerciseId)}
                aria-label={`Remove ${exerciseMap[exerciseId]?.name ?? "exercise"}`}
              >
                <X size={13} aria-hidden="true" />
              </button>
            </span>
          ))
        )}
      </div>

      <div className={styles.addRow}>
        <Button
          type="button"
          variant="ghost"
          size="small"
          icon={<Search size={14} />}
          onClick={() => onOpenPicker(workout.id)}
        >
          Add exercises
        </Button>
      </div>
    </div>
  );
};

export default function Admin2PlanCreatePage() {
  const history = useHistory();
  const [draft, setDraft] = useState<PlanDraft>(createPlanDraft);
  const [step, setStep] = useState(0);
  const [dirty, setDirty] = useState(false);
  const [catalog, setCatalog] = useState<Exercise[]>([]);
  const [exerciseMap, setExerciseMap] = useState<Record<string, Exercise>>({});
  const [pickerTarget, setPickerTarget] = useState<{ workoutId: string } | null>(null);
  const [attemptMessage, setAttemptMessage] = useState("");

  const update = useCallback((producer: (current: PlanDraft) => PlanDraft) => {
    setDraft((current) => producer(current));
    setDirty(true);
    setAttemptMessage("");
  }, []);

  useEffect(() => {
    let cancelled = false;

    fetchExercises(1, 100)
      .then((result) => {
        if (cancelled) {
          return;
        }
        setCatalog(result.exercises);
        const map: Record<string, Exercise> = {};
        result.exercises.forEach((exercise) => {
          map[exercise.id] = exercise;
        });
        setExerciseMap(map);
      })
      .catch(() => {
        if (!cancelled) {
          setCatalog([]);
        }
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

  const pickerTargetWorkout = useMemo(() => {
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
                  onOpenPicker={(workoutId) => setPickerTarget({ workoutId })}
                  onRemoveExercise={(workoutId, exerciseId) =>
                    update((current) =>
                      removeExerciseFromDay(current, week.id, workoutId, exerciseId)
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

      {pickerTarget && pickerTargetWorkout ? (
        <ExercisePickerDialog
          catalog={catalog}
          onClose={() => setPickerTarget(null)}
          onPick={(exerciseId) =>
            update((current) =>
              addExerciseToDay(current, pickerTargetWorkout.weekId, pickerTarget.workoutId, exerciseId)
            )
          }
        />
      ) : null}
    </>
  );
}