import { apiDelete, apiGet, apiPatch, apiPost, encodePathSegment } from "@utils/http_client";
import { AdminPagination } from "./admin_api";
import type { SetType } from "./exercise_library";
import { PREMIUM_LEVEL_1_PRODUCT_TYPE } from "./premium_level1_api";

export const ProgramType = {
  FREE: "free",
  PREMIUM: "premium",
  PERSONALIZED: "personalized"
} as const;

export type ProgramTypeValue = typeof ProgramType[keyof typeof ProgramType];

// PROGRAM_PRODUCT_TYPE identifies the product family a program belongs to
// (programs.product_type), which is orthogonal to ProgramType: the latter is
// the commercial/pricing category. The family decides which parts of the
// purchase lifecycle a program participates in and, for the management
// surfaces, which route a program is administered under. The value is kept in
// sync with the backend; the premium family constant is the single source
// already used by the Premium Level 1 product surfaces.
export const PROGRAM_PRODUCT_TYPE = {
  GENERIC: "generic",
  PREMIUM_LEVEL_1: PREMIUM_LEVEL_1_PRODUCT_TYPE
} as const;

export type ProgramProductTypeValue = typeof PROGRAM_PRODUCT_TYPE[keyof typeof PROGRAM_PRODUCT_TYPE];

export const PROGRAM_PRODUCT_TYPES: readonly ProgramProductTypeValue[] = [
  PROGRAM_PRODUCT_TYPE.GENERIC,
  PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1
];

// isProgramProductType narrows an unknown query value to a real family. Query
// strings are the only place a family value is ever read from the URL; storing
// it in a request payload would let the client choose a family, which the
// backend derives exclusively from the route.
export const isProgramProductType = (value: unknown): value is ProgramProductTypeValue =>
  PROGRAM_PRODUCT_TYPES.includes(value as ProgramProductTypeValue);

export const ProgramStatusEnum = {
  DRAFT: "draft",
  PUBLISHED: "published"
} as const;

export type ProgramStatusValue = typeof ProgramStatusEnum[keyof typeof ProgramStatusEnum];

// RYZE purchases are final: there is no refund workflow, so "refunded" is not
// a valid purchase status.
export const PurchaseStatusEnum = {
  PENDING: "pending",
  COMPLETED: "completed",
  FAILED: "failed"
} as const;

export type PurchaseStatus = typeof PurchaseStatusEnum[keyof typeof PurchaseStatusEnum];

export interface AdminProgram {
  id: string;
  trainer_id: string | null;
  name: string;
  description: string;
  type: ProgramTypeValue;
  status: ProgramStatusValue;
  product_type: ProgramProductTypeValue;
  price_minor_units: number;
  currency: string;
  created_at: string;
  updated_at: string;
}

export interface AdminProgramsList {
  programs: AdminProgram[];
  pagination: AdminPagination;
}

export interface AdminCommissionRule {
  id: string;
  trainer_id: string;
  commission_bps: number;
  valid_from: string;
  valid_until: string | null;
  created_at: string;
  updated_at: string;
}

export interface AdminCommissionResolution {
  commission_bps: number;
  is_override: boolean;
}

export interface AdminPricingInput {
  price_minor_units: number;
  currency: string;
}

export interface AdminPurchaseProgram {
  id: string;
  name: string;
  type: ProgramTypeValue;
  status: ProgramStatusValue;
  level: string | null;
  duration_weeks: number | null;
  frequency_per_week: number | null;
  training_type: string | null;
  price_minor_units: number;
  currency: string;
  deleted: boolean;
  created_at: string;
  updated_at: string;
}

// The admin purchase view exposes a curated slice of the purchase record:
// commission/payout data, the owning user id and all internal identifiers stay
// server-side and are never part of this contract.
export interface AdminPurchase {
  id: string;
  program_id: string;
  price_minor_units: number;
  currency: string;
  status: PurchaseStatus;
  test: boolean;
  access: boolean;
  created_at: string;
  updated_at: string;
  customer: {
    name: string;
    email: string;
  };
  program: AdminPurchaseProgram;
}

export interface AdminSalesSummary {
  completed_count: number;
  pending_count: number;
  failed_count: number;
  real_revenue_minor_units: number;
  test_purchase_count: number;
  programs_with_sales: number;
}

export interface AdminPurchaseListParams {
  page: number;
  limit: number;
  program_id?: string;
  // Batch program filter used by the program-management area: requests one
  // zero-filled sale summary per program id in a single aggregation.
  program_ids?: string[];
  status?: PurchaseStatus | "";
  test?: boolean;
  from?: string;
  to?: string;
}

export interface AdminPurchaseListResult {
  purchases: AdminPurchase[];
  summary: AdminSalesSummary;
  pagination: AdminPagination;
}

export interface AdminPurchaseDetailResponse {
  purchase: AdminPurchase;
}

export interface AdminProgramSale {
  program: AdminPurchaseProgram;
  completed_sales: number;
  // Completed non-test purchases: Test Mode purchases are reported separately
  // and never inflate the real sales figure.
  real_sales: number;
  revenue_minor_units: number;
  test_purchases: number;
  pending_purchases: number;
  failed_purchases: number;
}

export interface AdminProgramSalesListResult {
  sales: AdminProgramSale[];
  pagination: AdminPagination;
}

/* ------------------------------------------------------------------ */
/* Purchase reconciliation events                                       */
/* ------------------------------------------------------------------ */

// The reconciliation outcome vocabulary is fixed by the backend audit layer.
// These server codes are mapped to client-owned wording before rendering; the
// raw codes are never surfaced to the operator.
export const ReconciliationOutcomeEnum = {
  COMPLETED: "completed",
  ALREADY_COMPLETED: "already_completed",
  RECONCILED: "reconciled",
  CONFLICT_PAIR_RESOLVED: "conflict_pair_resolved",
  PURCHASE_MOVED_ON: "purchase_moved_on",
  ENTITLEMENT_INTEGRITY: "entitlement_integrity",
  TRANSIENT_FAILURE: "transient_failure",
  AMOUNT_MISMATCH: "amount_mismatch",
  CURRENCY_MISMATCH: "currency_mismatch",
  METHOD_MISMATCH: "method_mismatch",
  REFERENCE_MISMATCH: "reference_mismatch",
  PROVIDER_UNCERTAIN: "provider_uncertain",
  LATE_APPROVAL: "late_approval",
  PERMANENT_INVALID: "permanent_invalid",
  TEST_PURCHASE_EVENT: "test_purchase_event"
} as const;

export type ReconciliationOutcomeValue =
  typeof ReconciliationOutcomeEnum[keyof typeof ReconciliationOutcomeEnum];

export const reconciliationProviderLabel = (provider: string): string =>
  provider === "paypal" ? "PayPal" : "Stripe";

export type ReconciliationOutcomeCategory = "settled" | "needs-attention" | "rejected";

// Badge tone values mirror the Admin2StatusBadge contract.
export type AdminBadgeTone = "success" | "warning" | "danger" | "neutral";

export interface ReconciliationOutcomeMeta {
  label: string;
  category: ReconciliationOutcomeCategory;
  tone: AdminBadgeTone;
}

// Client-owned wording for every documented server outcome. The three
// "settled" outcomes are routine and excluded from the default open scope;
// everything else is flagged for investigation so the reconciliation view opens
// on what needs attention.
export const reconciliationOutcomeMeta = (outcome: string): ReconciliationOutcomeMeta => {
  switch (outcome) {
    case ReconciliationOutcomeEnum.COMPLETED:
      return { label: "Completed", category: "settled", tone: "success" };
    case ReconciliationOutcomeEnum.ALREADY_COMPLETED:
      return { label: "Already completed", category: "settled", tone: "success" };
    case ReconciliationOutcomeEnum.RECONCILED:
      return { label: "Reconciled", category: "settled", tone: "success" };
    case ReconciliationOutcomeEnum.CONFLICT_PAIR_RESOLVED:
      return { label: "Conflict resolved", category: "needs-attention", tone: "warning" };
    case ReconciliationOutcomeEnum.PURCHASE_MOVED_ON:
      return { label: "Purchase moved on", category: "needs-attention", tone: "warning" };
    case ReconciliationOutcomeEnum.ENTITLEMENT_INTEGRITY:
      return { label: "Entitlement integrity", category: "rejected", tone: "danger" };
    case ReconciliationOutcomeEnum.TRANSIENT_FAILURE:
      return { label: "Transient failure", category: "needs-attention", tone: "warning" };
    case ReconciliationOutcomeEnum.AMOUNT_MISMATCH:
      return { label: "Amount mismatch", category: "rejected", tone: "danger" };
    case ReconciliationOutcomeEnum.CURRENCY_MISMATCH:
      return { label: "Currency mismatch", category: "rejected", tone: "danger" };
    case ReconciliationOutcomeEnum.METHOD_MISMATCH:
      return { label: "Payment method mismatch", category: "rejected", tone: "danger" };
    case ReconciliationOutcomeEnum.REFERENCE_MISMATCH:
      return { label: "Reference mismatch", category: "rejected", tone: "danger" };
    case ReconciliationOutcomeEnum.PROVIDER_UNCERTAIN:
      return { label: "Uncertain delivery", category: "needs-attention", tone: "warning" };
    case ReconciliationOutcomeEnum.LATE_APPROVAL:
      return { label: "Late approval", category: "needs-attention", tone: "warning" };
    case ReconciliationOutcomeEnum.PERMANENT_INVALID:
      return { label: "Invalid delivery", category: "rejected", tone: "danger" };
    case ReconciliationOutcomeEnum.TEST_PURCHASE_EVENT:
      return { label: "Test purchase event", category: "needs-attention", tone: "neutral" };
    default:
      return { label: "Unknown outcome", category: "needs-attention", tone: "neutral" };
  }
};

export interface AdminReconciliationEvent {
  id: string;
  outcome: string;
  provider: string;
  event_id: string;
  created_at: string;
  purchase: AdminPurchase;
}

export interface AdminReconciliationListParams {
  page: number;
  limit: number;
  // "open" (default) surfaces the outcomes that need investigation; "all"
  // widens to the complete audit trail including routine settlements.
  scope?: "open" | "all";
  outcome?: ReconciliationOutcomeValue | "";
}

export interface AdminReconciliationListResult {
  events: AdminReconciliationEvent[];
  pagination: AdminPagination;
}

const reconciliationQuery = (params: AdminReconciliationListParams): string => {
  const query = new URLSearchParams({
    page: String(params.page),
    limit: String(params.limit)
  });
  if (params.scope) {
    query.set("scope", params.scope);
  }
  if (params.outcome) {
    query.set("outcome", params.outcome);
  }
  return query.toString();
};

export const fetchAdminReconciliationEvents = (
  params: AdminReconciliationListParams
): Promise<AdminReconciliationListResult> =>
  apiGet<AdminReconciliationListResult>(
    `/admin/purchases/reconciliation-events?${reconciliationQuery(params)}`
  );

const purchaseQuery = (params: AdminPurchaseListParams): string => {
  const query = new URLSearchParams({
    page: String(params.page),
    limit: String(params.limit)
  });
  if (params.program_id) {
    query.set("program_id", params.program_id);
  }
  if (params.program_ids && params.program_ids.length > 0) {
    query.set("program_ids", params.program_ids.join(","));
  }
  if (params.status) {
    query.set("status", params.status);
  }
  if (params.test !== undefined) {
    query.set("test", String(params.test));
  }
  if (params.from) {
    query.set("from", params.from);
  }
  if (params.to) {
    query.set("to", params.to);
  }
  return query.toString();
};

export const fetchAdminPurchases = (params: AdminPurchaseListParams): Promise<AdminPurchaseListResult> =>
  apiGet<AdminPurchaseListResult>(`/admin/purchases?${purchaseQuery(params)}`);

export const fetchAdminPurchase = (id: string): Promise<AdminPurchase> =>
  apiGet<AdminPurchaseDetailResponse>(`/admin/purchases/${encodePathSegment(id)}`).then((result) => result.purchase);

export const fetchAdminProgramSales = (
  params: AdminPurchaseListParams
): Promise<AdminProgramSalesListResult> =>
  apiGet<AdminProgramSalesListResult>(`/admin/purchases/programs?${purchaseQuery(params)}`);

// fetchAdminSalesByProgramIds requests one zero-filled sale summary per program
// id in a single aggregation. It is used by the program-management area to show
// per-program sales without leaking sale data through the general program
// endpoints.
export const fetchAdminSalesByProgramIds = (program_ids: string[]): Promise<AdminProgramSale[]> =>
  fetchAdminProgramSales({ page: 1, limit: program_ids.length, program_ids }).then(
    (result) => result.sales
  );

export interface AdminFeedback {
  id: string;
  user_id: string;
  user_name: string;
  rating: number;
  message: string;
  created_at: string;
}

export interface AdminFeedbackListResult {
  available: boolean;
  feedback: AdminFeedback[];
  reason: string;
}

// programTypeLabel names the commercial pricing category (programs.type). It is
// deliberately independent of the product family: FREE/PREMIUM/PERSONALIZED are
// the pricing categories, while programFamilyLabel names the product family.
export const programTypeLabel = (type: ProgramTypeValue): string => {
  if (type === ProgramType.FREE) {
    return "Free Plan";
  }
  if (type === ProgramType.PREMIUM) {
    return "Premium Plan";
  }
  return "Personalized Plan";
};

export const programTypeShortLabel = (type: ProgramTypeValue): string => {
  if (type === ProgramType.FREE) {
    return "Free";
  }
  if (type === ProgramType.PREMIUM) {
    return "Premium";
  }
  return "Personalized";
};

// programFamilyLabel names the product family (programs.product_type). The
// family is a lifecycle discriminator, not a pricing category: a Premium Level 1
// program sells under the `premium` pricing category but belongs to its own
// family because it participates in the questionnaire/nutrition lifecycle.
export const programFamilyLabel = (family: ProgramProductTypeValue): string => {
  if (family === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1) {
    return "Premium · Level 1";
  }
  return "Generic";
};

export const programFamilyShortLabel = (family: ProgramProductTypeValue): string => {
  if (family === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1) {
    return "Premium L1";
  }
  return "Generic";
};

// programFamilyPath maps a family to its administration route segment. The
// backend exposes one route group per family and derives the family exclusively
// from this segment, so every management call must address the family that way.
export const programFamilyPath = (family: ProgramProductTypeValue): string => {
  if (family === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1) {
    return "premium-level-1";
  }
  return "generic";
};

export const formatAdminPrice = (minorUnits: number, currency: string): string => {
  const value = (minorUnits ?? 0) / 100;
  return new Intl.NumberFormat("en-GB", {
    style: "currency",
    currency: currency || "EUR"
  }).format(value);
};

export const formatCommissionBPS = (bps: number): string => `${(bps ?? 0) / 100}%`;

export const fetchAdminPrograms = (page: number, limit: number, type?: ProgramTypeValue) => {
  const typeQuery = type ? `&type=${type}` : "";
  return apiGet<AdminProgramsList>(`/programs?page=${page}&limit=${limit}${typeQuery}`);
};

export const fetchAdminProgram = (id: string) => apiGet<AdminProgram>(`/admin/programs/${encodePathSegment(id)}`);

export const updateAdminProgramPricing = (id: string, input: AdminPricingInput) =>
  apiPatch<AdminProgram>(`/admin/programs/${encodePathSegment(id)}/pricing`, input);

export const updateCommissionRule = (trainerId: string, commissionBps: number) =>
  apiPatch<AdminCommissionRule>(`/admin/trainers/${encodePathSegment(trainerId)}/commission`, {
    commission_bps: commissionBps
  });

export const deleteCommissionRule = (trainerId: string) =>
  apiDelete<never>(`/admin/trainers/${encodePathSegment(trainerId)}/commission`);

export const fetchCommissionResolution = (trainerId: string) =>
  apiGet<AdminCommissionResolution>(`/admin/trainers/${encodePathSegment(trainerId)}/commission/resolve`);

/* ------------------------------------------------------------------ */
/* Generic programs (platform-owned catalogue plans)                   */
/* ------------------------------------------------------------------ */

export type GenericProgramLevel = "Beginner" | "Intermediate" | "Advanced";

// Controlled vocabulary stored verbatim in programs.training_type. Display
// casing is part of the contract and shared with the public catalog filter.
export type GenericTrainingType =
  | "Hypertrophy"
  | "Strength"
  | "HYROX"
  | "CrossFit"
  | "Fat Loss"
  | "At Home";

export const GENERIC_TRAINING_TYPES: GenericTrainingType[] = [
  "Hypertrophy",
  "Strength",
  "HYROX",
  "CrossFit",
  "Fat Loss",
  "At Home"
];

export interface GenericProgramSummary {
  id: string;
  name: string;
  description: string;
  type: ProgramTypeValue;
  product_type: ProgramProductTypeValue;
  status: ProgramStatusValue;
  level: GenericProgramLevel | null;
  duration_weeks: number | null;
  frequency_per_week: number | null;
  training_type: GenericTrainingType | null;
  price_minor_units: number;
  currency: string;
  created_at: string;
  updated_at: string;
}

export interface GenericProgramSet {
  set_number: number;
  set_type: SetType;
  reps: number | null;
  weight_kg: number | null;
  rir: number | null;
  rpe: number | null;
  rest_seconds: number | null;
  tempo: string;
}

export interface GenericProgramExercise {
  id: string;
  name: string;
  description: string;
  catalog_instructions: string;
  target_muscles: string;
  equipment: string;
  difficulty: string;
  video_url: string;
  image_url: string;
  position: number;
  instructions: string;
  notes: string;
  sets: GenericProgramSet[];
}

export interface GenericProgramWorkout {
  position: number;
  exercises: GenericProgramExercise[];
}

export interface GenericProgramWeek {
  week_number: number;
  workouts: GenericProgramWorkout[];
}

export interface GenericProgramDetail extends GenericProgramSummary {
  weeks: GenericProgramWeek[];
}

export interface GenericProgramSetInput {
  set_type: SetType;
  reps: number | null;
  weight_kg: number | null;
  rir: number | null;
  rpe: number | null;
  rest_seconds: number | null;
  tempo: string;
}

export interface GenericProgramExerciseInput {
  exercise_id: string;
  instructions: string;
  notes: string;
  sets: GenericProgramSetInput[];
}

export interface GenericProgramWorkoutInput {
  exercises: GenericProgramExerciseInput[];
}

export interface GenericProgramWeekInput {
  workouts: GenericProgramWorkoutInput[];
}

export interface GenericProgramInput {
  name: string;
  description: string;
  type: ProgramTypeValue;
  status: ProgramStatusValue;
  level: GenericProgramLevel | "";
  duration_weeks: number;
  frequency_per_week: number;
  training_type: GenericTrainingType | "";
  price_minor_units: number;
  currency: string;
  weeks: GenericProgramWeekInput[];
}

export interface GenericProgramsList {
  programs: GenericProgramSummary[];
  pagination: AdminPagination;
}

export interface GenericProgramListParams {
  page: number;
  limit: number;
  search?: string;
  type?: ProgramTypeValue;
  level?: GenericProgramLevel;
  training_type?: GenericTrainingType;
}

// The family is the first argument of every generic program call: the request
// is routed to the family's route group and the backend derives the family from
// that route. A client can never choose a family through a payload field.
export const fetchGenericPrograms = (
  family: ProgramProductTypeValue,
  params: GenericProgramListParams
): Promise<GenericProgramsList> => {
  const query = new URLSearchParams({
    page: String(params.page),
    limit: String(params.limit)
  });
  if (params.search) {
    query.set("search", params.search);
  }
  if (params.type) {
    query.set("type", params.type);
  }
  if (params.level) {
    query.set("level", params.level);
  }
  if (params.training_type) {
    query.set("training_type", params.training_type);
  }
  return apiGet<GenericProgramsList>(`/programs/${programFamilyPath(family)}?${query.toString()}`);
};

export const fetchGenericProgram = (family: ProgramProductTypeValue, id: string) =>
  apiGet<GenericProgramDetail>(`/programs/${programFamilyPath(family)}/${encodePathSegment(id)}`);

export const createGenericProgram = (family: ProgramProductTypeValue, input: GenericProgramInput) =>
  apiPost<GenericProgramDetail>(`/programs/${programFamilyPath(family)}`, input);

export const updateGenericProgram = (family: ProgramProductTypeValue, id: string, input: GenericProgramInput) =>
  apiPatch<GenericProgramDetail>(`/programs/${programFamilyPath(family)}/${encodePathSegment(id)}`, input);

export const publishGenericProgram = (family: ProgramProductTypeValue, id: string) =>
  apiPost<GenericProgramSummary>(`/programs/${programFamilyPath(family)}/${encodePathSegment(id)}/publish`, {});

export const deleteGenericProgram = (family: ProgramProductTypeValue, id: string) =>
  apiDelete<never>(`/programs/${programFamilyPath(family)}/${encodePathSegment(id)}`);

/**
 * Resolves the admin feedback list. Feedback is not part of the product yet,
 * so this always reports the data as unavailable. Keep `AdminFeedback`
 * aligned with the future GET /admin/feedback response when the feature
 * ships.
 */
export async function fetchAdminFeedback(): Promise<AdminFeedbackListResult> {
  return {
    available: false,
    feedback: [],
    reason: "Feedback is not part of the community scope yet."
  };
}

// --- Exercise catalogue administration ---

// The persisted catalogue is the single source the public Exercise Library,
// trainers and administrators all read from. Publication is explicit: new
// entries start as drafts, and only published entries are publicly visible.
export const EXERCISE_STATUS = {
  DRAFT: "draft",
  PUBLISHED: "published",
  ARCHIVED: "archived"
} as const;

export type ExerciseStatusValue = typeof EXERCISE_STATUS[keyof typeof EXERCISE_STATUS];

export const EXERCISE_STATUS_LABELS: Record<ExerciseStatusValue, string> = {
  draft: "Draft",
  published: "Published",
  archived: "Archived"
};

export const isExerciseStatus = (value: string): value is ExerciseStatusValue =>
  value === EXERCISE_STATUS.DRAFT ||
  value === EXERCISE_STATUS.PUBLISHED ||
  value === EXERCISE_STATUS.ARCHIVED;

export interface AdminExercise {
  id: string;
  name: string;
  source: string;
  source_id: string;
  body_part: string;
  description: string;
  instructions: string;
  instruction_steps: string[];
  target_muscles: string;
  primary_muscle_group: string;
  secondary_muscle_groups: string;
  equipment: string;
  difficulty: string;
  movement_category: string;
  video_url: string;
  image_url: string;
  animation_url: string;
  source_media_id: string;
  media_attribution: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface AdminExercisesList {
  exercises: AdminExercise[];
  pagination: AdminPagination;
}

export interface AdminExerciseFilters {
  query?: string;
  status?: ExerciseStatusValue | "";
  bodyPart?: string;
}

// Editable catalogue fields. Names are immutable after creation because
// workout references and public links may already point at the entry.
export interface AdminExerciseInput {
  description: string;
  instructions: string;
  instruction_steps: string[];
  target_muscles: string;
  primary_muscle_group: string;
  secondary_muscle_groups: string;
  equipment: string;
  difficulty: string;
  movement_category: string;
  body_part: string;
  image_url: string;
  animation_url: string;
  source_media_id: string;
  media_attribution: string;
}

export interface AdminExerciseCreateInput extends AdminExerciseInput {
  name: string;
}

export interface AdminExerciseUpdateInput extends AdminExerciseInput {
  status: ExerciseStatusValue;
}

export interface AdminExerciseMutationResult {
  exercise: AdminExercise;
}

export const fetchAdminExercises = (page: number, limit: number, filters: AdminExerciseFilters = {}) => {
  const query = new URLSearchParams();
  query.set("page", String(page));
  query.set("limit", String(limit));
  const search = filters.query?.trim() ?? "";
  if (search) {
    query.set("q", search);
  }
  if (filters.status) {
    query.set("status", filters.status);
  }
  if (filters.bodyPart) {
    query.set("body_part", filters.bodyPart);
  }
  return apiGet<AdminExercisesList>(`/admin/exercises?${query.toString()}`);
};

export const createAdminExercise = (input: AdminExerciseCreateInput) =>
  apiPost<AdminExerciseMutationResult>("/admin/exercises", input);

export const updateAdminExercise = (id: string, input: AdminExerciseUpdateInput) =>
  apiPatch<AdminExerciseMutationResult>(`/admin/exercises/${encodePathSegment(id)}`, input);