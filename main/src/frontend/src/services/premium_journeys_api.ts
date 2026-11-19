import { apiPost } from "@utils/http_client";

import type { MarketplaceProgram } from "./marketplace_api";
import type { PremiumLevel2AnswerValue } from "./premium_level_2_format";

export type PremiumObjective = "gain_muscle" | "strength_conditioning" | "lose_fat" | "sport_specific";

export interface PremiumMatchInput {
  objective: PremiumObjective;
  sex: "female" | "male";
  weight_kg: number;
  height_cm: number;
  age: number;
}

export interface PremiumMatch {
  match: { id: string; objective: PremiumObjective; created_at: string };
  program: MarketplaceProgram & { nutrition_mode: string | null; cover_image_url?: string };
}

// RYZE picks a published Premium Level 1 package for the objective at random,
// never one the client already owns (404 NO_MATCHING_PLAN, 409
// ALL_PLANS_OWNED otherwise).
export const matchPremiumLevel1 = (input: PremiumMatchInput) => apiPost<PremiumMatch>("/me/premium-level-1/match", input);
