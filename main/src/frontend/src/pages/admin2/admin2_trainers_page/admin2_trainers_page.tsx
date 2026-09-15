import { useCallback, useEffect, useState } from "react";
import type { FormEvent } from "react";
import { RefreshCw, Settings2 } from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Admin2Modal } from "@/components/admin2/admin2_modal/admin2_modal";
import { Admin2Table } from "@/components/admin2/admin2_table/admin2_table";
import { Admin2Pagination } from "@/components/admin2/admin2_pagination/admin2_pagination";
import { Button } from "@/components/button/button";
import { fullName, fetchAdminTrainers, type AdminTrainer } from "@/services/admin_api";
import {
  deleteCommissionRule,
  fetchCommissionResolution,
  formatCommissionBPS,
  updateCommissionRule
} from "@/services/admin2_api";

import styles from "./admin2_trainers_page.module.css";

const TrainerCommissionDialog = ({
  trainer,
  onClose,
  onSaved
}: {
  trainer: AdminTrainer;
  onClose: () => void;
  onSaved: () => void;
}) => {
  const [resolvedBps, setResolvedBps] = useState<number | null>(null);
  const [isOverride, setIsOverride] = useState(false);
  const [bps, setBps] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const loadResolution = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");
    try {
      const resolution = await fetchCommissionResolution(trainer.id);
      setResolvedBps(resolution.commission_bps);
      setIsOverride(resolution.is_override);
      setBps(String(resolution.commission_bps));
    } catch {
      setErrorMessage("Unable to load the commission settings.");
    } finally {
      setLoading(false);
    }
  }, [trainer.id]);

  useEffect(() => {
    void loadResolution();
  }, [loadResolution]);

  const handleSave = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }

    const parsed = Number(bps);
    if (!Number.isInteger(parsed) || parsed < 0 || parsed > 10000) {
      setErrorMessage("Commission must be an integer between 0 and 10000 bps (0% to 100%).");
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      await updateCommissionRule(trainer.id, parsed);
      await loadResolution();
      onSaved();
    } catch (error) {
      setErrorMessage(
        error instanceof Error ? error.message : "Unable to save the commission settings."
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleRemoveOverride = async () => {
    if (submitting || !isOverride) {
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      await deleteCommissionRule(trainer.id);
      await loadResolution();
      onSaved();
    } catch (error) {
      setErrorMessage(
        error instanceof Error ? error.message : "Unable to remove the commission override."
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Admin2Modal
      title="Commission settings"
      description={`Configure the platform commission for ${fullName(trainer)}.`}
      onClose={onClose}
    >
      {loading ? (
        <p className={styles.muted}>Loading commission…</p>
      ) : (
        <>
          <div className={styles.resolutionBlock}>
            <p className={styles.resolutionValue}>
              {resolvedBps === null ? "—" : formatCommissionBPS(resolvedBps)}
            </p>
            <p className={styles.resolutionNote}>
              Effective platform commission on paid program sales.{" "}
              {isOverride
                ? "A trainer-specific override is applied."
                : "The global default applies — no trainer-specific override."}
            </p>
          </div>

          <form onSubmit={handleSave} noValidate>
            <div className={styles.formRow}>
              <label className={styles.formLabel} htmlFor="commission-bps">
                Trainer-specific commission (bps)
              </label>
              <input
                id="commission-bps"
                className={styles.formInput}
                type="number"
                min="0"
                max="10000"
                step="1"
                value={bps}
                onChange={(event) => setBps(event.target.value)}
                aria-label="Commission in basis points"
              />
            </div>
            <p className={styles.formHint} style={{ marginTop: "0.35rem" }}>
              1 bps = 0.01%. A value of 2000 means a 20% platform commission.
            </p>

            {errorMessage ? (
              <p className={styles.formError} role="alert" style={{ marginTop: "0.5rem" }}>
                {errorMessage}
              </p>
            ) : null}

            <div className={styles.modalActions}>
              {isOverride ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="small"
                  onClick={handleRemoveOverride}
                  disabled={submitting}
                >
                  Remove override
                </Button>
              ) : null}
              <Button type="button" variant="ghost" size="small" onClick={onClose}>
                Cancel
              </Button>
              <Button type="submit" size="small" disabled={submitting}>
                {submitting ? "Saving…" : "Save commission"}
              </Button>
            </div>
          </form>
        </>
      )}
    </Admin2Modal>
  );
};

export default function Admin2TrainersPage() {
  const [trainers, setTrainers] = useState<AdminTrainer[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [commissionTrainer, setCommissionTrainer] = useState<AdminTrainer | null>(null);

  const load = useCallback(
    async (targetPage: number) => {
      setLoading(true);
      setErrorMessage("");
      try {
        const result = await fetchAdminTrainers(targetPage, 20);
        setTrainers(result.trainers);
        setTotal(result.pagination.total);
        setTotalPages(result.pagination.total_pages);
        setPage(result.pagination.page);
      } catch {
        setErrorMessage("Unable to load trainers. Please try again.");
      } finally {
        setLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    void load(1);
  }, [load]);

  return (
    <>
      <Admin2PageHeader
        eyebrow="People"
        title="Trainers"
        description="The trainer network and each partner's commercial configuration."
      />

      <Admin2Section title="Trainer network" subtitle={`${total} trainers in the platform`}>
        {loading ? (
          <div className={styles.pageState}>
            <p className={styles.muted}>Loading trainers…</p>
          </div>
        ) : errorMessage ? (
          <div className={styles.pageState}>
            <p className={styles.muted}>{errorMessage}</p>
            <div style={{ marginTop: "0.75rem" }}>
              <Button variant="secondary" size="small" onClick={() => void load(page)} icon={<RefreshCw size={15} />}>
                Retry
              </Button>
            </div>
          </div>
        ) : (
          <Admin2Table
            columns={[
              {
                key: "name",
                label: "Trainer",
                render: (trainer: AdminTrainer) => {
                  const name = fullName(trainer);
                  return name !== "—" ? name : <span className={styles.muted}>—</span>;
                }
              },
              {
                key: "email",
                label: "Email",
                className: styles.muted,
                render: (trainer: AdminTrainer) => trainer.email
              },
              {
                key: "status",
                label: "Status",
                render: (trainer: AdminTrainer) => (
                  <Admin2StatusBadge
                    label={trainer.status === "active" ? "Active" : "Disabled"}
                    tone={trainer.status === "active" ? "success" : "danger"}
                  />
                )
              },
              {
                key: "commission",
                label: "",
                className: styles.muted,
                render: (trainer: AdminTrainer) => (
                  <Button
                    variant="ghost"
                    size="small"
                    icon={<Settings2 size={14} />}
                    onClick={() => setCommissionTrainer(trainer)}
                  >
                    Commission
                  </Button>
                )
              }
            ]}
            rows={trainers}
            rowKey={(trainer) => trainer.id}
            emptyRow={<span className={styles.muted}>No trainers registered yet.</span>}
          />
        )}

        {!loading && !errorMessage ? (
          <Admin2Pagination
            page={page}
            totalPages={totalPages}
            pageSize={20}
            totalItems={total}
            onChange={(nextPage) => void load(nextPage)}
          />
        ) : null}
      </Admin2Section>

      {commissionTrainer ? (
        <TrainerCommissionDialog
          trainer={commissionTrainer}
          onClose={() => setCommissionTrainer(null)}
          onSaved={() => undefined}
        />
      ) : null}
    </>
  );
}