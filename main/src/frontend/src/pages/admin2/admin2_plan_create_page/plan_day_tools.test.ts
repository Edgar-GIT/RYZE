import { describe, expect, it } from "vitest";

import {
  addExercisesToDay,
  clearDay,
  createPlanDraft,
  duplicateDay,
  findDay,
  moveDay,
  pasteDayAsNew,
  pasteDayInto,
  syncWeeks,
  updateDayDetails
} from "./plan_builder_model";

// Day tools (owner request, 2026-11-22): copies never share ids with the
// source; moving keeps positions consecutive.
const draft = () => {
  let plan = syncWeeks({ ...createPlanDraft(), duration_weeks: "2", frequency_per_week: "3" });
  const week = plan.weeks[0];
  plan = updateDayDetails(plan, week.id, week.workouts[0].id, { name: "Push" });
  return addExercisesToDay(plan, week.id, week.workouts[0].id, ["ex-1", "ex-2"]);
};

describe("plan day tools", () => {
  it("duplicates a day right after it with new ids", () => {
    const plan = draft();
    const week = plan.weeks[0];
    const next = duplicateDay(plan, week.id, week.workouts[0].id).weeks[0];
    expect(next.workouts.map((day) => day.name)).toEqual(["Push", "Push", "", ""]);
    expect(next.workouts[1].id).not.toBe(next.workouts[0].id);
    expect(next.workouts[1].exercises[0].id).not.toBe(next.workouts[0].exercises[0].id);
    expect(next.workouts.map((day) => day.position)).toEqual([1, 2, 3, 4]);
  });

  it("pastes into another week, as a new day or over a day", () => {
    const plan = draft();
    const source = findDay(plan, plan.weeks[0].id, plan.weeks[0].workouts[0].id)!;
    const target = plan.weeks[1];
    const asNew = pasteDayAsNew(plan, target.id, source).weeks[1];
    expect(asNew.workouts).toHaveLength(4);
    expect(asNew.workouts[3].exercises.map((exercise) => exercise.exercise_id)).toEqual(["ex-1", "ex-2"]);
    const over = pasteDayInto(plan, target.id, target.workouts[2].id, source).weeks[1];
    expect(over.workouts[2].name).toBe("Push");
    expect(over.workouts[2].id).toBe(target.workouts[2].id);
  });

  it("moves and clears days", () => {
    const plan = draft();
    const week = plan.weeks[0];
    const moved = moveDay(plan, week.id, week.workouts[0].id, 1).weeks[0];
    expect(moved.workouts.map((day) => day.name)).toEqual(["", "Push", ""]);
    expect(moveDay(plan, week.id, week.workouts[0].id, -1)).toEqual(plan);
    expect(clearDay(plan, week.id, week.workouts[0].id).weeks[0].workouts[0].exercises).toEqual([]);
  });
});
