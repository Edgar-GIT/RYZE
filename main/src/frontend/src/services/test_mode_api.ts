import { apiGet, apiPost } from "@utils/http_client";

import type { Purchase } from "@/services/purchases_api";

/**
 * Admin Test Mode contract. Test Mode is established entirely server-side:
 * entering mints a real persona access token and a `ryze_test_session` cookie,
 * so the effective identity can never be forged through client state. While
 * active, every purchasable Generic Program renders as Free and purchases are
 * completed instantly by the backend without any payment provider.
 */

export interface TestModeStatus {
  active: boolean;
  persona?: string;
}

export interface TestModeEnterResult {
  persona: string;
  return_path: string;
}

export interface TestModeExitResult {
  return_path: string;
}

export const TestModePersonas = {
  CLIENT: "client",
  TRAINER: "trainer"
} as const;

export type TestModePersona = (typeof TestModePersonas)[keyof typeof TestModePersonas];

// Status is a public read-only endpoint: it never runs authentication so the
// banner can always reflect the active session, and it never reveals the token.
export const fetchTestModeStatus = (): Promise<TestModeStatus> =>
  apiGet<TestModeStatus>("/auth/test-mode");

// Entering is restricted server-side to the Technical Administrator
// (TECHNICAL_ADMINISTRATOR role). The persona is always one of the predefined
// identities, never an arbitrary user.
export const enterTestMode = (
  persona: TestModePersona,
  returnPath: string
): Promise<TestModeEnterResult> =>
  apiPost<TestModeEnterResult>("/admin/auth/test-mode", {
    persona,
    return_path: returnPath
  });

// Exiting intentionally requires no admin authentication: the admin session is
// cleared on enter, so the `ryze_test_session` cookie is the only trustworthy
// marker. The backend restores the original admin identity.
export const exitTestMode = (): Promise<TestModeExitResult> =>
  apiPost<TestModeExitResult>("/admin/auth/test-mode/exit", {});

// Test purchases complete a zero-price purchase and a real entitlement for the
// persona. They are capped to an active Test Mode session bound to the
// authenticated persona user and never touch a payment provider.
export const completeTestPurchase = (programId: string): Promise<Purchase> =>
  apiPost<Purchase>(`/auth/test-mode/programs/${programId}/purchase`, {});