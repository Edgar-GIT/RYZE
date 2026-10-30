import { apiGet, encodePathSegment } from "@utils/http_client";

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

/** product_type is the product family: "generic" or "premium_level_1". It is
 *  public product metadata and is never client data. */
export interface MarketplaceProgramProduct {
  product_type: string;
}

export interface MarketplaceProgram extends MarketplaceProgramProduct {
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

export type MarketplaceSort = "created_at" | "updated_at" | "name";

/** MarketplaceFilters is the server-side filter contract. Values inside one
 *  dimension are ORed; dimensions are ANDed by the backend. */
export interface MarketplaceFilters {
  trainingTypes?: string[];
  levels?: string[];
  frequencies?: number[];
  durationRanges?: Array<[number, number]>;
}

export interface MarketplaceProgramParams extends MarketplaceFilters {
  q?: string;
  type?: string;
  sort?: MarketplaceSort;
  order?: "asc" | "desc";
  page: number;
  limit: number;
}

const commaList = (values: readonly (string | number)[] | undefined): string =>
  (values ?? []).filter((value) => value !== "").join(",");

export const fetchMarketplacePrograms = async (
  params: MarketplaceProgramParams
): Promise<MarketplaceListResult> => {
  const query = new URLSearchParams({
    scope: "generic",
    page: String(params.page),
    limit: String(params.limit)
  });
  if (params.q) {
    query.set("q", params.q);
  }
  if (params.type) {
    query.set("type", params.type);
  }
  if (params.trainingTypes?.length) {
    query.set("training_type", commaList(params.trainingTypes));
  }
  if (params.levels?.length) {
    query.set("level", commaList(params.levels));
  }
  if (params.frequencies?.length) {
    query.set("frequency", commaList(params.frequencies));
  }
  if (params.durationRanges?.length) {
    query.set(
      "duration_ranges",
      params.durationRanges.map(([min, max]) => `${min}-${max}`).join(",")
    );
  }
  if (params.sort) {
    query.set("sort", params.sort);
  }
  if (params.order) {
    query.set("order", params.order);
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

// The service pages need every published programme to compute the category
// price range, so they collect the full filtered catalog through bounded,
// server-paginated requests.
export const fetchAllMarketplacePrograms = async (
  filters: MarketplaceFilters = {},
  pageSize = 100
): Promise<{ programs: MarketplaceProgram[]; total: number }> => {
  const collected: MarketplaceProgram[] = [];
  let page = 1;
  let total = 0;

  for (let guard = 0; guard < 100; guard += 1) {
    const result = await fetchMarketplacePrograms({ ...filters, page, limit: pageSize });
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
  apiGet<MarketplaceProgramDetail>(`/programs/${encodePathSegment(id)}`);
// PREMIUM_SCOPE is the catalog scope for the Premium Level 1 product family.
const PREMIUM_SCOPE = "premium_level_1";

// fetchPremiumLevel1Programs returns the published Premium Level 1 packages.
// The Premium entry point reads real product data from the backend rather than
// hardcoding product metadata.
export const fetchPremiumLevel1Programs = async (
  limit = 20
): Promise<{ programs: MarketplaceProgram[]; total: number }> => {
  const query = new URLSearchParams({
    scope: PREMIUM_SCOPE,
    page: "1",
    limit: String(limit)
  });

  const data = await apiGet<{
    programs: MarketplaceProgram[];
    pagination: { page: number; limit: number; total: number };
  }>(`/programs?${query.toString()}`);

  return { programs: data.programs, total: data.pagination.total };
};

// fetchAllPremiumLevel1Programs walks every page so the catalogue never hides
// packages behind an arbitrary first-page limit.
export const fetchAllPremiumLevel1Programs = async (
  pageSize = 100
): Promise<{ programs: MarketplaceProgram[]; total: number }> => {
  const collected: MarketplaceProgram[] = [];
  let page = 1;
  let total = 0;

  for (let guard = 0; guard < 100; guard += 1) {
    const query = new URLSearchParams({
      scope: PREMIUM_SCOPE,
      page: String(page),
      limit: String(pageSize)
    });
    const data = await apiGet<{
      programs: MarketplaceProgram[];
      pagination: { page: number; limit: number; total: number };
    }>(`/programs?${query.toString()}`);
    collected.push(...data.programs);
    total = data.pagination.total;
    if (collected.length >= total || data.programs.length === 0) {
      break;
    }
    page += 1;
  }

  return { programs: collected, total };
};
