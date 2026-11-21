import { apiDelete, apiGet, apiPatch, apiPut, apiUpload } from "@utils/http_client";

import type { MemberUser } from "@/components/app_shell/member_shell";

/** Sets the display name ("" clears it) and, when given, a new @username. */
export const updateProfile = (displayName: string, username: string): Promise<MemberUser> =>
  apiPatch<MemberUser>("/me/profile", { display_name: displayName, username });

/** Uploads a new profile photo (JPEG or PNG; stored re-encoded, 512 px). */
export const uploadAvatar = (file: File): Promise<MemberUser> => {
  const form = new FormData();
  form.append("file", file);
  return apiUpload<MemberUser>("/me/avatar", form);
};

export const removeAvatar = (): Promise<MemberUser> => apiDelete<MemberUser>("/me/avatar");

// The name people see (owner decision, 2026-11-21): the nickname, else the
// @username, never the email or the registered first and last name.
export const publicNameOf = (user: Pick<MemberUser, "display_name" | "username">): string =>
  user.display_name?.trim() || user.username;

/** Whether the person set a nickname (the @username is then shown under it;
 *  without one the @username is the name, shown once). */
export const hasNickname = (user: Pick<MemberUser, "display_name">): boolean => Boolean(user.display_name?.trim());

/** Records the @username the person chose (first visit of an account whose
 *  @username was generated, or a later change). */
export const chooseUsername = (username: string): Promise<MemberUser> =>
  apiPut<MemberUser>("/me/username", { username: username.trim().replace(/^@/, "").toLowerCase() });

export const USERNAME_PATTERN = /^[a-z0-9][a-z0-9._]{2,29}$/;

export const isValidUsername = (value: string): boolean => {
  const name = value.trim().replace(/^@/, "").toLowerCase();
  return USERNAME_PATTERN.test(name) && !name.includes("..") && !name.endsWith(".");
};

/** A social network on a profile: the stored handle and the link the server
 *  builds from it. */
export interface SocialLink {
  platform: import("@/components/social_icons/social_icons").SocialPlatform;
  handle: string;
  url: string;
}

/** The public "about" of the account, as its owner edits it. */
export interface ProfileDetails {
  bio: string;
  /** YYYY-MM-DD or "" (only the age is shown). */
  birth_date: string;
  /** The year the person started training (shown as years of training). */
  training_since: number | null;
  weight_kg: number | null;
  socials: SocialLink[];
}

export const fetchProfileDetails = (): Promise<ProfileDetails> => apiGet<ProfileDetails>("/me/profile/details");

/** Replaces the whole "about"; an empty value clears a field or a network. */
export const saveProfileDetails = (details: {
  bio: string;
  birth_date: string;
  training_since: number | null;
  weight_kg: number | null;
  socials: Record<string, string>;
}): Promise<ProfileDetails> => apiPut<ProfileDetails>("/me/profile/details", details);
