import { apiGet, encodePathSegment } from "@utils/http_client";

// EXP, lifter level and record medals, computed by the server from the
// completed workout sessions (see docs/INSTRUCTIONS/api.MD, "Achievements").

export type LifterLevelKey =
  | "rookie"
  | "beginner"
  | "average"
  | "dedicated"
  | "advanced"
  | "elite"
  | "master"
  | "legend";

export interface LifterLevel {
  rank: number;
  key: LifterLevelKey;
  min_xp: number;
}

export interface RecordMedal {
  id: string;
  session_id: string;
  exercise_id: string;
  exercise_name: string;
  weight_kg: number;
  reps: number;
  previous_best_kg: number;
  earned_at: string;
}

export interface AchievementRules {
  workout_xp: number;
  set_xp: number;
  max_sets_counted: number;
  record_xp: number;
  sessions_per_day: number;
}

export interface Achievements {
  xp: number;
  level: LifterLevel;
  next_level: LifterLevel | null;
  levels: LifterLevel[];
  rules: AchievementRules;
  workouts_counted: number;
  medal_count: number;
  medals: RecordMedal[];
}

export interface SessionAward {
  award: {
    session_id: string;
    completed_sets: number;
    xp_earned: number;
    counted: boolean;
    breakdown: { workout: number; sets: number; records: number };
    medals: RecordMedal[];
    xp_before: number;
    xp_after: number;
  };
  level_before: LifterLevel;
  level_after: LifterLevel;
  next_level: LifterLevel | null;
  rules: AchievementRules;
}

export const fetchAchievements = () => apiGet<Achievements>("/me/achievements");

export const fetchSessionAward = (sessionId: string) =>
  apiGet<SessionAward>(`/me/achievements/sessions/${encodePathSegment(sessionId)}`);

// Progress inside the current level, 0–1 (1 at the top level).
export const levelProgress = (xp: number, level: LifterLevel, next: LifterLevel | null): number => {
  if (!next) return 1;
  const span = next.min_xp - level.min_xp;
  if (span <= 0) return 1;
  return Math.min(1, Math.max(0, (xp - level.min_xp) / span));
};
