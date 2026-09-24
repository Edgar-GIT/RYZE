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

export interface Purchase {
  id: string;
  program_id: string;
  price_minor_units: number;
  currency: string;
  commission_bps: number;
  platform_amount: number;
  trainer_amount: number;
  status: PurchaseStatus;
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

export const createPurchaseIntent = (programId: string): Promise<Purchase> =>
  apiPost<Purchase>(`/me/programs/${programId}/purchase`, {});

export const initiatePayment = (
  purchaseId: string,
  paymentMethod: string
): Promise<PaymentInitiation> =>
  apiPost<PaymentInitiation>(`/me/purchases/${purchaseId}/payment`, {
    payment_method: paymentMethod
  });

export const capturePayment = (purchaseId: string, orderId: string): Promise<Purchase> =>
  apiPost<Purchase>(`/me/purchases/${purchaseId}/capture`, { order_id: orderId });

export const fetchMyPurchases = (): Promise<Purchase[]> =>
  apiGet<Purchase[]>("/me/purchases");

export const fetchEntitlements = (): Promise<Entitlement[]> =>
  apiGet<Entitlement[]>("/me/entitlements");