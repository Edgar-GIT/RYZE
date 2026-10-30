import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@utils/http_client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@utils/http_client")>();
  return {
    ...actual,
    apiGet: vi.fn()
  };
});

import { apiGet } from "@utils/http_client";

import {
  fetchAllMarketplacePrograms,
  fetchAllPremiumLevel1Programs,
  fetchMarketplacePrograms,
  type MarketplaceProgram
} from "./marketplace_api";

const mockedGet = vi.mocked(apiGet);

const program = (id: string): MarketplaceProgram => ({
  id,
  trainer_id: null,
  name: `Program ${id}`,
  description: "",
  type: "free",
  status: "published",
  level: "Beginner",
  duration_weeks: 4,
  frequency_per_week: 3,
  training_type: "Strength",
  price_minor_units: 0,
  currency: "EUR",
  product_type: "generic",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z"
});

beforeEach(() => {
  mockedGet.mockReset();
});

const queryOf = (path: string): URLSearchParams => new URL(path, "http://ryze.test").searchParams;

describe("fetchMarketplacePrograms", () => {
  it("sends every filter as a server-side query parameter", async () => {
    mockedGet.mockResolvedValue({
      programs: [],
      pagination: { page: 2, limit: 12, total: 0 }
    });

    await fetchMarketplacePrograms({
      page: 2,
      limit: 12,
      trainingTypes: ["Strength", "Hypertrophy"],
      levels: ["Beginner", "Advanced"],
      frequencies: [3, 4],
      durationRanges: [[1, 2], [5, 8]],
      sort: "updated_at",
      order: "desc"
    });

    const params = queryOf(mockedGet.mock.calls[0][0]);
    expect(params.get("scope")).toBe("generic");
    expect(params.get("page")).toBe("2");
    expect(params.get("limit")).toBe("12");
    expect(params.get("training_type")).toBe("Strength,Hypertrophy");
    expect(params.get("level")).toBe("Beginner,Advanced");
    expect(params.get("frequency")).toBe("3,4");
    expect(params.get("duration_ranges")).toBe("1-2,5-8");
    expect(params.get("sort")).toBe("updated_at");
    expect(params.get("order")).toBe("desc");
  });
});

describe("fetchAllMarketplacePrograms", () => {
  it("walks pages while preserving the active filters", async () => {
    mockedGet
      .mockResolvedValueOnce({
        programs: [program("1"), program("2")],
        pagination: { page: 1, limit: 2, total: 3 }
      })
      .mockResolvedValueOnce({
        programs: [program("3")],
        pagination: { page: 2, limit: 2, total: 3 }
      });

    const result = await fetchAllMarketplacePrograms({ levels: ["Beginner"] }, 2);

    expect(result.programs.map((entry) => entry.id)).toEqual(["1", "2", "3"]);
    expect(mockedGet).toHaveBeenCalledTimes(2);
    for (const [path] of mockedGet.mock.calls) {
      expect(queryOf(path).get("level")).toBe("Beginner");
    }
  });
});

describe("fetchAllPremiumLevel1Programs", () => {
  it("walks every Premium Level 1 page", async () => {
    mockedGet
      .mockResolvedValueOnce({
        programs: [program("1")],
        pagination: { page: 1, limit: 1, total: 2 }
      })
      .mockResolvedValueOnce({
        programs: [program("2")],
        pagination: { page: 2, limit: 1, total: 2 }
      });

    const result = await fetchAllPremiumLevel1Programs(1);

    expect(result.programs.map((entry) => entry.id)).toEqual(["1", "2"]);
    expect(mockedGet).toHaveBeenCalledTimes(2);
    expect(queryOf(mockedGet.mock.calls[1][0]).get("page")).toBe("2");
    expect(queryOf(mockedGet.mock.calls[1][0]).get("scope")).toBe("premium_level_1");
  });
});
