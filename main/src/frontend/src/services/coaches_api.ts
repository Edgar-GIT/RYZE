import { apiGet, apiPost, encodePathSegment } from "@utils/http_client";

import type { MarketplaceProgram } from "@/services/marketplace_api";

/**
 * Public coach showcase. Trainer-owned programs are ranked by views ("Featured
 * coach programs"); each one names its coach, whose public profile lists every
 * program they published. Only coaches with at least one published program
 * are public, and only their id and registered name are exposed.
 */

export interface PublicCoach {
  id: string;
  name: string;
}

export interface CoachProgram extends MarketplaceProgram {
  views: number;
  coach: PublicCoach;
}

export interface CoachProfile {
  coach: PublicCoach & { coach_since: string; total_views: number };
  programs: CoachProgram[];
}

export const fetchFeaturedCoachPrograms = async (limit = 6): Promise<CoachProgram[]> => {
  const data = await apiGet<{ programs: CoachProgram[] }>(`/coaches/featured-programs?limit=${limit}`);
  return data.programs ?? [];
};

export const fetchCoachProfile = (trainerId: string) =>
  apiGet<CoachProfile>(`/coaches/${encodePathSegment(trainerId)}`);

/**
 * Counts one view of a program page. The server deduplicates (once per viewer
 * and program every 30 minutes) and ignores the coach's own visits, so the
 * page can call this on every open. Failures are irrelevant to the visitor.
 */
export const recordProgramView = async (programId: string): Promise<void> => {
  try {
    await apiPost<unknown>(`/programs/${encodePathSegment(programId)}/views`, {});
  } catch {
    // A missed view never affects the page.
  }
};

/** Up to two initials for the coach avatar. */
export const coachInitials = (name: string): string => {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  const first = parts[0][0] ?? "";
  const last = parts.length > 1 ? (parts[parts.length - 1][0] ?? "") : "";
  return (first + last).toUpperCase();
};
