import { Dumbbell } from "lucide-react";

import {
  exerciseMediaView,
  exerciseThumbnailView,
  type ExerciseCatalogEntry
} from "@/services/exercise_library";
import { joinClassNames } from "@utils/class_names";

import styles from "./plan_builder.module.css";

// BuilderExercise is what the builder needs to show an exercise: the
// catalogue identity, its muscles and the media references (when the
// deployment serves exercise media).
export interface BuilderExercise {
  id: string;
  name: string;
  primary_muscle_group: string;
  body_part: string;
  equipment: string;
  difficulty: string;
  description: string;
  instructions: string;
  instruction_steps: string[] | null;
  image_url: string | null;
  animation_url: string | null;
  video_url: string | null;
  media_attribution: string;
  media_available: boolean;
}

export const fromCatalogEntry = (entry: ExerciseCatalogEntry): BuilderExercise => ({
  id: entry.id,
  name: entry.name,
  primary_muscle_group: entry.primary_muscle_group,
  body_part: entry.body_part,
  equipment: entry.equipment,
  difficulty: entry.difficulty,
  description: entry.description,
  instructions: entry.instructions,
  instruction_steps: entry.instruction_steps,
  image_url: entry.image_url,
  animation_url: entry.animation_url,
  video_url: entry.video_url,
  media_attribution: entry.media_attribution,
  media_available: entry.media_available
});

export const capitalise = (value: string) =>
  value
    .split(" ")
    .map((word) => (word ? word[0].toUpperCase() + word.slice(1) : word))
    .join(" ");

// ExerciseThumb shows the still picture of an exercise (never the animation:
// that only plays in the preview), or a neutral tile when media is off.
export const ExerciseThumb = ({ exercise, size = "md" }: { exercise: BuilderExercise | undefined; size?: "sm" | "md" | "lg" }) => {
  const view = exercise ? exerciseThumbnailView(exercise) : { kind: "none" as const, url: "" };
  return (
    <span className={joinClassNames(styles.thumb, styles[`thumb_${size}`])} aria-hidden="true">
      {view.kind === "image" ? <img src={view.url} alt="" loading="lazy" decoding="async" /> : <Dumbbell />}
    </span>
  );
};

// ExerciseAnimation plays the demonstration (GIF or video) in the preview.
export const ExerciseAnimation = ({ exercise, unavailable }: { exercise: BuilderExercise; unavailable: string }) => {
  const view = exerciseMediaView(exercise);
  if (view.kind === "video") {
    return (
      <div className={styles.animation}>
        <video src={view.url} autoPlay loop muted playsInline />
      </div>
    );
  }
  if (view.kind === "image") {
    return (
      <div className={styles.animation}>
        <img src={view.url} alt="" />
        {view.attribution ? <span className={styles.attribution}>{view.attribution}</span> : null}
      </div>
    );
  }
  return (
    <div className={joinClassNames(styles.animation, styles.animationEmpty)}>
      <Dumbbell aria-hidden="true" />
      <span>{unavailable}</span>
    </div>
  );
};
