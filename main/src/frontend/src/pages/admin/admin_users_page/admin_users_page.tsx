import { Edit3, KeyRound, RefreshCw, Search, ShieldOff, UserPlus } from "lucide-react";
import type { FormEvent, ReactNode } from "react";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/button/button";
import {
  createAdminUser,
  disableAdminUser,
  fetchAdminDeletedUsers,
  fetchAdminUsers,
  fullName,
  formatAdminDate,
  reactivateAdminUser,
  resetAdminUserPassword,
  updateAdminUser,
  type AdminUser
} from "@/services/admin_api";
import { ApiError } from "@utils/http_client";
import { joinClassNames } from "@utils/class_names";

import styles from "./admin_users_page.module.css";

const USERS_PER_PAGE = 20;

type ActiveTab = "active" | "deleted";

type Dialog =
  | { mode: "create" }
  | { mode: "edit"; user: AdminUser }
  | { mode: "password"; user: AdminUser }
  | null;

const errorMessage = (error: unknown, fallback: string): string => {
  if (error instanceof ApiError) {
    if (error.code === "EMAIL_ALREADY_REGISTERED") {
      return "A user with this email already exists.";
    }
    if (error.code === "VALIDATION_ERROR") {
      return "Please check your input and try again.";
    }
    return error.message || fallback;
  }
  return fallback;
};

const ModalShell = ({
  title,
  description,
  onClose,
  children
}: {
  title: string;
  description: string;
  onClose: () => void;
  children: ReactNode;
}) => (
  <div className={styles.modalOverlay} role="dialog" aria-modal="true" aria-label={title}>
    <div className={styles.modalCard}>
      <header className={styles.modalHeader}>
        <h2 className={styles.modalTitle}>{title}</h2>
        <p className={styles.modalDescription}>{description}</p>
      </header>
      {children}
    </div>
  </div>
);

const ModalFormActions = ({ onCancel, submitting }: { onCancel: () => void; submitting: boolean }) => (
  <div className={styles.modalActions}>
    <Button type="button" variant="ghost" onClick={onCancel}>
      Cancel
    </Button>
    <Button type="submit" disabled={submitting}>
      {submitting ? "Saving…" : "Save changes"}
    </Button>
  </div>
);

const CreateUserDialog = ({ onClose, onSuccess }: { onClose: () => void; onSuccess: () => void }) => {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }

    const form = new FormData(event.currentTarget);
    const email = String(form.get("email") ?? "").trim();
    const password = String(form.get("password") ?? "");
    const first_name = String(form.get("first_name") ?? "").trim();
    const last_name = String(form.get("last_name") ?? "").trim();

    if (!email || !password) {
      setError("Email and password are required.");
      return;
    }

    setSubmitting(true);
    setError("");

    try {
      await createAdminUser({ email, password, first_name, last_name });
      onSuccess();
    } catch (caught: unknown) {
      setError(errorMessage(caught, "Unable to create user."));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <ModalShell title="Create user" description="Create a new active account." onClose={onClose}>
      <form onSubmit={handleSubmit} noValidate>
        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="create-user-email">Email</label>
          <input className={styles.formInput} id="create-user-email" name="email" type="email" required autoComplete="email" />
        </div>
        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="create-user-password">Password</label>
          <input className={styles.formInput} id="create-user-password" name="password" type="password" required autoComplete="new-password" />
        </div>
        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="create-user-first">First name</label>
          <input className={styles.formInput} id="create-user-first" name="first_name" type="text" autoComplete="given-name" />
        </div>
        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="create-user-last">Last name</label>
          <input className={styles.formInput} id="create-user-last" name="last_name" type="text" autoComplete="family-name" />
        </div>
        {error ? (
          <p className={styles.errorBanner} role="alert">{error}</p>
        ) : null}
        <div className={styles.modalActions}>
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={submitting}>
            {submitting ? "Creating…" : "Create account"}
          </Button>
        </div>
      </form>
    </ModalShell>
  );
};

const EditUserDialog = ({ user, onClose, onSuccess }: { user: AdminUser; onClose: () => void; onSuccess: () => void }) => {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }

    const form = new FormData(event.currentTarget);
    const email = String(form.get("email") ?? "").trim() || undefined;
    const first_name = String(form.get("first_name") ?? "").trim() || undefined;
    const last_name = String(form.get("last_name") ?? "").trim() || undefined;

    setSubmitting(true);
    setError("");

    try {
      await updateAdminUser(user.id, { email, first_name, last_name });
      onSuccess();
    } catch (caught: unknown) {
      setError(errorMessage(caught, "Unable to update user."));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <ModalShell title="Edit user" description={`Update the profile details for ${user.email}.`} onClose={onClose}>
      <form onSubmit={handleSubmit} noValidate>
        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="edit-user-email">Email</label>
          <input className={styles.formInput} id="edit-user-email" name="email" type="email" defaultValue={user.email} autoComplete="email" />
        </div>
        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="edit-user-first">First name</label>
          <input className={styles.formInput} id="edit-user-first" name="first_name" type="text" defaultValue={user.first_name} autoComplete="given-name" />
        </div>
        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="edit-user-last">Last name</label>
          <input className={styles.formInput} id="edit-user-last" name="last_name" type="text" defaultValue={user.last_name} autoComplete="family-name" />
        </div>
        {error ? (
          <p className={styles.errorBanner} role="alert">{error}</p>
        ) : null}
        <ModalFormActions onCancel={onClose} submitting={submitting} />
      </form>
    </ModalShell>
  );
};

const ResetPasswordDialog = ({ user, onClose, onSuccess }: { user: AdminUser; onClose: () => void; onSuccess: () => void }) => {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }

    const form = new FormData(event.currentTarget);
    const newPassword = String(form.get("new_password") ?? "");

    if (!newPassword) {
      setError("Please enter a new password.");
      return;
    }

    setSubmitting(true);
    setError("");

    try {
      await resetAdminUserPassword(user.id, newPassword);
      onSuccess();
    } catch (caught: unknown) {
      setError(errorMessage(caught, "Unable to reset password."));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <ModalShell title="Reset password" description={`Replace the current password for ${user.email}. This invalidates existing sessions.`} onClose={onClose}>
      <form onSubmit={handleSubmit} noValidate>
        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="reset-password">New password</label>
          <input className={styles.formInput} id="reset-password" name="new_password" type="password" required autoComplete="new-password" />
        </div>
        {error ? (
          <p className={styles.errorBanner} role="alert">{error}</p>
        ) : null}
        <div className={styles.modalActions}>
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={submitting}>
            {submitting ? "Resetting…" : "Reset password"}
          </Button>
        </div>
      </form>
    </ModalShell>
  );
};

export const AdminUsersPage = () => {
  const [activeTab, setActiveTab] = useState<ActiveTab>("active");
  const [page, setPage] = useState(1);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [pagination, setPagination] = useState<{ page: number; total_pages: number; total: number } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [dialog, setDialog] = useState<Dialog>(null);

  const load = useCallback(async (targetPage: number, tab: ActiveTab) => {
    setLoading(true);
    setError(null);
    try {
      const result = tab === "active"
        ? await fetchAdminUsers(targetPage, USERS_PER_PAGE)
        : await fetchAdminDeletedUsers(targetPage, USERS_PER_PAGE);
      setUsers(result.users);
      setPagination(result.pagination);
      setPage(result.pagination.page);
    } catch {
      setError("Unable to load users.");
      setUsers([]);
      setPagination(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(page, activeTab);
  }, [activeTab, page, load]);

  const filteredUsers = search.trim()
    ? users.filter(
      (user) =>
        user.email.toLowerCase().includes(search.toLowerCase()) ||
        fullName(user).toLowerCase().includes(search.toLowerCase())
    )
    : users;

  const resetDialogAndRefresh = async () => {
    setDialog(null);
    await load(page, activeTab);
  };

  const handleDisable = async (id: string) => {
    try {
      await disableAdminUser(id);
      await resetDialogAndRefresh();
    } catch {
      setError("Unable to disable user.");
    }
  };

  const handleReactivate = async (id: string) => {
    try {
      await reactivateAdminUser(id);
      await resetDialogAndRefresh();
    } catch {
      setError("Unable to reactivate user.");
    }
  };

  return (
    <div className={styles.usersPage}>
      <header className={styles.header}>
        <div className={styles.headerText}>
          <p className={styles.eyebrow}>Management</p>
          <h1 className={styles.title}>Users</h1>
          <p className={styles.description}>
            Active accounts and disabled identities.
          </p>
        </div>
        <div className={styles.headerActions}>
          <Button type="button" onClick={() => setDialog({ mode: "create" })}>
            <UserPlus aria-hidden="true" />
            Create user
          </Button>
        </div>
      </header>

      <div className={styles.tabs} role="tablist" aria-label="User tabs">
        {(["active", "deleted"] as const).map((tab) => (
          <button
            key={tab}
            type="button"
            role="tab"
            aria-selected={activeTab === tab}
            className={joinClassNames(styles.tab, activeTab === tab && styles.tabActive)}
            onClick={() => { setActiveTab(tab); setPage(1); setSearch(""); }}
          >
            {tab === "active" ? "Active" : "Deleted"}
          </button>
        ))}
      </div>

      <div className={styles.toolbar}>
        <div className={styles.searchWrap}>
          <Search className={styles.searchIcon} aria-hidden="true" />
          <input
            type="search"
            className={styles.searchInput}
            placeholder="Search by name or email…"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            aria-label="Filter current page"
          />
        </div>
        {pagination ? (
          <p className={styles.toolbarInfo}>
            Page {pagination.page} · {pagination.total.toLocaleString("en")} total
          </p>
        ) : null}
      </div>

      {error ? (
        <p className={styles.errorBanner} role="alert">{error}</p>
      ) : null}

      {loading ? (
        <div className={styles.tableCard}>
          <p className={styles.loadingBanner}>Loading users…</p>
        </div>
      ) : filteredUsers.length === 0 ? (
        <div className={styles.tableCard}>
          <div className={styles.emptyState}>
            <p className={styles.emptyStateEmphasis}>
              {search.trim()
                ? "No users match your search on this page."
                : activeTab === "deleted"
                  ? "No disabled users."
                  : "No users yet."}
            </p>
          </div>
        </div>
      ) : (
        <>
          <div className={styles.tableCard}>
            <table className={styles.usersTable}>
              <thead>
                <tr>
                  <th scope="col">Name</th>
                  <th scope="col">Email</th>
                  <th scope="col">Created</th>
                  <th scope="col" aria-label="Actions" style={{ width: "7rem" }} />
                </tr>
              </thead>
              <tbody>
                {filteredUsers.map((user) => (
                  <tr key={user.id}>
                    <td className={styles.nameCell}>{fullName(user)}</td>
                    <td className={styles.emailCell}>{user.email}</td>
                    <td>{formatAdminDate(user.created_at)}</td>
                    <td>
                      <div className={styles.actionsCell}>
                        {activeTab === "active" ? (
                          <>
                            <button
                              type="button"
                              className={styles.iconButton}
                              aria-label={`Edit ${user.email}`}
                              title="Edit"
                              onClick={() => setDialog({ mode: "edit", user })}
                            >
                              <Edit3 aria-hidden="true" />
                            </button>
                            <button
                              type="button"
                              className={joinClassNames(styles.iconButton, styles.iconButtonDanger)}
                              aria-label={`Disable ${user.email}`}
                              title="Disable"
                              onClick={() => handleDisable(user.id)}
                            >
                              <ShieldOff aria-hidden="true" />
                            </button>
                            <button
                              type="button"
                              className={styles.iconButton}
                              aria-label={`Reset password for ${user.email}`}
                              title="Reset password"
                              onClick={() => setDialog({ mode: "password", user })}
                            >
                              <KeyRound aria-hidden="true" />
                            </button>
                          </>
                        ) : (
                          <button
                            type="button"
                            className={styles.iconButton}
                            aria-label={`Reactivate ${user.email}`}
                            title="Reactivate"
                            onClick={() => handleReactivate(user.id)}
                          >
                            <RefreshCw aria-hidden="true" />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {pagination && pagination.total_pages > 1 ? (
            <div className={styles.pagination}>
              <p className={styles.paginationInfo}>Page {pagination.page} of {pagination.total_pages}</p>
              {pagination.page > 1 ? (
                <Button type="button" variant="ghost" onClick={() => setPage((p) => Math.max(1, p - 1))}>
                  Previous
                </Button>
              ) : null}
              {pagination.page < pagination.total_pages ? (
                <Button type="button" variant="ghost" onClick={() => setPage((p) => p + 1)}>
                  Next
                </Button>
              ) : null}
            </div>
          ) : null}
        </>
      )}

      {dialog?.mode === "create" ? (
        <CreateUserDialog onClose={() => setDialog(null)} onSuccess={resetDialogAndRefresh} />
      ) : null}
      {dialog?.mode === "edit" ? (
        <EditUserDialog user={dialog.user} onClose={() => setDialog(null)} onSuccess={resetDialogAndRefresh} />
      ) : null}
      {dialog?.mode === "password" ? (
        <ResetPasswordDialog user={dialog.user} onClose={() => setDialog(null)} onSuccess={resetDialogAndRefresh} />
      ) : null}
    </div>
  );
};