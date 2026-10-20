import { apiGet } from "@utils/http_client";
import { AdminPagination } from "./admin_api";

// Controlled vocabulary of the global exercise catalog. These values are
// stored verbatim by the backend and every filter is validated against them,
// so the frontend exposes exactly the same surface.
export const EXERCISE_MUSCLE_GROUPS = [
  "Chest",
  "Back",
  "Shoulders",
  "Biceps",
  "Triceps",
  "Forearms",
  "Abs",
  "Obliques",
  "Core",
  "Quads",
  "Hamstrings",
  "Glutes",
  "Calves",
  "Traps",
  "Lower Back",
  "Hips",
  "Full Body"
] as const;

export type ExerciseMuscleGroup = (typeof EXERCISE_MUSCLE_GROUPS)[number];

// The dataset body parts are stored lowercase and every filter is a substring
// match against those values, so the option values stay verbatim while the
// labels are presented capitalised.
export const EXERCISE_BODY_PART_OPTIONS = [
  { value: "chest", label: "Chest" },
  { value: "back", label: "Back" },
  { value: "shoulders", label: "Shoulders" },
  { value: "upper arms", label: "Upper Arms" },
  { value: "lower arms", label: "Lower Arms" },
  { value: "upper legs", label: "Upper Legs" },
  { value: "lower legs", label: "Lower Legs" },
  { value: "waist", label: "Waist" },
  { value: "neck", label: "Neck" },
  { value: "cardio", label: "Cardio" }
] as const;

export const exerciseBodyPartLabel = (value: string): string => {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }
  const option = EXERCISE_BODY_PART_OPTIONS.find((entry) => entry.value === trimmed.toLowerCase());
  return option ? option.label : trimmed;
};

export const MOVEMENT_CATEGORIES = [
  "Compound",
  "Isolation",
  "Core",
  "Plyometric",
  "Cardio",
  "Mobility",
  "Stretching"
] as const;

export type MovementCategory = (typeof MOVEMENT_CATEGORIES)[number];

export const DIFFICULTY_LEVELS = ["Beginner", "Intermediate", "Advanced"] as const;

export type DifficultyLevel = (typeof DIFFICULTY_LEVELS)[number];

export const SET_TYPES = ["warmup", "working", "drop", "backoff", "failure"] as const;

export type SetType = (typeof SET_TYPES)[number];

export const SET_TYPE_LABELS: Record<SetType, string> = {
  warmup: "Warm-up",
  working: "Working",
  drop: "Drop",
  backoff: "Back-off",
  failure: "Failure"
};

export const EXERCISE_EQUIPMENT_OPTIONS = [
  "Barbell",
  "Dumbbells",
  "Kettlebell",
  "Bench",
  "Cable Machine",
  "Bodyweight",
  "Pull-Up Bar",
  "Dip Bars",
  "Weight Belt",
  "Leg Press Machine",
  "Leg Extension Machine",
  "Leg Curl Machine",
  "Calf Raise Machine",
  "Treadmill",
  "Jump Rope",
  "Medicine Ball",
  "Box"
] as const;

export interface ExerciseCatalogEntry {
  id: string;
  name: string;
  body_part: string;
  description: string;
  instructions: string;
  instruction_steps: string[] | null;
  target_muscles: string;
  primary_muscle_group: string;
  secondary_muscle_groups: string;
  equipment: string;
  difficulty: string;
  movement_category: string;
  video_url: string | null;
  image_url: string | null;
  animation_url: string | null;
  three_d_key: string | null;
  media_attribution: string;
  media_available: boolean;
  created_at: string;
  updated_at: string;
}

export interface ExerciseAlternative {
  id: string;
  exercise_id: string;
  alternative_exercise_id: string;
  alternative_name: string;
}

export interface ExerciseDetail {
  exercise: ExerciseCatalogEntry;
  alternatives: ExerciseAlternative[];
}

export interface ExerciseLibraryResult {
  exercises: ExerciseCatalogEntry[];
  pagination: AdminPagination;
}

export interface ExerciseLibraryFilters {
  query?: string;
  bodyPart?: string;
  muscle?: string;
  equipment?: string;
  difficulty?: string;
  category?: string;
}

export const countActiveLibraryFilters = (filters: ExerciseLibraryFilters): number =>
  [filters.bodyPart, filters.muscle, filters.equipment, filters.difficulty, filters.category].filter(
    (value) => value && value.trim()
  ).length;

const appendParam = (params: string[], key: string, value: string | undefined): void => {
  const trimmed = value?.trim();
  if (trimmed) {
    params.push(`${key}=${encodeURIComponent(trimmed)}`);
  }
};

// fetchExerciseLibrary browses the public catalogue. A non-empty query goes to
// the search endpoint (the list endpoint ignores free-text queries); both
// endpoints accept the same pagination and filter parameters.
export const fetchExerciseLibrary = (
  filters: ExerciseLibraryFilters,
  page: number,
  limit: number
): Promise<ExerciseLibraryResult> => {
  const params: string[] = [`page=${page}`, `limit=${limit}`];
  appendParam(params, "body_part", filters.bodyPart);
  appendParam(params, "muscle", filters.muscle);
  appendParam(params, "equipment", filters.equipment);
  appendParam(params, "difficulty", filters.difficulty);
  appendParam(params, "category", filters.category);

  const query = filters.query?.trim();
  const path = query
    ? `/exercises/search?q=${encodeURIComponent(query)}&${params.join("&")}`
    : `/exercises?${params.join("&")}`;

  return apiGet<ExerciseLibraryResult>(path);
};

export const fetchExerciseDetail = (exerciseId: string): Promise<ExerciseDetail> =>
  apiGet<ExerciseDetail>(`/exercises/${encodeURIComponent(exerciseId)}`);

// exerciseMetaLine renders the compact one-line descriptor used in lists and
// previews: primary muscle group, movement category, equipment and difficulty.
export const exerciseMetaLine = (entry: Pick<ExerciseCatalogEntry, "primary_muscle_group" | "equipment" | "difficulty">): string =>
  [entry.primary_muscle_group, entry.equipment, entry.difficulty]
    .map((value) => value?.trim())
    .filter(Boolean)
    .join(" · ") || "Catalog exercise";

export const exerciseMuscles = (entry: Pick<ExerciseCatalogEntry, "primary_muscle_group" | "secondary_muscle_groups">): string =>
  [entry.primary_muscle_group, entry.secondary_muscle_groups]
    .map((value) => value?.trim())
    .filter(Boolean)
    .join(" · ");

export interface ExerciseMediaView {
  kind: "image" | "video" | "none";
  url: string;
  attribution: string;
}

const VIDEO_FILE_PATTERN = /\.(mp4|webm|mov|m4v)(\?.*)?$/i;

// exerciseMediaView resolves the renderable media for a catalogue entry. The
// backend only serves media references while media exposure is enabled and
// reports it through media_available; without that flag every entry renders the
// unavailable state. The animated demonstration (a GIF) is preferred over the
// still image because it shows the movement.
export const exerciseMediaView = (
  entry: Pick<ExerciseCatalogEntry, "media_available" | "media_attribution" | "animation_url" | "image_url" | "video_url">
): ExerciseMediaView => {
  if (!entry.media_available) {
    return { kind: "none", url: "", attribution: "" };
  }

  const candidates = [entry.animation_url, entry.image_url, entry.video_url];
  const url = candidates.find((value) => value && value.trim())?.trim() ?? "";
  if (!url) {
    return { kind: "none", url: "", attribution: "" };
  }

  return {
    kind: VIDEO_FILE_PATTERN.test(url) ? "video" : "image",
    url,
    attribution: entry.media_attribution?.trim() ?? ""
  };
};