import { apiGet, encodePathSegment } from "@utils/http_client";

/**
 * The purchase document RYZE issues for every completed purchase (real or
 * Test Mode), served by `GET /me/invoices` and
 * `GET /me/purchases/:purchaseId/invoice`. Until certified invoicing software
 * is connected it is a purchase receipt with no tax value
 * (`document_type: "purchase_receipt"`, `fiscal_value: false`). Amounts are
 * minor units. The owner's account id is never part of the response.
 */
export interface PurchaseInvoice {
  id: string;
  number: string;
  series: "RYZE" | "TEST";
  year: number;
  sequence: number;
  purchase_id: string;
  program_id: string;
  test: boolean;
  status: "issued";
  issued_at: string;
  purchased_at: string;
  seller_name: string;
  seller_tax_id: string | null;
  seller_address: string | null;
  buyer_name: string;
  buyer_tax_id: string | null;
  description: string;
  product_type: string;
  quantity: number;
  unit_price_minor_units: number;
  discount_minor_units: number;
  tax_rate_bps: number;
  tax_minor_units: number;
  /** "M10": VAT exempt under article 53 of the Portuguese VAT Code. */
  tax_exemption_code: string | null;
  total_minor_units: number;
  currency: string;
  payment_method: string | null;
  document_type: "purchase_receipt";
  fiscal_value: boolean;
}

export const fetchMyInvoices = (): Promise<PurchaseInvoice[]> => apiGet<PurchaseInvoice[]>("/me/invoices");

/** The document of one of the signed-in user's purchases (404 for anyone else's). */
export const fetchPurchaseInvoice = (purchaseId: string): Promise<PurchaseInvoice> =>
  apiGet<PurchaseInvoice>(`/me/purchases/${encodePathSegment(purchaseId)}/invoice`);

/** The receipt page of a purchase (also linked from the receipt email). */
export const receiptPath = (purchaseId: string): string => `/account/purchases/${encodeURIComponent(purchaseId)}/receipt`;
