import type {
  PremiumLevel2NutritionInput,
  PremiumLevel2NutritionPlan
} from "@/services/premium_level_2_api";
import { uid, type PlanDraftCuratedItem, type PlanDraftCuratedNutrition } from "./plan_builder_model";

const emptyItem = (): PlanDraftCuratedItem => ({
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

export const emptyCuratedNutritionDraft = (): PlanDraftCuratedNutrition => ({
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
      items: [emptyItem()]
    }
  ]
});

export const nutritionPlanToDraft = (
  plan: PremiumLevel2NutritionPlan
): PlanDraftCuratedNutrition => ({
  dietary_pattern: plan.dietary_pattern,
  summary: plan.summary,
  prep_guidance: plan.prep_guidance,
  hydration_litres: String(plan.hydration_litres),
  hydration_note: plan.hydration_note,
  cautions: plan.cautions.join("\n"),
  meals: plan.meals.map((meal) => ({
    id: uid(),
    label: meal.label,
    kind: meal.kind,
    notes: meal.notes ?? "",
    items: meal.items.map((food) => ({
      id: uid(),
      food_name: food.food_name,
      category: food.category,
      quantity: String(food.quantity),
      unit: food.unit,
      calories: String(food.calories),
      protein_grams: String(food.protein_grams),
      carbohydrate_grams: String(food.carbohydrate_grams),
      fat_grams: String(food.fat_grams),
      fiber_grams: String(food.fiber_grams),
      substitution_note: food.substitution_note ?? ""
    }))
  }))
});

const numberOrZero = (value: string): number => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
};

export const nutritionDraftToInput = (
  draft: PlanDraftCuratedNutrition
): PremiumLevel2NutritionInput => ({
  dietary_pattern: draft.dietary_pattern.trim() || "balanced",
  summary: draft.summary.trim(),
  prep_guidance: draft.prep_guidance.trim(),
  hydration_litres: numberOrZero(draft.hydration_litres),
  hydration_note: draft.hydration_note.trim(),
  cautions: draft.cautions
    .split("\n")
    .map((caution) => caution.trim())
    .filter(Boolean),
  meals: draft.meals.map((meal) => ({
    label: meal.label.trim(),
    kind: meal.kind,
    notes: meal.notes.trim(),
    foods: meal.items.map((item) => ({
      name: item.food_name.trim(),
      category: item.category.trim(),
      quantity: numberOrZero(item.quantity),
      unit: item.unit.trim(),
      calories: numberOrZero(item.calories),
      protein_grams: numberOrZero(item.protein_grams),
      carbohydrate_grams: numberOrZero(item.carbohydrate_grams),
      fat_grams: numberOrZero(item.fat_grams),
      fiber_grams: numberOrZero(item.fiber_grams),
      substitution_note: item.substitution_note.trim()
    }))
  }))
});

export const validateNutritionDraft = (draft: PlanDraftCuratedNutrition): string[] => {
  const messages: string[] = [];
  if (!draft.summary.trim()) messages.push("The nutrition plan needs a summary.");
  if (!draft.prep_guidance.trim()) messages.push("The nutrition plan needs preparation guidance.");
  if (!draft.hydration_note.trim()) messages.push("The nutrition plan needs a hydration note.");
  const hydration = Number(draft.hydration_litres);
  if (!Number.isFinite(hydration) || hydration <= 0 || hydration > 20) {
    messages.push("Hydration must be between 0.1 and 20 litres.");
  }
  if (draft.meals.length === 0) messages.push("The nutrition plan needs at least one meal.");
  draft.meals.forEach((meal, mealIndex) => {
    if (!meal.label.trim()) messages.push(`Meal ${mealIndex + 1} needs a label.`);
    if (meal.items.length === 0) messages.push(`Meal ${mealIndex + 1} needs at least one food.`);
    meal.items.forEach((item, itemIndex) => {
      if (!item.food_name.trim() || !item.category.trim() || !item.unit.trim()) {
        messages.push(`Meal ${mealIndex + 1}, food ${itemIndex + 1} needs a name, category and unit.`);
      }
      if (!(Number(item.quantity) > 0)) {
        messages.push(`Meal ${mealIndex + 1}, food ${itemIndex + 1} needs a positive quantity.`);
      }
      for (const value of [
        item.calories,
        item.protein_grams,
        item.carbohydrate_grams,
        item.fat_grams,
        item.fiber_grams
      ]) {
        if (value.trim() === "" || !Number.isFinite(Number(value)) || Number(value) < 0) {
          messages.push(`Meal ${mealIndex + 1}, food ${itemIndex + 1} has an invalid nutrition value.`);
          break;
        }
      }
    });
  });
  return messages;
};
