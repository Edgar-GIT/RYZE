import { apiDelete, apiPatch, apiUpload } from "@utils/http_client";

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

// The name people see: the display name, else the registered name.
export const publicNameOf = (user: Pick<MemberUser, "display_name" | "first_name" | "last_name" | "email">): string =>
  user.display_name?.trim() || `${user.first_name} ${user.last_name}`.trim() || user.email;

export const USERNAME_PATTERN = /^[a-z0-9][a-z0-9._]{2,29}$/;

export const isValidUsername = (value: string): boolean => {
  const name = value.trim().replace(/^@/, "").toLowerCase();
  return USERNAME_PATTERN.test(name) && !name.includes("..") && !name.endsWith(".");
};
