import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/services/purchases_api", () => ({
  createPurchaseIntent: vi.fn(),
  capturePayment: vi.fn(),
  fetchEntitlements: vi.fn(),
  fetchMyPurchases: vi.fn(),
  fetchPaymentMethods: vi.fn(),
  initiatePayment: vi.fn()
}));

vi.mock("@/services/test_mode_api", () => ({
  completeTestPurchase: vi.fn()
}));

import {
  capturePayment,
  createPurchaseIntent,
  fetchEntitlements,
  fetchMyPurchases,
  fetchPaymentMethods,
  initiatePayment
} from "@/services/purchases_api";
import { completeTestPurchase } from "@/services/test_mode_api";
import { ApiError } from "@utils/http_client";
import type { Entitlement, Purchase, PurchaseHistoryEntry } from "@/services/purchases_api";

import { useProgramCheckout } from "./use_program_checkout";

const PROGRAM_ID = "11111111-1111-1111-1111-111111111111";

const mockedCapturePayment = vi.mocked(capturePayment);
const mockedCreatePurchaseIntent = vi.mocked(createPurchaseIntent);
const mockedFetchEntitlements = vi.mocked(fetchEntitlements);
const mockedFetchMyPurchases = vi.mocked(fetchMyPurchases);
const mockedFetchPaymentMethods = vi.mocked(fetchPaymentMethods);
const mockedInitiatePayment = vi.mocked(initiatePayment);
const mockedCompleteTestPurchase = vi.mocked(completeTestPurchase);

const pendingPurchase = (id: string): Purchase => ({
  id,
  program_id: PROGRAM_ID,
  price_minor_units: 1000,
  currency: "EUR",
  status: "pending",
  payment_method: "card"
});

const pendingHistoryEntry = (id: string, method: string): PurchaseHistoryEntry => ({
  ...pendingPurchase(id),
  payment_method: method,
  test: false,
  access: false,
  created_at: "",
  program: {
    id: PROGRAM_ID,
    name: "Program",
    description: "",
    type: "premium",
    status: "published",
    price_minor_units: 1000,
    currency: "EUR",
    created_at: "",
    updated_at: ""
  }
});

const entitlementsFor = (programId: string): Entitlement[] => [
  {
    id: "entitlement-1",
    program_id: programId,
    program: {
      id: programId,
      name: "Program",
      description: "",
      type: "premium",
      status: "published",
      price_minor_units: 1000,
      currency: "EUR",
      product_type: "generic",
      created_at: "",
      updated_at: ""
    },
    created_at: "",
    updated_at: ""
  }
];

const renderCheckout = (options?: { testModeActive?: boolean; enabled?: boolean }) =>
  renderHook(() =>
    useProgramCheckout({
      programId: PROGRAM_ID,
      free: false,
      testModeActive: options?.testModeActive ?? false,
      enabled: options?.enabled ?? false
    })
  );

const originalLocation = window.location;

beforeEach(() => {
  window.history.replaceState({}, "", "/");
  mockedFetchPaymentMethods.mockResolvedValue([]);
});

afterEach(() => {
  cleanup();
  Object.defineProperty(window, "location", {
    configurable: true,
    value: originalLocation
  });
});

describe("useProgramCheckout", () => {
  it("resolves an unauthenticated buyer as guest", async () => {
    mockedFetchEntitlements.mockRejectedValue(new ApiError(401, "UNAUTHORIZED", "unauthorized"));

    const { result } = renderCheckout({ enabled: true });

    await waitFor(() => expect(result.current.state.status).toBe("guest"));
  });

  it("completes the capture return leg with the provider identifiers", async () => {
    window.history.replaceState({}, "", "/?purchase_id=purchase-1&session_id=cs_1");
    mockedCapturePayment.mockResolvedValue({ ...pendingPurchase("purchase-1"), status: "completed" });

    const { result } = renderCheckout();

    await waitFor(() => expect(result.current.state.status).toBe("success"));
    expect(mockedCapturePayment).toHaveBeenCalledWith("purchase-1", "cs_1");
  });

  it("recovers a raced capture by reflecting the settled access state", async () => {
    window.history.replaceState({}, "", "/?purchase_id=purchase-2&token=paypal-order-1");
    mockedCapturePayment.mockRejectedValue(new ApiError(409, "PURCHASE_NOT_PENDING", "conflict"));
    mockedFetchEntitlements.mockResolvedValue(entitlementsFor(PROGRAM_ID));

    const { result } = renderCheckout();

    await waitFor(() => expect(result.current.state.status).toBe("success"));
  });

  it("treats a duplicate-entitlement initiation as owned access", async () => {
    mockedCreatePurchaseIntent.mockResolvedValue(pendingPurchase("purchase-3"));
    mockedInitiatePayment.mockRejectedValue(new ApiError(409, "DUPLICATE_ENTITLEMENT", "pair already resolved"));
    mockedFetchEntitlements.mockResolvedValue(entitlementsFor(PROGRAM_ID));

    const { result } = renderCheckout();

    await act(async () => {
      await result.current.buy();
    });

    expect(result.current.state.status).toBe("owned");
    expect(mockedInitiatePayment).toHaveBeenCalledWith("purchase-3", "card");
  });

  it("returns to the ready state when the provider reports a cancellation", async () => {
    window.history.replaceState({}, "", "/?status=cancelled&purchase_id=purchase-4");

    const { result } = renderCheckout();

    await waitFor(() => expect(result.current.state.status).toBe("cancelled"));
    expect(result.current.state).toEqual({ status: "cancelled", purchaseId: "purchase-4" });
    expect(mockedCapturePayment).not.toHaveBeenCalled();
  });

  it("resumes a duplicate pending purchase with its recorded method, not the stale selection", async () => {
    const pending = pendingHistoryEntry("purchase-5", "card");
    mockedFetchPaymentMethods.mockResolvedValue([
      { method: "card", label: "Card" },
      { method: "paypal", label: "PayPal" }
    ]);
    mockedCreatePurchaseIntent.mockRejectedValue(new ApiError(409, "DUPLICATE_PURCHASE", "pending exists"));
    mockedFetchMyPurchases.mockResolvedValue([pending]);
    mockedInitiatePayment.mockResolvedValue({
      payment_id: "payment-5",
      checkout_url: "https://provider.test/checkout",
      status: "pending",
      purchase_id: "purchase-5"
    });
    const assign = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...window.location, assign }
    });

    const { result } = renderCheckout();
    await waitFor(() => expect(result.current.paymentMethods).toHaveLength(2));

    act(() => result.current.selectMethod("paypal"));
    await act(async () => {
      await result.current.buy();
    });

    expect(mockedInitiatePayment).toHaveBeenCalledWith("purchase-5", "card");
  });

  it("refreshes ownership after an external purchase event", async () => {
    mockedFetchEntitlements.mockResolvedValueOnce([]).mockResolvedValueOnce(entitlementsFor(PROGRAM_ID));

    const { result } = renderCheckout({ enabled: true });
    await waitFor(() => expect(result.current.state.status).toBe("ready"));

    await act(async () => {
      expect(await result.current.refreshOwnership()).toBe("owned");
    });
  });

  it("reflects owned access when a Test Mode purchase conflicts", async () => {
    mockedCompleteTestPurchase.mockRejectedValue(new ApiError(409, "DUPLICATE_ENTITLEMENT", "owned"));
    mockedFetchEntitlements.mockResolvedValue(entitlementsFor(PROGRAM_ID));

    const { result } = renderCheckout({ testModeActive: true });

    await act(async () => {
      await result.current.buy();
    });

    expect(result.current.state.status).toBe("owned");
  });
});
