import { Fragment } from "react";
import { Plus, Repeat, Trash2 } from "lucide-react";

import { Button } from "@/components/button/button";
import { Admin2Modal } from "@/components/admin2/admin2_modal/admin2_modal";
import { ExercisePreviewBody } from "@/components/exercise_library/exercise_preview_body";
import { joinClassNames } from "@utils/class_names";
import {
  SET_TYPES,
  SET_TYPE_LABELS,
  type ExerciseAlternative,
  type ExerciseCatalogEntry,
  type SetType
} from "@/services/exercise_library";
import { type PlanDraftExercise, type PlanDraftSet } from "./plan_builder_model";

import styles from "./admin2_plan_create_page.module.css";

interface PlanSetGridProps {
  assignment: PlanDraftExercise;
  onUpdateSet: (
    assignmentId: string,
    setId: string,
    patch: Partial<Omit<PlanDraftSet, "id" | "set_number">>
  ) => void;
  onRemoveSet: (assignmentId: string, setId: string) => void;
  onAddSet: (assignmentId: string) => void;
}

// PlanSetGrid renders the per-set prescription table shared by the build
// workflow: one normalized set per row with typed training fields. It keeps
// its behaviour inert — every mutation is delegated upwards.
export const PlanSetGrid = ({ assignment, onUpdateSet, onRemoveSet, onAddSet }: PlanSetGridProps) => (
  <>
    <div className={styles.setGrid}>
      <span className={styles.setHeader}>Set</span>
      <span className={styles.setHeader}>Type</span>
      <span className={styles.setHeader}>Reps</span>
      <span className={styles.setHeader}>Weight (kg)</span>
      <span className={styles.setHeader}>RIR</span>
      <span className={styles.setHeader}>RPE</span>
      <span className={styles.setHeader}>Rest (s)</span>
      <span className={styles.setHeader}>Tempo</span>
      <span className={styles.setHeader} aria-hidden="true" />

      {assignment.sets.map((set) => (
        <Fragment key={set.id}>
          <span className={styles.setNumber}>{set.set_number}</span>
          <select
            className={styles.setSelect}
            value={set.set_type}
            onChange={(event) =>
              onUpdateSet(assignment.id, set.id, { set_type: event.target.value as SetType })
            }
            aria-label={`Type of set ${set.set_number}`}
          >
            {SET_TYPES.map((type) => (
              <option key={type} value={type}>
                {SET_TYPE_LABELS[type]}
              </option>
            ))}
          </select>
          <input
            className={styles.setInput}
            type="number"
            inputMode="numeric"
            min={0}
            placeholder="—"
            value={set.reps}
            onChange={(event) => onUpdateSet(assignment.id, set.id, { reps: event.target.value })}
            aria-label={`Reps of set ${set.set_number}`}
          />
          <input
            className={styles.setInput}
            type="number"
            inputMode="decimal"
            min={0}
            step={0.5}
            placeholder="—"
            value={set.weight_kg}
            onChange={(event) => onUpdateSet(assignment.id, set.id, { weight_kg: event.target.value })}
            aria-label={`Weight of set ${set.set_number}`}
          />
          <input
            className={styles.setInput}
            type="number"
            inputMode="numeric"
            min={0}
            max={5}
            step={1}
            placeholder="—"
            value={set.rir}
            onChange={(event) => onUpdateSet(assignment.id, set.id, { rir: event.target.value })}
            aria-label={`RIR of set ${set.set_number}`}
          />
          <input
            className={styles.setInput}
            type="number"
            inputMode="decimal"
            min={0}
            max={10}
            step={0.5}
            placeholder="—"
            value={set.rpe}
            onChange={(event) => onUpdateSet(assignment.id, set.id, { rpe: event.target.value })}
            aria-label={`RPE of set ${set.set_number}`}
          />
          <input
            className={styles.setInput}
            type="number"
            inputMode="numeric"
            min={0}
            step={15}
            placeholder="—"
            value={set.rest_seconds}
            onChange={(event) =>
              onUpdateSet(assignment.id, set.id, { rest_seconds: event.target.value })
            }
            aria-label={`Rest seconds of set ${set.set_number}`}
          />
          <input
            className={styles.setInput}
            type="text"
            inputMode="numeric"
            pattern="[0-9]*"
            maxLength={4}
            placeholder="2010"
            value={set.tempo}
            onChange={(event) => onUpdateSet(assignment.id, set.id, { tempo: event.target.value })}
            aria-label={`Tempo of set ${set.set_number}`}
          />
          <button
            type="button"
            className={styles.setRemove}
            disabled={assignment.sets.length <= 1}
            onClick={() => onRemoveSet(assignment.id, set.id)}
            aria-label={`Remove set ${set.set_number}`}
            title={assignment.sets.length <= 1 ? "An exercise needs at least one set" : "Remove set"}
          >
            <Trash2 size={13} aria-hidden="true" />
          </button>
        </Fragment>
      ))}
    </div>
    <div className={styles.addRow}>
      <Button type="button" variant="ghost" size="small" icon={<Plus size={14} />} onClick={() => onAddSet(assignment.id)}>
        Add set
      </Button>
    </div>
  </>
);

interface PlanExerciseEditorModalProps {
  assignment: PlanDraftExercise;
  exercise: ExerciseCatalogEntry | null;
  alternatives: ExerciseAlternative[];
  onClose: () => void;
  onUpdateInstructions: (assignmentId: string, value: string) => void;
  onUpdateSet: (
    assignmentId: string,
    setId: string,
    patch: Partial<Omit<PlanDraftSet, "id" | "set_number">>
  ) => void;
  onRemoveSet: (assignmentId: string, setId: string) => void;
  onAddSet: (assignmentId: string) => void;
  onReplace: (assignmentId: string) => void;
  onRemove: (assignmentId: string) => void;
}

// PlanExerciseEditorModal is the detailed editor for a single exercise
// assignment: catalogue preview beside the editable per-day prescription.
// Replacing and removing are actions handled by the parent page.
export const PlanExerciseEditorModal = ({
  assignment,
  exercise,
  alternatives,
  onClose,
  onUpdateInstructions,
  onUpdateSet,
  onRemoveSet,
  onAddSet,
  onReplace,
  onRemove
}: PlanExerciseEditorModalProps) => (
  <Admin2Modal
    title={exercise ? exercise.name : "Exercise details"}
    description="Review the catalogue entry, then tailor this day's prescription."
    onClose={onClose}
    className={styles.editorModal}
  >
    <div className={styles.editorLayout}>
      <section className={styles.editorPreview} aria-label="Catalogue preview">
        {exercise ? (
          <ExercisePreviewBody exercise={exercise} alternatives={alternatives} />
        ) : (
          <p className={styles.editorLoading}>Loading exercise details…</p>
        )}
      </section>

      <section className={styles.editorPanel} aria-label="Workout prescription">
        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="exercise-instruction">
            Exercise instruction
          </label>
          <textarea
            id="exercise-instruction"
            className={joinClassNames(styles.formInput, styles.formTextarea, styles.instructionTextarea)}
            value={assignment.instructions}
            maxLength={400}
            placeholder="Tailor the movement instructions for this day."
            onChange={(event) => onUpdateInstructions(assignment.id, event.target.value)}
          />
          {!assignment.instructions.trim() ? (
            <p className={styles.formHint}>
              Nothing set: members will see the catalogue instructions for this exercise.
            </p>
          ) : null}
        </div>

        <div className={styles.setSection}>
          <span className={styles.formLabel}>Sets</span>
          <PlanSetGrid
            assignment={assignment}
            onUpdateSet={onUpdateSet}
            onRemoveSet={onRemoveSet}
            onAddSet={onAddSet}
          />
        </div>
      </section>
    </div>

    <div className={styles.editorFooter}>
      <div className={styles.editorFooterLeft}>
        <Button type="button" variant="ghost" size="small" icon={<Repeat size={14} />} onClick={() => onReplace(assignment.id)}>
          Replace exercise
        </Button>
        <Button
          type="button"
          variant="danger"
          size="small"
          icon={<Trash2 size={14} />}
          iconPosition="left"
          onClick={() => onRemove(assignment.id)}
        >
          Remove from day
        </Button>
      </div>
      <Button type="button" variant="primary" size="small" onClick={onClose}>
        Done
      </Button>
    </div>
  </Admin2Modal>
);