import { Check, ChevronDown, ChevronUp, Download, FileText, X } from "lucide-react";
import { Fragment, useCallback, useEffect, useRef, useState } from "react";

import { Button } from "@/components/button/button";
import {
  ApplicationStatus,
  approveAdminTrainerApplication,
  downloadAdminApplicationDocument,
  fetchAdminApplicationDecisions,
  fetchAdminApplicationDocuments,
  fetchAdminTrainerApplications,
  formatAdminDate,
  fullName,
  rejectAdminTrainerApplication,
  type AdminApplicationDecision,
  type AdminApplicationDocument,
  type AdminTrainerApplication
} from "@/services/admin_api";
import { APPLICATION_DOCUMENT_KIND_LABELS, type ApplicationDocumentKindValue } from "@/services/trainer_api";
import { joinClassNames } from "@utils/class_names";
import { formatBytes } from "@utils/format_bytes";

import styles from "./admin_trainer_applications_page.module.css";

const APPLICATIONS_PER_PAGE = 20;

type StatusFilter = "ALL" | "PENDING" | "APPROVED" | "REJECTED";

interface ApplicationDetail {
  loading: boolean;
  error: string;
  documents: AdminApplicationDocument[];
  decisions: AdminApplicationDecision[];
  downloadingId: string | null;
}

const kindLabel = (kind: string): string =>
  APPLICATION_DOCUMENT_KIND_LABELS[kind as ApplicationDocumentKindValue] ?? kind;

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

  const [expandedId, setExpandedId] = useState<string | null>(null);
  const expandedRef = useRef<string | null>(null);
  const [detail, setDetail] = useState<ApplicationDetail | null>(null);
  const [decisionReason, setDecisionReason] = useState("");

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

  const closeReview = () => {
    expandedRef.current = null;
    setExpandedId(null);
    setDetail(null);
    setDecisionReason("");
  };

  const toggleReview = async (application: AdminTrainerApplication) => {
    if (expandedId === application.id) {
      closeReview();
      return;
    }

    expandedRef.current = application.id;
    setExpandedId(application.id);
    setDecisionReason("");
    setDetail({ loading: true, error: "", documents: [], decisions: [], downloadingId: null });

    try {
      const [documentsResult, decisionsResult] = await Promise.all([
        fetchAdminApplicationDocuments(application.id),
        fetchAdminApplicationDecisions(application.id)
      ]);
      if (expandedRef.current !== application.id) {
        return;
      }
      setDetail({
        loading: false,
        error: "",
        documents: documentsResult.documents,
        decisions: decisionsResult.decisions,
        downloadingId: null
      });
    } catch {
      if (expandedRef.current === application.id) {
        setDetail({ loading: false, error: "Unable to load the review details.", documents: [], decisions: [], downloadingId: null });
      }
    }
  };

  const handleDownload = async (document: AdminApplicationDocument) => {
    const applicationId = expandedRef.current;
    if (!applicationId) {
      return;
    }
    setDetail((current) => (current ? { ...current, downloadingId: document.id } : current));
    setError(null);
    try {
      await downloadAdminApplicationDocument(applicationId, document.id, document.original_filename);
    } catch {
      setError("Unable to download the document.");
    } finally {
      setDetail((current) => (current ? { ...current, downloadingId: null } : current));
    }
  };

  const handleApprove = async (id: string, reason?: string) => {
    setActionInProgress(id);
    setError(null);
    try {
      await approveAdminTrainerApplication(id, reason);
      closeReview();
      await load(page, statusFilter);
    } catch {
      setError("Unable to approve application.");
    } finally {
      setActionInProgress(null);
    }
  };

  const handleReject = async (id: string, reason?: string) => {
    setActionInProgress(id);
    setError(null);
    try {
      await rejectAdminTrainerApplication(id, reason);
      closeReview();
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
                  <th scope="col" aria-label="Review" style={{ width: "6rem" }} />
                </tr>
              </thead>
              <tbody>
                {applications.map((application) => (
                  <Fragment key={application.id}>
                    <tr>
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
                              onClick={() => void handleApprove(application.id)}
                            >
                              <Check aria-hidden="true" />
                            </button>
                            <button
                              type="button"
                              className={joinClassNames(styles.iconButton, styles.actionReject)}
                              aria-label={`Reject ${fullName(application.user)}`}
                              title="Reject"
                              disabled={actionInProgress === application.id}
                              onClick={() => void handleReject(application.id)}
                            >
                              <X aria-hidden="true" />
                            </button>
                          </div>
                        ) : null}
                      </td>
                      <td>
                        <button
                          type="button"
                          className={styles.expandButton}
                          aria-expanded={expandedId === application.id}
                          onClick={() => void toggleReview(application)}
                        >
                          {expandedId === application.id ? <ChevronUp size={14} aria-hidden="true" /> : <ChevronDown size={14} aria-hidden="true" />}
                          Review
                        </button>
                      </td>
                    </tr>
                    {expandedId === application.id ? (
                      <tr className={styles.detailRow}>
                        <td colSpan={6}>
                          {!detail || detail.loading ? (
                            <p className={styles.detailState}>Loading review details…</p>
                          ) : detail.error ? (
                            <p className={styles.detailState}>{detail.error}</p>
                          ) : (
                            <div className={styles.detailPanel}>
                              <div className={styles.detailColumn}>
                                <h2 className={styles.detailTitle}>Evidence documents</h2>
                                {detail.documents.length === 0 ? (
                                  <p className={styles.detailState}>
                                    No documents were attached to this application.
                                  </p>
                                ) : (
                                  <ul className={styles.documentList}>
                                    {detail.documents.map((document) => (
                                      <li key={document.id} className={styles.documentRow}>
                                        <FileText size={15} aria-hidden="true" className={styles.documentIcon} />
                                        <span className={styles.documentInfo}>
                                          <span className={styles.documentName}>{document.original_filename}</span>
                                          <span className={styles.documentMeta}>
                                            {kindLabel(document.kind)} · {formatBytes(document.byte_size)} ·{" "}
                                            {formatAdminDate(document.created_at)}
                                          </span>
                                        </span>
                                        <button
                                          type="button"
                                          className={styles.iconButton}
                                          aria-label={`Download ${document.original_filename}`}
                                          title="Download"
                                          disabled={detail.downloadingId === document.id}
                                          onClick={() => void handleDownload(document)}
                                        >
                                          <Download size={15} aria-hidden="true" />
                                        </button>
                                      </li>
                                    ))}
                                  </ul>
                                )}
                              </div>

                              <div className={styles.detailColumn}>
                                <h2 className={styles.detailTitle}>Decision history</h2>
                                {detail.decisions.length === 0 ? (
                                  <p className={styles.detailState}>No decisions recorded yet.</p>
                                ) : (
                                  <ul className={styles.decisionList}>
                                    {detail.decisions.map((decision) => (
                                      <li key={decision.id} className={styles.decisionItem}>
                                        <span className={styles.decisionTransition}>
                                          {decision.from_status} → {decision.to_status}
                                        </span>
                                        {decision.reason ? (
                                          <p className={styles.decisionReason}>{decision.reason}</p>
                                        ) : null}
                                        <span className={styles.decisionMeta}>{formatAdminDate(decision.created_at)}</span>
                                      </li>
                                    ))}
                                  </ul>
                                )}

                                {application.status === ApplicationStatus.PENDING ? (
                                  <div className={styles.reasonField}>
                                    <label className={styles.reasonLabel} htmlFor={`decision-reason-${application.id}`}>
                                      Decision note (optional, stored in the review history)
                                    </label>
                                    <textarea
                                      id={`decision-reason-${application.id}`}
                                      className={styles.reasonInput}
                                      rows={2}
                                      maxLength={500}
                                      value={decisionReason}
                                      onChange={(event) => setDecisionReason(event.target.value)}
                                    />
                                    <div className={styles.reasonActions}>
                                      <Button
                                        variant="secondary"
                                        size="small"
                                        disabled={actionInProgress === application.id}
                                        onClick={() => void handleApprove(application.id, decisionReason)}
                                      >
                                        Approve with note
                                      </Button>
                                      <Button
                                        variant="secondary"
                                        size="small"
                                        disabled={actionInProgress === application.id}
                                        onClick={() => void handleReject(application.id, decisionReason)}
                                      >
                                        Reject with note
                                      </Button>
                                    </div>
                                  </div>
                                ) : null}
                              </div>
                            </div>
                          )}
                        </td>
                      </tr>
                    ) : null}
                  </Fragment>
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