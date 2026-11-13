import { describe, expect, it } from "vitest";

import type { WorkoutSession, WorkoutSessionSet } from "@/services/workout_sessions_api";

import {
  currentStreakWeeks,
  estimatedOneRepMax,
  exerciseSeries,
  formatClock,
  personalRecords,
  previousSets,
  summarizeSession,
  trainedExercises,
  weeklyBuckets
} from "./training_insights";

let counter = 0;
const set = (weight: number | null, reps: number | null, status: WorkoutSessionSet["status"] = "completed"): WorkoutSessionSet => ({
  id: `set-${(counter += 1)}`,
  set_number: counter,
  set_type: "working",
  prescribed_reps: 8,
  prescribed_weight_kg: null,
  prescribed_rir: null,
  prescribed_rpe: null,
  prescribed_rest_seconds: 90,
  prescribed_tempo: "",
  actual_reps: reps,
  actual_weight_kg: weight,
  actual_rir: null,
  actual_rpe: null,
  status,
  completed_at: null,
  rest_ends_at: null
});

const session = (id: string, startedAt: string, exercises: Array<[string, WorkoutSessionSet[]]>, status: WorkoutSession["status"] = "completed"): WorkoutSession => ({
  id,
  program_id: "p",
  program_workout_id: "w",
  program_name: "Plan",
  week_number: 1,
  workout_position: 1,
  status,
  started_at: startedAt,
  paused_at: null,
  completed_at: null,
  abandoned_at: null,
  elapsed_seconds: 3000,
  total_sets: 0,
  completed_sets: 0,
  skipped_sets: 0,
  exercises: exercises.map(([exerciseId, sets], index) => ({
    id: `${id}-${exerciseId}`,
    exercise_id: exerciseId,
    position: index + 1,
    name: exerciseId === "bench" ? "Bench press" : "Squat",
    instructions: "",
    notes: "",
    sets
  }))
});

describe("training insights", () => {
  it("summarizes volume, reps and the best set of completed sets only", () => {
    const summary = summarizeSession(
      session("s1", "2026-11-10T10:00:00Z", [["bench", [set(60, 10), set(70, 6), set(80, 3, "skipped"), set(null, null, "pending")]]])
    );
    expect(summary.volumeKg).toBe(60 * 10 + 70 * 6);
    expect(summary.reps).toBe(16);
    expect(summary.completedSets).toBe(2);
    expect(summary.skippedSets).toBe(1);
    expect(summary.pendingSets).toBe(1);
    expect(summary.exercises[0].best).toEqual({ weight: 70, reps: 6 });
  });

  it("estimates a one-rep max with Epley and keeps single reps as they are", () => {
    expect(estimatedOneRepMax(100, 1)).toBe(100);
    expect(estimatedOneRepMax(100, 10)).toBeCloseTo(133.33, 1);
    expect(estimatedOneRepMax(0, 5)).toBe(0);
  });

  it("finds the previous sets from the most recent earlier session", () => {
    const history = [
      session("old", "2026-11-01T10:00:00Z", [["bench", [set(50, 10)]]]),
      session("recent", "2026-11-05T10:00:00Z", [["bench", [set(60, 8), set(60, 8)]]]),
      session("abandoned", "2026-11-06T10:00:00Z", [["bench", [set(90, 1)]]], "abandoned")
    ];
    const current = { id: "now", started_at: "2026-11-10T10:00:00Z" };
    const previous = previousSets(history, current);
    expect(previous.get("bench")?.map((entry) => entry.actual_weight_kg)).toEqual([60, 60]);
    expect(previous.has("squat")).toBe(false);
  });

  it("reports weight and estimated records, never for a first-time exercise", () => {
    const history = [session("before", "2026-11-01T10:00:00Z", [["bench", [set(80, 5)]]])];
    const heavier = session("now", "2026-11-10T10:00:00Z", [
      ["bench", [set(85, 3)]],
      ["squat", [set(100, 5)]]
    ]);
    const records = personalRecords(heavier, history);
    expect(records).toHaveLength(1);
    expect(records[0]).toMatchObject({ exerciseId: "bench", weight: 85, kind: "weight", previous: 80 });

    const moreReps = session("now2", "2026-11-11T10:00:00Z", [["bench", [set(80, 8)]]]);
    expect(personalRecords(moreReps, history)[0]).toMatchObject({ kind: "e1rm", weight: 80, reps: 8 });

    const same = session("now3", "2026-11-12T10:00:00Z", [["bench", [set(80, 5)]]]);
    expect(personalRecords(same, history)).toHaveLength(0);
  });

  it("buckets completed workouts per week and counts the streak", () => {
    const now = new Date("2026-11-12T12:00:00Z"); // Thursday
    const history = [
      session("a", "2026-11-10T10:00:00Z", [["bench", [set(60, 10)]]]),
      session("b", "2026-11-04T10:00:00Z", [["bench", [set(60, 10)]]]),
      session("c", "2026-10-28T10:00:00Z", [["bench", [set(60, 10)]]]),
      session("x", "2026-11-11T10:00:00Z", [["bench", [set(60, 10)]]], "abandoned")
    ];
    const buckets = weeklyBuckets(history, 4, now);
    expect(buckets.map((bucket) => bucket.workouts)).toEqual([0, 1, 1, 1]);
    expect(buckets[3].volumeKg).toBe(600);
    expect(currentStreakWeeks(history, now)).toBe(3);
  });

  it("builds an exercise series and lists trained exercises by frequency", () => {
    const history = [
      session("b", "2026-11-05T10:00:00Z", [["bench", [set(62.5, 8)]], ["squat", [set(100, 5)]]]),
      session("a", "2026-11-01T10:00:00Z", [["bench", [set(60, 8)]]])
    ];
    expect(exerciseSeries(history, "bench").map((point) => point.bestWeight)).toEqual([60, 62.5]);
    expect(trainedExercises(history).map((entry) => entry.exerciseId)).toEqual(["bench", "squat"]);
  });

  it("formats clocks with and without hours", () => {
    expect(formatClock(65)).toBe("1:05");
    expect(formatClock(3725)).toBe("1:02:05");
  });
});
