export interface ProgramSearchResult {
  id: string;
  title: string;
  downloads: number;
  rating: number;
}

/**
 * Resolves a catalog search query. Program records do not exist in the database
 * yet (the marketplace currently renders placeholder cards), so every query
 * resolves to an empty list. Once the backend search endpoint is available this
 * function will call it through `@utils/http_client` and return the matching
 * programs; keep the `ProgramSearchResult` contract aligned with that response.
 */
export async function searchPrograms(_query: string): Promise<ProgramSearchResult[]> {
  return [];
}