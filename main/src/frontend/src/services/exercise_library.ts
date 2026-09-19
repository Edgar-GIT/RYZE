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
  description: string;
  instructions: string;
  target_muscles: string;
  primary_muscle_group: string;
  secondary_muscle_groups: string;
  equipment: string;
  difficulty: string;
  movement_category: string;
  video_url: string | null;
  image_url: string | null;
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
  muscle?: string;
  equipment?: string;
  difficulty?: string;
  category?: string;
}

export const countActiveLibraryFilters = (filters: ExerciseLibraryFilters): number =>
  [filters.muscle, filters.equipment, filters.difficulty, filters.category].filter(
    (value) => value && value.trim()
  ).length;

const appendParam = (params: string[], key: string, value: string | undefined): void => {
  const trimmed = value?.trim();
  if (trimmed) {
    params.push(`${key}=${encodeURIComponent(trimmed)}`);
  }
};

export const fetchExerciseLibrary = (
  filters: ExerciseLibraryFilters,
  page: number,
  limit: number
): Promise<ExerciseLibraryResult> => {
  const params: string[] = [`page=${page}`, `limit=${limit}`];
  appendParam(params, "q", filters.query);
  appendParam(params, "muscle", filters.muscle);
  appendParam(params, "equipment", filters.equipment);
  appendParam(params, "difficulty", filters.difficulty);
  appendParam(params, "category", filters.category);
  return apiGet<ExerciseLibraryResult>(`/exercises?${params.join("&")}`);
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