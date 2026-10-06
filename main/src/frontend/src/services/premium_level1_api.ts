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

// NUTRITION_RESTRICTIONS_UNSATISFIABLE is returned when no food in the plan's
// catalog survives the client's recorded restrictions. It is a refusal, not a
// crash: the delivered plan is untouched and no intake is ever altered.
const NUTRITION_RESTRICTIONS_UNSATISFIABLE = "NUTRITION_RESTRICTIONS_UNSATISFIABLE";

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

/** NutritionMacros is the shared energy-and-macro shape used by the plan, each
 *  meal and each food. */
export interface NutritionMacros {
  calories: number;
  protein_grams: number;
  carbs_grams: number;
  fat_grams: number;
  fiber_grams: number;
}

/** NutritionMealKind distinguishes the main meals the intake asked for from the
 *  smaller snack occasions. */
export type NutritionMealKind = "meal" | "snack";

export interface NutritionMealItem {
  position: number;
  food_name: string;
  category: string;
  quantity: number;
  unit: string;
  macros: NutritionMacros;
  /** Present only where the engine swapped the food. It names the equivalent the
   *  client may use instead, so a substitution is visible rather than silent. */
  substitution_note?: string;
}

export interface NutritionMeal {
  position: number;
  label: string;
  kind: NutritionMealKind;
  percent_of_daily: number;
  macros: NutritionMacros;
  notes?: string;
  items: NutritionMealItem[];
}

/** NutritionExclusionReasonCode explains why a food was kept out of the plan.
 *  It is a controlled server token, never raw questionnaire text. */
export type NutritionExclusionReason =
  | "allergy"
  | "intolerance"
  | "diet"
  | "excluded_food"
  | "disliked_food"
  | "insufficient_alternatives";

export interface NutritionExclusion {
  reason_code: NutritionExclusionReason;
  token: string;
}

export interface NutritionPlan {
  version: number;
  engine_version: number;
  fingerprint: string;
  dietary_pattern: string;
  maintenance_calories: number;
  target_calories: number;
  /** Daily carries the targets for the whole day, including the total energy. */
  daily: NutritionMacros;
  protein_percent: number;
  carbohydrate_percent: number;
  fat_percent: number;
  meals_per_day: number;
  snacks_per_day: number;
  meals: NutritionMeal[];
  /** Exclusions are the audit of what the plan removed and why. They are what
   *  makes the dietary guarantee inspectable instead of implicit. */
  exclusions: NutritionExclusion[];
  hydration: { daily_litres: number; note: string };
  cautions: string[];
  prep_guidance: string;
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
// delivered plan. An active plan whose questionnaire has moved on is superseded
// with a new version rather than rewritten, so the previous version stays
// recoverable; that is why the server alone decides when a refresh happens.
export const generateNutritionAssignment = (programId: string): Promise<NutritionAssignmentStatus> =>
  apiPost<NutritionAssignmentStatus>(`/me/programs/${programId}/nutrition/generate`, {});

// nutritionGenerationErrorMessage translates a failed generation into wording
// the client owns. The server's reason token is never rendered directly: only
// this mapping decides what the interface says, so a server wording change can
// never leak into the product. A refusal is reported as a refusal, never as a
// problem with the client's answers or their payment.
export const nutritionGenerationErrorMessage = (error: unknown): string => {
  if (error instanceof ApiError && error.code === NUTRITION_RESTRICTIONS_UNSATISFIABLE) {
    return "This programme's food list cannot cover every restriction you recorded. Your questionnaire, your plan and your purchase are all unchanged.";
  }
  return "We could not build your nutrition programme yet. Your purchase is unaffected — please try again.";
};