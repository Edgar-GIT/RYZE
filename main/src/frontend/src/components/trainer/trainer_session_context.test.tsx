import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/services/trainer_api", () => ({
  fetchTrainerAccess: vi.fn()
}));

import { fetchTrainerAccess } from "@/services/trainer_api";
import { ApiError } from "@utils/http_client";

import { TrainerSessionProvider, useTrainerSession } from "./trainer_session_context";

const mockedFetchAccess = vi.mocked(fetchTrainerAccess);

// Probe renders the resolved session state so assertions can observe what the
// provider actually exposes to every trainer route.
const Probe = () => {
  const { access, unauthenticated, loaded } = useTrainerSession();
  return (
    <div>
      <span data-testid="loaded">{String(loaded)}</span>
      <span data-testid="unauthenticated">{String(unauthenticated)}</span>
      <span data-testid="is-trainer">{access ? String(access.is_trainer) : "none"}</span>
      <span data-testid="application">{access?.application?.status ?? "none"}</span>
    </div>
  );
};

const renderProvider = () =>
  render(
    <TrainerSessionProvider>
      <Probe />
    </TrainerSessionProvider>
  );

beforeEach(() => {
  mockedFetchAccess.mockReset();
});

// Auto-cleanup is only registered when Vitest globals are enabled, so the
// mounted tree is removed explicitly between tests.
afterEach(() => {
  cleanup();
});

describe("TrainerSessionProvider", () => {
  it("exposes the trainer access payload once the backend resolves", async () => {
    mockedFetchAccess.mockResolvedValue({
      is_trainer: true,
      application: { id: "app-1", status: "APPROVED", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" }
    });

    renderProvider();

    expect(screen.getByTestId("loaded").textContent).toBe("false");

    await waitFor(() => expect(screen.getByTestId("loaded").textContent).toBe("true"));
    expect(screen.getByTestId("unauthenticated").textContent).toBe("false");
    expect(screen.getByTestId("is-trainer").textContent).toBe("true");
    expect(screen.getByTestId("application").textContent).toBe("APPROVED");
    expect(mockedFetchAccess).toHaveBeenCalledTimes(1);
  });

  it("marks the session unauthenticated on a 401", async () => {
    mockedFetchAccess.mockRejectedValue(new ApiError(401, "AUTHENTICATION_REQUIRED", "Authentication required."));

    renderProvider();

    await waitFor(() => expect(screen.getByTestId("loaded").textContent).toBe("true"));
    expect(screen.getByTestId("unauthenticated").textContent).toBe("true");
    expect(screen.getByTestId("is-trainer").textContent).toBe("none");
  });

  it("does not treat a non-401 failure as unauthenticated", async () => {
    mockedFetchAccess.mockRejectedValue(new ApiError(500, "INTERNAL_ERROR", "Internal server error."));

    renderProvider();

    await waitFor(() => expect(screen.getByTestId("loaded").textContent).toBe("true"));
    expect(screen.getByTestId("unauthenticated").textContent).toBe("false");
    expect(screen.getByTestId("is-trainer").textContent).toBe("none");
  });
});
