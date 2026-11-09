import { describe, expect, it } from "vitest";

import type { PremiumLevel2NutritionPlan } from "@/services/premium_level_2_api";
import {
  emptyCuratedNutritionDraft,
  nutritionDraftToInput,
  nutritionPlanToDraft,
  validateNutritionDraft
} from "./l2_nutrition_mapping";

const plan: PremiumLevel2NutritionPlan = {
  id: "plan-1",
  request_id: "request-1",
  version: 3,
  status: "draft",
  dietary_pattern: "balanced",
  summary: "Summary",
  prep_guidance: "Prep",
  hydration_litres: 2.5,
  hydration_note: "Water",
  cautions: ["Caution one"],
  meals: [
    {
      id: "meal-1",
      plan_id: "plan-1",
      position: 1,
      label: "Breakfast",
      kind: "meal",
      notes: null,
      items: [
        {
          id: "food-1",
          plan_id: "plan-1",
          meal_id: "meal-1",
          position: 1,
          food_name: "Oats",
          category: "carbohydrate",
          quantity: 80,
          unit: "g",
          calories: 300,
          protein_grams: 10,
          carbohydrate_grams: 50,
          fat_grams: 6,
          fiber_grams: 8,
          substitution_note: null
        }
      ]
    }
  ],
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z"
};

describe("premium level 2 nutrition mapping", () => {
  it("round-trips a saved plan into the editable draft and back", () => {
    const draft = nutritionPlanToDraft(plan);
    expect(draft.meals[0].items[0].food_name).toBe("Oats");
    const input = nutritionDraftToInput(draft);
    expect(input.hydration_litres).toBe(2.5);
    expect(input.meals[0].foods[0]).toMatchObject({
      name: "Oats",
      quantity: 80,
      calories: 300,
      fiber_grams: 8
    });
    expect(input.cautions).toEqual(["Caution one"]);
  });

  it("accepts a complete draft", () => {
    const draft = nutritionPlanToDraft(plan);
    expect(validateNutritionDraft(draft)).toEqual([]);
  });

  it("reports missing copy and invalid food values", () => {
    const draft = emptyCuratedNutritionDraft();
    draft.meals[0].items[0].quantity = "0";
    const messages = validateNutritionDraft(draft);
    expect(messages).toContain("The nutrition plan needs a summary.");
    expect(messages).toContain("The nutrition plan needs preparation guidance.");
    expect(messages.some((message) => message.includes("positive quantity"))).toBe(true);
  });
});
