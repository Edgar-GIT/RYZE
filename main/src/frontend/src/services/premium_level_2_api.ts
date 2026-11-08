import { apiGet, apiPost, apiPut, encodePathSegment } from "@utils/http_client";

import type { AdminPagination } from "./admin_api";
import type { MarketplaceProgram } from "./marketplace_api";
import type { WorkoutSetTypeValue } from "./trainer_api";

// Premium Level 2 is a purchased, trainer-delivered service. The public offer
// is an ordinary published program; the request, intake, authored program and
// nutrition exist only after the purchase entitlement is active. Nothing in
// this module decides authorisation: every read and write is verified by the
// backend from the authenticated session.

export const PREMIUM_LEVEL_2_PRODUCT_TYPE = "premium_level_2";

export type PremiumLevel2RequestStatus =
  | "awaiting_intake"
  | "awaiting_trainer_assignment"
  | "in_progress"
  | "ready_for_delivery"
  | "delivered"
  | "cancelled";

export const PREMIUM_LEVEL_2_STATUS_LABELS: Record<PremiumLevel2RequestStatus, string> = {
  awaiting_intake: "Intake required",
  awaiting_trainer_assignment: "Trainer allocation pending",
  in_progress: "Trainer working on your plan",
  ready_for_delivery: "Ready for delivery",
  delivered: "Delivered",
  cancelled: "Cancelled"
};

export const isPremiumLevel2Status = (value: string): value is PremiumLevel2RequestStatus =>
  value in PREMIUM_LEVEL_2_STATUS_LABELS;

export interface PremiumLevel2QuestionOption {
  value: string;
  label: string;
}

export interface PremiumLevel2Question {
  key: string;
  label: string;
  type: "text" | "textarea" | "number" | "select" | "multiselect" | "boolean";
  required: boolean;
  help?: string;
  placeholder?: string;
  options?: PremiumLevel2QuestionOption[];
  min?: number;
  max?: number;
  step?: number;
  max_length?: number;
}

export interface PremiumLevel2QuestionSection {
  key: string;
  title: string;
  description: string;
  questions: PremiumLevel2Question[];
}

export interface PremiumLevel2Catalog {
  version: number;
  sections: PremiumLevel2QuestionSection[];
}

export interface PremiumLevel2Request {
  id: string;
  program_id: string;
  offer_name: string;
  offer_description: string;
  offer_price_minor_units: number;
  offer_currency: string;
  status: PremiumLevel2RequestStatus;
  intake_version: number;
  trainer_id: string | null;
  trainer_name: string;
  training_program_id: string | null;
  training_program_name: string | null;
  nutrition_plan_version: number | null;
  nutrition_plan_status: string | null;
  delivered_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface StaffPremiumLevel2Request extends PremiumLevel2Request {
  client_first_name: string;
  client_last_name: string;
  client_email: string;
}

export interface PremiumLevel2Intake {
  id: string;
  version: number;
  answers: Record<string, unknown>;
  submitted_at: string;
}

export interface PremiumLevel2PlanFood {
  id: string;
  plan_id: string;
  meal_id: string;
  position: number;
  food_name: string;
  category: string;
  quantity: number;
  unit: string;
  calories: number;
  protein_grams: number;
  carbohydrate_grams: number;
  fat_grams: number;
  fiber_grams: number;
  substitution_note: string | null;
}

export interface PremiumLevel2PlanMeal {
  id: string;
  plan_id: string;
  position: number;
  label: string;
  kind: "meal" | "snack";
  notes: string | null;
  items: PremiumLevel2PlanFood[];
}

export interface PremiumLevel2NutritionPlan {
  id: string;
  request_id: string;
  version: number;
  status: "draft" | "delivered";
  dietary_pattern: string;
  summary: string;
  prep_guidance: string;
  hydration_litres: number;
  hydration_note: string;
  cautions: string[];
  meals: PremiumLevel2PlanMeal[];
  created_at: string;
  updated_at: string;
}

export interface PremiumLevel2Event {
  event_type: string;
  actor_kind: string;
  from_status: string | null;
  to_status: string | null;
  created_at: string;
}

export interface PremiumLevel2ClientDetail {
  request: PremiumLevel2Request;
  intake: PremiumLevel2Intake | null;
  nutrition_plan: PremiumLevel2NutritionPlan | null;
}

export interface PremiumLevel2StaffDetail {
  request: StaffPremiumLevel2Request;
  intake: PremiumLevel2Intake | null;
  nutrition_plan: PremiumLevel2NutritionPlan | null;
  events: PremiumLevel2Event[];
}

export interface PremiumLevel2RequestsList {
  requests: StaffPremiumLevel2Request[];
  pagination: AdminPagination;
}

export interface PremiumLevel2FoodInput {
  name: string;
  category: string;
  quantity: number;
  unit: string;
  calories: number;
  protein_grams: number;
  carbohydrate_grams: number;
  fat_grams: number;
  fiber_grams: number;
  substitution_note: string;
}

export interface PremiumLevel2MealInput {
  label: string;
  kind: "meal" | "snack";
  notes: string;
  foods: PremiumLevel2FoodInput[];
}

export interface PremiumLevel2NutritionInput {
  dietary_pattern: string;
  summary: string;
  prep_guidance: string;
  hydration_litres: number;
  hydration_note: string;
  cautions: string[];
  meals: PremiumLevel2MealInput[];
}

export interface PremiumLevel2ProgramOption {
  id: string;
  name: string;
  description: string;
  status: string;
}

export const fetchPremiumLevel2Programs = async (limit = 20): Promise<MarketplaceProgram[]> => {
  const query = new URLSearchParams({ scope: PREMIUM_LEVEL_2_PRODUCT_TYPE, page: "1", limit: String(limit) });
  const data = await apiGet<{ programs: MarketplaceProgram[] }>(`/programs?${query.toString()}`);
  return data.programs;
};

export const fetchPremiumLevel2Catalog = (): Promise<PremiumLevel2Catalog> =>
  apiGet<PremiumLevel2Catalog>("/me/premium-level-2/catalog");

export const fetchMyPremiumLevel2Requests = async (): Promise<PremiumLevel2Request[]> => {
  const data = await apiGet<{ requests: PremiumLevel2Request[] }>("/me/premium-level-2");
  return data.requests;
};

export const fetchMyPremiumLevel2Request = (requestId: string): Promise<PremiumLevel2ClientDetail> =>
  apiGet<PremiumLevel2ClientDetail>(`/me/premium-level-2/${encodePathSegment(requestId)}`);

export const submitPremiumLevel2Intake = (
  requestId: string,
  answers: Record<string, unknown>
): Promise<{ request: PremiumLevel2Request; intake: PremiumLevel2Intake }> =>
  apiPut<{ request: PremiumLevel2Request; intake: PremiumLevel2Intake }>(
    `/me/premium-level-2/${encodePathSegment(requestId)}/intake`,
    { answers }
  );

export const fetchMyPremiumLevel2Nutrition = (
  requestId: string
): Promise<{ request: PremiumLevel2Request; plan: PremiumLevel2NutritionPlan }> =>
  apiGet<{ request: PremiumLevel2Request; plan: PremiumLevel2NutritionPlan }>(
    `/me/premium-level-2/${encodePathSegment(requestId)}/nutrition`
  );

const staffQuery = (page: number, limit: number, status: string): string => {
  const query = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (status) {
    query.set("status", status);
  }
  return query.toString();
};

export const fetchAdminPremiumLevel2Requests = (
  page: number,
  limit: number,
  status = ""
): Promise<PremiumLevel2RequestsList> =>
  apiGet<PremiumLevel2RequestsList>(`/admin/premium-level-2/requests?${staffQuery(page, limit, status)}`);

export const fetchAdminPremiumLevel2Request = (requestId: string): Promise<PremiumLevel2StaffDetail> =>
  apiGet<PremiumLevel2StaffDetail>(`/admin/premium-level-2/requests/${encodePathSegment(requestId)}`);

export const assignPremiumLevel2Trainer = (
  requestId: string,
  trainerId: string
): Promise<StaffPremiumLevel2Request> =>
  apiPost<StaffPremiumLevel2Request>(
    `/admin/premium-level-2/requests/${encodePathSegment(requestId)}/trainer`,
    { trainer_id: trainerId }
  );

export const fetchTrainerPremiumLevel2Requests = (
  page: number,
  limit: number,
  status = ""
): Promise<PremiumLevel2RequestsList> =>
  apiGet<PremiumLevel2RequestsList>(`/trainer/premium-level-2/requests?${staffQuery(page, limit, status)}`);

export const fetchTrainerPremiumLevel2Request = (requestId: string): Promise<PremiumLevel2StaffDetail> =>
  apiGet<PremiumLevel2StaffDetail>(`/trainer/premium-level-2/requests/${encodePathSegment(requestId)}`);

export const attachPremiumLevel2TrainingProgram = (
  requestId: string,
  programId: string
): Promise<StaffPremiumLevel2Request> =>
  apiPut<StaffPremiumLevel2Request>(
    `/trainer/premium-level-2/requests/${encodePathSegment(requestId)}/training-program`,
    { program_id: programId }
  );

export const savePremiumLevel2Nutrition = (
  requestId: string,
  input: PremiumLevel2NutritionInput
): Promise<PremiumLevel2NutritionPlan> =>
  apiPut<PremiumLevel2NutritionPlan>(
    `/trainer/premium-level-2/requests/${encodePathSegment(requestId)}/nutrition`,
    input
  );

export const deliverPremiumLevel2Request = (requestId: string): Promise<StaffPremiumLevel2Request> =>
  apiPost<StaffPremiumLevel2Request>(
    `/trainer/premium-level-2/requests/${encodePathSegment(requestId)}/deliver`,
    {}
  );

// The plan builder and the trainer workspace reuse the same nutrition shape.
// Keeping this conversion in one place prevents the two surfaces from drifting
// when the prescription model changes.
export const mealKindOf = (kind: string): "meal" | "snack" => (kind === "snack" ? "snack" : "meal");

export const setTypeLabel = (type: WorkoutSetTypeValue): string => {
  switch (type) {
    case "warmup":
      return "Warm-up";
    case "working":
      return "Working";
    case "drop":
      return "Drop";
    case "backoff":
      return "Back-off";
    case "failure":
      return "Failure";
    default:
      return type;
  }
};
