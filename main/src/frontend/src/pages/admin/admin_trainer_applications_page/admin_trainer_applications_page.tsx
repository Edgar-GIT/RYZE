import { Check, X } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/button/button";
import {
  ApplicationStatus,
  approveAdminTrainerApplication,
  fetchAdminTrainerApplications,
  formatAdminDate,
  fullName,
  rejectAdminTrainerApplication,
  type AdminTrainerApplication
} from "@/services/admin_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./admin_trainer_applications_page.module.css";

const APPLICATIONS_PER_PAGE = 20;

type StatusFilter = "ALL" | "PENDING" | "APPROVED" | "REJECTED";

const statusBadgeClass = (status: string): string => {
  if (status === ApplicationStatus.PENDING) {
    return styles.statusPending;
  }
  if (status === ApplicationStatus.APPROVED) {
    return styles.statusApproved;
  }
  return styles.statusRejected;
};

export const AdminTrainerApplicationsPage = () => {
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("ALL");
  const [page, setPage] = useState(1);
  const [applications, setApplications] = useState<AdminTrainerApplication[]>([]);
  const [pagination, setPagination] = useState<{ page: number; total_pages: number; total: number } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionInProgress, setActionInProgress] = useState<string | null>(null);

  const load = useCallback(async (targetPage: number, status: StatusFilter) => {
    setLoading(true);
    setError(null);
    try {
      const result = await fetchAdminTrainerApplications(targetPage, APPLICATIONS_PER_PAGE, status === "ALL" ? undefined : status);
      setApplications(result.applications);
      setPagination(result.pagination);
      setPage(result.pagination.page);
    } catch {
      setError("Unable to load trainer applications.");
      setApplications([]);
      setPagination(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(page, statusFilter);
  }, [statusFilter, page, load]);

  const handleApprove = async (id: string) => {
    setActionInProgress(id);
    setError(null);
    try {
      await approveAdminTrainerApplication(id);
      await load(page, statusFilter);
    } catch {
      setError("Unable to approve application.");
    } finally {
      setActionInProgress(null);
    }
  };

  const handleReject = async (id: string) => {
    setActionInProgress(id);
    setError(null);
    try {
      await rejectAdminTrainerApplication(id);
      await load(page, statusFilter);
    } catch {
      setError("Unable to reject application.");
    } finally {
      setActionInProgress(null);
    }
  };

  return (
    <div className={styles.applicationsPage}>
      <header className={styles.header}>
        <div className={styles.headerText}>
          <p className={styles.eyebrow}>Management</p>
          <h1 className={styles.title}>Trainer applications</h1>
          <p className={styles.description}>
            Review trainer applications, approve or reject requests.
          </p>
        </div>
      </header>

      <div className={styles.tabs} role="tablist" aria-label="Application status filter">
        {(["ALL", "PENDING", "APPROVED", "REJECTED"] as const).map((status) => (
          <button
            key={status}
            type="button"
            role="tab"
            aria-selected={statusFilter === status}
            className={joinClassNames(styles.tab, statusFilter === status && styles.tabActive)}
            onClick={() => { setStatusFilter(status); setPage(1); }}
          >
            {status === "ALL"
              ? "All"
              : status === "PENDING"
                ? "Pending"
                : status === "APPROVED"
                  ? "Approved"
                  : "Rejected"}
          </button>
        ))}
      </div>

      {error ? (
        <p className={styles.errorBanner} role="alert">{error}</p>
      ) : null}

      {loading ? (
        <div className={styles.tableCard}>
          <p className={styles.loadingBanner}>Loading applications…</p>
        </div>
      ) : applications.length === 0 ? (
        <div className={styles.tableCard}>
          <div className={styles.emptyState}>
            <p className={styles.emptyStateEmphasis}>
              {statusFilter === "ALL"
                ? "No applications yet."
                : `No ${statusFilter.toLowerCase()} applications.`}
            </p>
          </div>
        </div>
      ) : (
        <>
          <div className={styles.tableCard}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Applicant</th>
                  <th scope="col">Email</th>
                  <th scope="col">Status</th>
                  <th scope="col">Submitted</th>
                  <th scope="col" aria-label="Actions" style={{ width: "7rem" }} />
                </tr>
              </thead>
              <tbody>
                {applications.map((application) => (
                  <tr key={application.id}>
                    <td className={styles.nameCell}>{fullName(application.user)}</td>
                    <td>{application.user.email}</td>
                    <td>
                      <span className={joinClassNames(styles.statusBadge, statusBadgeClass(application.status))}>
                        {application.status.charAt(0) + application.status.slice(1).toLowerCase()}
                      </span>
                    </td>
                    <td>{formatAdminDate(application.created_at)}</td>
                    <td>
                      {application.status === ApplicationStatus.PENDING ? (
                        <div className={styles.actionsCell}>
                          <button
                            type="button"
                            className={joinClassNames(styles.iconButton, styles.actionApprove)}
                            aria-label={`Approve ${fullName(application.user)}`}
                            title="Approve"
                            disabled={actionInProgress === application.id}
                            onClick={() => handleApprove(application.id)}
                          >
                            <Check aria-hidden="true" />
                          </button>
                          <button
                            type="button"
                            className={joinClassNames(styles.iconButton, styles.actionReject)}
                            aria-label={`Reject ${fullName(application.user)}`}
                            title="Reject"
                            disabled={actionInProgress === application.id}
                            onClick={() => handleReject(application.id)}
                          >
                            <X aria-hidden="true" />
                          </button>
                        </div>
                      ) : null}
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
    </div>
  );
};