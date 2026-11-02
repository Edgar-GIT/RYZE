import { apiGet, apiPatch, apiPost, encodePathSegment } from "@utils/http_client";

export type WorkoutSetStatus = "pending" | "completed" | "skipped";
export type WorkoutSessionStatus = "in_progress" | "paused" | "completed" | "abandoned";

export interface WorkoutSessionSet {
  id: string;
  set_number: number;
  set_type: string;
  prescribed_reps: number | null;
  prescribed_weight_kg: number | null;
  prescribed_rir: number | null;
  prescribed_rpe: number | null;
  prescribed_rest_seconds: number | null;
  prescribed_tempo: string;
  actual_reps: number | null;
  actual_weight_kg: number | null;
  actual_rir: number | null;
  actual_rpe: number | null;
  status: WorkoutSetStatus;
  completed_at: string | null;
  rest_ends_at: string | null;
}

export interface WorkoutSessionExercise {
  id: string;
  exercise_id: string;
  position: number;
  name: string;
  instructions: string;
  notes: string;
  sets: WorkoutSessionSet[];
}

export interface WorkoutSession {
  id: string;
  program_id: string;
  program_workout_id: string;
  program_name: string;
  week_number: number;
  workout_position: number;
  status: WorkoutSessionStatus;
  started_at: string;
  paused_at: string | null;
  completed_at: string | null;
  abandoned_at: string | null;
  elapsed_seconds: number;
  total_sets: number;
  completed_sets: number;
  skipped_sets: number;
  exercises: WorkoutSessionExercise[];
}

export interface WorkoutSetInput {
  actual_reps?: number | null;
  actual_weight_kg?: number | null;
  actual_rir?: number | null;
  actual_rpe?: number | null;
  status: WorkoutSetStatus;
}

const sessionPath = (sessionId: string): string =>
  `/me/workout-sessions/${encodePathSegment(sessionId)}`;

export const startWorkoutSession = (
  programId: string,
  workoutId: string
): Promise<WorkoutSession> =>
  apiPost<WorkoutSession>(
    `/me/programs/${encodePathSegment(programId)}/workouts/${encodePathSegment(workoutId)}/sessions`,
    {}
  );

export const fetchActiveWorkoutSession = (): Promise<WorkoutSession | null> =>
  apiGet<WorkoutSession | null>("/me/workout-sessions/active");

export const fetchWorkoutSession = (sessionId: string): Promise<WorkoutSession> =>
  apiGet<WorkoutSession>(sessionPath(sessionId));

export const submitWorkoutSet = (
  sessionId: string,
  setId: string,
  input: WorkoutSetInput
): Promise<WorkoutSession> =>
  apiPatch<WorkoutSession>(`${sessionPath(sessionId)}/sets/${encodePathSegment(setId)}`, input);

const transition = (sessionId: string, action: string): Promise<WorkoutSession> =>
  apiPost<WorkoutSession>(`${sessionPath(sessionId)}/${action}`, {});

export const pauseWorkoutSession = (sessionId: string): Promise<WorkoutSession> =>
  transition(sessionId, "pause");

export const resumeWorkoutSession = (sessionId: string): Promise<WorkoutSession> =>
  transition(sessionId, "resume");

export const completeWorkoutSession = (sessionId: string): Promise<WorkoutSession> =>
  transition(sessionId, "complete");

export const abandonWorkoutSession = (sessionId: string): Promise<WorkoutSession> =>
  transition(sessionId, "abandon");

export const fetchWorkoutHistory = (limit = 20): Promise<WorkoutSession[]> =>
  apiGet<WorkoutSession[]>(`/me/workout-sessions?limit=${limit}`);
