import { apiGet, apiPost, ApiError } from "@utils/http_client";

// PREMIUM_LEVEL_1_PRODUCT_TYPE identifies the Premium Level 1 product family.
// It is the value the backend stores in programs.product_type. The frontend
// only reads it to explain the package; access is always decided server-side.
export const PREMIUM_LEVEL_1_PRODUCT_TYPE = "premium_level_1";

// NUTRITION_PLAN_NOT_FOUND is returned for a program the caller does not own,
// for a program without a nutrition plan, and for a plan that does not exist
// yet. The three cases are deliberately indistinguishable so the endpoint is
// never a purchase oracle.
const NUTRITION_PLAN_NOT_FOUND = "NUTRITION_NOT_FOUND";

// QUESTIONNAIRE_LOCKED is returned when the client tries to change a
// questionnaire that is immutable because the purchase already completed.
const QUESTIONNAIRE_LOCKED = "QUESTIONNAIRE_LOCKED";

// isQuestionnaireLockedError reports the server's post-purchase lock. The
// questionnaire is a purchased, immutable artefact once payment completed, so
// the client must stop editing and render the read-only state instead of
// retrying the submission.
export const isQuestionnaireLockedError = (error: unknown): boolean =>
  error instanceof ApiError && error.code === QUESTIONNAIRE_LOCKED;

// QuestionnaireQuestion is one entry of the server-owned intake contract. The
// client renders its form from this catalog, so the questions and the validation
// rules can never drift apart.
export interface QuestionnaireQuestion {
  field: string;
  label: string;
  type: string;
  required: boolean;
  options?: string[];
  min?: number;
  max?: number;
  max_length?: number;
  max_entries?: number;
  sensitive?: boolean;
  help?: string;
}

export interface QuestionnaireCatalog {
  version: number;
  questions: QuestionnaireQuestion[];
}

/** QuestionnaireRequirement is the authoritative intake state for one program.
 *  `locked` is the server's decision: the client must never decide editability
 *  from its own local state. */
export interface QuestionnaireRequirement {
  program_id: string;
  required: boolean;
  submitted: boolean;
  version: number;
  locked: boolean;
  schema_version: number;
  min_schema_version: number;
  submitted_at?: string;
}

/** QuestionnaireAnswerValue is a single answer. The questionnaire contract uses
 *  a handful of shapes, so the value stays a primitive or a list of primitives. */
export type QuestionnaireAnswerValue = string | number | string[];

export type QuestionnaireAnswers = Record<string, QuestionnaireAnswerValue>;

export interface QuestionnaireSubmitInput {
  answers: QuestionnaireAnswers;
}

export type NutritionAssignmentState = "pending" | "processing" | "completed" | "failed";

export interface NutritionEnergyTargets {
  maintenance_calories: number;
  target_calories: number;
  daily_protein_grams: number;
  daily_carbs_grams: number;
  daily_fat_grams: number;
}

export interface NutritionMacronutrients {
  protein_grams: number;
  carbohydrate_grams: number;
  fat_grams: number;
  fiber_grams: number;
  protein_percent: number;
  carbohydrate_percent: number;
  fat_percent: number;
  calories_per_gram: { protein: number; carbohydrate: number; fat: number };
}

export interface NutritionMealSlot {
  label: string;
  percent_of_daily: number;
  approx_calories: number;
  approx_protein_grams: number;
}

export interface NutritionPlan {
  version: number;
  fingerprint: string;
  energy_targets: NutritionEnergyTargets;
  macronutrients: NutritionMacronutrients;
  meal_plan: {
    meals_per_day: number;
    snacks_per_day: number;
    distribution: NutritionMealSlot[];
    prep_guidance: string;
  };
  hydration: { daily_litres: number; note: string };
  dietary_rules: {
    pattern: string;
    excluded_foods: string[];
    disliked_foods: string[];
    allergies: string[];
    intolerances: string[];
    medical_conditions: string[];
    injuries: string[];
    limitations: string[];
  };
  cautions: string[];
  summary: string;
}

/** NutritionAssignmentStatus is the server-owned assignment lifecycle. The
 *  client renders this state directly: it must never claim a plan is ready
 *  unless the backend reports `completed`. */
export interface NutritionAssignmentStatus {
  program_id: string;
  status: NutritionAssignmentState;
  version: number;
  questionnaire_version: number;
  out_of_date: boolean;
  plan: NutritionPlan | null;
}

// fetchQuestionnaireCatalog returns the server-owned intake contract. It is
// authenticated so the questionnaire surface is never served anonymously.
export const fetchQuestionnaireCatalog = (): Promise<QuestionnaireCatalog> =>
  apiGet<QuestionnaireCatalog>("/me/questionnaire/questions");

// fetchQuestionnaireRequirement reports whether the intake is required, already
// stored, and — authoritatively — whether it is locked.
export const fetchQuestionnaireRequirement = (programId: string): Promise<QuestionnaireRequirement> =>
  apiGet<QuestionnaireRequirement>(`/me/programs/${programId}/questionnaire`);

// submitQuestionnaire validates and stores the intake. It rejects with
// QUESTIONNAIRE_LOCKED once the purchase completed; the caller must then render
// the read-only state rather than retrying.
export const submitQuestionnaire = (
  programId: string,
  input: QuestionnaireSubmitInput
): Promise<QuestionnaireRequirement> =>
  apiPost<QuestionnaireRequirement>(`/me/programs/${programId}/questionnaire`, input);

// fetchNutritionAssignment returns the assignment state for a program the
// caller owns. A missing plan and an unowned program both surface as
// NUTRITION_NOT_FOUND.
export const fetchNutritionAssignment = (programId: string): Promise<NutritionAssignmentStatus> =>
  apiGet<NutritionAssignmentStatus>(`/me/programs/${programId}/nutrition`);

// generateNutritionAssignment runs the server-side generation step. It is
// idempotent, so retrying after a network failure converges on the same single
// delivered plan. There is deliberately no regenerate action: the plan is part
// of a purchased package and is immutable once delivered.
export const generateNutritionAssignment = (programId: string): Promise<NutritionAssignmentStatus> =>
  apiPost<NutritionAssignmentStatus>(`/me/programs/${programId}/nutrition/generate`, {});