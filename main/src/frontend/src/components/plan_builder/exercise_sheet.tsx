import { Plus, Trash2, X } from "lucide-react";
import { useEffect, useState } from "react";

import type { PlanDraftExercise, PlanDraftSet } from "@/pages/admin2/admin2_plan_create_page/plan_builder_model";
import { fetchExerciseDetail, SET_TYPES, type SetType } from "@/services/exercise_library";

import type { BuilderCopy } from "./copy";
import { capitalise, ExerciseAnimation, fromCatalogEntry, type BuilderExercise } from "./exercise_media";
import { RestField } from "./rest_field";
import styles from "./plan_builder.module.css";

interface ExerciseSheetProps {
  copy: BuilderCopy;
  exercise: BuilderExercise | undefined;
  assignment: PlanDraftExercise;
  onDetails: (patch: Partial<Pick<PlanDraftExercise, "instructions" | "notes">>) => void;
  onSet: (setId: string, patch: Partial<Omit<PlanDraftSet, "id" | "set_number">>) => void;
  onAddSet: () => void;
  onRemoveSet: (setId: string) => void;
  onExerciseLoaded: (exercise: BuilderExercise) => void;
  onClose: () => void;
}

// ExerciseSheet opens from an exercise of a training day: the movement plays,
// the catalogue explains it, and every prescription field (set type, reps,
// weight, rest, RIR, RPE, tempo, client instructions and notes) is editable.
export const ExerciseSheet = ({
  copy,
  exercise,
  assignment,
  onDetails,
  onSet,
  onAddSet,
  onRemoveSet,
  onExerciseLoaded,
  onClose
}: ExerciseSheetProps) => {
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.body.style.overflow = previous;
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [onClose]);

  useEffect(() => {
    if (loaded) return;
    let cancelled = false;
    fetchExerciseDetail(assignment.exercise_id)
      .then((detail) => {
        if (!cancelled) {
          onExerciseLoaded(fromCatalogEntry(detail.exercise));
          setLoaded(true);
        }
      })
      .catch(() => setLoaded(true));
    return () => {
      cancelled = true;
    };
  }, [assignment.exercise_id, loaded, onExerciseLoaded]);

  const name = exercise?.name ?? "…";

  return (
    <div className={styles.overlay}>
      <button type="button" className={styles.backdrop} aria-label={copy.close} tabIndex={-1} onClick={onClose} />
      <div className={styles.sheet} role="dialog" aria-modal="true" aria-label={name}>
        <header className={styles.pickerHead}>
          <h2>{name}</h2>
          <button type="button" className={styles.iconButton} onClick={onClose} aria-label={copy.close} autoFocus>
            <X aria-hidden="true" />
          </button>
        </header>

        <div className={styles.sheetBody}>
          <div className={styles.sheetMedia}>
            {exercise ? <ExerciseAnimation exercise={exercise} unavailable={copy.mediaUnavailable} /> : null}
            {exercise ? (
              <dl className={styles.previewFacts}>
                {exercise.primary_muscle_group || exercise.body_part ? (
                  <div>
                    <dt>{copy.muscles}</dt>
                    <dd>{capitalise(exercise.primary_muscle_group || exercise.body_part)}</dd>
                  </div>
                ) : null}
                {exercise.equipment ? (
                  <div>
                    <dt>{copy.equipment}</dt>
                    <dd>{capitalise(exercise.equipment)}</dd>
                  </div>
                ) : null}
              </dl>
            ) : null}
            {exercise?.instruction_steps && exercise.instruction_steps.length > 0 ? (
              <>
                <h3 className={styles.previewSub}>{copy.howTo}</h3>
                <ol className={styles.howToSteps}>
                  {exercise.instruction_steps.map((step, index) => (
                    <li key={index}>{step}</li>
                  ))}
                </ol>
              </>
            ) : exercise?.description ? (
              <p className={styles.previewText}>{exercise.description}</p>
            ) : null}
          </div>

          <div className={styles.sheetForm}>
            <h3 className={styles.previewSub}>{copy.sheetPrescription}</h3>
            <div className={styles.fullSets} role="table" aria-label={copy.sheetPrescription}>
              <div className={styles.fullSetsHead} role="row">
                <span role="columnheader">{copy.set}</span>
                <span role="columnheader">{copy.setType}</span>
                <span role="columnheader">{copy.reps}</span>
                <span role="columnheader">{copy.weight}</span>
                <span role="columnheader">{copy.rest}</span>
                <span role="columnheader">{copy.rir}</span>
                <span role="columnheader">{copy.rpe}</span>
                <span role="columnheader">{copy.tempo}</span>
                <span role="columnheader" />
              </div>
              {assignment.sets.map((set) => (
                <div key={set.id} className={styles.fullSetsRow} role="row">
                  <span className={styles.setNumber} role="cell">
                    {set.set_number}
                  </span>
                  <span role="cell">
                    <select
                      className={styles.cellInput}
                      value={set.set_type}
                      aria-label={`${copy.setType} ${set.set_number}`}
                      onChange={(event) => onSet(set.id, { set_type: event.target.value as SetType })}
                    >
                      {SET_TYPES.map((type) => (
                        <option key={type} value={type}>
                          {copy.setTypes[type] ?? type}
                        </option>
                      ))}
                    </select>
                  </span>
                  <span role="cell">
                    <input
                      className={styles.cellInput}
                      inputMode="numeric"
                      value={set.reps}
                      aria-label={`${copy.reps} ${set.set_number}`}
                      onChange={(event) => onSet(set.id, { reps: event.target.value.replace(/[^\d]/g, "") })}
                    />
                  </span>
                  <span role="cell">
                    <input
                      className={styles.cellInput}
                      inputMode="decimal"
                      value={set.weight_kg}
                      aria-label={`${copy.weight} ${set.set_number}`}
                      onChange={(event) => onSet(set.id, { weight_kg: event.target.value.replace(",", ".").replace(/[^\d.]/g, "") })}
                    />
                  </span>
                  <span role="cell">
                    <RestField
                      value={set.rest_seconds}
                      label={`${copy.rest} ${set.set_number}`}
                      placeholder={copy.restPlaceholder}
                      onChange={(value) => onSet(set.id, { rest_seconds: value })}
                    />
                  </span>
                  <span role="cell">
                    <input
                      className={styles.cellInput}
                      inputMode="numeric"
                      value={set.rir}
                      aria-label={`${copy.rir} ${set.set_number}`}
                      onChange={(event) => onSet(set.id, { rir: event.target.value.replace(/[^\d]/g, "") })}
                    />
                  </span>
                  <span role="cell">
                    <input
                      className={styles.cellInput}
                      inputMode="decimal"
                      value={set.rpe}
                      aria-label={`${copy.rpe} ${set.set_number}`}
                      onChange={(event) => onSet(set.id, { rpe: event.target.value.replace(",", ".").replace(/[^\d.]/g, "") })}
                    />
                  </span>
                  <span role="cell">
                    <input
                      className={styles.cellInput}
                      value={set.tempo}
                      maxLength={20}
                      placeholder="3010"
                      aria-label={`${copy.tempo} ${set.set_number}`}
                      onChange={(event) => onSet(set.id, { tempo: event.target.value })}
                    />
                  </span>
                  <span role="cell">
                    <button
                      type="button"
                      className={styles.iconButtonSmall}
                      aria-label={copy.removeSet(set.set_number)}
                      disabled={assignment.sets.length <= 1}
                      onClick={() => onRemoveSet(set.id)}
                    >
                      <Trash2 aria-hidden="true" />
                    </button>
                  </span>
                </div>
              ))}
            </div>
            <button type="button" className={styles.ghostButton} onClick={onAddSet}>
              <Plus aria-hidden="true" /> {copy.addSet}
            </button>

            <label className={styles.field}>
              <span className={styles.label}>{copy.instructionsLabel}</span>
              <textarea
                className={styles.textarea}
                rows={3}
                maxLength={2000}
                value={assignment.instructions}
                placeholder={copy.instructionsPlaceholder}
                onChange={(event) => onDetails({ instructions: event.target.value })}
              />
            </label>
            <label className={styles.field}>
              <span className={styles.label}>{copy.coachNotes}</span>
              <textarea
                className={styles.textarea}
                rows={2}
                maxLength={2000}
                value={assignment.notes}
                placeholder={copy.notesPlaceholder}
                onChange={(event) => onDetails({ notes: event.target.value })}
              />
            </label>
          </div>
        </div>

        <footer className={styles.pickerFoot}>
          <button type="button" className={styles.primaryButton} onClick={onClose}>
            {copy.done(0)}
          </button>
        </footer>
      </div>
    </div>
  );
};
