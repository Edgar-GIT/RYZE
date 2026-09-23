import { apiGet } from "@utils/http_client";

/**
 * Read-only public marketplace contract. This service talks to the public
 * catalog endpoints (`/programs` + `/programs/:id`) with `scope=generic`, so it
 * only ever receives platform-owned published programs. Prices are part of the
 * API contract and are rendered by the marketplace to communicate the current
 * sale price before purchase checks.
 */

export const FREE_PROGRAM_TYPE = "free";

/**
 * Formats a price in minor units into a client-facing label. Free programs are
 * always rendered as "Free"; paid programs use the EUR localised currency
 * format. Money is handled exclusively in integer minor units.
 */
export const formatMarketplacePrice = (
  minorUnits: number,
  currency: string,
  type: string
): string => {
  if (type === FREE_PROGRAM_TYPE || !minorUnits) {
    return "Free";
  }
  return new Intl.NumberFormat("en-GB", {
    style: "currency",
    currency: currency || "EUR"
  }).format(minorUnits / 100);
};

export interface MarketplaceProgram {
  id: string;
  trainer_id: string | null;
  name: string;
  description: string;
  type: string;
  status: string;
  level: string | null;
  duration_weeks: number | null;
  frequency_per_week: number | null;
  training_type: string | null;
  price_minor_units: number;
  currency: string;
  created_at: string;
  updated_at: string;
}

export interface MarketplaceSet {
  set_number: number;
  set_type: string;
  reps: number | null;
  weight_kg: number | null;
  rir: number | null;
  rpe: number | null;
  rest_seconds: number | null;
  tempo: string;
}

export interface MarketplaceExercise {
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
  sets: MarketplaceSet[];
}

export interface MarketplaceWorkout {
  position: number;
  exercises: MarketplaceExercise[];
}

export interface MarketplaceWeek {
  week_number: number;
  workouts: MarketplaceWorkout[];
}

export interface MarketplaceProgramDetail extends MarketplaceProgram {
  weeks: MarketplaceWeek[];
}

export interface MarketplaceListResult {
  programs: MarketplaceProgram[];
  total: number;
  page: number;
  limit: number;
}

export interface MarketplaceProgramParams {
  q?: string;
  type?: string;
  training_type?: string;
  level?: string;
  frequency?: number;
  duration_min?: number;
  duration_max?: number;
  sort?: string;
  order?: string;
  page: number;
  limit: number;
}

export const fetchMarketplacePrograms = async (
  params: MarketplaceProgramParams
): Promise<MarketplaceListResult> => {
  const query = new URLSearchParams({
    scope: "generic",
    page: String(params.page),
    limit: String(params.limit)
  });
  for (const [key, value] of [
    ["q", params.q],
    ["type", params.type],
    ["training_type", params.training_type],
    ["level", params.level],
    ["sort", params.sort],
    ["order", params.order]
  ] as const) {
    if (value) {
      query.set(key, value);
    }
  }
  for (const [key, value] of [
    ["frequency", params.frequency],
    ["duration_min", params.duration_min],
    ["duration_max", params.duration_max]
  ] as const) {
    if (value && value > 0) {
      query.set(key, String(value));
    }
  }

  const data = await apiGet<{
    programs: MarketplaceProgram[];
    pagination: { page: number; limit: number; total: number };
  }>(`/programs?${query.toString()}`);

  return {
    programs: data.programs,
    total: data.pagination.total,
    page: data.pagination.page,
    limit: data.pagination.limit
  };
};

// The marketplace is an infinite-scroll page with no numbered pagination, so
// the full generic catalog is collected through bounded page fetches.
export const fetchAllMarketplacePrograms = async (
  pageSize = 100
): Promise<{ programs: MarketplaceProgram[]; total: number }> => {
  const collected: MarketplaceProgram[] = [];
  let page = 1;
  let total = 0;

  for (let guard = 0; guard < 100; guard += 1) {
    const result = await fetchMarketplacePrograms({ page, limit: pageSize });
    collected.push(...result.programs);
    total = result.total;
    if (collected.length >= total || result.programs.length === 0) {
      break;
    }
    page += 1;
  }

  return { programs: collected, total };
};

export const fetchMarketplaceProgram = (id: string): Promise<MarketplaceProgramDetail> =>
  apiGet<MarketplaceProgramDetail>(`/programs/${id}`);