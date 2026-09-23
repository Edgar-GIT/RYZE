import { fetchMarketplacePrograms } from "./marketplace_api";

export interface ProgramSearchResult {
  id: string;
  title: string;
  trainingType: string | null;
  level: string | null;
  durationWeeks: number | null;
  frequencyPerWeek: number | null;
}

/**
 * Resolves a live marketplace search query against the public generic catalog
 * (`GET /programs?scope=generic&q=...`). Only published platform-owned programs
 * are returned; the result contract carries the real product metadata that the
 * dropdown renders instead of fabricated popularity numbers.
 */
export async function searchPrograms(query: string): Promise<ProgramSearchResult[]> {
  const result = await fetchMarketplacePrograms({ q: query, page: 1, limit: 8 });
  return result.programs.map((program) => ({
    id: program.id,
    title: program.name,
    trainingType: program.training_type,
    level: program.level,
    durationWeeks: program.duration_weeks,
    frequencyPerWeek: program.frequency_per_week
  }));
}