import { Repeat, ArrowRight } from "lucide-react";
import { useState } from "react";

import {
  exerciseMediaView,
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
  titleAs?: "h1" | "h3";
}

// ExercisePreviewBody is the read-only catalogue panel reused by the library
// picker dialog, the public exercise pages and the exercise detail editor. It
// never mutates anything.
export const ExercisePreviewBody = ({
  exercise,
  alternatives = [],
  onSelectAlternative,
  alternativeActionLabel = "View",
  titleAs: TitleTag = "h3"
}: ExercisePreviewBodyProps) => {
  const media = exerciseMediaView(exercise);
  const steps = (exercise.instruction_steps ?? []).filter((step) => step.trim());

  return (
    <div className={styles.body}>
      {media.kind === "video" ? (
        <video
          src={media.url}
          controls
          loop
          muted
          playsInline
          className={styles.media}
          aria-label={`${exercise.name} demonstration`}
        />
      ) : media.kind === "image" ? (
        <img src={media.url} alt={`${exercise.name} demonstration`} className={styles.media} loading="lazy" />
      ) : (
        <ExerciseMediaPlaceholder label="Demonstration video" />
      )}
      {media.attribution ? <p className={styles.attribution}>{media.attribution}</p> : null}

      <div className={styles.header}>
        <TitleTag className={styles.name}>{exercise.name}</TitleTag>
        <p className={styles.meta}>{exerciseMetaLine(exercise)}</p>
        <div className={styles.badges}>
          <span className={styles.badge}>{exercise.movement_category || "Exercise"}</span>
          <span className={styles.badge}>{exercise.difficulty || "Difficulty"}</span>
        </div>
      </div>

      <p className={styles.muscles}>{exerciseMuscles(exercise)}</p>

      <p className={styles.description}>{exercise.description}</p>

      {steps.length > 0 ? (
        <div className={styles.section}>
          <h4 className={styles.sectionTitle}>Instructions</h4>
          <ol className={styles.steps}>
            {steps.map((step, index) => (
              <li key={`${index}-${step.slice(0, 24)}`} className={styles.step}>
                {step}
              </li>
            ))}
          </ol>
        </div>
      ) : exercise.instructions ? (
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