import {
  PROGRAM_PRODUCT_TYPE,
  ProgramStatusEnum,
  ProgramType,
  type GenericProgramDetail,
  type GenericProgramInput,
  type GenericProgramLevel,
  type GenericTrainingType,
  type ProgramObjectiveValue,
  type ProgramNutritionModeValue,
  type ProgramProductTypeValue,
  type ProgramStatusValue,
  type ProgramTypeValue,
  type GenericCuratedNutritionItemInput,
  type GenericCuratedNutritionMealInput
} from "@/services/admin2_api";
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
  // Training-day details: a name ("Push"), a short description and the
  // muscle group the day focuses on (one of DAY_FOCUS_VALUES or "").
  name: string;
  description: string;
  focus: string;
  exercises: PlanDraftExercise[];
}

export interface PlanDraftWeek {
  id: string;
  week_number: number;
  workouts: PlanDraftWorkout[];
}

export interface PlanDraftCuratedItem {
  id: string;
  food_name: string;
  category: string;
  quantity: string;
  unit: string;
  calories: string;
  protein_grams: string;
  carbohydrate_grams: string;
  fat_grams: string;
  fiber_grams: string;
  substitution_note: string;
}

export interface PlanDraftCuratedMeal {
  id: string;
  label: string;
  kind: "meal" | "snack";
  notes: string;
  items: PlanDraftCuratedItem[];
}

export interface PlanDraftCuratedNutrition {
  dietary_pattern: string;
  summary: string;
  prep_guidance: string;
  hydration_litres: string;
  hydration_note: string;
  cautions: string;
  meals: PlanDraftCuratedMeal[];
}

export interface PlanDraft {
  name: string;
  description: string;
  type: ProgramTypeValue;
  product_type: ProgramProductTypeValue;
  objective: ProgramObjectiveValue | "";
  nutrition_mode: ProgramNutritionModeValue | "";
  curated_nutrition: PlanDraftCuratedNutrition | null;
  status: ProgramStatusValue;
  price: string;
  currency: string;
  level: GenericProgramLevel | "";
  training_type: GenericTrainingType | "";
  duration_weeks: string;
  frequency_per_week: string;
  // Cover picture: the uploads key the API returned and its public URL.
  cover_image: string;
  cover_url: string;
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

const createCuratedItem = (): PlanDraftCuratedItem => ({
  id: uid(),
  food_name: "",
  category: "",
  quantity: "",
  unit: "g",
  calories: "",
  protein_grams: "",
  carbohydrate_grams: "",
  fat_grams: "",
  fiber_grams: "",
  substitution_note: ""
});

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

// A new exercise starts with a common prescription (3 sets of 10, 90 s rest)
// that the coach adjusts inline.
const createExerciseAssignment = (exerciseId: string, position: number): PlanDraftExercise => ({
  id: uid(),
  exercise_id: exerciseId,
  position,
  sets: [1, 2, 3].map((setNumber) => ({ ...createSet(setNumber), reps: "10", rest_seconds: "90" })),
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

// createPlanDraft starts a new plan in the given product family. The family is a
// property of what is being built, not of the serialized payload: it chooses
// which administration route saves the plan, and it is never editable once the
// plan exists because the backend derives it from the route addressed.
export const createPlanDraft = (
  productType: ProgramProductTypeValue = PROGRAM_PRODUCT_TYPE.GENERIC
): PlanDraft => ({
  name: "",
  description: "",
  type:
    productType === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_2
      ? ProgramType.PREMIUM
      : ProgramType.FREE,
  product_type: productType,
  objective: "",
  nutrition_mode:
    productType === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1 ? "curated" : "",
  curated_nutrition:
    productType === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1
      ? {
          dietary_pattern: "balanced",
          summary: "",
          prep_guidance: "",
          hydration_litres: "2.5",
          hydration_note: "",
          cautions: "",
          meals: [
            {
              id: uid(),
              label: "Breakfast",
              kind: "meal",
              notes: "",
              items: [createCuratedItem()]
            }
          ]
        }
      : null,
  status: ProgramStatusEnum.DRAFT,
  price: "0.00",
  currency: "EUR",
  level: "",
  training_type: "",
  duration_weeks: productType === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_2 ? "" : "4",
  frequency_per_week: productType === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_2 ? "" : "3",
  cover_image: "",
  cover_url: "",
  weeks: []
});

// --- Training-day focus --------------------------------------------------

// DAY_FOCUS_VALUES is the closed muscle-group vocabulary of a training day
// (it matches the API and the database constraint).
export const DAY_FOCUS_VALUES = [
  "full_body",
  "upper_body",
  "lower_body",
  "push",
  "pull",
  "legs",
  "chest",
  "back",
  "shoulders",
  "arms",
  "core",
  "glutes",
  "cardio",
  "mobility"
] as const;

export type DayFocus = (typeof DAY_FOCUS_VALUES)[number];

const createDay = (position: number): PlanDraftWorkout => ({
  id: uid(),
  position,
  name: "",
  description: "",
  focus: "",
  exercises: []
});

const createWeek = (weekNumber: number, days: number): PlanDraftWeek => ({
  id: uid(),
  week_number: weekNumber,
  workouts: Array.from({ length: days }, (_, index) => createDay(index + 1))
});

const weekHasContent = (week: PlanDraftWeek) =>
  week.workouts.some(
    (workout) => workout.exercises.length > 0 || workout.name.trim() !== "" || workout.description.trim() !== ""
  );

// syncWeeks makes the structure match the number of weeks chosen in the first
// step: missing weeks are created with one empty training day per weekly
// session, and surplus weeks are removed only while they are still empty (a
// week with content is never discarded silently). Returns the new draft.
export const syncWeeks = (draft: PlanDraft): PlanDraft => {
  const target = Math.min(52, Math.max(1, Number(draft.duration_weeks) || 1));
  const days = Math.min(7, Math.max(1, Number(draft.frequency_per_week) || 3));
  let weeks = [...draft.weeks];
  while (weeks.length < target) {
    weeks.push(createWeek(weeks.length + 1, days));
  }
  while (weeks.length > target && !weekHasContent(weeks[weeks.length - 1])) {
    weeks = weeks.slice(0, -1);
  }
  return { ...draft, weeks, duration_weeks: String(weeks.length) };
};

const cloneWorkouts = (workouts: PlanDraftWorkout[]): PlanDraftWorkout[] =>
  workouts.map((workout) => ({
    ...workout,
    id: uid(),
    exercises: workout.exercises.map((assignment) => ({
      ...assignment,
      id: uid(),
      sets: assignment.sets.map((set) => ({ ...set, id: uid() }))
    }))
  }));

// copyWeekForward replaces every following week with a copy of the given
// week (the usual way to build a block of identical weeks).
export const copyWeekForward = (draft: PlanDraft, weekId: string): PlanDraft => {
  const index = draft.weeks.findIndex((week) => week.id === weekId);
  if (index < 0) return draft;
  const source = draft.weeks[index];
  return {
    ...draft,
    weeks: draft.weeks.map((week, position) =>
      position > index ? { ...week, workouts: cloneWorkouts(source.workouts) } : week
    )
  };
};

export const updateDayDetails = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  patch: Partial<Pick<PlanDraftWorkout, "name" | "description" | "focus">>
): PlanDraft => mapWorkout(draft, weekId, workoutId, (workout) => ({ ...workout, ...patch }));

// addExercisesToDay appends several catalogue exercises at once (exercises
// already in the day are skipped: the API rejects duplicates per day).
export const addExercisesToDay = (
  draft: PlanDraft,
  weekId: string,
  workoutId: string,
  exerciseIds: string[]
): PlanDraft =>
  exerciseIds.reduce((current, exerciseId) => addExerciseToDay(current, weekId, workoutId, exerciseId), draft);

export const addWeek = (draft: PlanDraft): PlanDraft => {
  const days = Math.min(7, Math.max(1, Number(draft.frequency_per_week) || 3));
  const weeks = [...draft.weeks, createWeek(draft.weeks.length + 1, days)];
  return { ...draft, weeks, duration_weeks: String(weeks.length) };
};

export const removeWeek = (draft: PlanDraft, weekId: string): PlanDraft => {
  const weeks = draft.weeks
    .filter((week) => week.id !== weekId)
    .map((week, index) => ({ ...week, week_number: index + 1 }));
  return { ...draft, weeks, duration_weeks: String(Math.max(weeks.length, 1)) };
};

export const addDay = (draft: PlanDraft, weekId: string): PlanDraft => ({
  ...draft,
  weeks: draft.weeks.map((week) =>
    week.id === weekId
      ? {
          ...week,
          workouts: [...week.workouts, createDay(week.workouts.length + 1)]
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
      // A new set repeats the previous one, which is what a coach types next.
      const last = assignment.sets[assignment.sets.length - 1];
      const next = last
        ? { ...last, id: uid(), set_number: assignment.sets.length + 1 }
        : createSet(assignment.sets.length + 1);
      return {
        ...assignment,
        sets: [...assignment.sets, next]
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
  const isLevel2 = draft.product_type === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_2;
  if (!isLevel2 && draft.weeks.length === 0) {
    messages.push("The plan needs at least one week.");
  }
  if (!isLevel2 && countExercises(draft) === 0) {
    messages.push("The plan needs at least one exercise across its training days.");
  }

  if (!isLevel2) {
    const duration = optionalNumber(draft.duration_weeks);
    if (duration !== null && (duration < 1 || duration > 52)) {
      messages.push("The plan duration must be between 1 and 52 weeks.");
    }
    const frequency = optionalNumber(draft.frequency_per_week);
    if (frequency !== null && (frequency < 1 || frequency > 7)) {
      messages.push("The frequency must be between 1 and 7 training days per week.");
    }
  }

  if (draft.product_type === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1) {
    if (!draft.objective) {
      messages.push("Choose the package objective.");
    }
    if (draft.nutrition_mode === "curated") {
      validateCuratedNutrition(draft.curated_nutrition, messages);
    }
  }

  return { valid: messages.length === 0, messages };
};

const validateCuratedNutrition = (
  nutrition: PlanDraftCuratedNutrition | null,
  messages: string[]
): void => {
  if (!nutrition) {
    messages.push("A curated package needs a nutrition plan.");
    return;
  }
  if (!nutrition.summary.trim()) {
    messages.push("The nutrition plan needs a summary.");
  }
  if (!nutrition.prep_guidance.trim()) {
    messages.push("The nutrition plan needs preparation guidance.");
  }
  if (!nutrition.hydration_note.trim()) {
    messages.push("The nutrition plan needs a hydration note.");
  }
  const hydration = optionalNumber(nutrition.hydration_litres);
  if (hydration === null || hydration <= 0 || hydration > 20) {
    messages.push("Hydration must be between 0.1 and 20 litres.");
  }
  if (nutrition.meals.length === 0) {
    messages.push("The nutrition plan needs at least one meal.");
  }
  nutrition.meals.forEach((meal, index) => {
    if (!meal.label.trim()) {
      messages.push(`Meal ${index + 1} needs a label.`);
    }
    if (meal.items.length === 0) {
      messages.push(`Meal ${index + 1} needs at least one food.`);
    }
    meal.items.forEach((item, itemIndex) => {
      if (!item.food_name.trim() || !item.category.trim() || !item.unit.trim()) {
        messages.push(`Meal ${index + 1}, food ${itemIndex + 1} needs a name, category and unit.`);
      }
      if ((optionalNumber(item.quantity) ?? 0) <= 0) {
        messages.push(`Meal ${index + 1}, food ${itemIndex + 1} needs a positive quantity.`);
      }
      for (const value of [
        item.calories,
        item.protein_grams,
        item.carbohydrate_grams,
        item.fat_grams,
        item.fiber_grams
      ]) {
        if (optionalNumber(value) === null) {
          messages.push(`Meal ${index + 1}, food ${itemIndex + 1} has an invalid nutrition value.`);
          break;
        }
      }
    });
  });
};

const optionalNumber = (value: string): number | null => {
  const trimmed = value.trim();
  if (trimmed === "") {
    return null;
  }
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : null;
};

// toGenericProgramInput serializes the client-side PlanDraft into the generic
// program API payload. Structural ordering keys (week/week workout/exercise
// position) are derived server-side from array order and are never sent; the
// optional level, duration and frequency metadata are only sent when set, and
// the server maps empty values back to "not set".
export const toGenericProgramInput = (draft: PlanDraft): GenericProgramInput => {
  const isLevel2 = draft.product_type === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_2;
  return {
  name: draft.name.trim(),
  description: draft.description.trim(),
  type: draft.product_type === PROGRAM_PRODUCT_TYPE.GENERIC ? ProgramType.FREE : ProgramType.PREMIUM,
  status: draft.status,
  objective: draft.product_type === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1 ? draft.objective : "",
  nutrition_mode:
    draft.product_type === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1 ? draft.nutrition_mode : "",
  curated_nutrition:
    draft.product_type === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1 &&
    draft.nutrition_mode === "curated" &&
    draft.curated_nutrition
      ? toCuratedNutritionInput(draft.curated_nutrition)
      : null,
  level: isLevel2 ? "" : draft.level,
  training_type: isLevel2 ? "" : draft.training_type,
  duration_weeks: isLevel2 ? 0 : optionalNumber(draft.duration_weeks) ?? 0,
  frequency_per_week: isLevel2 ? 0 : optionalNumber(draft.frequency_per_week) ?? 0,
  // The price follows the family's catalogue category on the server; the
  // value sent here is ignored.
  price_minor_units: 0,
  currency: "EUR",
  cover_image: draft.cover_image,
  weeks: isLevel2
    ? []
    : draft.weeks.map((week) => ({
    workouts: week.workouts.map((workout) => ({
      name: workout.name.trim(),
      description: workout.description.trim(),
      focus: workout.focus,
      exercises: workout.exercises.map((assignment) => ({
        exercise_id: assignment.exercise_id,
        instructions: assignment.instructions,
        notes: assignment.notes,
        sets: assignment.sets.map((set) => ({
          set_type: set.set_type,
          reps: optionalNumber(set.reps),
          weight_kg: optionalNumber(set.weight_kg),
          rir: optionalNumber(set.rir),
          rpe: optionalNumber(set.rpe),
          rest_seconds: optionalNumber(set.rest_seconds),
          tempo: set.tempo
        }))
      }))
    }))
  }))
  };
};

const toCuratedNutritionInput = (nutrition: PlanDraftCuratedNutrition) => ({
  dietary_pattern: nutrition.dietary_pattern.trim() || "balanced",
  summary: nutrition.summary.trim(),
  prep_guidance: nutrition.prep_guidance.trim(),
  hydration_litres: optionalNumber(nutrition.hydration_litres) ?? 0,
  hydration_note: nutrition.hydration_note.trim(),
  cautions: nutrition.cautions
    .split("\n")
    .map((caution) => caution.trim())
    .filter(Boolean),
  meals: nutrition.meals.map<GenericCuratedNutritionMealInput>((meal) => ({
    label: meal.label.trim(),
    kind: meal.kind,
    notes: meal.notes.trim(),
    items: meal.items.map<GenericCuratedNutritionItemInput>((item) => ({
      food_name: item.food_name.trim(),
      category: item.category.trim(),
      quantity: optionalNumber(item.quantity) ?? 0,
      unit: item.unit.trim(),
      calories: optionalNumber(item.calories) ?? 0,
      protein_grams: optionalNumber(item.protein_grams) ?? 0,
      carbohydrate_grams: optionalNumber(item.carbohydrate_grams) ?? 0,
      fat_grams: optionalNumber(item.fat_grams) ?? 0,
      fiber_grams: optionalNumber(item.fiber_grams) ?? 0,
      substitution_note: item.substitution_note.trim()
    }))
  }))
});

// detailToPlanDraft maps an existing generic program (full structure response)
// back into an editable PlanDraft. Children receive fresh local ids because the
// update payload never carries ids: the server matches children by ordering and
// preserves untouched rows. Numeric prescriptions are surfaced as raw strings so
// the existing input fields keep working unchanged.
export const detailToPlanDraft = (detail: GenericProgramDetail): PlanDraft => ({
  name: detail.name,
  description: detail.description,
  type: detail.type,
  product_type: detail.product_type,
  objective: detail.objective ?? "",
  nutrition_mode: detail.nutrition_mode ?? "",
  curated_nutrition: detail.curated_nutrition
    ? {
        dietary_pattern: detail.curated_nutrition.dietary_pattern,
        summary: detail.curated_nutrition.summary,
        prep_guidance: detail.curated_nutrition.prep_guidance,
        hydration_litres: String(detail.curated_nutrition.hydration_litres),
        hydration_note: detail.curated_nutrition.hydration_note,
        cautions: detail.curated_nutrition.cautions.join("\n"),
        meals: detail.curated_nutrition.meals.map((meal) => ({
          id: uid(),
          label: meal.label,
          kind: meal.kind,
          notes: meal.notes ?? "",
          items: meal.items.map((item) => ({
            id: uid(),
            food_name: item.food_name,
            category: item.category,
            quantity: String(item.quantity),
            unit: item.unit,
            calories: String(item.calories),
            protein_grams: String(item.protein_grams),
            carbohydrate_grams: String(item.carbohydrate_grams),
            fat_grams: String(item.fat_grams),
            fiber_grams: String(item.fiber_grams),
            substitution_note: item.substitution_note ?? ""
          }))
        }))
      }
    : null,
  status: detail.status,
  price: (detail.price_minor_units / 100).toFixed(2),
  currency: detail.currency,
  level: detail.level ?? "",
  training_type: detail.training_type ?? "",
  duration_weeks: detail.duration_weeks === null ? "" : String(detail.duration_weeks),
  frequency_per_week: detail.frequency_per_week === null ? "" : String(detail.frequency_per_week),
  cover_image: detail.cover_image ?? "",
  cover_url: detail.cover_image_url ?? "",
  weeks: detail.weeks.map((week) => ({
    id: uid(),
    week_number: week.week_number,
    workouts: week.workouts.map((workout) => ({
      id: uid(),
      position: workout.position,
      name: workout.name ?? "",
      description: workout.description ?? "",
      focus: workout.focus ?? "",
      exercises: workout.exercises.map((assignment) => ({
        id: uid(),
        exercise_id: assignment.id,
        position: assignment.position,
        instructions: assignment.instructions,
        notes: assignment.notes,
        sets: assignment.sets.map((set) => ({
          id: uid(),
          set_number: set.set_number,
          reps: set.reps === null ? "" : String(set.reps),
          weight_kg: set.weight_kg === null ? "" : String(set.weight_kg),
          rir: set.rir === null ? "" : String(set.rir),
          rpe: set.rpe === null ? "" : String(set.rpe),
          rest_seconds: set.rest_seconds === null ? "" : String(set.rest_seconds),
          tempo: set.tempo,
          set_type: set.set_type
        }))
      }))
    }))
  }))
});