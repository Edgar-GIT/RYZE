import { apiDelete, apiGet, apiPatch } from "@utils/http_client";
import { AdminPagination } from "./admin_api";

export const ProgramType = {
  FREE: "free",
  PREMIUM: "premium",
  PERSONALIZED: "personalized"
} as const;

export type ProgramTypeValue = typeof ProgramType[keyof typeof ProgramType];

export const ProgramStatusEnum = {
  DRAFT: "draft",
  PUBLISHED: "published"
} as const;

export type ProgramStatusValue = typeof ProgramStatusEnum[keyof typeof ProgramStatusEnum];

export const PurchaseStatusEnum = {
  PENDING: "pending",
  COMPLETED: "completed",
  FAILED: "failed",
  REFUNDED: "refunded"
} as const;

export type PurchaseStatus = typeof PurchaseStatusEnum[keyof typeof PurchaseStatusEnum];

export interface AdminProgram {
  id: string;
  trainer_id: string | null;
  name: string;
  description: string;
  type: ProgramTypeValue;
  status: ProgramStatusValue;
  price_minor_units: number;
  currency: string;
  created_at: string;
  updated_at: string;
}

export interface AdminProgramsList {
  programs: AdminProgram[];
  pagination: AdminPagination;
}

export interface Exercise {
  id: string;
  name: string;
  description: string;
  target_muscles: string;
  equipment: string;
  difficulty: string;
  video_url: string | null;
  image_url: string | null;
}

export interface ExercisesList {
  exercises: Exercise[];
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

export interface AdminPurchase {
  id: string;
  user_id: string;
  program_id: string;
  program_name: string;
  price_minor_units: number;
  currency: string;
  status: PurchaseStatus;
  trainer_id: string | null;
  trainer_name: string;
  commission_bps: number;
  platform_amount_minor_units: number;
  trainer_amount_minor_units: number;
  created_at: string;
}

export interface AdminPurchaseListResult {
  available: boolean;
  purchases: AdminPurchase[];
  reason: string;
}

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

export const programTypeLabel = (type: ProgramTypeValue): string => {
  if (type === ProgramType.PREMIUM) {
    return "Premium · Level 1";
  }
  if (type === ProgramType.PERSONALIZED) {
    return "Premium · Level 2";
  }
  return "Generic Plan";
};

export const programTypeShortLabel = (type: ProgramTypeValue): string => {
  if (type === ProgramType.PREMIUM) {
    return "Premium L1";
  }
  if (type === ProgramType.PERSONALIZED) {
    return "Premium L2";
  }
  return "Generic";
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

export const fetchAdminProgram = (id: string) => apiGet<AdminProgram>(`/admin/programs/${id}`);

export const updateAdminProgramPricing = (id: string, input: AdminPricingInput) =>
  apiPatch<AdminProgram>(`/admin/programs/${id}/pricing`, input);

export const fetchExercises = (page: number, limit: number) =>
  apiGet<ExercisesList>(`/exercises?page=${page}&limit=${limit}`);

export const searchExercises = (query: string, page: number, limit: number) => {
  const q = encodeURIComponent(query.trim());
  const queryString = q ? `&q=${q}` : "";
  return apiGet<ExercisesList>(`/exercises/search?page=${page}&limit=${limit}${queryString}`);
};

export const updateCommissionRule = (trainerId: string, commissionBps: number) =>
  apiPatch<AdminCommissionRule>(`/admin/trainers/${trainerId}/commission`, {
    commission_bps: commissionBps
  });

export const deleteCommissionRule = (trainerId: string) =>
  apiDelete<never>(`/admin/trainers/${trainerId}/commission`);

export const fetchCommissionResolution = (trainerId: string) =>
  apiGet<AdminCommissionResolution>(`/admin/trainers/${trainerId}/commission/resolve`);

/**
 * Resolves the admin purchase list. There is no admin sales endpoint in the
 * backend yet, so this always reports the data as unavailable instead of
 * fabricating records. Keep `AdminPurchase` aligned with the future
 * GET /admin/purchases response; pages consuming this contract must render
 * the honest unavailable state they receive.
 */
export async function fetchAdminPurchases(): Promise<AdminPurchaseListResult> {
  return {
    available: false,
    purchases: [],
    reason: "The admin sales API is not implemented yet."
  };
}

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