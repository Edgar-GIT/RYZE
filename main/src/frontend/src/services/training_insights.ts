// Training insights computed from workout sessions (the session API already
// returns every exercise and set). Pure functions: no requests, no state.
import type { WorkoutSession, WorkoutSessionSet } from "@/services/workout_sessions_api";

export interface BestSet {
  weight: number;
  reps: number;
}

export interface ExerciseSummary {
  exerciseId: string;
  name: string;
  completedSets: number;
  totalSets: number;
  volumeKg: number;
  reps: number;
  best: BestSet | null;
}

export interface SessionSummary {
  durationSeconds: number;
  volumeKg: number;
  completedSets: number;
  skippedSets: number;
  pendingSets: number;
  totalSets: number;
  reps: number;
  exercises: ExerciseSummary[];
}

export interface PersonalRecord {
  exerciseId: string;
  name: string;
  weight: number;
  reps: number;
  // "weight": heaviest weight ever lifted on the exercise; "e1rm": best
  // estimated one-rep max (a heavier effort at more reps).
  kind: "weight" | "e1rm";
  previous: number;
}

const isDone = (set: WorkoutSessionSet): boolean => set.status === "completed";

const weightOf = (set: WorkoutSessionSet): number => set.actual_weight_kg ?? 0;
const repsOf = (set: WorkoutSessionSet): number => set.actual_reps ?? 0;

// Epley estimate of a one-rep max. A single rep is the lift itself.
export const estimatedOneRepMax = (weight: number, reps: number): number => {
  if (weight <= 0 || reps <= 0) {
    return 0;
  }
  return reps === 1 ? weight : weight * (1 + reps / 30);
};

const better = (a: BestSet | null, b: BestSet): BestSet =>
  !a || b.weight > a.weight || (b.weight === a.weight && b.reps > a.reps) ? b : a;

export const summarizeSession = (session: WorkoutSession): SessionSummary => {
  const exercises: ExerciseSummary[] = session.exercises.map((exercise) => {
    const done = exercise.sets.filter(isDone);
    let best: BestSet | null = null;
    for (const set of done) {
      if (weightOf(set) > 0 || repsOf(set) > 0) {
        best = better(best, { weight: weightOf(set), reps: repsOf(set) });
      }
    }
    return {
      exerciseId: exercise.exercise_id,
      name: exercise.name,
      completedSets: done.length,
      totalSets: exercise.sets.length,
      volumeKg: done.reduce((sum, set) => sum + weightOf(set) * repsOf(set), 0),
      reps: done.reduce((sum, set) => sum + repsOf(set), 0),
      best
    };
  });
  const allSets = session.exercises.flatMap((exercise) => exercise.sets);
  return {
    durationSeconds: session.elapsed_seconds,
    volumeKg: exercises.reduce((sum, exercise) => sum + exercise.volumeKg, 0),
    completedSets: allSets.filter(isDone).length,
    skippedSets: allSets.filter((set) => set.status === "skipped").length,
    pendingSets: allSets.filter((set) => set.status === "pending").length,
    totalSets: allSets.length,
    reps: exercises.reduce((sum, exercise) => sum + exercise.reps, 0),
    exercises
  };
};

// Sessions that count as training done, oldest first.
const completedBefore = (history: WorkoutSession[], session: { id: string; started_at: string }) =>
  history
    .filter((entry) => entry.id !== session.id && entry.status === "completed" && entry.started_at < session.started_at)
    .sort((a, b) => a.started_at.localeCompare(b.started_at));

// previousSets returns, per exercise, the completed sets of the most recent
// earlier session that trained it (the "previous" column of a workout).
export const previousSets = (
  history: WorkoutSession[],
  session: { id: string; started_at: string }
): Map<string, WorkoutSessionSet[]> => {
  const result = new Map<string, WorkoutSessionSet[]>();
  const earlier = completedBefore(history, session).reverse();
  for (const entry of earlier) {
    for (const exercise of entry.exercises) {
      if (result.has(exercise.exercise_id)) {
        continue;
      }
      const done = exercise.sets.filter(isDone);
      if (done.length > 0) {
        result.set(exercise.exercise_id, done);
      }
    }
  }
  return result;
};

// personalRecords compares a session with everything completed before it. An
// exercise done for the first time sets no record: there is nothing to beat.
export const personalRecords = (session: WorkoutSession, history: WorkoutSession[]): PersonalRecord[] => {
  const earlier = completedBefore(history, session);
  const bestWeight = new Map<string, number>();
  const bestE1rm = new Map<string, number>();
  for (const entry of earlier) {
    for (const exercise of entry.exercises) {
      for (const set of exercise.sets.filter(isDone)) {
        const weight = weightOf(set);
        bestWeight.set(exercise.exercise_id, Math.max(bestWeight.get(exercise.exercise_id) ?? 0, weight));
        bestE1rm.set(
          exercise.exercise_id,
          Math.max(bestE1rm.get(exercise.exercise_id) ?? 0, estimatedOneRepMax(weight, repsOf(set)))
        );
      }
    }
  }
  const records: PersonalRecord[] = [];
  for (const exercise of session.exercises) {
    const previousWeight = bestWeight.get(exercise.exercise_id);
    const previousE1rm = bestE1rm.get(exercise.exercise_id);
    if (previousWeight === undefined || previousE1rm === undefined) {
      continue;
    }
    let topWeight: BestSet | null = null;
    let topE1rm: BestSet | null = null;
    for (const set of exercise.sets.filter(isDone)) {
      const candidate = { weight: weightOf(set), reps: repsOf(set) };
      if (candidate.weight <= 0) {
        continue;
      }
      topWeight = better(topWeight, candidate);
      if (!topE1rm || estimatedOneRepMax(candidate.weight, candidate.reps) > estimatedOneRepMax(topE1rm.weight, topE1rm.reps)) {
        topE1rm = candidate;
      }
    }
    if (topWeight && topWeight.weight > previousWeight) {
      records.push({ exerciseId: exercise.exercise_id, name: exercise.name, ...topWeight, kind: "weight", previous: previousWeight });
    } else if (topE1rm && estimatedOneRepMax(topE1rm.weight, topE1rm.reps) > previousE1rm + 0.01) {
      records.push({ exerciseId: exercise.exercise_id, name: exercise.name, ...topE1rm, kind: "e1rm", previous: previousE1rm });
    }
  }
  return records;
};

const startOfWeek = (date: Date): Date => {
  const day = (date.getDay() + 6) % 7; // Monday = 0
  const start = new Date(date.getFullYear(), date.getMonth(), date.getDate() - day);
  start.setHours(0, 0, 0, 0);
  return start;
};

export interface WeekBucket {
  weekStart: Date;
  workouts: number;
  volumeKg: number;
  seconds: number;
}

// weeklyBuckets counts completed workouts per week (Monday start) for the
// last `weeks` weeks, oldest first, including empty weeks.
export const weeklyBuckets = (history: WorkoutSession[], weeks = 8, now = new Date()): WeekBucket[] => {
  const current = startOfWeek(now);
  const buckets: WeekBucket[] = [];
  for (let i = weeks - 1; i >= 0; i -= 1) {
    const weekStart = new Date(current);
    weekStart.setDate(current.getDate() - i * 7);
    buckets.push({ weekStart, workouts: 0, volumeKg: 0, seconds: 0 });
  }
  for (const session of history) {
    if (session.status !== "completed") {
      continue;
    }
    const week = startOfWeek(new Date(session.started_at)).getTime();
    const bucket = buckets.find((entry) => entry.weekStart.getTime() === week);
    if (bucket) {
      const summary = summarizeSession(session);
      bucket.workouts += 1;
      bucket.volumeKg += summary.volumeKg;
      bucket.seconds += summary.durationSeconds;
    }
  }
  return buckets;
};

export interface ExercisePoint {
  sessionId: string;
  date: string;
  bestWeight: number;
  bestReps: number;
  e1rm: number;
  volumeKg: number;
}

// exerciseSeries is the history of one exercise, oldest first: the best set
// and volume of every completed session that trained it.
export const exerciseSeries = (history: WorkoutSession[], exerciseId: string): ExercisePoint[] =>
  history
    .filter((session) => session.status === "completed")
    .sort((a, b) => a.started_at.localeCompare(b.started_at))
    .flatMap((session) => {
      const exercise = session.exercises.find((entry) => entry.exercise_id === exerciseId);
      const done = exercise?.sets.filter(isDone) ?? [];
      if (done.length === 0) {
        return [];
      }
      let best: BestSet | null = null;
      let e1rm = 0;
      for (const set of done) {
        best = better(best, { weight: weightOf(set), reps: repsOf(set) });
        e1rm = Math.max(e1rm, estimatedOneRepMax(weightOf(set), repsOf(set)));
      }
      return [
        {
          sessionId: session.id,
          date: session.started_at,
          bestWeight: best?.weight ?? 0,
          bestReps: best?.reps ?? 0,
          e1rm,
          volumeKg: done.reduce((sum, set) => sum + weightOf(set) * repsOf(set), 0)
        }
      ];
    });

export interface TrainedExercise {
  exerciseId: string;
  name: string;
  sessions: number;
}

// trainedExercises lists exercises with completed sets, most trained first.
export const trainedExercises = (history: WorkoutSession[]): TrainedExercise[] => {
  const counts = new Map<string, TrainedExercise>();
  for (const session of history) {
    if (session.status !== "completed") {
      continue;
    }
    for (const exercise of session.exercises) {
      if (!exercise.sets.some(isDone)) {
        continue;
      }
      const entry = counts.get(exercise.exercise_id) ?? { exerciseId: exercise.exercise_id, name: exercise.name, sessions: 0 };
      entry.sessions += 1;
      counts.set(exercise.exercise_id, entry);
    }
  }
  return [...counts.values()].sort((a, b) => b.sessions - a.sessions || a.name.localeCompare(b.name));
};

// currentStreakWeeks counts consecutive weeks (ending this week or last
// week) with at least one completed workout.
export const currentStreakWeeks = (history: WorkoutSession[], now = new Date()): number => {
  const buckets = weeklyBuckets(history, 52, now);
  let index = buckets.length - 1;
  if (buckets[index]?.workouts === 0) {
    index -= 1; // this week may simply not have a workout yet
  }
  let streak = 0;
  while (index >= 0 && buckets[index].workouts > 0) {
    streak += 1;
    index -= 1;
  }
  return streak;
};

export const formatClock = (seconds: number): string => {
  const safe = Math.max(0, Math.floor(seconds));
  const hours = Math.floor(safe / 3600);
  const minutes = Math.floor((safe % 3600) / 60);
  const rest = safe % 60;
  const mm = String(minutes).padStart(hours > 0 ? 2 : 1, "0");
  const ss = String(rest).padStart(2, "0");
  return hours > 0 ? `${hours}:${mm}:${ss}` : `${mm}:${ss}`;
};

export const formatWeight = (kg: number, intlLocale: string): string =>
  new Intl.NumberFormat(intlLocale, { maximumFractionDigits: 1 }).format(kg);
