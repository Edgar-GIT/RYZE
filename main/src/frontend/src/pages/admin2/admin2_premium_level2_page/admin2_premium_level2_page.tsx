import { AlertTriangle, CheckCircle2, RefreshCw, Search, UserRound, Users } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge, type Admin2BadgeTone } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { PremiumLevel2IntakeAnswers } from "@/components/premium_level2_intake/intake_answers";
import { Button } from "@/components/button/button";
import { fetchAdminTrainers, formatAdminDate, type AdminTrainer, type AdminPagination } from "@/services/admin_api";
import {
  assignPremiumLevel2Trainer,
  fetchAdminPremiumLevel2Catalog,
  fetchAdminPremiumLevel2Request,
  fetchAdminPremiumLevel2Requests,
  PREMIUM_LEVEL_2_STATUS_LABELS,
  type PremiumLevel2Catalog,
  type PremiumLevel2RequestStatus,
  type PremiumLevel2StaffDetail,
  type StaffPremiumLevel2Request
} from "@/services/premium_level_2_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./admin2_premium_level2_page.module.css";

const PAGE_SIZE = 20;

const STATUS_OPTIONS: Array<{ value: PremiumLevel2RequestStatus | ""; label: string }> = [
  { value: "", label: "All statuses" },
  { value: "awaiting_intake", label: PREMIUM_LEVEL_2_STATUS_LABELS.awaiting_intake },
  { value: "awaiting_trainer_assignment", label: PREMIUM_LEVEL_2_STATUS_LABELS.awaiting_trainer_assignment },
  { value: "in_progress", label: PREMIUM_LEVEL_2_STATUS_LABELS.in_progress },
  { value: "ready_for_delivery", label: PREMIUM_LEVEL_2_STATUS_LABELS.ready_for_delivery },
  { value: "delivered", label: PREMIUM_LEVEL_2_STATUS_LABELS.delivered },
  { value: "cancelled", label: PREMIUM_LEVEL_2_STATUS_LABELS.cancelled }
];

const statusTone = (status: PremiumLevel2RequestStatus): Admin2BadgeTone => {
  switch (status) {
    case "delivered":
      return "success";
    case "ready_for_delivery":
      return "warning";
    case "cancelled":
      return "danger";
    default:
      return "neutral";
  }
};

const clientName = (request: StaffPremiumLevel2Request): string =>
  [request.client_first_name, request.client_last_name].filter(Boolean).join(" ") || request.client_email;

export default function Admin2PremiumLevel2Page() {
  const [requests, setRequests] = useState<StaffPremiumLevel2Request[]>([]);
  const [pagination, setPagination] = useState<AdminPagination | null>(null);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<PremiumLevel2RequestStatus | "">("awaiting_trainer_assignment");
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [listError, setListError] = useState("");

  const [catalog, setCatalog] = useState<PremiumLevel2Catalog | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<PremiumLevel2StaffDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");

  const [trainers, setTrainers] = useState<AdminTrainer[]>([]);
  const [trainerId, setTrainerId] = useState("");
  const [assigning, setAssigning] = useState(false);
  const [assignError, setAssignError] = useState("");
  const [assignSuccess, setAssignSuccess] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 300);
    return () => clearTimeout(timer);
  }, [search]);

  const loadRequests = useCallback(async () => {
    setLoading(true);
    setListError("");
    try {
      const result = await fetchAdminPremiumLevel2Requests(page, PAGE_SIZE, status, debouncedSearch);
      setRequests(result.requests);
      setPagination(result.pagination);
    } catch {
      setRequests([]);
      setPagination(null);
      setListError("Unable to load Premium Level 2 requests.");
    } finally {
      setLoading(false);
    }
  }, [page, status, debouncedSearch]);

  useEffect(() => {
    void loadRequests();
  }, [loadRequests]);

  useEffect(() => {
    let cancelled = false;
    fetchAdminPremiumLevel2Catalog()
      .then((value) => {
        if (!cancelled) setCatalog(value);
      })
      .catch(() => {
        // Answers still render with humanised keys.
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    fetchAdminTrainers(1, 100)
      .then((result) => {
        if (!cancelled) {
          setTrainers(result.trainers.filter((trainer) => trainer.status === "active"));
        }
      })
      .catch(() => {
        // Allocation stays unavailable if the trainer list cannot be loaded.
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const openRequest = useCallback(async (requestId: string) => {
    setDetail(null);
    setDetailError("");
    setAssignError("");
    setAssignSuccess("");
    setDetailLoading(true);
    try {
      const loaded = await fetchAdminPremiumLevel2Request(requestId);
      setDetail(loaded);
      setTrainerId(loaded.request.trainer_id ?? "");
    } catch {
      setDetailError("Unable to load this request.");
    } finally {
      setDetailLoading(false);
    }
  }, []);

  useEffect(() => {
    if (selectedId) {
      void openRequest(selectedId);
    }
  }, [openRequest, selectedId]);

  const assign = async () => {
    if (!detail || !trainerId) return;
    setAssigning(true);
    setAssignError("");
    setAssignSuccess("");
    try {
      const updated = await assignPremiumLevel2Trainer(detail.request.id, trainerId);
      setDetail((current) => (current ? { ...current, request: updated } : current));
      setAssignSuccess("Trainer allocated. The trainer now sees this request in their workspace.");
      await loadRequests();
    } catch {
      setAssignError("Unable to allocate the trainer. Please try again.");
    } finally {
      setAssigning(false);
    }
  };

  const selected = useMemo(
    () => requests.find((request) => request.id === selectedId) ?? detail?.request ?? null,
    [detail, requests, selectedId]
  );

  return (
    <>
      <Admin2PageHeader
        eyebrow="Programs"
        title="Premium Level 2 requests"
        description="Review submitted intakes and allocate the trainer who will author and deliver the service."
      />

      <div className={styles.layout}>
        <Admin2Section
          title="Requests"
          subtitle={loading ? "Loading…" : `${pagination?.total ?? 0} requests`}
        >
          <div className={styles.filters}>
            <div className={styles.searchBox}>
              <Search size={15} className={styles.searchIcon} aria-hidden="true" />
              <input
                className={styles.searchInput}
                type="search"
                value={search}
                placeholder="Search offer, client name or email…"
                aria-label="Search Premium Level 2 requests"
                onChange={(event) => {
                  setSearch(event.target.value);
                  setPage(1);
                }}
              />
            </div>
            <label className={styles.filterField}>
              <span>Status</span>
              <select
                className={styles.select}
                value={status}
                onChange={(event) => {
                  setStatus(event.target.value as PremiumLevel2RequestStatus | "");
                  setPage(1);
                }}
              >
                {STATUS_OPTIONS.map((option) => (
                  <option key={option.value || "all"} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
          </div>

          {loading ? (
            <p className={styles.muted}>Loading requests…</p>
          ) : listError ? (
            <div className={styles.stateBlock}>
              <p className={styles.error}>{listError}</p>
              <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => void loadRequests()}>
                Try again
              </Button>
            </div>
          ) : requests.length === 0 ? (
            <p className={styles.muted}>No requests match this filter.</p>
          ) : (
            <div className={styles.requestList}>
              {requests.map((request) => (
                <button
                  key={request.id}
                  type="button"
                  className={joinClassNames(
                    styles.requestRow,
                    selectedId === request.id && styles.requestRowActive
                  )}
                  onClick={() => setSelectedId(request.id)}
                >
                  <span className={styles.requestMain}>
                    <strong>{request.offer_name}</strong>
                    <span>
                      {clientName(request)} · {request.client_email}
                    </span>
                  </span>
                  <span className={styles.requestMeta}>
                    <Admin2StatusBadge
                      label={PREMIUM_LEVEL_2_STATUS_LABELS[request.status]}
                      tone={statusTone(request.status)}
                    />
                    <span className={styles.requestDate}>{formatAdminDate(request.updated_at)}</span>
                  </span>
                </button>
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
        </Admin2Section>

        <Admin2Section
          title={selected ? selected.offer_name : "Request detail"}
          subtitle={selected ? clientName(selected) : "Select a request to review the intake"}
        >
          {!selectedId ? (
            <div className={styles.emptyDetail}>
              <Users size={22} aria-hidden="true" />
              <p className={styles.muted}>Choose a request from the list.</p>
            </div>
          ) : detailLoading ? (
            <p className={styles.muted}>Loading request…</p>
          ) : detailError || !detail ? (
            <div className={styles.stateBlock}>
              <p className={styles.error}>{detailError || "Request unavailable."}</p>
              <Button variant="secondary" size="small" onClick={() => void openRequest(selectedId)}>
                Try again
              </Button>
            </div>
          ) : (
            <div className={styles.detail}>
              <div className={styles.metaGrid}>
                <div>
                  <span className={styles.metaLabel}>Client</span>
                  <strong>{clientName(detail.request)}</strong>
                  <span className={styles.muted}>{detail.request.client_email}</span>
                </div>
                <div>
                  <span className={styles.metaLabel}>Status</span>
                  <Admin2StatusBadge
                    label={PREMIUM_LEVEL_2_STATUS_LABELS[detail.request.status]}
                    tone={statusTone(detail.request.status)}
                  />
                </div>
                <div>
                  <span className={styles.metaLabel}>Intake revision</span>
                  <strong>{detail.request.intake_version || "Not submitted"}</strong>
                </div>
                <div>
                  <span className={styles.metaLabel}>Delivered</span>
                  <strong>{detail.request.delivered_at ? formatAdminDate(detail.request.delivered_at) : "Not yet"}</strong>
                </div>
              </div>

              <section className={styles.assignPanel}>
                <div className={styles.assignHeading}>
                  <UserRound size={16} aria-hidden="true" />
                  <h3>Trainer allocation</h3>
                </div>
                <p className={styles.muted}>
                  {detail.request.trainer_name
                    ? `Allocated to ${detail.request.trainer_name}. Reassigning clears the attached programme until the new trainer saves one.`
                    : "Assign an active trainer after the intake is submitted."}
                </p>
                <div className={styles.assignRow}>
                  <select
                    className={styles.select}
                    value={trainerId}
                    disabled={assigning}
                    onChange={(event) => setTrainerId(event.target.value)}
                    aria-label="Select trainer"
                  >
                    <option value="">Choose a trainer</option>
                    {trainers.map((trainer) => (
                      <option key={trainer.id} value={trainer.id}>
                        {[trainer.first_name, trainer.last_name].filter(Boolean).join(" ") || trainer.email}
                      </option>
                    ))}
                  </select>
                  <Button
                    variant="primary"
                    size="small"
                    disabled={!trainerId || assigning || trainerId === detail.request.trainer_id}
                    onClick={() => void assign()}
                  >
                    {assigning ? "Allocating…" : detail.request.trainer_id ? "Reassign trainer" : "Allocate trainer"}
                  </Button>
                </div>
                {assignError ? <p className={styles.error}>{assignError}</p> : null}
                {assignSuccess ? (
                  <p className={styles.success}>
                    <CheckCircle2 size={15} aria-hidden="true" />
                    {assignSuccess}
                  </p>
                ) : null}
              </section>

              <section className={styles.detailSection}>
                <h3>Intake answers</h3>
                <PremiumLevel2IntakeAnswers intake={detail.intake} catalog={catalog} />
              </section>

              <section className={styles.detailSection}>
                <h3>Delivery state</h3>
                <dl className={styles.answerGrid}>
                  <div className={styles.answerRow}>
                    <dt>Training programme</dt>
                    <dd>
                      {detail.request.training_program_id ? (
                        <Link
                          to={`/admin2/plans/${detail.request.training_program_id}?family=generic`}
                          className={styles.link}
                        >
                          {detail.request.training_program_name || "Open programme"}
                        </Link>
                      ) : (
                        "Not attached"
                      )}
                    </dd>
                  </div>
                  <div className={styles.answerRow}>
                    <dt>Nutrition plan</dt>
                    <dd>
                      {detail.nutrition_plan
                        ? `Version ${detail.nutrition_plan.version} · ${detail.nutrition_plan.status}`
                        : "Not authored"}
                    </dd>
                  </div>
                </dl>
              </section>

              <section className={styles.detailSection}>
                <h3>History</h3>
                {detail.events.length === 0 ? (
                  <p className={styles.muted}>No events recorded yet.</p>
                ) : (
                  <ol className={styles.timeline}>
                    {detail.events.map((event, index) => (
                      <li key={`${event.event_type}-${event.created_at}-${index}`}>
                        <span className={styles.timelineType}>{event.event_type.replaceAll("_", " ")}</span>
                        <span className={styles.muted}>
                          {event.actor_kind} · {formatAdminDate(event.created_at)}
                          {event.to_status
                            ? ` · ${PREMIUM_LEVEL_2_STATUS_LABELS[event.to_status as PremiumLevel2RequestStatus] ?? event.to_status}`
                            : ""}
                        </span>
                      </li>
                    ))}
                  </ol>
                )}
              </section>

              {detail.request.status === "ready_for_delivery" ? (
                <p className={styles.warning}>
                  <AlertTriangle size={15} aria-hidden="true" />
                  The trainer must deliver the service from their workspace.
                </p>
              ) : null}
            </div>
          )}
        </Admin2Section>
      </div>
    </>
  );
}
