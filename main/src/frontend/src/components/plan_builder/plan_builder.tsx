import {
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  CalendarDays,
  Check,
  CircleAlert,
  Copy as CopyIcon,
  Dumbbell,
  ImagePlus,
  Lock,
  Minus,
  Plus,
  Trash2,
  X
} from "lucide-react";
import { useCallback, useMemo, useRef, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";

import { useI18n } from "@/i18n/i18n";
import { CuratedNutritionEditor } from "@/pages/admin2/admin2_plan_create_page/curated_nutrition_editor";
import {
  addDay,
  addExercisesToDay,
  addSetToExercise,
  addWeek,
  copyWeekForward,
  countDays,
  countExercises,
  countSets,
  DAY_FOCUS_VALUES,
  moveExerciseInDay,
  removeDay,
  removeExerciseFromDay,
  removeSetFromExercise,
  removeWeek,
  syncWeeks,
  updateDayDetails,
  updateExerciseAssignment,
  updateSetInExercise,
  validatePlan,
  type PlanDraft,
  type PlanDraftExercise,
  type PlanDraftWorkout
} from "@/pages/admin2/admin2_plan_create_page/plan_builder_model";
import {
  GENERIC_TRAINING_TYPES,
  PROGRAM_OBJECTIVES,
  type GenericProgramLevel,
  type UploadedImage
} from "@/services/admin2_api";
import { joinClassNames } from "@utils/class_names";
import { publicFileUrl } from "@utils/http_client";

import { BUILDER_COPY, type BuilderCopy } from "./copy";
import { ExercisePicker } from "./exercise_picker";
import { ExerciseSheet } from "./exercise_sheet";
import { capitalise, ExerciseThumb, type BuilderExercise } from "./exercise_media";
import { RestField } from "./rest_field";
import styles from "./plan_builder.module.css";

export interface BuilderCategory {
  id: string;
  label: string;
  priceMinor: number;
  description: string;
}

type StepId = "basics" | "structure" | "nutrition" | "review";

interface PlanBuilderProps {
  heading: { kicker: string; title: string; backTo: string; backLabel: string };
  draft: PlanDraft;
  onChange: (producer: (current: PlanDraft) => PlanDraft) => void;
  categories: BuilderCategory[];
  category: string;
  // Undefined when the category can no longer change (the plan exists).
  onCategoryChange?: (id: string) => void;
  // Premium Level 1: package objective and nutrition model.
  showPackageOptions: boolean;
  packageOptionsLocked: boolean;
  hasStructure: boolean;
  hasNutrition: boolean;
  exercises: Record<string, BuilderExercise>;
  rememberExercises: (list: BuilderExercise[]) => void;
  uploadCover: (file: File) => Promise<UploadedImage>;
  status: "draft" | "published";
  saving: boolean;
  message: { tone: "error" | "success"; text: string } | null;
  onSave: (publish: boolean) => void;
  onDelete?: () => void;
  deleting?: boolean;
  lockedNote?: string;
}

const MAX_COVER_BYTES = 5 * 1024 * 1024;
const COVER_TYPES = ["image/jpeg", "image/png", "image/webp"];
const LEVELS: GenericProgramLevel[] = ["Beginner", "Intermediate", "Advanced"];

const Stepper = ({
  label,
  value,
  min,
  max,
  onChange,
  copy
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (next: number) => void;
  copy: BuilderCopy;
}) => (
  <div className={styles.stepper} role="group" aria-label={label}>
    <button type="button" aria-label={`${copy.decrease} ${label}`} disabled={value <= min} onClick={() => onChange(value - 1)}>
      <Minus aria-hidden="true" />
    </button>
    <output aria-live="polite">{value}</output>
    <button type="button" aria-label={`${copy.increase} ${label}`} disabled={value >= max} onClick={() => onChange(value + 1)}>
      <Plus aria-hidden="true" />
    </button>
  </div>
);

// PlanBuilder is the shared plan creation workflow of ADMIN_2 (generic, Level
// 1 and Level 2) and of trainers. Everything that classifies the plan is
// chosen first (category, cover, goal, difficulty, weeks, days); the weeks are
// then created automatically and filled day by day, with exercises picked from
// the catalogue and prescribed inline. Prices are fixed by category and only
// shown, never typed.
export const PlanBuilder = ({
  heading,
  draft,
  onChange,
  categories,
  category,
  onCategoryChange,
  showPackageOptions,
  packageOptionsLocked,
  hasStructure,
  hasNutrition,
  exercises,
  rememberExercises,
  uploadCover,
  status,
  saving,
  message,
  onSave,
  onDelete,
  deleting = false,
  lockedNote
}: PlanBuilderProps) => {
  const { locale } = useI18n();
  const copy = BUILDER_COPY[locale];
  const money = useMemo(
    () => new Intl.NumberFormat(locale === "pt" ? "pt-PT" : "en-IE", { style: "currency", currency: "EUR" }),
    [locale]
  );
  const priceText = (minor: number) => (minor === 0 ? copy.free : money.format(minor / 100));

  const steps = useMemo<StepId[]>(
    () => ["basics", ...(hasStructure ? (["structure"] as StepId[]) : []), ...(hasNutrition ? (["nutrition"] as StepId[]) : []), "review"],
    [hasStructure, hasNutrition]
  );
  const [stepIndex, setStepIndex] = useState(0);
  const step = steps[Math.min(stepIndex, steps.length - 1)];
  const [weekIndex, setWeekIndex] = useState(0);
  const [picker, setPicker] = useState<{ weekId: string; workoutId: string; label: string } | null>(null);
  const [sheet, setSheet] = useState<{ weekId: string; workoutId: string; assignmentId: string } | null>(null);
  const [coverState, setCoverState] = useState<"idle" | "uploading" | "error">("idle");
  const [copied, setCopied] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const goTo = (index: number) => {
    const target = steps[index];
    if (target === "structure") {
      onChange(syncWeeks);
    }
    setStepIndex(index);
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const selectedCategory = categories.find((option) => option.id === category);

  // --- Cover ---------------------------------------------------------------
  const chooseCover = async (file: File | undefined) => {
    if (!file) return;
    if (!COVER_TYPES.includes(file.type) || file.size > MAX_COVER_BYTES) {
      setCoverState("error");
      return;
    }
    setCoverState("uploading");
    try {
      const uploaded = await uploadCover(file);
      onChange((current) => ({ ...current, cover_image: uploaded.key, cover_url: uploaded.url }));
      setCoverState("idle");
    } catch {
      setCoverState("error");
    } finally {
      if (fileRef.current) fileRef.current.value = "";
    }
  };

  // --- Checklist -------------------------------------------------------------
  const checklist = useMemo(() => {
    const items: { label: string; ok: boolean; required: boolean }[] = [
      { label: copy.name, ok: draft.name.trim() !== "", required: true },
      { label: copy.description, ok: draft.description.trim() !== "", required: true },
      { label: copy.coverTitle, ok: draft.cover_image !== "", required: false }
    ];
    if (hasStructure) {
      items.push({ label: copy.goal, ok: draft.training_type !== "", required: false });
      items.push({ label: copy.level, ok: draft.level !== "", required: false });
      items.push({ label: copy.stats.exercises, ok: countExercises(draft) > 0, required: true });
    }
    if (showPackageOptions && draft.nutrition_mode === "curated") {
      items.push({ label: copy.objective, ok: draft.objective !== "", required: true });
    }
    return items;
  }, [copy, draft, hasStructure, showPackageOptions]);

  // Nutrition problems come from the shared validation (English wording).
  const nutritionProblems = useMemo(
    () =>
      hasNutrition
        ? validatePlan(draft).messages.filter((text) => /nutrition|meal|hydration|food/i.test(text))
        : [],
    [draft, hasNutrition]
  );
  const ready = checklist.every((item) => !item.required || item.ok) && nutritionProblems.length === 0;
  const canSaveDraft = draft.name.trim() !== "" && draft.description.trim() !== "";

  // --- Structure helpers -------------------------------------------------------
  const week = draft.weeks[Math.min(weekIndex, Math.max(draft.weeks.length - 1, 0))];

  const findAssignment = (weekId: string, workoutId: string, assignmentId: string) => {
    const workout = draft.weeks.find((entry) => entry.id === weekId)?.workouts.find((entry) => entry.id === workoutId);
    return workout?.exercises.find((entry) => entry.id === assignmentId);
  };

  const rememberOne = useCallback((exercise: BuilderExercise) => rememberExercises([exercise]), [rememberExercises]);

  const renderExercise = (weekId: string, workout: PlanDraftWorkout, assignment: PlanDraftExercise, index: number) => {
    const exercise = exercises[assignment.exercise_id];
    const name = exercise?.name ?? "…";
    return (
      <li key={assignment.id} className={styles.exercise}>
        <div className={styles.exerciseHead}>
          <button
            type="button"
            className={styles.exerciseOpen}
            onClick={() => setSheet({ weekId, workoutId: workout.id, assignmentId: assignment.id })}
            aria-label={copy.openExercise(name)}
          >
            <ExerciseThumb exercise={exercise} size="md" />
            <span className={styles.exerciseName}>
              <strong>{name}</strong>
              <span>{capitalise(exercise?.primary_muscle_group || exercise?.body_part || "")}</span>
            </span>
          </button>
          <div className={styles.exerciseTools}>
            <button
              type="button"
              className={styles.iconButtonSmall}
              aria-label={copy.moveUp}
              disabled={index === 0}
              onClick={() => onChange((current) => moveExerciseInDay(current, weekId, workout.id, assignment.id, -1))}
            >
              <ArrowUp aria-hidden="true" />
            </button>
            <button
              type="button"
              className={styles.iconButtonSmall}
              aria-label={copy.moveDown}
              disabled={index === workout.exercises.length - 1}
              onClick={() => onChange((current) => moveExerciseInDay(current, weekId, workout.id, assignment.id, 1))}
            >
              <ArrowDown aria-hidden="true" />
            </button>
            <button
              type="button"
              className={styles.iconButtonSmall}
              aria-label={`${copy.removeExercise}: ${name}`}
              onClick={() => onChange((current) => removeExerciseFromDay(current, weekId, workout.id, assignment.id))}
            >
              <Trash2 aria-hidden="true" />
            </button>
          </div>
        </div>

        <div className={styles.sets} role="table" aria-label={`${name} — ${copy.sheetPrescription}`}>
          <div className={styles.setsHead} role="row">
            <span role="columnheader">{copy.set}</span>
            <span role="columnheader">{copy.reps}</span>
            <span role="columnheader">{copy.weight}</span>
            <span role="columnheader">{copy.rest}</span>
            <span role="columnheader">{copy.rir}</span>
            <span role="columnheader" />
          </div>
          {assignment.sets.map((set) => (
            <div key={set.id} className={styles.setsRow} role="row">
              <span className={styles.setNumber} role="cell">
                {set.set_type === "warmup" ? "W" : set.set_number}
              </span>
              <span role="cell">
                <input
                  className={styles.cellInput}
                  inputMode="numeric"
                  placeholder="—"
                  value={set.reps}
                  aria-label={`${copy.reps}, ${copy.set} ${set.set_number}`}
                  onChange={(event) =>
                    onChange((current) =>
                      updateSetInExercise(current, weekId, workout.id, assignment.id, set.id, {
                        reps: event.target.value.replace(/[^\d]/g, "")
                      })
                    )
                  }
                />
              </span>
              <span role="cell">
                <input
                  className={styles.cellInput}
                  inputMode="decimal"
                  placeholder="—"
                  value={set.weight_kg}
                  aria-label={`${copy.weight}, ${copy.set} ${set.set_number}`}
                  onChange={(event) =>
                    onChange((current) =>
                      updateSetInExercise(current, weekId, workout.id, assignment.id, set.id, {
                        weight_kg: event.target.value.replace(",", ".").replace(/[^\d.]/g, "")
                      })
                    )
                  }
                />
              </span>
              <span role="cell">
                <RestField
                  value={set.rest_seconds}
                  label={`${copy.rest}, ${copy.set} ${set.set_number}`}
                  placeholder={copy.restPlaceholder}
                  onChange={(value) =>
                    onChange((current) =>
                      updateSetInExercise(current, weekId, workout.id, assignment.id, set.id, { rest_seconds: value })
                    )
                  }
                />
              </span>
              <span role="cell">
                <input
                  className={styles.cellInput}
                  inputMode="numeric"
                  placeholder="—"
                  value={set.rir}
                  aria-label={`${copy.rir}, ${copy.set} ${set.set_number}`}
                  onChange={(event) =>
                    onChange((current) =>
                      updateSetInExercise(current, weekId, workout.id, assignment.id, set.id, {
                        rir: event.target.value.replace(/[^\d]/g, "")
                      })
                    )
                  }
                />
              </span>
              <span role="cell">
                <button
                  type="button"
                  className={styles.iconButtonSmall}
                  aria-label={copy.removeSet(set.set_number)}
                  disabled={assignment.sets.length <= 1}
                  onClick={() => onChange((current) => removeSetFromExercise(current, weekId, workout.id, assignment.id, set.id))}
                >
                  <X aria-hidden="true" />
                </button>
              </span>
            </div>
          ))}
        </div>
        <div className={styles.exerciseFoot}>
          <button
            type="button"
            className={styles.textButton}
            onClick={() => onChange((current) => addSetToExercise(current, weekId, workout.id, assignment.id))}
          >
            <Plus aria-hidden="true" /> {copy.addSet}
          </button>
          <input
            className={styles.noteInput}
            value={assignment.notes}
            maxLength={2000}
            placeholder={copy.notesPlaceholder}
            aria-label={`${copy.notes}: ${name}`}
            onChange={(event) =>
              onChange((current) =>
                updateExerciseAssignment(current, weekId, workout.id, assignment.id, { notes: event.target.value })
              )
            }
          />
        </div>
      </li>
    );
  };

  const renderDay = (weekId: string, workout: PlanDraftWorkout, index: number) => {
    const dayLabel = copy.day(index + 1);
    return (
      <article key={workout.id} className={styles.day} aria-label={workout.name || dayLabel}>
        <header className={styles.dayHead}>
          <span className={styles.dayBadge}>{dayLabel}</span>
          <input
            className={styles.dayName}
            value={workout.name}
            maxLength={80}
            placeholder={copy.dayNamePlaceholder}
            aria-label={`${copy.dayName} — ${dayLabel}`}
            onChange={(event) =>
              onChange((current) => updateDayDetails(current, weekId, workout.id, { name: event.target.value }))
            }
          />
          <button
            type="button"
            className={styles.iconButtonSmall}
            aria-label={`${copy.removeDay}: ${dayLabel}`}
            onClick={() => onChange((current) => removeDay(current, weekId, workout.id))}
          >
            <Trash2 aria-hidden="true" />
          </button>
        </header>

        <div className={styles.focusRow} role="radiogroup" aria-label={`${copy.dayFocus} — ${dayLabel}`}>
          {DAY_FOCUS_VALUES.map((focus) => (
            <button
              key={focus}
              type="button"
              role="radio"
              aria-checked={workout.focus === focus}
              className={styles.focusChip}
              onClick={() =>
                onChange((current) =>
                  updateDayDetails(current, weekId, workout.id, { focus: workout.focus === focus ? "" : focus })
                )
              }
            >
              {copy.focus[focus]}
            </button>
          ))}
        </div>

        <textarea
          className={styles.dayDescription}
          rows={2}
          maxLength={1000}
          value={workout.description}
          placeholder={copy.dayDescriptionPlaceholder}
          aria-label={`${copy.dayDescription} — ${dayLabel}`}
          onChange={(event) =>
            onChange((current) => updateDayDetails(current, weekId, workout.id, { description: event.target.value }))
          }
        />

        {workout.exercises.length === 0 ? (
          <p className={styles.emptyDay}>{copy.noExercises}</p>
        ) : (
          <ol className={styles.exercises}>
            {workout.exercises.map((assignment, position) => renderExercise(weekId, workout, assignment, position))}
          </ol>
        )}

        <button
          type="button"
          className={styles.addExercise}
          onClick={() => setPicker({ weekId, workoutId: workout.id, label: workout.name.trim() || dayLabel })}
        >
          <Plus aria-hidden="true" /> {copy.addExercise}
        </button>
      </article>
    );
  };

  // --- Steps -------------------------------------------------------------------
  const basics = (
    <div className={styles.stepBody}>
      <section className={styles.panel} aria-labelledby="builder-category">
        <h2 className={styles.panelTitle} id="builder-category">
          {copy.categoryTitle}
        </h2>
        <div className={styles.categories} role="radiogroup" aria-labelledby="builder-category">
          {categories.map((option) => {
            const selected = option.id === category;
            const disabled = !onCategoryChange && !selected;
            return (
              <button
                key={option.id}
                type="button"
                role="radio"
                aria-checked={selected}
                disabled={disabled}
                className={joinClassNames(styles.category, selected && styles.categoryActive)}
                onClick={() => onCategoryChange?.(option.id)}
              >
                <span className={styles.categoryPrice}>{priceText(option.priceMinor)}</span>
                <strong>{option.label}</strong>
                <span>{option.description}</span>
                {selected ? <Check className={styles.categoryCheck} aria-hidden="true" /> : null}
              </button>
            );
          })}
        </div>
        {!onCategoryChange ? (
          <p className={styles.hint}>
            <Lock aria-hidden="true" /> {copy.categoryLocked}
          </p>
        ) : null}
      </section>

      <div className={styles.basicsGrid}>
        <section className={styles.panel} aria-labelledby="builder-cover">
          <h2 className={styles.panelTitle} id="builder-cover">
            {copy.coverTitle}
          </h2>
          <div className={styles.cover}>
            {draft.cover_url ? (
              <img src={publicFileUrl(draft.cover_url)} alt="" className={styles.coverImage} />
            ) : (
              <button type="button" className={styles.coverEmpty} onClick={() => fileRef.current?.click()}>
                <ImagePlus aria-hidden="true" />
                <strong>{copy.coverChoose}</strong>
                <span>{copy.coverTypes}</span>
              </button>
            )}
            {coverState === "uploading" ? <span className={styles.coverBusy}>{copy.coverUploading}</span> : null}
          </div>
          <input
            ref={fileRef}
            type="file"
            accept={COVER_TYPES.join(",")}
            className={styles.srOnly}
            tabIndex={-1}
            aria-label={copy.coverChoose}
            onChange={(event) => void chooseCover(event.target.files?.[0])}
          />
          <div className={styles.coverActions}>
            <button type="button" className={styles.ghostButton} disabled={coverState === "uploading"} onClick={() => fileRef.current?.click()}>
              <ImagePlus aria-hidden="true" /> {draft.cover_url ? copy.coverReplace : copy.coverChoose}
            </button>
            {draft.cover_url ? (
              <button
                type="button"
                className={styles.textButton}
                onClick={() => onChange((current) => ({ ...current, cover_image: "", cover_url: "" }))}
              >
                {copy.coverRemove}
              </button>
            ) : null}
          </div>
          <p className={styles.hint}>{coverState === "error" ? <span className={styles.errorText}>{copy.coverError}</span> : copy.coverHint}</p>
        </section>

        <section className={styles.panel} aria-label={copy.name}>
          <label className={styles.field}>
            <span className={styles.label}>{copy.name}</span>
            <input
              className={styles.bigInput}
              value={draft.name}
              maxLength={255}
              placeholder={copy.namePlaceholder}
              onChange={(event) => onChange((current) => ({ ...current, name: event.target.value }))}
            />
          </label>
          <label className={styles.field}>
            <span className={styles.label}>
              {copy.description}
              <em className={styles.counter}>{draft.description.length}/5000</em>
            </span>
            <textarea
              className={styles.textarea}
              rows={5}
              maxLength={5000}
              value={draft.description}
              placeholder={copy.descriptionPlaceholder}
              onChange={(event) => onChange((current) => ({ ...current, description: event.target.value }))}
            />
          </label>
        </section>
      </div>

      {hasStructure ? (
        <section className={styles.panel} aria-labelledby="builder-shape">
          <h2 className={styles.panelTitle} id="builder-shape">
            {copy.goal} · {copy.level}
          </h2>
          <div className={styles.field}>
            <span className={styles.label}>{copy.goal}</span>
            <div className={styles.choiceRow} role="radiogroup" aria-label={copy.goal}>
              {GENERIC_TRAINING_TYPES.map((goal) => (
                <button
                  key={goal}
                  type="button"
                  role="radio"
                  aria-checked={draft.training_type === goal}
                  className={styles.choice}
                  onClick={() =>
                    onChange((current) => ({ ...current, training_type: current.training_type === goal ? "" : goal }))
                  }
                >
                  {copy.goals[goal] ?? goal}
                </button>
              ))}
            </div>
          </div>
          <div className={styles.field}>
            <span className={styles.label}>{copy.level}</span>
            <div className={styles.choiceRow} role="radiogroup" aria-label={copy.level}>
              {LEVELS.map((level) => (
                <button
                  key={level}
                  type="button"
                  role="radio"
                  aria-checked={draft.level === level}
                  className={styles.choice}
                  onClick={() => onChange((current) => ({ ...current, level: current.level === level ? "" : level }))}
                >
                  {copy.levels[level]}
                </button>
              ))}
            </div>
          </div>
          <div className={styles.countGrid}>
            <div className={styles.field}>
              <span className={styles.label}>
                <CalendarDays aria-hidden="true" /> {copy.weeks}
              </span>
              <Stepper
                label={copy.weeks}
                value={Number(draft.duration_weeks) || 1}
                min={1}
                max={52}
                copy={copy}
                onChange={(next) => onChange((current) => ({ ...current, duration_weeks: String(next) }))}
              />
              <span className={styles.hint}>{copy.weeksHint}</span>
            </div>
            <div className={styles.field}>
              <span className={styles.label}>
                <Dumbbell aria-hidden="true" /> {copy.daysPerWeek}
              </span>
              <Stepper
                label={copy.daysPerWeek}
                value={Number(draft.frequency_per_week) || 3}
                min={1}
                max={7}
                copy={copy}
                onChange={(next) => onChange((current) => ({ ...current, frequency_per_week: String(next) }))}
              />
              <span className={styles.hint}>{copy.daysHint}</span>
            </div>
          </div>
        </section>
      ) : (
        <p className={styles.note}>{copy.level2Note}</p>
      )}

      {showPackageOptions ? (
        <section className={styles.panel} aria-labelledby="builder-package">
          <h2 className={styles.panelTitle} id="builder-package">
            {copy.objective} · {copy.nutritionMode}
          </h2>
          <div className={styles.choiceRow} role="radiogroup" aria-label={copy.nutritionMode}>
            {(["curated", "questionnaire_generated"] as const).map((mode) => (
              <button
                key={mode}
                type="button"
                role="radio"
                aria-checked={draft.nutrition_mode === mode}
                disabled={packageOptionsLocked}
                className={joinClassNames(styles.choice, styles.choiceWide)}
                onClick={() =>
                  onChange((current) => ({
                    ...current,
                    nutrition_mode: mode,
                    objective: mode === "curated" ? current.objective : ""
                  }))
                }
              >
                <strong>{mode === "curated" ? copy.nutritionCurated : copy.nutritionQuestionnaire}</strong>
                <span>{mode === "curated" ? copy.nutritionCuratedText : copy.nutritionQuestionnaireText}</span>
              </button>
            ))}
          </div>
          {draft.nutrition_mode === "curated" ? (
            <div className={styles.field}>
              <span className={styles.label}>{copy.objective}</span>
              <div className={styles.choiceRow} role="radiogroup" aria-label={copy.objective}>
                {PROGRAM_OBJECTIVES.map((objective) => (
                  <button
                    key={objective}
                    type="button"
                    role="radio"
                    aria-checked={draft.objective === objective}
                    disabled={packageOptionsLocked}
                    className={styles.choice}
                    onClick={() => onChange((current) => ({ ...current, objective }))}
                  >
                    {copy.objectives[objective] ?? objective}
                  </button>
                ))}
              </div>
            </div>
          ) : null}
          {packageOptionsLocked ? (
            <p className={styles.hint}>
              <Lock aria-hidden="true" /> {copy.metadataLocked}
            </p>
          ) : null}
        </section>
      ) : null}
    </div>
  );

  const structure = week ? (
    <div className={styles.stepBody}>
      <div className={styles.weekBar}>
        <div className={styles.weekTabs} role="tablist" aria-label={copy.weeks}>
          {draft.weeks.map((entry, index) => {
            const exercisesInWeek = entry.workouts.reduce((sum, workout) => sum + workout.exercises.length, 0);
            return (
              <button
                key={entry.id}
                type="button"
                role="tab"
                aria-selected={entry.id === week.id}
                className={styles.weekTab}
                onClick={() => {
                  setWeekIndex(index);
                  setCopied(false);
                }}
              >
                <strong>{copy.week(index + 1)}</strong>
                <span>
                  {entry.workouts.length} · {exercisesInWeek}
                </span>
              </button>
            );
          })}
          <button
            type="button"
            className={joinClassNames(styles.weekTab, styles.weekTabAdd)}
            aria-label={copy.addWeek}
            disabled={draft.weeks.length >= 52}
            onClick={() => {
              onChange(addWeek);
              setWeekIndex(draft.weeks.length);
            }}
          >
            <Plus aria-hidden="true" />
          </button>
        </div>
        <div className={styles.weekActions}>
          {weekIndex < draft.weeks.length - 1 ? (
            <button
              type="button"
              className={styles.ghostButton}
              onClick={() => {
                onChange((current) => copyWeekForward(current, week.id));
                setCopied(true);
              }}
            >
              <CopyIcon aria-hidden="true" /> {copy.copyForward}
            </button>
          ) : null}
          {draft.weeks.length > 1 ? (
            <button
              type="button"
              className={styles.textButton}
              onClick={() => {
                onChange((current) => removeWeek(current, week.id));
                setWeekIndex(Math.max(0, weekIndex - 1));
              }}
            >
              <Trash2 aria-hidden="true" /> {copy.removeWeek}
            </button>
          ) : null}
        </div>
      </div>
      {copied ? (
        <p className={styles.success} role="status">
          <Check aria-hidden="true" /> {copy.copyForwardDone}
        </p>
      ) : null}

      <div className={styles.days} role="tabpanel" aria-label={copy.week(weekIndex + 1)}>
        {week.workouts.map((workout, index) => renderDay(week.id, workout, index))}
        <button
          type="button"
          className={styles.addDay}
          disabled={week.workouts.length >= 14}
          onClick={() => onChange((current) => addDay(current, week.id))}
        >
          <Plus aria-hidden="true" /> {copy.addDay}
        </button>
      </div>
    </div>
  ) : null;

  const nutrition =
    hasNutrition && draft.curated_nutrition ? (
      <div className={styles.stepBody}>
        <section className={styles.panel}>
          <CuratedNutritionEditor
            nutrition={draft.curated_nutrition}
            onChange={(next) => onChange((current) => ({ ...current, curated_nutrition: next }))}
          />
        </section>
      </div>
    ) : null;

  const review = (
    <div className={styles.stepBody}>
      <div className={styles.reviewGrid}>
        <section className={styles.summary} aria-labelledby="builder-summary">
          <div className={styles.summaryCover}>
            {draft.cover_url ? <img src={publicFileUrl(draft.cover_url)} alt="" /> : <Dumbbell aria-hidden="true" />}
            {selectedCategory ? <span className={styles.summaryTag}>{selectedCategory.label}</span> : null}
          </div>
          <div className={styles.summaryBody}>
            <h2 id="builder-summary">{draft.name.trim() || copy.namePlaceholder}</h2>
            <p>{draft.description.trim() || copy.descriptionPlaceholder}</p>
            <div className={styles.summaryChips}>
              {draft.training_type ? <span>{copy.goals[draft.training_type] ?? draft.training_type}</span> : null}
              {draft.level ? <span>{copy.levels[draft.level]}</span> : null}
              {hasStructure ? <span>{copy.weeksCount(draft.weeks.length || Number(draft.duration_weeks) || 1)}</span> : null}
              <span className={styles.statusChip} data-status={status}>
                {status === "published" ? copy.published : copy.draft}
              </span>
            </div>
            <dl className={styles.priceLine}>
              <dt>{copy.price}</dt>
              <dd>
                {selectedCategory ? priceText(selectedCategory.priceMinor) : "—"} <em>{copy.fixedPrice}</em>
              </dd>
            </dl>
          </div>
        </section>

        <div className={styles.reviewSide}>
          {hasStructure ? (
            <dl className={styles.stats}>
              <div>
                <dt>{copy.stats.weeks}</dt>
                <dd>{draft.weeks.length}</dd>
              </div>
              <div>
                <dt>{copy.stats.days}</dt>
                <dd>{countDays(draft)}</dd>
              </div>
              <div>
                <dt>{copy.stats.exercises}</dt>
                <dd>{countExercises(draft)}</dd>
              </div>
              <div>
                <dt>{copy.stats.sets}</dt>
                <dd>{countSets(draft)}</dd>
              </div>
            </dl>
          ) : null}

          <section className={styles.panel} aria-labelledby="builder-checklist">
            <h2 className={styles.panelTitle} id="builder-checklist">
              {copy.checklist}
            </h2>
            <ul className={styles.checklist}>
              {checklist.map((item) => (
                <li key={item.label} data-ok={item.ok} data-required={item.required}>
                  {item.ok ? <Check aria-hidden="true" /> : <CircleAlert aria-hidden="true" />}
                  <span>{item.label}</span>
                </li>
              ))}
              {nutritionProblems.map((problem) => (
                <li key={problem} data-ok={false} data-required>
                  <CircleAlert aria-hidden="true" />
                  <span>{problem}</span>
                </li>
              ))}
            </ul>
            {ready ? <p className={styles.success}>{copy.ready}</p> : null}
          </section>
        </div>
      </div>

      {hasStructure && draft.weeks.length > 0 ? (
        <section className={styles.panel} aria-label={copy.steps.structure}>
          <ol className={styles.overview}>
            {draft.weeks.map((entry, index) => (
              <li key={entry.id}>
                <strong>{copy.week(index + 1)}</strong>
                <ul>
                  {entry.workouts.map((workout, position) => (
                    <li key={workout.id}>
                      <span>{workout.name.trim() || copy.day(position + 1)}</span>
                      {workout.focus ? <em>{copy.focus[workout.focus]}</em> : null}
                      <b>{workout.exercises.length > 0 ? `${workout.exercises.length} ×` : copy.restDay}</b>
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ol>
        </section>
      ) : null}
    </div>
  );

  const content: Record<StepId, ReactNode> = { basics, structure, nutrition, review };
  const isLast = stepIndex === steps.length - 1;

  const pickerWorkout = picker
    ? draft.weeks.find((entry) => entry.id === picker.weekId)?.workouts.find((entry) => entry.id === picker.workoutId)
    : undefined;
  const sheetAssignment = sheet ? findAssignment(sheet.weekId, sheet.workoutId, sheet.assignmentId) : undefined;

  return (
    <div className={styles.builder}>
      <header className={styles.header}>
        <Link className={styles.back} to={heading.backTo}>
          <ArrowLeft aria-hidden="true" /> {heading.backLabel}
        </Link>
        <p className={styles.kicker}>{heading.kicker}</p>
        <h1 className={styles.title}>{heading.title}</h1>
      </header>

      <nav className={styles.steps} aria-label={copy.stepOf(stepIndex + 1, steps.length)}>
        <ol>
          {steps.map((id, index) => (
            <li key={id}>
              <button
                type="button"
                aria-current={index === stepIndex ? "step" : undefined}
                className={joinClassNames(styles.stepButton, index < stepIndex && styles.stepDone)}
                onClick={() => goTo(index)}
              >
                <span className={styles.stepIndex}>{index < stepIndex ? <Check aria-hidden="true" /> : index + 1}</span>
                <span>{copy.steps[id]}</span>
              </button>
            </li>
          ))}
        </ol>
      </nav>

      {lockedNote ? (
        <p className={styles.lockedNote}>
          <Lock aria-hidden="true" /> {lockedNote}
        </p>
      ) : null}

      {content[step]}

      {message ? (
        <p className={message.tone === "error" ? styles.errorBox : styles.success} role={message.tone === "error" ? "alert" : "status"}>
          {message.text}
        </p>
      ) : null}

      <footer className={styles.footer}>
        {stepIndex > 0 ? (
          <button type="button" className={styles.ghostButton} onClick={() => goTo(stepIndex - 1)}>
            <ArrowLeft aria-hidden="true" /> {copy.back}
          </button>
        ) : (
          <span />
        )}
        <div className={styles.footerActions}>
          {isLast && onDelete ? (
            <button
              type="button"
              className={styles.dangerButton}
              disabled={deleting || saving}
              onClick={() => {
                if (window.confirm(copy.deleteConfirm)) onDelete();
              }}
            >
              <Trash2 aria-hidden="true" /> {deleting ? copy.deleting : copy.deletePlan}
            </button>
          ) : null}
          {isLast ? (
            <>
              {status === "draft" ? (
                <button type="button" className={styles.ghostButton} disabled={saving || !canSaveDraft || Boolean(lockedNote)} onClick={() => onSave(false)}>
                  {saving ? copy.saving : copy.saveDraft}
                </button>
              ) : null}
              <button
                type="button"
                className={styles.primaryButton}
                disabled={saving || !ready || Boolean(lockedNote)}
                onClick={() => onSave(true)}
              >
                <Check aria-hidden="true" /> {saving ? copy.saving : status === "published" ? copy.saveChanges : copy.publish}
              </button>
            </>
          ) : (
            <>
              {status === "draft" && canSaveDraft ? (
                <button type="button" className={styles.textButton} disabled={saving || Boolean(lockedNote)} onClick={() => onSave(false)}>
                  {saving ? copy.saving : copy.saveDraft}
                </button>
              ) : null}
              <button type="button" className={styles.primaryButton} onClick={() => goTo(stepIndex + 1)}>
                {copy.next} <ArrowRight aria-hidden="true" />
              </button>
            </>
          )}
        </div>
      </footer>

      {picker && pickerWorkout ? (
        <ExercisePicker
          copy={copy}
          dayLabel={picker.label}
          addedIds={pickerWorkout.exercises.map((assignment) => assignment.exercise_id)}
          onAdd={(exercise) => {
            rememberExercises([exercise]);
            onChange((current) => addExercisesToDay(current, picker.weekId, picker.workoutId, [exercise.id]));
          }}
          onClose={() => setPicker(null)}
        />
      ) : null}

      {sheet && sheetAssignment ? (
        <ExerciseSheet
          copy={copy}
          exercise={exercises[sheetAssignment.exercise_id]}
          assignment={sheetAssignment}
          onExerciseLoaded={rememberOne}
          onDetails={(patch) =>
            onChange((current) => updateExerciseAssignment(current, sheet.weekId, sheet.workoutId, sheet.assignmentId, patch))
          }
          onSet={(setId, patch) =>
            onChange((current) => updateSetInExercise(current, sheet.weekId, sheet.workoutId, sheet.assignmentId, setId, patch))
          }
          onAddSet={() => onChange((current) => addSetToExercise(current, sheet.weekId, sheet.workoutId, sheet.assignmentId))}
          onRemoveSet={(setId) =>
            onChange((current) => removeSetFromExercise(current, sheet.weekId, sheet.workoutId, sheet.assignmentId, setId))
          }
          onClose={() => setSheet(null)}
        />
      ) : null}
    </div>
  );
};
