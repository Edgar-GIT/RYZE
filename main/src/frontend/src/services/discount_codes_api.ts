import { apiGet, apiPost, encodePathSegment } from "@utils/http_client";

// Discount codes (owner request, 2026-11-24): both administrators create
// them; buyers enter one at checkout. The server validates and applies it.

export type DiscountScope = "all" | "premium_level_1" | "premium_level_2" | "trainer_plans";

export interface DiscountCode {
  id: string;
  code: string;
  percent_off: number;
  applies_to: DiscountScope;
  starts_at: string;
  expires_at: string;
  max_uses: number | null;
  uses: number;
  disabled_at: string | null;
  created_at: string;
}

export interface DiscountCodeInput {
  code: string;
  percent_off: number;
  applies_to: DiscountScope;
  // 1 to 720 hours (30 days).
  duration_hours: number;
  max_uses: number | null;
}

export const fetchDiscountCodes = async (): Promise<DiscountCode[]> =>
  (await apiGet<{ codes: DiscountCode[] }>("/admin/discount-codes")).codes;

export const createDiscountCode = (input: DiscountCodeInput): Promise<DiscountCode> =>
  apiPost<DiscountCode>("/admin/discount-codes", input);

export const disableDiscountCode = (id: string): Promise<null> =>
  apiPost<null>(`/admin/discount-codes/${encodePathSegment(id)}/disable`, {});

export interface DiscountPreview {
  code: string;
  percent_off: number;
  discount_minor_units: number;
  total_minor_units: number;
  original_minor_units: number;
  currency: string;
  expires_at: string;
}

export const previewDiscountCode = (code: string, programId: string): Promise<DiscountPreview> =>
  apiGet<DiscountPreview>(
    `/me/discount-codes/preview?code=${encodeURIComponent(code.trim())}&program_id=${encodeURIComponent(programId)}`
  );

// Why a code was refused (the server's DISCOUNT_INVALID detail).
export const DISCOUNT_REASON: Record<"en" | "pt", Record<string, string>> = {
  en: {
    not_found: "This code does not exist.",
    expired: "This code has expired.",
    not_applicable: "This code does not apply to this plan.",
    used_up: "This code has no uses left.",
    already_used: "You already used this code.",
    locked: "The payment already started; the price can no longer change.",
    generic: "We could not check the code. Try again."
  },
  pt: {
    not_found: "Este código não existe.",
    expired: "Este código já expirou.",
    not_applicable: "Este código não se aplica a este plano.",
    used_up: "Este código já esgotou.",
    already_used: "Já usaste este código.",
    locked: "O pagamento já começou; o preço já não pode mudar.",
    generic: "Não foi possível verificar o código. Tenta de novo."
  }
};
