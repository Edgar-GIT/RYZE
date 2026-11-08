import { beforeEach, describe, expect, it, vi } from "vitest";

import { apiGet, apiPost, apiPut } from "@utils/http_client";
import {
  assignPremiumLevel2Trainer,
  fetchAdminPremiumLevel2Requests,
  fetchPremiumLevel2Programs,
  submitPremiumLevel2Intake
} from "./premium_level_2_api";

vi.mock("@utils/http_client", () => ({
  apiGet: vi.fn(),
  apiPost: vi.fn(),
  apiPut: vi.fn(),
  encodePathSegment: (value: string) => encodeURIComponent(value)
}));

const mockedGet = vi.mocked(apiGet);
const mockedPost = vi.mocked(apiPost);
const mockedPut = vi.mocked(apiPut);

describe("premium level 2 api", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reads only the premium level 2 public scope", async () => {
    mockedGet.mockResolvedValue({ programs: [] });
    await fetchPremiumLevel2Programs();
    const query = new URL(mockedGet.mock.calls[0][0], "http://localhost").searchParams;
    expect(query.get("scope")).toBe("premium_level_2");
  });

  it("submits the intake through the owner-scoped PUT route", async () => {
    mockedPut.mockResolvedValue({ request: {}, intake: {} });
    await submitPremiumLevel2Intake("request/1", { primary_goal: "fat_loss" });
    expect(mockedPut).toHaveBeenCalledWith("/me/premium-level-2/request%2F1/intake", {
      answers: { primary_goal: "fat_loss" }
    });
  });

  it("sends the selected trainer through the admin request route", async () => {
    mockedPost.mockResolvedValue({});
    await assignPremiumLevel2Trainer("request-1", "trainer-1");
    expect(mockedPost).toHaveBeenCalledWith(
      "/admin/premium-level-2/requests/request-1/trainer",
      { trainer_id: "trainer-1" }
    );
  });

  it("keeps staff queue filters and pagination in the query", async () => {
    mockedGet.mockResolvedValue({ requests: [], pagination: { page: 2, limit: 25, total: 0, total_pages: 0 } });
    await fetchAdminPremiumLevel2Requests(2, 25, "awaiting_intake");
    const query = new URL(mockedGet.mock.calls[0][0], "http://localhost").searchParams;
    expect(query.get("page")).toBe("2");
    expect(query.get("limit")).toBe("25");
    expect(query.get("status")).toBe("awaiting_intake");
  });
});
