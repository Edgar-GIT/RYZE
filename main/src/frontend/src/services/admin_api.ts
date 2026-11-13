import { apiGet, apiPatch, apiPost, downloadFile, encodePathSegment } from "@utils/http_client";

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

// Evidence documents and the append-only decision history of one trainer
// application, as exposed to reviewers.
export interface AdminApplicationDocument {
  id: string;
  kind: string;
  original_filename: string;
  content_type: string;
  byte_size: number;
  created_at: string;
}

export interface AdminApplicationDecision {
  id: string;
  application_id: string;
  admin_id: string;
  from_status: string;
  to_status: string;
  reason: string;
  created_at: string;
}

export interface AdminApplicationDocumentsList {
  documents: AdminApplicationDocument[];
}

export interface AdminApplicationDecisionsList {
  decisions: AdminApplicationDecision[];
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
) => apiPatch<AdminUser>(`/admin/users/${encodePathSegment(id)}`, input);

export const disableAdminUser = (id: string) => apiPatch<never>(`/admin/users/${encodePathSegment(id)}/disable`, {});

export const reactivateAdminUser = (id: string) => apiPost<AdminUser>(`/admin/users/${encodePathSegment(id)}/reactivate`, {});

export const resetAdminUserPassword = (id: string, newPassword: string) =>
  apiPost<never>(`/admin/users/${encodePathSegment(id)}/password`, { new_password: newPassword });

export const fetchAdminTrainers = (page: number, limit: number) =>
  apiGet<AdminTrainersList>(`/admin/trainers?page=${page}&limit=${limit}`);

export const fetchAdminDeletedTrainers = (page: number, limit: number) =>
  apiGet<AdminTrainersList>(`/admin/trainers/deleted?page=${page}&limit=${limit}`);

export const disableAdminTrainer = (id: string) => apiPatch<never>(`/admin/trainers/${encodePathSegment(id)}/disable`, {});

export const reactivateAdminTrainer = (id: string) => apiPost<AdminTrainer>(`/admin/trainers/${encodePathSegment(id)}/reactivate`, {});

export const fetchAdminTrainerApplications = (page: number, limit: number, status?: string) => {
  const statusQuery = status && status !== "ALL" ? `&status=${status}` : "";
  return apiGet<AdminTrainerApplicationsList>(`/admin/trainer-applications?page=${page}&limit=${limit}${statusQuery}`);
};

// Review decisions may carry an optional reason; an empty reason is omitted
// rather than sent as an empty string.
const decisionReasonBody = (reason?: string): Record<string, string> => {
  const trimmed = reason?.trim();
  return trimmed ? { reason: trimmed } : {};
};

export const approveAdminTrainerApplication = (id: string, reason?: string) =>
  apiPost<AdminTrainerApplication>(`/admin/trainer-applications/${encodePathSegment(id)}/approve`, decisionReasonBody(reason));

export const rejectAdminTrainerApplication = (id: string, reason?: string) =>
  apiPost<AdminTrainerApplication>(`/admin/trainer-applications/${encodePathSegment(id)}/reject`, decisionReasonBody(reason));

export const fetchAdminApplicationDocuments = (applicationId: string) =>
  apiGet<AdminApplicationDocumentsList>(
    `/admin/trainer-applications/${encodePathSegment(applicationId)}/documents`
  );

export const fetchAdminApplicationDecisions = (applicationId: string) =>
  apiGet<AdminApplicationDecisionsList>(
    `/admin/trainer-applications/${encodePathSegment(applicationId)}/decisions`
  );

export const downloadAdminApplicationDocument = (
  applicationId: string,
  documentId: string,
  fallbackName: string
): Promise<void> =>
  downloadFile(
    `/admin/trainer-applications/${encodePathSegment(applicationId)}/documents/${encodePathSegment(documentId)}/download`,
    fallbackName
  );

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
// Technical Administrator system status (GET /admin/system/status).
export interface AdminSystemStatus {
  version: string;
  started_at: string;
  uptime_seconds: number;
  database: { reachable: boolean; schema_version?: number; schema_dirty?: boolean; open_connections?: number };
  email_queue: { status: string; count: number }[];
  visible_notifications: number;
  open_support: number;
  features: {
    email_delivery: boolean;
    google_sign_in: boolean;
    payment_methods: string[];
    exercise_media: boolean;
    rate_limiting: boolean;
    test_mode: boolean;
    contact_delivery: boolean;
    secure_cookies: boolean;
    default_platform_commission_bps: number;
    session_lifetime_seconds: number;
  };
}

export const fetchAdminSystemStatus = () => apiGet<AdminSystemStatus>("/admin/system/status");

// formatDuration turns seconds into "3d 4h", "5h 12m" or "8m".
export const formatDuration = (seconds: number): string => {
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  if (days > 0) return `${days}d ${hours % 24}h`;
  if (hours > 0) return `${hours}h ${minutes % 60}m`;
  return `${Math.max(minutes, 0)}m`;
};

// --- Audit log (ADMIN_1) ---

export type AdminAuditOutcome = "success" | "denied" | "failed";

export interface AdminAuditEntry {
  id: string;
  admin_id: "ADMIN_1" | "ADMIN_2";
  action: string;
  method: string;
  route: string;
  target_id: string | null;
  status_code: number;
  outcome: AdminAuditOutcome;
  client_ip: string | null;
  created_at: string;
}

export interface AdminAuditList {
  entries: AdminAuditEntry[];
  pagination: { page: number; limit: number; total: number; total_pages: number };
}

export interface AdminAuditFilters {
  admin?: "ADMIN_1" | "ADMIN_2";
  outcome?: AdminAuditOutcome;
}

export const fetchAdminAuditLog = (page: number, limit: number, filters: AdminAuditFilters = {}) => {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (filters.admin) params.set("admin", filters.admin);
  if (filters.outcome) params.set("outcome", filters.outcome);
  return apiGet<AdminAuditList>(`/admin/logs?${params.toString()}`);
};

// Readable names of the recorded actions. Unknown actions (a route added
// later) fall back to the raw "<method> <route>" the API stores.
const AUDIT_ACTION_LABELS: Record<string, string> = {
  "admin.sign_in": "Signed in to the console",
  "test_mode.enter": "Entered Test Mode",
  "user.create": "Created a user",
  "user.update": "Edited a user",
  "user.disable": "Disabled a user",
  "user.reactivate": "Reactivated a user",
  "user.password_reset": "Reset a user's password",
  "trainer.create": "Created a trainer",
  "trainer.update": "Edited a trainer",
  "trainer.disable": "Disabled a trainer",
  "trainer.reactivate": "Reactivated a trainer",
  "trainer_application.approve": "Approved a trainer application",
  "trainer_application.reject": "Rejected a trainer application",
  "trainer_application.document_download": "Downloaded an application document",
  "feedback.approve": "Approved member feedback",
  "feedback.reject": "Rejected member feedback",
  "support.reply": "Replied to a support conversation",
  "support.mark_read": "Marked a support conversation read",
  "support.close": "Closed a support conversation",
  "support.reopen": "Reopened a support conversation",
  "exercise.create": "Created an exercise",
  "exercise.update": "Edited an exercise",
  "premium_level_2.assign_trainer": "Assigned a Premium Level 2 trainer",
  "program.pricing_update": "Changed a program price",
  "commission.update": "Set a trainer commission",
  "commission.delete": "Removed a trainer commission",
  "generic_program.create": "Created a generic program",
  "generic_program.update": "Edited a generic program",
  "generic_program.publish": "Published a generic program",
  "generic_program.delete": "Deleted a generic program",
  "premium_level_1_program.create": "Created a Premium Level 1 package",
  "premium_level_1_program.update": "Edited a Premium Level 1 package",
  "premium_level_1_program.publish": "Published a Premium Level 1 package",
  "premium_level_1_program.delete": "Deleted a Premium Level 1 package",
  "premium_level_2_program.create": "Created a Premium Level 2 offer",
  "premium_level_2_program.update": "Edited a Premium Level 2 offer",
  "premium_level_2_program.publish": "Published a Premium Level 2 offer",
  "premium_level_2_program.delete": "Deleted a Premium Level 2 offer"
};

export const describeAuditAction = (action: string): string => AUDIT_ACTION_LABELS[action] ?? action;

export const formatAdminDateTime = (iso: string): string => {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return "—";
  }
  return date.toLocaleString("en-GB", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit", second: "2-digit" });
};
