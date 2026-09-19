import { Repeat, ArrowRight } from "lucide-react";

import {
  exerciseMetaLine,
  exerciseMuscles,
  type ExerciseAlternative,
  type ExerciseCatalogEntry
} from "@/services/exercise_library";

import { ExerciseMediaPlaceholder } from "./exercise_media_placeholder";

import styles from "./exercise_preview_body.module.css";

interface ExercisePreviewBodyProps {
  exercise: ExerciseCatalogEntry;
  alternatives?: ExerciseAlternative[];
  onSelectAlternative?: (exerciseId: string) => void;
  alternativeActionLabel?: string;
}

// ExercisePreviewBody is the read-only catalogue panel reused by the library
// picker dialog and the exercise detail editor. It never mutates anything.
export const ExercisePreviewBody = ({
  exercise,
  alternatives = [],
  onSelectAlternative,
  alternativeActionLabel = "View"
}: ExercisePreviewBodyProps) => {
  const hasMedia = Boolean(exercise.video_url || exercise.image_url);

  return (
    <div className={styles.body}>
      {hasMedia ? (
        <img
          src={exercise.video_url ?? exercise.image_url ?? ""}
          alt={exercise.name}
          className={styles.media}
        />
      ) : (
        <ExerciseMediaPlaceholder label="Demonstration video" />
      )}

      <div className={styles.header}>
        <h3 className={styles.name}>{exercise.name}</h3>
        <p className={styles.meta}>{exerciseMetaLine(exercise)}</p>
        <div className={styles.badges}>
          <span className={styles.badge}>{exercise.movement_category || "Exercise"}</span>
          <span className={styles.badge}>{exercise.difficulty || "Difficulty"}</span>
        </div>
      </div>

      <p className={styles.muscles}>{exerciseMuscles(exercise)}</p>

      <p className={styles.description}>{exercise.description}</p>

      {exercise.instructions ? (
        <div className={styles.section}>
          <h4 className={styles.sectionTitle}>Instructions</h4>
          <p className={styles.instructions}>{exercise.instructions}</p>
        </div>
      ) : null}

      {alternatives.length > 0 ? (
        <div className={styles.section}>
          <h4 className={styles.sectionTitle}>Alternatives</h4>
          <ul className={styles.alternatives}>
            {alternatives.map((alternative) => (
              <li key={alternative.id} className={styles.alternative}>
                <span className={styles.alternativeIcon}>
                  <Repeat size={14} />
                </span>
                <span className={styles.alternativeName}>{alternative.alternative_name}</span>
                {onSelectAlternative ? (
                  <button
                    type="button"
                    className={styles.alternativeAction}
                    onClick={() => onSelectAlternative(alternative.alternative_exercise_id)}
                  >
                    {alternativeActionLabel}
                    <ArrowRight size={14} />
                  </button>
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
};