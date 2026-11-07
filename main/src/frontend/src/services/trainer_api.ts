import { apiDelete, apiGet, apiPatch, apiPost, apiUpload, downloadFile, encodePathSegment } from "@utils/http_client";

import { AdminPagination } from "./admin_api";
import type { WorkoutSession } from "./workout_sessions_api";

// The trainer session state comes from the backend access endpoint, which is
// the only source of truth for "is this user a trainer" and for the latest
// application outcome. Nothing is inferred from client-side state.

export const TrainerApplicationStatus = {
  PENDING: "PENDING",
  APPROVED: "APPROVED",
  REJECTED: "REJECTED"
} as const;

export type TrainerApplicationStatusValue =
  typeof TrainerApplicationStatus[keyof typeof TrainerApplicationStatus];

export interface TrainerOwnApplication {
  id: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface TrainerAccessStatus {
  is_trainer: boolean;
  application: TrainerOwnApplication | null;
}

export interface TrainerUserSummary {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  created_at: string;
  updated_at: string;
}

export interface TrainerProfile {
  id: string;
  user_id: string;
  user: TrainerUserSummary;
  created_at: string;
  updated_at: string;
}

export interface TrainerClient {
  id: string;
  trainer_id: string;
  user_id: string;
  user: TrainerUserSummary;
  created_at: string;
  updated_at: string;
}

export interface TrainerClientsList {
  clients: TrainerClient[];
  pagination: AdminPagination;
}

// TrainerClientCandidate is the minimal identity returned by the exact-email
// lookup. It only confirms who will be added; it is not a user profile.
export interface TrainerClientCandidate {
  user_id: string;
  email: string;
  first_name: string;
  last_name: string;
}

export const TrainerProgramType = {
  FREE: "free",
  PREMIUM: "premium",
  PERSONALIZED: "personalized"
} as const;

export type TrainerProgramTypeValue = typeof TrainerProgramType[keyof typeof TrainerProgramType];

export const TrainerProgramStatus = {
  DRAFT: "draft",
  PUBLISHED: "published"
} as const;

export type TrainerProgramStatusValue =
  typeof TrainerProgramStatus[keyof typeof TrainerProgramStatus];

export interface TrainerProgram {
  id: string;
  trainer_id: string | null;
  name: string;
  description: string;
  type: TrainerProgramTypeValue;
  status: TrainerProgramStatusValue;
  price_minor_units: number;
  currency: string;
  created_at: string;
  updated_at: string;
}

export interface TrainerProgramsList {
  programs: TrainerProgram[];
  pagination: AdminPagination;
}

export interface TrainerProgramInput {
  name: string;
  description: string;
  type: TrainerProgramTypeValue;
  status: TrainerProgramStatusValue;
  price_minor_units: number;
}

export type TrainerProgramUpdateInput = Partial<TrainerProgramInput>;

export interface TrainerWorkout {
  id: string;
  program_week_id: string;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface TrainerWeek {
  id: string;
  program_id: string;
  week_number: number;
  workouts: TrainerWorkout[];
  created_at: string;
  updated_at: string;
}

export interface TrainerExerciseSummary {
  id: string;
  name: string;
  description: string;
  target_muscles: string;
  equipment: string;
  difficulty: string;
  video_url: string;
  image_url: string;
}

export const WorkoutSetType = {
  WARMUP: "warmup",
  WORKING: "working",
  DROP: "drop",
  BACKOFF: "backoff",
  FAILURE: "failure"
} as const;

export type WorkoutSetTypeValue = typeof WorkoutSetType[keyof typeof WorkoutSetType];

export interface TrainerWorkoutSet {
  set_number: number;
  set_type: WorkoutSetTypeValue;
  reps: number | null;
  weight_kg: number | null;
  rir: number | null;
  rpe: number | null;
  rest_seconds: number | null;
  tempo: string;
}

export interface TrainerWorkoutExercise {
  id: string;
  program_workout_id: string;
  position: number;
  instructions: string;
  notes: string;
  exercise: TrainerExerciseSummary;
  sets: TrainerWorkoutSet[];
  created_at: string;
  updated_at: string;
}

export interface TrainerWorkoutSetInput {
  set_type: WorkoutSetTypeValue;
  reps: number | null;
  weight_kg: number | null;
  rir: number | null;
  rpe: number | null;
  rest_seconds: number | null;
  tempo: string;
}

export interface TrainerPrescriptionInput {
  instructions: string;
  notes: string;
  sets: TrainerWorkoutSetInput[];
}

export const fetchTrainerAccess = (): Promise<TrainerAccessStatus> =>
  apiGet<TrainerAccessStatus>("/trainer/access");

export const submitTrainerApplication = (): Promise<TrainerOwnApplication> =>
  apiPost<TrainerOwnApplication>("/trainer/apply", {});

// Optional evidence documents attached to a PENDING application. Kinds are the
// backend's controlled vocabulary; the server validates the file itself, the
// client only carries it.
export const ApplicationDocumentKind = {
  CERTIFICATION: "certification",
  IDENTIFICATION: "identification",
  INSURANCE: "insurance",
  OTHER: "other"
} as const;

export type ApplicationDocumentKindValue =
  typeof ApplicationDocumentKind[keyof typeof ApplicationDocumentKind];

export const APPLICATION_DOCUMENT_KIND_LABELS: Record<ApplicationDocumentKindValue, string> = {
  certification: "Certification",
  identification: "Identification",
  insurance: "Insurance",
  other: "Other"
};

export interface TrainerApplicationDocument {
  id: string;
  kind: string;
  original_filename: string;
  content_type: string;
  byte_size: number;
  created_at: string;
}

export interface TrainerApplicationDocumentsList {
  documents: TrainerApplicationDocument[];
}

export const fetchTrainerApplicationDocuments = (): Promise<TrainerApplicationDocumentsList> =>
  apiGet<TrainerApplicationDocumentsList>("/trainer/application/documents");

export const uploadTrainerApplicationDocument = (
  kind: ApplicationDocumentKindValue,
  file: File
): Promise<TrainerApplicationDocument> => {
  const form = new FormData();
  form.append("kind", kind);
  form.append("file", file);
  return apiUpload<TrainerApplicationDocument>("/trainer/application/documents", form);
};

export const deleteTrainerApplicationDocument = (documentId: string): Promise<never> =>
  apiDelete<never>(`/trainer/application/documents/${encodePathSegment(documentId)}`);

export const downloadTrainerApplicationDocument = (documentId: string, fallbackName: string): Promise<void> =>
  downloadFile(`/trainer/application/documents/${encodePathSegment(documentId)}/download`, fallbackName);

export const fetchTrainerProfile = (): Promise<TrainerProfile> =>
  apiGet<TrainerProfile>("/trainer/profile");

export const fetchTrainerClients = (page: number, limit: number): Promise<TrainerClientsList> =>
  apiGet<TrainerClientsList>(`/trainer/clients?page=${page}&limit=${limit}`);

export const lookupTrainerClient = (email: string): Promise<TrainerClientCandidate> =>
  apiGet<TrainerClientCandidate>(`/trainer/clients/lookup?email=${encodeURIComponent(email)}`);

export const addTrainerClient = (userId: string): Promise<TrainerClient> =>
  apiPost<TrainerClient>("/trainer/clients", { user_id: userId });

export const removeTrainerClient = (userId: string): Promise<never> =>
  apiDelete<never>(`/trainer/clients/${encodeURIComponent(userId)}`);

export const fetchTrainerPrograms = (page: number, limit: number): Promise<TrainerProgramsList> =>
  apiGet<TrainerProgramsList>(`/trainer/programs?page=${page}&limit=${limit}`);

export const fetchTrainerProgram = (programId: string): Promise<TrainerProgram> =>
  apiGet<TrainerProgram>(`/trainer/programs/${encodeURIComponent(programId)}`);

export const createTrainerProgram = (input: TrainerProgramInput): Promise<TrainerProgram> =>
  apiPost<TrainerProgram>("/trainer/programs", input);

export const updateTrainerProgram = (
  programId: string,
  input: TrainerProgramUpdateInput
): Promise<TrainerProgram> =>
  apiPatch<TrainerProgram>(`/trainer/programs/${encodeURIComponent(programId)}`, input);

export const publishTrainerProgram = (programId: string): Promise<TrainerProgram> =>
  apiPost<TrainerProgram>(`/trainer/programs/${encodeURIComponent(programId)}/publish`, {});

export const deleteTrainerProgram = (programId: string): Promise<never> =>
  apiDelete<never>(`/trainer/programs/${encodeURIComponent(programId)}`);

const structurePath = (programId: string): string =>
  `/trainer/programs/${encodeURIComponent(programId)}`;

export const fetchProgramWeeks = (programId: string): Promise<TrainerWeek[]> =>
  apiGet<{ weeks: TrainerWeek[] }>(`${structurePath(programId)}/weeks`).then((data) => data.weeks);

export const createProgramWeek = (programId: string): Promise<TrainerWeek> =>
  apiPost<TrainerWeek>(`${structurePath(programId)}/weeks`, {});

export const deleteProgramWeek = (programId: string, weekId: string): Promise<never> =>
  apiDelete<never>(`${structurePath(programId)}/weeks/${encodeURIComponent(weekId)}`);

export const reorderProgramWeeks = (programId: string, ids: string[]): Promise<never> =>
  apiPatch<never>(`${structurePath(programId)}/weeks/order`, { ids });

export const createProgramWorkout = (programId: string, weekId: string): Promise<TrainerWorkout> =>
  apiPost<TrainerWorkout>(`${structurePath(programId)}/weeks/${encodeURIComponent(weekId)}/workouts`, {});

export const deleteProgramWorkout = (
  programId: string,
  weekId: string,
  workoutId: string
): Promise<never> =>
  apiDelete<never>(
    `${structurePath(programId)}/weeks/${encodeURIComponent(weekId)}/workouts/${encodeURIComponent(workoutId)}`
  );

export const reorderProgramWorkouts = (
  programId: string,
  weekId: string,
  ids: string[]
): Promise<never> =>
  apiPatch<never>(`${structurePath(programId)}/weeks/${encodeURIComponent(weekId)}/workouts/order`, {
    ids
  });

export const fetchWorkoutExercises = (
  programId: string,
  weekId: string,
  workoutId: string
): Promise<TrainerWorkoutExercise[]> =>
  apiGet<{ exercises: TrainerWorkoutExercise[] }>(
    `${structurePath(programId)}/weeks/${encodeURIComponent(weekId)}/workouts/${encodeURIComponent(workoutId)}/exercises`
  ).then((data) => data.exercises);

export const addWorkoutExercise = (
  programId: string,
  weekId: string,
  workoutId: string,
  exerciseId: string
): Promise<TrainerWorkoutExercise> =>
  apiPost<TrainerWorkoutExercise>(
    `${structurePath(programId)}/weeks/${encodeURIComponent(weekId)}/workouts/${encodeURIComponent(workoutId)}/exercises`,
    { exercise_id: exerciseId }
  );

export const updateWorkoutExercisePrescription = (
  programId: string,
  weekId: string,
  workoutId: string,
  workoutExerciseId: string,
  input: TrainerPrescriptionInput
): Promise<TrainerWorkoutExercise> =>
  apiPatch<TrainerWorkoutExercise>(
    `${structurePath(programId)}/weeks/${encodeURIComponent(weekId)}/workouts/${encodeURIComponent(workoutId)}/exercises/${encodeURIComponent(workoutExerciseId)}`,
    input
  );

export const removeWorkoutExercise = (
  programId: string,
  weekId: string,
  workoutId: string,
  workoutExerciseId: string
): Promise<never> =>
  apiDelete<never>(
    `${structurePath(programId)}/weeks/${encodeURIComponent(weekId)}/workouts/${encodeURIComponent(workoutId)}/exercises/${encodeURIComponent(workoutExerciseId)}`
  );

export const reorderWorkoutExercises = (
  programId: string,
  weekId: string,
  workoutId: string,
  ids: string[]
): Promise<never> =>
  apiPatch<never>(
    `${structurePath(programId)}/weeks/${encodeURIComponent(weekId)}/workouts/${encodeURIComponent(workoutId)}/exercises/order`,
    { ids }
  );


export interface TrainerClientAccessAssignment {
  assignment_id?: string;
  program_id: string;
  program_name: string;
  program_status: string;
  assigned_at: string;
  owned_by_current_trainer: boolean;
  access_status: "active" | "unavailable";
}

export interface TrainerClientProgramAccess {
  program_id: string;
  program_name: string;
  program_status: string;
  granted_at: string;
  expires_at: string | null;
  access_status: "active" | "unavailable";
}

export interface TrainerClientStatistics {
  has_active_assignment: boolean;
  has_active_program: boolean;
  current_program_name: string;
  total_executions: number;
  unique_workouts_completed: number;
  total_workouts_in_program: number;
  completion_percentage: number;
  last_workout_date: string | null;
  completed_sessions: number;
  abandoned_sessions: number;
  completed_sets: number;
  skipped_sets: number;
  total_active_seconds: number;
}

export interface TrainerClientSessionSummary {
  id: string;
  program_id: string;
  program_name: string;
  week_number: number;
  workout_position: number;
  status: "in_progress" | "paused" | "completed" | "abandoned";
  started_at: string;
  completed_at: string | null;
  abandoned_at: string | null;
  elapsed_seconds: number;
  total_sets: number;
  completed_sets: number;
  skipped_sets: number;
}

export interface TrainerClientExerciseSummary {
  exercise_id: string;
  name: string;
  session_count: number;
  last_performed_at: string;
}

export interface TrainerClientOverview {
  access: {
    assignments: TrainerClientAccessAssignment[];
    purchases: TrainerClientProgramAccess[];
    free_activations: TrainerClientProgramAccess[];
  };
  statistics: TrainerClientStatistics;
  active_session: TrainerClientSessionSummary | null;
  recent_sessions: TrainerClientSessionSummary[];
  legacy: {
    count: number;
    last_completed_at: string | null;
  };
  exercises: TrainerClientExerciseSummary[];
}

export interface TrainerClientSessionsPage {
  sessions: TrainerClientSessionSummary[];
  pagination: AdminPagination;
}

export interface TrainerClientExerciseHistorySet {
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
  status: "pending" | "completed" | "skipped";
  completed_at: string | null;
}

export interface TrainerClientExerciseHistorySession {
  session_id: string;
  program_id: string;
  program_name: string;
  week_number: number;
  workout_position: number;
  completed_at: string | null;
  exercise_id: string;
  exercise_name: string;
  instructions: string;
  notes: string;
  sets: TrainerClientExerciseHistorySet[];
}

export interface TrainerClientExerciseHistory {
  exercise_id: string;
  name: string;
  sessions: TrainerClientExerciseHistorySession[];
}

export interface TrainerClientProgramAssignment {
  id: string;
  trainer_id: string;
  program_id: string;
  user_id: string;
  program: {
    id: string;
    trainer_id: string;
    name: string;
    description: string;
    type: TrainerProgramTypeValue;
    status: TrainerProgramStatusValue;
    created_at: string;
    updated_at: string;
  };
  created_at: string;
  updated_at: string;
}

export const fetchTrainerClient = (userId: string): Promise<TrainerClient> =>
  apiGet<TrainerClient>(`/trainer/clients/${encodeURIComponent(userId)}`);

export const fetchTrainerClientOverview = (userId: string): Promise<TrainerClientOverview> =>
  apiGet<TrainerClientOverview>(`/trainer/clients/${encodeURIComponent(userId)}/overview`);

export const fetchTrainerClientStatistics = (userId: string): Promise<TrainerClientStatistics> =>
  apiGet<TrainerClientStatistics>(`/trainer/clients/${encodeURIComponent(userId)}/statistics`);

export const fetchTrainerClientWorkoutSessions = (
  userId: string,
  page: number,
  limit: number
): Promise<TrainerClientSessionsPage> =>
  apiGet<TrainerClientSessionsPage>(
    `/trainer/clients/${encodeURIComponent(userId)}/workout-sessions?page=${page}&limit=${limit}`
  );

export const fetchTrainerClientWorkoutSession = (
  userId: string,
  sessionId: string
): Promise<WorkoutSession> =>
  apiGet<WorkoutSession>(
    `/trainer/clients/${encodeURIComponent(userId)}/workout-sessions/${encodeURIComponent(sessionId)}`
  );

export const fetchTrainerClientExerciseHistory = (
  userId: string,
  exerciseId: string,
  limit = 20
): Promise<TrainerClientExerciseHistory> =>
  apiGet<TrainerClientExerciseHistory>(
    `/trainer/clients/${encodeURIComponent(userId)}/exercises/${encodeURIComponent(exerciseId)}/history?limit=${limit}`
  );

export const fetchTrainerClientPrograms = (userId: string): Promise<TrainerClientProgramAssignment[]> =>
  apiGet<{ programs: TrainerClientProgramAssignment[] }>(
    `/trainer/clients/${encodeURIComponent(userId)}/programs`
  ).then((data) => data.programs);

export const assignTrainerClientProgram = (
  userId: string,
  programId: string
): Promise<TrainerClientProgramAssignment> =>
  apiPost<TrainerClientProgramAssignment>(
    `/trainer/clients/${encodeURIComponent(userId)}/programs`,
    { program_id: programId }
  );

export const removeTrainerClientProgram = (userId: string, assignmentId: string): Promise<never> =>
  apiDelete<never>(
    `/trainer/clients/${encodeURIComponent(userId)}/programs/${encodeURIComponent(assignmentId)}`
  );
