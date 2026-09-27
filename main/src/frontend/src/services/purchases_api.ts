import { apiGet, apiPost } from "@utils/http_client";

/**
 * Authenticated purchase contract. This service talks to the private purchase
 * endpoints (`/me/programs/:id/purchase`, `/me/purchases/:id/payment` and
 * `/me/purchases/:id/capture`) plus the entitlement reader (`/me/entitlements`).
 *
 * Purchases snapshot the commercial values at creation time; the backend is
 * fully authoritative for amounts, currency and status. The browser callback
 * never proves a payment: after the buyer returns from PayPal, the frontend
 * calls capture with the order id and only the server-verified capture result
 * equates to a completed purchase.
 *
 * Endpoints return 401 when the session cookie is missing or expired.
 */

export type PurchaseStatus = "pending" | "completed" | "failed";

/**
 * Authorized payment method advertised by the backend. `method` is the
 * normalized identifier sent to the checkout API; `label` is the display name.
 * The backend decides which methods are available based on configured
 * providers — the frontend never hard-codes a payment method.
 */
export interface PaymentMethodInfo {
  method: string;
  label: string;
}

export interface Purchase {
  id: string;
  program_id: string;
  price_minor_units: number;
  currency: string;
  commission_bps: number;
  platform_amount: number;
  trainer_amount: number;
  status: PurchaseStatus;
  payment_method?: string | null;
}

/**
 * Safe program summary included in purchase history entries. It mirrors the
 * entitlement program summary and never carries internal trainer or payout
 * metadata.
 */
export interface PurchaseHistoryProgram {
  id: string;
  name: string;
  description: string;
  type: string;
  status: string;
  level?: string | null;
  training_type?: string | null;
  duration_weeks?: number | null;
  frequency_per_week?: number | null;
  price_minor_units: number;
  currency: string;
  created_at: string;
  updated_at: string;
}

/**
 * One entry of the authenticated user's purchase history, served by
 * `GET /me/purchases`. `access` is computed by the backend and only becomes
 * true for completed purchases of published, non-deleted programs. `test`
 * marks Test Mode purchases, which are zero-price and never touch a payment
 * provider. Commission and payout fields are never exposed by this endpoint.
 */
export interface PurchaseHistoryEntry {
  id: string;
  program_id: string;
  price_minor_units: number;
  currency: string;
  status: PurchaseStatus;
  payment_method?: string | null;
  test: boolean;
  access: boolean;
  created_at: string;
  program: PurchaseHistoryProgram;
}

export interface PaymentInitiation {
  payment_id: string;
  checkout_url?: string;
  status: string;
  purchase_id: string;
}

export interface EntitlementProgram {
  id: string;
  name: string;
  description: string;
  type: string;
  status: string;
  level?: string | null;
  training_type?: string | null;
  duration_weeks?: number | null;
  frequency_per_week?: number | null;
  price_minor_units: number;
  currency: string;
  created_at: string;
  updated_at: string;
}

export interface Entitlement {
  id: string;
  program_id: string;
  program: EntitlementProgram;
  created_at: string;
  updated_at: string;
}

/**
 * Client-safe structure of one entitled program, served by
 * `GET /me/programs/:programId`. Access is granted on the backend only when
 * the authenticated user holds an active entitlement and the program remains
 * published. A missing entitlement and an unavailable program are
 * indistinguishable and both appear as 404.
 */
export interface ProgramAccessDetail {
  id: string;
  trainer_id?: string | null;
  name: string;
  description: string;
  type: string;
  status: string;
  level?: string | null;
  duration_weeks?: number | null;
  frequency_per_week?: number | null;
  training_type?: string | null;
  price_minor_units: number;
  currency: string;
  created_at: string;
  updated_at: string;
  weeks: ProgramAccessWeek[];
}

export interface ProgramAccessWeek {
  week_number: number;
  workouts: ProgramAccessWorkout[];
}

export interface ProgramAccessWorkout {
  position: number;
  exercises: ProgramAccessExercise[];
}

export interface ProgramAccessExercise {
  name: string;
  description: string;
  instructions: string;
  target_muscles: string;
  equipment: string;
  difficulty: string;
  video_url: string;
  image_url: string;
  position: number;
  notes: string;
  sets: ProgramAccessSet[];
}

export interface ProgramAccessSet {
  set_number: number;
  set_type: string;
  reps?: number | null;
  weight_kg?: number | null;
  rir?: number | null;
  rpe?: number | null;
  rest_seconds?: number | null;
  tempo: string;
}

export const createPurchaseIntent = (programId: string): Promise<Purchase> =>
  apiPost<Purchase>(`/me/programs/${programId}/purchase`, {});

export const initiatePayment = (
  purchaseId: string,
  paymentMethod: string
): Promise<PaymentInitiation> =>
  apiPost<PaymentInitiation>(`/me/purchases/${purchaseId}/payment`, {
    payment_method: paymentMethod
  });

/**
 * Verifies an approved provider payment for a pending purchase and completes
 * it. `providerPaymentId` is the provider's own identifier for the payment — a
 * PayPal Order ID or a Stripe Checkout Session ID — taken from the provider
 * return URL. It is only a correlation hint: the backend re-verifies the
 * payment on the provider API before completing anything.
 */
export const capturePayment = (purchaseId: string, providerPaymentId: string): Promise<Purchase> =>
  apiPost<Purchase>(`/me/purchases/${purchaseId}/capture`, { provider_payment_id: providerPaymentId });

/**
 * Public, unauthenticated contract exposing the currently configured payment
 * methods (derived server-side from the providers present in the backend
 * configuration). Returns them in the backend's stable order.
 */
export const fetchPaymentMethods = (): Promise<PaymentMethodInfo[]> =>
  apiGet<PaymentMethodInfo[]>("/payments/methods");

export const fetchMyPurchases = (): Promise<PurchaseHistoryEntry[]> =>
  apiGet<PurchaseHistoryEntry[]>("/me/purchases");

export const fetchEntitlements = (): Promise<Entitlement[]> =>
  apiGet<Entitlement[]>("/me/entitlements");

export const fetchProgramAccess = (programId: string): Promise<ProgramAccessDetail> =>
  apiGet<ProgramAccessDetail>(`/me/programs/${programId}`);