import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";
import { AlertTriangle, ArrowRight, Edit, Plus, RefreshCw } from "lucide-react";
import type { FormEvent } from "react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Admin2Modal } from "@/components/admin2/admin2_modal/admin2_modal";
import { Button } from "@/components/button/button";
import { joinClassNames } from "@utils/class_names";
import {
  AdminProgram,
  fetchAdminPrograms,
  formatAdminPrice,
  programTypeLabel,
  ProgramType,
  type ProgramTypeValue,
  updateAdminProgramPricing
} from "@/services/admin2_api";
import { formatAdminDate } from "@/services/admin_api";

import styles from "./admin2_plans_page.module.css";

type PlansTab = "ALL" | ProgramTypeValue;

const PLANS_TABS: Array<{ id: PlansTab; label: string }> = [
  { id: "ALL", label: "All programs" },
  { id: ProgramType.FREE, label: "Generic" },
  { id: ProgramType.PREMIUM, label: "Premium · L1" },
  { id: ProgramType.PERSONALIZED, label: "Premium · L2" }
];

const tabFromQuery = (search: string): PlansTab => {
  const type = new URLSearchParams(search).get("type");
  if (type === ProgramType.FREE || type === ProgramType.PREMIUM || type === ProgramType.PERSONALIZED) {
    return type;
  }
  return "ALL";
};

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
    if (Number.isNaN(parsedPrice) || parsedPrice <= 0) {
      setErrorMessage("Paid programs must have a price of at least €1.00.");
      return;
    }

    const minorUnits = Math.round(parsedPrice * 100);
    if (minorUnits < 100) {
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
      const message =
        error instanceof Error ? error.message : "Unable to update the price. Please try again.";
      setErrorMessage(message);
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
          <label className={styles.formLabel} htmlFor="plan-price">
            Price
          </label>
          <input
            id="plan-price"
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

        <div className={styles.formRow}>
          <label className={styles.formLabel} htmlFor="plan-currency">
            Currency
          </label>
          <input
            id="plan-currency"
            className={styles.formInput}
            type="text"
            value={program.currency}
            disabled
            aria-label="Currency"
          />
        </div>

        {isFree ? (
          <p className={styles.formError} style={{ marginTop: "0.5rem" }}>
            Free programs are always €0.00.
          </p>
        ) : (
          <p className={styles.formError} style={{ marginTop: "0.5rem" }}>
            Paid programs use a minimum price of €1.00.
          </p>
        )}

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

export default function Admin2PlansPage() {
  const history = useHistory();
  const location = useLocation();
  const activeTab = tabFromQuery(location.search);

  const [programs, setPrograms] = useState<AdminProgram[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [editingProgram, setEditingProgram] = useState<AdminProgram | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");

    try {
      const result = await fetchAdminPrograms(1, 100, activeTab === "ALL" ? undefined : activeTab);
      setPrograms(result.programs);
      setTotal(result.pagination.total);
    } catch {
      setErrorMessage("Unable to load plans. Please try again.");
    } finally {
      setLoading(false);
    }
  }, [activeTab]);

  useEffect(() => {
    void load();
  }, [load]);

  const changeTab = (tab: PlansTab) => {
    const search = tab === "ALL" ? "" : `?type=${tab}`;
    history.push({ pathname: "/admin2/plans", search });
  };

  const emptyCopy = useMemo(() => {
    if (activeTab === "ALL") {
      return {
        title: "No published programs yet",
        message: "Programs appear here once they are published through the plan creation workspace."
      };
    }
    return {
      title: `No ${programTypeLabel(activeTab as ProgramTypeValue).toLowerCase()} programs`,
      message: "Publish a plan of this type to make it appear in this filter."
    };
  }, [activeTab]);

  return (
    <>
      <Admin2PageHeader
        eyebrow="Programs"
        title="Plans"
        description="Every published program across the RYZE product lines. Pricing is managed from here."
        actions={
          <Button to="/admin2/plans/create" size="small" icon={<Plus size={15} />}>
            New plan
          </Button>
        }
      />

      <div className={styles.toolbar}>
        <div className={styles.tabs} role="tablist" aria-label="Filter plans by type">
          {PLANS_TABS.map((tab) => (
            <button
              key={tab.id}
              type="button"
              role="tab"
              aria-selected={activeTab === tab.id}
              className={joinClassNames(styles.tab, activeTab === tab.id && styles.tabActive)}
              onClick={() => changeTab(tab.id)}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div className={styles.notice}>
        <AlertTriangle size={15} className={styles.noticeIcon} aria-hidden="true" />
        <span>
          <strong>Published programs only.</strong> Draft programs are created by trainers and are
          not exposed through the admin API yet.
        </span>
      </div>

      <Admin2Section
        title={activeTab === "ALL" ? "All plans" : programTypeLabel(activeTab as ProgramTypeValue)}
        subtitle={`${total} published program${total === 1 ? "" : "s"}`}
      >
        {loading ? (
          <div className={styles.pageState}>
            <p className={styles.tableMuted}>Loading plans…</p>
          </div>
        ) : errorMessage ? (
          <div className={styles.pageState}>
            <p className={styles.formError}>{errorMessage}</p>
            <div style={{ marginTop: "0.75rem" }}>
              <Button variant="secondary" size="small" onClick={() => void load()} icon={<RefreshCw size={15} />}>
                Retry
              </Button>
            </div>
          </div>
        ) : programs.length === 0 ? (
          <div className={styles.emptyRow}>
            <p className={styles.emptyTitle}>{emptyCopy.title}</p>
            <p className={styles.emptyMessage}>{emptyCopy.message}</p>
            <div className={styles.emptyAction}>
              <Button
                to="/admin2/plans/create"
                variant="secondary"
                size="small"
                icon={<ArrowRight size={15} />}
              >
                Open the plan workspace
              </Button>
            </div>
          </div>
        ) : (
          <>
            <div className={styles.tableWrap}>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <th>Program</th>
                    <th>Type</th>
                    <th>Price</th>
                    <th>Status</th>
                    <th>Published</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {programs.map((program) => (
                    <tr key={program.id}>
                      <td>
                        <Link to={`/admin2/plans/${program.id}`}>{program.name}</Link>
                      </td>
                      <td>
                        <Admin2StatusBadge
                          label={programTypeLabel(program.type)}
                          tone="success"
                          withDot={false}
                        />
                      </td>
                      <td>
                        <span className={styles.priceCell}>
                          {formatAdminPrice(program.price_minor_units, program.currency)}
                          {program.type === ProgramType.FREE ? (
                            <span className={styles.priceNote}>(free)</span>
                          ) : null}
                        </span>
                      </td>
                      <td>
                        <Admin2StatusBadge label="Published" tone="success" />
                      </td>
                      <td className={styles.tableMuted}>{formatAdminDate(program.created_at)}</td>
                      <td className={styles.tableAction}>
                        <Button
                          variant="ghost"
                          size="small"
                          icon={<Edit size={14} />}
                          onClick={() => setEditingProgram(program)}
                        >
                          Pricing
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className={styles.footer}>
              <p className={styles.footerInfo}>Showing up to 100 programs per load.</p>
            </div>
          </>
        )}
      </Admin2Section>

      {editingProgram ? (
        <PricingDialog
          program={editingProgram}
          onClose={() => setEditingProgram(null)}
          onSaved={() => void load()}
        />
      ) : null}
    </>
  );
}