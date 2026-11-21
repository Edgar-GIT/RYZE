import { apiDelete, apiGet, apiPost, encodePathSegment } from "@utils/http_client";

import type { LifterLevelKey } from "@/services/achievements_api";
import type { SocialLink } from "@/services/profile_api";

/** An account as profiles and lists show it: never the email or the
 *  registered name. */
export interface PublicPerson {
  username: string;
  nickname: string | null;
  avatar_url: string | null;
}

export interface PublicProfile extends PublicPerson {
  member_since: string;
  stats: {
    workouts: number;
    followers: number;
    following: number;
    level: { rank: number; key: LifterLevelKey | "" };
    medals: number;
    /** A coach's reach (views of their published plans); null for members. */
    views?: number | null;
  };
  /** The public coach profile of a trainer, when there is one. */
  coach_id: string | null;
  /** The optional "about": only what the person filled in. */
  about: {
    bio: string;
    age: number | null;
    training_years: number | null;
    weight_kg: number | null;
    socials: SocialLink[];
  } | null;
  viewer: { signed_in: boolean; is_self: boolean; following: boolean };
}

export const fetchPublicProfile = (username: string): Promise<PublicProfile> =>
  apiGet<PublicProfile>(`/profiles/${encodePathSegment(username)}`);

export const fetchFollowers = async (username: string): Promise<PublicPerson[]> =>
  (await apiGet<{ people: PublicPerson[] }>(`/profiles/${encodePathSegment(username)}/followers`)).people;

export const fetchFollowing = async (username: string): Promise<PublicPerson[]> =>
  (await apiGet<{ people: PublicPerson[] }>(`/profiles/${encodePathSegment(username)}/following`)).people;

export const followUser = (username: string) => apiPost<{ following: boolean }>(`/me/following/${encodePathSegment(username)}`, {});

export const unfollowUser = (username: string) => apiDelete<{ following: boolean }>(`/me/following/${encodePathSegment(username)}`);

/** The address of a public profile. */
export const profilePath = (username: string) => `/u/${encodeURIComponent(username)}`;
