import { useCallback, useEffect, useState } from "react";
import { Check, RefreshCw, Star, X } from "lucide-react";

import { Admin2EmptyState } from "@/components/admin2/admin2_empty_state/admin2_empty_state";
import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Pagination } from "@/components/admin2/admin2_pagination/admin2_pagination";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge, type Admin2BadgeTone } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Button } from "@/components/button/button";
import {
  approveAdminFeedback,
  fetchAdminFeedback,
  rejectAdminFeedback,
  type AdminFeedback,
  type AdminFeedbackListResult,
  type AdminFeedbackStatus
} from "@/services/admin2_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./admin2_feedback_page.module.css";

const PAGE_SIZE = 20;

const TABS: Array<{ status: AdminFeedbackStatus; label: string }> = [
  { status: "pending", label: "Pending" },
  { status: "approved", label: "Approved" },
  { status: "rejected", label: "Rejected" }
];

const STATUS_TONE: Record<AdminFeedbackStatus, Admin2BadgeTone> = {
  pending: "warning",
  approved: "success",
  rejected: "danger"
};

const formatDate = (value: string) =>
  new Date(value).toLocaleDateString(undefined, { day: "2-digit", month: "short", year: "numeric" });

// Admin2FeedbackPage is the member feedback review queue: submissions appear on
// the public Feedback page only after they are approved here.
export default function Admin2FeedbackPage() {
  const [status, setStatus] = useState<AdminFeedbackStatus>("pending");
  const [page, setPage] = useState(1);
  const [result, setResult] = useState<AdminFeedbackListResult | null>(null);
  const [error, setError] = useState("");
  const [busyId, setBusyId] = useState<string | null>(null);
  const [actionError, setActionError] = useState("");

  const load = useCallback(async () => {
    setError("");
    try {
      setResult(await fetchAdminFeedback({ status, page, limit: PAGE_SIZE }));
    } catch {
      setResult(null);
      setError("Feedback could not be loaded. Try again.");
    }
  }, [status, page]);

  useEffect(() => {
    void load();
  }, [load]);

  const review = async (item: AdminFeedback, decision: "approve" | "reject") => {
    setBusyId(item.id);
    setActionError("");
    try {
      await (decision === "approve" ? approveAdminFeedback(item.id) : rejectAdminFeedback(item.id));
      await load();
    } catch {
      setActionError("The review could not be saved. It may already have been reviewed; the list was refreshed.");
      await load();
    } finally {
      setBusyId(null);
    }
  };

  return (
    <>
      <Admin2PageHeader
        eyebrow="Engagement"
        title="Feedback"
        description="Ratings and messages from members. Approved feedback appears on the public Feedback page."
      />

      <Admin2Section title="Member feedback" subtitle="Review submissions before they are published">
        <div className={styles.tabs} role="tablist" aria-label="Feedback status">
          {TABS.map((tab) => (
            <button
              key={tab.status}
              type="button"
              role="tab"
              aria-selected={status === tab.status}
              className={joinClassNames(styles.tab, status === tab.status && styles.tabActive)}
              onClick={() => {
                setStatus(tab.status);
                setPage(1);
              }}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {actionError ? (
          <p className={styles.error} role="alert">
            {actionError}
          </p>
        ) : null}

        {error ? (
          <div className={styles.pageState}>
            <p className={styles.error}>{error}</p>
            <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => void load()}>
              Try again
            </Button>
          </div>
        ) : result === null ? (
          <div className={styles.pageState}>
            <p className={styles.muted}>Loading feedback…</p>
          </div>
        ) : result.feedback.length === 0 ? (
          <Admin2EmptyState
            icon={Star}
            title={status === "pending" ? "Nothing to review" : `No ${status} feedback`}
            message={status === "pending" ? "New member feedback will appear here." : "Reviewed feedback is listed here."}
          />
        ) : (
          <>
            <ul className={styles.list}>
              {result.feedback.map((item) => (
                <li key={item.id} className={styles.item}>
                  <div className={styles.itemHead}>
                    <strong>{item.display_name}</strong>
                    <span className={styles.stars} role="img" aria-label={`${item.rating} out of 5 stars`}>
                      {"★".repeat(item.rating)}
                      <span className={styles.starsOff}>{"★".repeat(5 - item.rating)}</span>
                    </span>
                    <span className={styles.muted}>{formatDate(item.created_at)}</span>
                    <Admin2StatusBadge label={item.status} tone={STATUS_TONE[item.status]} />
                  </div>
                  <p className={styles.message}>{item.message}</p>
                  {item.status === "pending" ? (
                    <div className={styles.actions}>
                      <Button
                        size="small"
                        icon={<Check size={15} />}
                        disabled={busyId === item.id}
                        onClick={() => void review(item, "approve")}
                      >
                        Approve
                      </Button>
                      <Button
                        variant="secondary"
                        size="small"
                        icon={<X size={15} />}
                        disabled={busyId === item.id}
                        onClick={() => void review(item, "reject")}
                      >
                        Reject
                      </Button>
                    </div>
                  ) : item.reviewed_at ? (
                    <p className={styles.muted}>
                      Reviewed {formatDate(item.reviewed_at)}
                      {item.reviewed_by ? ` by ${item.reviewed_by}` : ""}
                    </p>
                  ) : null}
                </li>
              ))}
            </ul>
            <Admin2Pagination
              page={result.pagination.page}
              totalPages={result.pagination.total_pages}
              pageSize={result.pagination.limit}
              totalItems={result.pagination.total}
              onChange={setPage}
            />
          </>
        )}
      </Admin2Section>
    </>
  );
}
