import { RefreshCw, ScrollText, ShieldAlert, ShieldCheck, ShieldX } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/button/button";
import {
  describeAuditAction,
  fetchAdminAuditLog,
  formatAdminDateTime,
  type AdminAuditEntry,
  type AdminAuditFilters,
  type AdminAuditOutcome
} from "@/services/admin_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./admin_audit_page.module.css";

const ENTRIES_PER_PAGE = 50;

const ADMIN_FILTERS: Array<{ value: AdminAuditFilters["admin"]; label: string }> = [
  { value: undefined, label: "Everyone" },
  { value: "ADMIN_1", label: "ADMIN_1 · Technical" },
  { value: "ADMIN_2", label: "ADMIN_2 · Management" }
];

const OUTCOME_FILTERS: Array<{ value: AdminAuditOutcome | undefined; label: string }> = [
  { value: undefined, label: "Any result" },
  { value: "success", label: "Done" },
  { value: "denied", label: "Denied" },
  { value: "failed", label: "Failed" }
];

const OUTCOME_VIEW: Record<AdminAuditOutcome, { label: string; className: string; Icon: typeof ShieldCheck }> = {
  success: { label: "Done", className: styles.outcomeSuccess, Icon: ShieldCheck },
  denied: { label: "Denied", className: styles.outcomeDenied, Icon: ShieldX },
  failed: { label: "Failed", className: styles.outcomeFailed, Icon: ShieldAlert }
};

// AdminAuditPage lists the immutable administrative audit log: every change an
// administrator made, every evidence download and every console sign-in,
// newest first. Records cannot be edited or deleted, here or in the database.
export const AdminAuditPage = () => {
  const [filters, setFilters] = useState<AdminAuditFilters>({});
  const [page, setPage] = useState(1);
  const [entries, setEntries] = useState<AdminAuditEntry[]>([]);
  const [pagination, setPagination] = useState<{ page: number; total_pages: number; total: number } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (targetPage: number, current: AdminAuditFilters) => {
    setLoading(true);
    setError(null);
    try {
      const result = await fetchAdminAuditLog(targetPage, ENTRIES_PER_PAGE, current);
      setEntries(result.entries);
      setPagination(result.pagination);
    } catch {
      setError("Unable to load the audit log.");
      setEntries([]);
      setPagination(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load(page, filters);
  }, [filters, page, load]);

  const applyFilter = (next: AdminAuditFilters) => {
    setFilters(next);
    setPage(1);
  };

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div className={styles.headerText}>
          <p className={styles.eyebrow}>Security</p>
          <h1 className={styles.title}>Audit log</h1>
          <p className={styles.description}>
            Every change made in the consoles, every document download and every sign-in. Entries cannot be edited or deleted.
          </p>
        </div>
        <Button type="button" variant="ghost" onClick={() => void load(page, filters)} disabled={loading}>
          <RefreshCw aria-hidden="true" className={styles.buttonIcon} /> Refresh
        </Button>
      </header>

      <div className={styles.filters}>
        <div className={styles.filterGroup} role="group" aria-label="Administrator">
          {ADMIN_FILTERS.map((option) => (
            <button
              key={option.label}
              type="button"
              aria-pressed={filters.admin === option.value}
              className={joinClassNames(styles.chip, filters.admin === option.value && styles.chipActive)}
              onClick={() => applyFilter({ ...filters, admin: option.value })}
            >
              {option.label}
            </button>
          ))}
        </div>
        <div className={styles.filterGroup} role="group" aria-label="Result">
          {OUTCOME_FILTERS.map((option) => (
            <button
              key={option.label}
              type="button"
              aria-pressed={filters.outcome === option.value}
              className={joinClassNames(styles.chip, filters.outcome === option.value && styles.chipActive)}
              onClick={() => applyFilter({ ...filters, outcome: option.value })}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      {error ? (
        <p className={styles.errorBanner} role="alert">{error}</p>
      ) : null}

      <div className={styles.tableCard}>
        {loading ? (
          <p className={styles.loadingBanner}>Loading the audit log…</p>
        ) : entries.length === 0 ? (
          <div className={styles.emptyState}>
            <ScrollText aria-hidden="true" />
            <p className={styles.emptyStateEmphasis}>Nothing recorded yet.</p>
            <p>Administrative actions appear here as soon as they happen.</p>
          </div>
        ) : (
          <table className={`${styles.table} app-stack-table`}>
            <thead>
              <tr>
                <th scope="col">What</th>
                <th scope="col">Who</th>
                <th scope="col">When</th>
                <th scope="col">Result</th>
                <th scope="col">Target</th>
                <th scope="col">Address</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => {
                const outcome = OUTCOME_VIEW[entry.outcome] ?? OUTCOME_VIEW.failed;
                return (
                  <tr key={entry.id}>
                    <td data-wide="">
                      <span className={styles.action}>{describeAuditAction(entry.action)}</span>
                      <code className={styles.route}>
                        {entry.method} {entry.route.replace(/^\/api\/v1/, "")}
                      </code>
                    </td>
                    <td data-label="Who">
                      <span className={joinClassNames(styles.adminBadge, entry.admin_id === "ADMIN_2" && styles.adminBadgeTwo)}>
                        {entry.admin_id}
                      </span>
                    </td>
                    <td data-label="When" className={styles.nowrap}>{formatAdminDateTime(entry.created_at)}</td>
                    <td data-label="Result">
                      <span className={joinClassNames(styles.outcome, outcome.className)} title={`HTTP ${entry.status_code}`}>
                        <outcome.Icon aria-hidden="true" /> {outcome.label}
                        <span className={styles.status}>{entry.status_code}</span>
                      </span>
                    </td>
                    <td data-label="Target">
                      {entry.target_id ? <code className={styles.target} title={entry.target_id}>{entry.target_id}</code> : "—"}
                    </td>
                    <td data-label="Address" className={styles.nowrap}>{entry.client_ip ?? "—"}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {pagination && pagination.total_pages > 1 ? (
        <div className={styles.pagination}>
          <p className={styles.paginationInfo}>
            Page {pagination.page} of {pagination.total_pages} · {pagination.total} entries
          </p>
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
    </div>
  );
};
