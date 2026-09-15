import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { AlertTriangle, ArrowLeft, Edit, RefreshCw } from "lucide-react";
import type { FormEvent } from "react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Admin2Modal } from "@/components/admin2/admin2_modal/admin2_modal";
import { Button } from "@/components/button/button";
import {
  AdminProgram,
  fetchAdminProgram,
  formatAdminPrice,
  programTypeLabel,
  ProgramType,
  updateAdminProgramPricing
} from "@/services/admin2_api";
import { formatAdminDate } from "@/services/admin_api";

import styles from "./admin2_plan_detail_page.module.css";

const PricingDialog = ({
  program,
  onClose,
  onSaved
}: {
  program: AdminProgram;
  onClose: () => void;
  onSaved: () => void;
}) => {
  const isFree = program.type === ProgramType.FREE;
  const [price, setPrice] = useState(String(program.price_minor_units / 100));
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting || isFree) {
      return;
    }

    const parsedPrice = Number(price);
    const minorUnits = Math.round(parsedPrice * 100);
    if (Number.isNaN(parsedPrice) || minorUnits < 100) {
      setErrorMessage("Paid programs must have a price of at least €1.00.");
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      await updateAdminProgramPricing(program.id, {
        price_minor_units: minorUnits,
        currency: program.currency
      });
      onSaved();
      onClose();
    } catch (error) {
      setErrorMessage(
        error instanceof Error ? error.message : "Unable to update the price. Please try again."
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Admin2Modal
      title="Edit pricing"
      description={`Update the marketplace price for ${program.name}.`}
      onClose={onClose}
    >
      <form onSubmit={handleSubmit} noValidate>
        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="detail-price">
            Price
          </label>
          <input
            id="detail-price"
            className={styles.formInput}
            type="number"
            min="0"
            step="0.01"
            value={isFree ? "0.00" : price}
            disabled={isFree}
            onChange={(event) => setPrice(event.target.value)}
            aria-label="Price in euros"
          />
        </div>

        {isFree ? (
          <p className={styles.formError} style={{ marginTop: "0.5rem" }}>
            Free programs are always €0.00.
          </p>
        ) : null}

        {errorMessage ? (
          <p className={styles.formError} role="alert" style={{ marginTop: "0.5rem" }}>
            {errorMessage}
          </p>
        ) : null}

        <div className={styles.modalActions}>
          <Button type="button" variant="ghost" size="small" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" size="small" disabled={submitting || isFree}>
            {submitting ? "Saving…" : "Save price"}
          </Button>
        </div>
      </form>
    </Admin2Modal>
  );
};

export default function Admin2PlanDetailPage() {
  const { programId } = useParams<{ programId: string }>();
  const [program, setProgram] = useState<AdminProgram | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [showPricing, setShowPricing] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");

    try {
      const result = await fetchAdminProgram(programId);
      setProgram(result);
    } catch {
      setErrorMessage("Unable to load this program. Please try again.");
    } finally {
      setLoading(false);
    }
  }, [programId]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <>
      <Admin2PageHeader
        eyebrow="Programs"
        title={program?.name ?? "Plan details"}
        description={program?.description ?? undefined}
        actions={
          <Button
            to="/admin2/plans"
            variant="ghost"
            size="small"
            icon={<ArrowLeft size={15} />}
            iconPosition="left"
          >
            Back to plans
          </Button>
        }
      />

      {loading ? (
        <div className={styles.pageState}>
          <p className={styles.tableMuted}>Loading plan…</p>
        </div>
      ) : errorMessage || !program ? (
        <div className={styles.pageState}>
          <p className={styles.formError}>{errorMessage}</p>
          <div style={{ marginTop: "0.75rem" }}>
            <Button variant="secondary" size="small" onClick={() => void load()} icon={<RefreshCw size={15} />}>
              Retry
            </Button>
          </div>
        </div>
      ) : (
        <div className={styles.split}>
<Admin2Section title="Program structure" subtitle="Where the plan content lives">
              <div className={styles.notice}>
                <AlertTriangle size={15} className={styles.noticeIcon} aria-hidden="true" />
                <span>
                  <strong>Not exposed yet.</strong> The admin program endpoint returns program
                  metadata only; weeks, workouts and exercises are not part of the response until the
                  structure API ships.
                </span>
              </div>
            </Admin2Section>

          <Admin2Section title="Summary" subtitle="Program facts">
            <div className={styles.detailList}>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Type</p>
                <p className={styles.rowValue}>{programTypeLabel(program.type)}</p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Status</p>
                <div className={styles.rowValue}>
                  <Admin2StatusBadge label={program.status === "published" ? "Published" : "Draft"} tone={program.status === "published" ? "success" : "warning"} />
                </div>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Price</p>
                <p className={styles.rowValue}>{formatAdminPrice(program.price_minor_units, program.currency)}</p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Currency</p>
                <p className={styles.rowValue}>{program.currency}</p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Created</p>
                <p className={styles.rowValue}>{formatAdminDate(program.created_at)}</p>
              </div>
              <div className={styles.row}>
                <p className={styles.rowLabel}>Last updated</p>
                <p className={styles.rowValue}>{formatAdminDate(program.updated_at)}</p>
              </div>
            </div>

            <div className={styles.actions}>
              <Button
                variant="secondary"
                size="small"
                icon={<Edit size={14} />}
                onClick={() => setShowPricing(true)}
                disabled={program.type === ProgramType.FREE}
              >
                Edit pricing
              </Button>
              <Link className={styles.formError} to="/admin2/plans" style={{ textDecoration: "none" }}>
                Back to the plan list
              </Link>
            </div>
          </Admin2Section>
        </div>
      )}

      {program && showPricing ? (
        <PricingDialog
          program={program}
          onClose={() => setShowPricing(false)}
          onSaved={() => void load()}
        />
      ) : null}
    </>
  );
}