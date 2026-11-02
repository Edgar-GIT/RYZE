import { apiGet } from "@utils/http_client";

/** Safe derived workout statistics for the authenticated client. */
export interface ClientStatistics {
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

export const fetchClientStatistics = (): Promise<ClientStatistics> =>
  apiGet<ClientStatistics>("/me/statistics");
