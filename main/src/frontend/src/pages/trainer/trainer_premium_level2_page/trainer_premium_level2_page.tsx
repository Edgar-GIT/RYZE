import { ChevronRight, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Admin2StatusBadge, type Admin2BadgeTone } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { fetchTrainerPremiumLevel2Requests, PREMIUM_LEVEL_2_STATUS_LABELS, type PremiumLevel2RequestStatus, type StaffPremiumLevel2Request } from "@/services/premium_level_2_api";
import { formatAdminDate, type AdminPagination } from "@/services/admin_api";

import styles from "./trainer_premium_level2_page.module.css";

const PAGE_SIZE = 20;

const statusTone = (status: PremiumLevel2RequestStatus): Admin2BadgeTone => {
  if (status === "delivered") return "success";
  if (status === "ready_for_delivery") return "warning";
  if (status === "cancelled") return "danger";
  return "neutral";
};

const clientName = (request: StaffPremiumLevel2Request): string =>
  [request.client_first_name, request.client_last_name].filter(Boolean).join(" ") || request.client_email;

export const TrainerPremiumLevel2Page = () => {
  const [requests, setRequests] = useState<StaffPremiumLevel2Request[]>([]);
  const [pagination, setPagination] = useState<AdminPagination | null>(null);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<PremiumLevel2RequestStatus | "">("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const result = await fetchTrainerPremiumLevel2Requests(page, PAGE_SIZE, status);
      setRequests(result.requests);
      setPagination(result.pagination);
    } catch {
      setRequests([]);
      setPagination(null);
      setError("Unable to load your Premium Level 2 requests.");
    } finally {
      setLoading(false);
    }
  }, [page, status]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <p className={styles.eyebrow}>Premium Level 2</p>
          <h1 className={styles.title}>Client services</h1>
          <p className={styles.subtitle}>
            Read each client intake, author the training programme and nutrition plan, then deliver
            the service.
          </p>
        </div>
        <select
          className={styles.select}
          value={status}
          aria-label="Filter by status"
          onChange={(event) => {
            setStatus(event.target.value as PremiumLevel2RequestStatus | "");
            setPage(1);
          }}
        >
          <option value="">All statuses</option>
          <option value="in_progress">{PREMIUM_LEVEL_2_STATUS_LABELS.in_progress}</option>
          <option value="ready_for_delivery">{PREMIUM_LEVEL_2_STATUS_LABELS.ready_for_delivery}</option>
          <option value="delivered">{PREMIUM_LEVEL_2_STATUS_LABELS.delivered}</option>
        </select>
      </header>

      {loading ? (
        <p className={styles.muted}>Loading your services…</p>
      ) : error ? (
        <div className={styles.stateBlock}>
          <p className={styles.error}>{error}</p>
          <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => void load()}>
            Try again
          </Button>
        </div>
      ) : requests.length === 0 ? (
        <p className={styles.muted}>No Premium Level 2 services are allocated to you.</p>
      ) : (
        <div className={styles.list}>
          {requests.map((request) => (
            <Link key={request.id} to={`/trainer/premium-level-2/${request.id}`} className={styles.row}>
              <span className={styles.rowMain}>
                <strong>{request.offer_name}</strong>
                <span>
                  {clientName(request)} · {request.client_email}
                </span>
                <span className={styles.rowMeta}>
                  Intake revision {request.intake_version || "not submitted"} · updated{" "}
                  {formatAdminDate(request.updated_at)}
                </span>
              </span>
              <span className={styles.rowSide}>
                <Admin2StatusBadge
                  label={PREMIUM_LEVEL_2_STATUS_LABELS[request.status]}
                  tone={statusTone(request.status)}
                />
                <ChevronRight size={16} aria-hidden="true" />
              </span>
            </Link>
          ))}
        </div>
      )}

      {pagination && pagination.total_pages > 1 ? (
        <div className={styles.pager}>
          <Button
            variant="secondary"
            size="small"
            disabled={pagination.page <= 1}
            onClick={() => setPage((current) => Math.max(1, current - 1))}
          >
            Previous
          </Button>
          <span className={styles.muted}>
            Page {pagination.page} of {pagination.total_pages}
          </span>
          <Button
            variant="secondary"
            size="small"
            disabled={pagination.page >= pagination.total_pages}
            onClick={() => setPage((current) => current + 1)}
          >
            Next
          </Button>
        </div>
      ) : null}
    </div>
  );
};
