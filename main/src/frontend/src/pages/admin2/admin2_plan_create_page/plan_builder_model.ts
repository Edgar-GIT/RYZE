import { ProgramStatusEnum, ProgramType, type ProgramStatusValue, type ProgramTypeValue } from "@/services/admin2_api";

export interface PlanDraftWorkout {
  id: string;
  position: number;
  exercise_ids: string[];
}

export interface PlanDraftWeek {
  id: string;
  week_number: number;
  workouts: PlanDraftWorkout[];
}

export interface PlanDraft {
  name: string;
  description: string;
  type: ProgramTypeValue;
  status: ProgramStatusValue;
  price: string;
  currency: string;
  weeks: PlanDraftWeek[];
}

export interface PlanValidation {
  valid: boolean;
  messages: string[];
}

export const uid = (): string => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `id-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
};

export const createPlanDraft = (): PlanDraft => ({
  name: "",
  description: "",
  type: ProgramType.FREE,
  status: ProgramStatusEnum.DRAFT,
  price: "0.00",
  currency: "EUR",
  weeks: [{ id: uid(), week_number: 1, workouts: [] }]
});

export const addWeek = (draft: PlanDraft): PlanDraft => ({
  ...draft,
  weeks: [
    ...draft.weeks,
    {
      id: uid(),
      week_number: draft.weeks.length + 1,
      workouts: []
    }
  ]
});

export const removeWeek = (draft: PlanDraft, weekId: string): PlanDraft => {
  const weeks = draft.weeks
    .filter((week) => week.id !== weekId)
    .map((week, index) => ({ ...week, week_number: index + 1 }));
  return { ...draft, weeks };
};

export const addDay = (draft: PlanDraft, weekId: string): PlanDraft => ({
  ...draft,
  weeks: draft.weeks.map((week) =>
    week.id === weekId
      ? {
          ...week,
          workouts: [
            ...week.workouts,
            { id: uid(), position: week.workouts.length + 1, exercise_ids: [] }
          ]
        }
      : week
  )
});

export const removeDay = (draft: PlanDraft, weekId: string, workoutId: string): PlanDraft => ({
  ...draft,
  weeks: draft.weeks.map((week) => {
    if (week.id !== weekId) {
      return week;
    }
    const workouts = week.workouts
      .filter((workout) => workout.id !== workoutId)
      .map((workout, index) => ({ ...workout, position: index + 1 }));
    return { ...week, workouts };
  })
});

export const addExerciseToDay = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  exerciseId: string
): PlanDraft => ({
  ...draft,
  weeks: draft.weeks.map((week) =>
    week.id === weekId
      ? {
          ...week,
          workouts: week.workouts.map((workout) =>
            workout.id === workoutId && !workout.exercise_ids.includes(exerciseId)
              ? { ...workout, exercise_ids: [...workout.exercise_ids, exerciseId] }
              : workout
          )
        }
      : week
  )
});

export const removeExerciseFromDay = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  exerciseId: string
): PlanDraft => ({
  ...draft,
  weeks: draft.weeks.map((week) =>
    week.id === weekId
      ? {
          ...week,
          workouts: week.workouts.map((workout) =>
            workout.id === workoutId
              ? { ...workout, exercise_ids: workout.exercise_ids.filter((id) => id !== exerciseId) }
              : workout
          )
        }
      : week
  )
});

export const countWeeks = (draft: PlanDraft): number => draft.weeks.length;
export const countDays = (draft: PlanDraft): number =>
  draft.weeks.reduce((acc, week) => acc + week.workouts.length, 0);
export const countExercises = (draft: PlanDraft): number =>
  draft.weeks.reduce(
    (acc, week) => acc + week.workouts.reduce((inner, workout) => inner + workout.exercise_ids.length, 0),
    0
  );

export const validatePlan = (draft: PlanDraft): PlanValidation => {
  const messages: string[] = [];

  if (!draft.name.trim()) {
    messages.push("The plan needs a name.");
  }
  if (!draft.description.trim()) {
    messages.push("The plan needs a description.");
  }
  if (draft.weeks.length === 0) {
    messages.push("The plan needs at least one week.");
  }
  if (countExercises(draft) === 0) {
    messages.push("The plan needs at least one exercise across its training days.");
  }
  if (draft.type !== ProgramType.FREE) {
    const parsedPrice = Number(draft.price);
    if (Number.isNaN(parsedPrice) || Math.round(parsedPrice * 100) < 100) {
      messages.push("Paid plans must have a price of at least €1.00.");
    }
  }

  return { valid: messages.length === 0, messages };
};