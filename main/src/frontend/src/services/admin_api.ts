import { apiGet, apiPatch, apiPost } from "@utils/http_client";

// AdminIdentity is the authenticated administrator returned by GET
// /admin/auth/me. The role is always derived server-side from the configured
// admin identity, never from the client.
export interface AdminIdentity {
  id: string;
  role: string;
}

export interface AdminPagination {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface AdminUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  created_at: string;
  updated_at: string;
}

export interface AdminTrainer {
  id: string;
  user_id: string;
  email: string;
  first_name: string;
  last_name: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface AdminTrainerApplication {
  id: string;
  status: string;
  user_id: string;
  user: AdminUser;
  created_at: string;
  updated_at: string;
}

export interface AdminUsersList {
  users: AdminUser[];
  pagination: AdminPagination;
}

export interface AdminTrainersList {
  trainers: AdminTrainer[];
  pagination: AdminPagination;
}

export interface AdminTrainerApplicationsList {
  applications: AdminTrainerApplication[];
  pagination: AdminPagination;
}

export const ApplicationStatus = {
  PENDING: "PENDING",
  APPROVED: "APPROVED",
  REJECTED: "REJECTED"
} as const;

export const ADMIN_1 = "ADMIN_1";
export const ADMIN_2 = "ADMIN_2";

export const TECHNICAL_ADMINISTRATOR = "TECHNICAL_ADMINISTRATOR";
export const MANAGEMENT_ADMINISTRATOR = "MANAGEMENT_ADMINISTRATOR";

export const fetchAdminIdentity = () => apiGet<AdminIdentity>("/admin/auth/me");

export const signOutAdmin = () => apiPost<never>("/admin/auth/logout", {});

export const fetchAdminUsers = (page: number, limit: number) =>
  apiGet<AdminUsersList>(`/admin/users?page=${page}&limit=${limit}`);

export const fetchAdminDeletedUsers = (page: number, limit: number) =>
  apiGet<AdminUsersList>(`/admin/users/deleted?page=${page}&limit=${limit}`);

export const createAdminUser = (input: { email: string; password: string; first_name: string; last_name: string }) =>
  apiPost<AdminUser>("/admin/users", input);

export const updateAdminUser = (
  id: string,
  input: { email?: string; first_name?: string; last_name?: string }
) => apiPatch<AdminUser>(`/admin/users/${id}`, input);

export const disableAdminUser = (id: string) => apiPatch<never>(`/admin/users/${id}/disable`, {});

export const reactivateAdminUser = (id: string) => apiPost<AdminUser>(`/admin/users/${id}/reactivate`, {});

export const resetAdminUserPassword = (id: string, newPassword: string) =>
  apiPost<never>(`/admin/users/${id}/password`, { new_password: newPassword });

export const fetchAdminTrainers = (page: number, limit: number) =>
  apiGet<AdminTrainersList>(`/admin/trainers?page=${page}&limit=${limit}`);

export const fetchAdminDeletedTrainers = (page: number, limit: number) =>
  apiGet<AdminTrainersList>(`/admin/trainers/deleted?page=${page}&limit=${limit}`);

export const disableAdminTrainer = (id: string) => apiPatch<never>(`/admin/trainers/${id}/disable`, {});

export const reactivateAdminTrainer = (id: string) => apiPost<AdminTrainer>(`/admin/trainers/${id}/reactivate`, {});

export const fetchAdminTrainerApplications = (page: number, limit: number, status?: string) => {
  const statusQuery = status && status !== "ALL" ? `&status=${status}` : "";
  return apiGet<AdminTrainerApplicationsList>(`/admin/trainer-applications?page=${page}&limit=${limit}${statusQuery}`);
};

export const approveAdminTrainerApplication = (id: string) =>
  apiPost<AdminTrainerApplication>(`/admin/trainer-applications/${id}/approve`, {});

export const rejectAdminTrainerApplication = (id: string) =>
  apiPost<AdminTrainerApplication>(`/admin/trainer-applications/${id}/reject`, {});

export const adminRoleName = (role: string): string => {
  if (role === TECHNICAL_ADMINISTRATOR) {
    return "Technical Administrator";
  }
  if (role === MANAGEMENT_ADMINISTRATOR) {
    return "Management Administrator";
  }
  return "Administrator";
};

export const formatAdminDate = (iso: string): string => {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return "—";
  }
  return date.toLocaleDateString("en-GB", { day: "2-digit", month: "short", year: "numeric" });
};

export const fullName = (user: { first_name: string; last_name: string }): string => {
  const name = [user.first_name, user.last_name].filter(Boolean).join(" ").trim();
  return name || "—";
};