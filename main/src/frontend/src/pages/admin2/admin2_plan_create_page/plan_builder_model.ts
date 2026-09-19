import { ProgramStatusEnum, ProgramType, type ProgramStatusValue, type ProgramTypeValue } from "@/services/admin2_api";
import type { SetType } from "@/services/exercise_library";

export interface PlanDraftSet {
  id: string;
  set_number: number;
  reps: string;
  weight_kg: string;
  rir: string;
  rpe: string;
  rest_seconds: string;
  tempo: string;
  set_type: SetType;
}

export interface PlanDraftExercise {
  id: string;
  exercise_id: string;
  position: number;
  sets: PlanDraftSet[];
  instructions: string;
  notes: string;
}

export interface PlanDraftWorkout {
  id: string;
  position: number;
  exercises: PlanDraftExercise[];
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

const createSet = (setNumber: number): PlanDraftSet => ({
  id: uid(),
  set_number: setNumber,
  reps: "",
  weight_kg: "",
  rir: "",
  rpe: "",
  rest_seconds: "",
  tempo: "",
  set_type: "working"
});

const createExerciseAssignment = (exerciseId: string, position: number): PlanDraftExercise => ({
  id: uid(),
  exercise_id: exerciseId,
  position,
  sets: [createSet(1)],
  instructions: "",
  notes: ""
});

const mapWorkout = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  update: (workout: PlanDraftWorkout) => PlanDraftWorkout
): PlanDraft => ({
  ...draft,
  weeks: draft.weeks.map((week) =>
    week.id === weekId
      ? {
          ...week,
          workouts: week.workouts.map((workout) => (workout.id === workoutId ? update(workout) : workout))
        }
      : week
  )
});

const renumberSets = (sets: PlanDraftSet[]): PlanDraftSet[] =>
  sets.map((set, index) => ({
    ...set,
    set_number: index + 1
  }));

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
            { id: uid(), position: week.workouts.length + 1, exercises: [] }
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
): PlanDraft =>
  mapWorkout(draft, weekId, workoutId, (workout) => {
    if (workout.exercises.some((assignment) => assignment.exercise_id === exerciseId)) {
      return workout;
    }
    return {
      ...workout,
      exercises: [
        ...workout.exercises,
        createExerciseAssignment(exerciseId, workout.exercises.length + 1)
      ]
    };
  });

export const removeExerciseFromDay = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  assignmentId: string
): PlanDraft =>
  mapWorkout(draft, weekId, workoutId, (workout) => {
    const exercises = workout.exercises
      .filter((assignment) => assignment.id !== assignmentId)
      .map((assignment, index) => ({ ...assignment, position: index + 1 }));
    return { ...workout, exercises };
  });

export const moveExerciseInDay = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  assignmentId: string,
  direction: -1 | 1
): PlanDraft =>
  mapWorkout(draft, weekId, workoutId, (workout) => {
    const index = workout.exercises.findIndex((assignment) => assignment.id === assignmentId);
    const target = index + direction;
    if (index < 0 || target < 0 || target >= workout.exercises.length) {
      return workout;
    }
    const reordered = [...workout.exercises];
    [reordered[index], reordered[target]] = [reordered[target], reordered[index]];
    return {
      ...workout,
      exercises: reordered.map((assignment, position) => ({ ...assignment, position: position + 1 }))
    };
  });

export const replaceExerciseInDay = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  assignmentId: string,
  exerciseId: string
): PlanDraft =>
  mapWorkout(draft, weekId, workoutId, (workout) => ({
    ...workout,
    exercises: workout.exercises.map((assignment) =>
      assignment.id === assignmentId && assignment.exercise_id !== exerciseId
        ? { ...assignment, exercise_id: exerciseId, sets: [createSet(1)] }
        : assignment
    )
  }));

export const updateExerciseAssignment = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  assignmentId: string,
  patch: Partial<Pick<PlanDraftExercise, "instructions" | "notes">>
): PlanDraft =>
  mapWorkout(draft, weekId, workoutId, (workout) => ({
    ...workout,
    exercises: workout.exercises.map((assignment) =>
      assignment.id === assignmentId ? { ...assignment, ...patch } : assignment
    )
  }));

export const addSetToExercise = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  assignmentId: string
): PlanDraft =>
  mapWorkout(draft, weekId, workoutId, (workout) => ({
    ...workout,
    exercises: workout.exercises.map((assignment) => {
      if (assignment.id !== assignmentId) {
        return assignment;
      }
      return {
        ...assignment,
        sets: [...assignment.sets, createSet(assignment.sets.length + 1)]
      };
    })
  }));

export const removeSetFromExercise = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  assignmentId: string,
  setId: string
): PlanDraft =>
  mapWorkout(draft, weekId, workoutId, (workout) => ({
    ...workout,
    exercises: workout.exercises.map((assignment) => {
      if (assignment.id !== assignmentId || assignment.sets.length <= 1) {
        return assignment;
      }
      return {
        ...assignment,
        sets: renumberSets(assignment.sets.filter((set) => set.id !== setId))
      };
    })
  }));

export const updateSetInExercise = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  assignmentId: string,
  setId: string,
  patch: Partial<Omit<PlanDraftSet, "id" | "set_number">>
): PlanDraft =>
  mapWorkout(draft, weekId, workoutId, (workout) => ({
    ...workout,
    exercises: workout.exercises.map((assignment) => {
      if (assignment.id !== assignmentId) {
        return assignment;
      }
      return {
        ...assignment,
        sets: assignment.sets.map((set) => (set.id === setId ? { ...set, ...patch } : set))
      };
    })
  }));

export const countWeeks = (draft: PlanDraft): number => draft.weeks.length;
export const countDays = (draft: PlanDraft): number =>
  draft.weeks.reduce((acc, week) => acc + week.workouts.length, 0);
export const countExercises = (draft: PlanDraft): number =>
  draft.weeks.reduce(
    (acc, week) => acc + week.workouts.reduce((inner, workout) => inner + workout.exercises.length, 0),
    0
  );
export const countSets = (draft: PlanDraft): number =>
  draft.weeks.reduce(
    (acc, week) =>
      acc +
      week.workouts.reduce(
        (inner, workout) => inner + workout.exercises.reduce((sets, assignment) => sets + assignment.sets.length, 0),
        0
      ),
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