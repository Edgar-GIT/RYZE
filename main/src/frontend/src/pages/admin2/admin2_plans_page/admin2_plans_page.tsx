import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";
import { ArrowRight, Edit, Plus, RefreshCw, Search, Trash2 } from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Admin2Modal } from "@/components/admin2/admin2_modal/admin2_modal";
import { Button } from "@/components/button/button";
import { joinClassNames } from "@utils/class_names";
import {
  deleteGenericProgram,
  fetchAdminSalesByProgramIds,
  fetchGenericPrograms,
  formatAdminPrice,
  isProgramProductType,
  PROGRAM_PRODUCT_TYPE,
  programFamilyLabel,
  programTypeLabel,
  ProgramStatusEnum,
  ProgramType,
  type AdminProgramSale,
  type GenericProgramLevel,
  type GenericProgramSummary,
  type ProgramProductTypeValue,
  type ProgramTypeValue
} from "@/services/admin2_api";
import { formatAdminDate, type AdminPagination } from "@/services/admin_api";

import styles from "./admin2_plans_page.module.css";

const PAGE_SIZE = 25;

type PlanFamilyTab = ProgramProductTypeValue;

// These tabs filter `product_type`, the product family dimension. Each family is
// administered through its own backend route (scoped to `trainer_id IS NULL`
// plus one family), so the list always shows exactly one family at a time and
// never a merged view. They are deliberately not labelled as pricing categories:
// `type` (free/premium/personalized) is the commercial category and lives in the
// per-row table plus the pricing-category dropdown filter.
const PLANS_FAMILY_TABS: Array<{ id: PlanFamilyTab; label: string }> = [
  { id: PROGRAM_PRODUCT_TYPE.GENERIC, label: programFamilyLabel(PROGRAM_PRODUCT_TYPE.GENERIC) },
  {
    id: PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1,
    label: programFamilyLabel(PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1)
  }
];

const TYPE_FILTERS: Array<{ id: ProgramTypeValue | ""; label: string }> = [
  { id: "", label: "All pricing categories" },
  { id: ProgramType.FREE, label: "Free" },
  { id: ProgramType.PREMIUM, label: "Premium" },
  { id: ProgramType.PERSONALIZED, label: "Personalized" }
];

const LEVEL_FILTERS: Array<{ id: GenericProgramLevel | ""; label: string }> = [
  { id: "", label: "All levels" },
  { id: "Beginner", label: "Beginner" },
  { id: "Intermediate", label: "Intermediate" },
  { id: "Advanced", label: "Advanced" }
];

// familyFromQuery reads the family from the URL. The family is never inventable
// client-side beyond the closed PRODUCT_TYPE list: an unknown value (or no value
// at all) falls back to the generic family.
const familyFromQuery = (search: string): PlanFamilyTab => {
  const family = new URLSearchParams(search).get("family");
  return isProgramProductType(family) ? family : PROGRAM_PRODUCT_TYPE.GENERIC;
};

const familyPhrase = (family: PlanFamilyTab): string =>
  family === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1 ? "premium level 1" : "generic";

const levelLabel = (level: GenericProgramLevel | null): string => level ?? "—";

const frequencyLabel = (frequency: number | null): string =>
  frequency === null ? "—" : `${frequency} day${frequency === 1 ? "" : "s"}/week`;

const durationLabel = (duration: number | null): string =>
  duration === null ? "—" : `${duration} week${duration === 1 ? "" : "s"}`;

const DeleteDialog = ({
  program,
  onClose,
  onConfirm,
  submitting,
  errorMessage
}: {
  program: GenericProgramSummary;
  onClose: () => void;
  onConfirm: () => void;
  submitting: boolean;
  errorMessage: string;
}) => (
  <Admin2Modal
    title="Delete plan"
    description={`Remove "${program.name}" from the RYZE catalogue.`}
    onClose={onClose}
  >
    <p className={styles.formHint}>
      The plan is soft-deleted: it disappears from the catalogue and the admin list, but its data
      is preserved and can be restored by the platform.
    </p>

    {errorMessage ? (
      <p className={styles.formError} role="alert" style={{ marginTop: "0.5rem" }}>
        {errorMessage}
      </p>
    ) : null}

    <div className={styles.modalActions}>
      <Button type="button" variant="ghost" size="small" onClick={onClose}>
        Cancel
      </Button>
      <Button type="button" variant="danger" size="small" onClick={onConfirm} disabled={submitting}>
        {submitting ? "Deleting…" : "Delete plan"}
      </Button>
    </div>
  </Admin2Modal>
);

export default function Admin2PlansPage() {
  const history = useHistory();
  const location = useLocation();
  const activeFamily = familyFromQuery(location.search);

  const [programs, setPrograms] = useState<GenericProgramSummary[]>([]);
  const [pagination, setPagination] = useState<AdminPagination | null>(null);
  const [salesByProgram, setSalesByProgram] = useState<Record<string, AdminProgramSale>>({});
  const [salesError, setSalesError] = useState(false);
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState<ProgramTypeValue | "">("");
  const [level, setLevel] = useState<GenericProgramLevel | "">("");
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<GenericProgramSummary | null>(null);
  const [deleteError, setDeleteError] = useState("");
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query), 300);
    return () => clearTimeout(timer);
  }, [query]);

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");

    try {
      const result = await fetchGenericPrograms(activeFamily, {
        page,
        limit: PAGE_SIZE,
        search: debouncedQuery.trim() || undefined,
        type: typeFilter || undefined,
        level: level || undefined
      });
      setPrograms(result.programs);
      setPagination(result.pagination);

      if (result.programs.length === 0) {
        setSalesByProgram({});
        setSalesError(false);
      } else {
        try {
          const sales = await fetchAdminSalesByProgramIds(result.programs.map((program) => program.id));
          setSalesByProgram(
            Object.fromEntries(sales.map((sale) => [sale.program.id, sale]))
          );
          setSalesError(false);
        } catch {
          setSalesError(true);
        }
      }
    } catch {
      setErrorMessage("Unable to load plans. Please try again.");
    } finally {
      setLoading(false);
    }
  }, [page, debouncedQuery, activeFamily, typeFilter, level]);

  useEffect(() => {
    void load();
  }, [load]);

  const changeFamily = (family: PlanFamilyTab) => {
    setPage(1);
    history.push({ pathname: "/admin2/plans", search: `?family=${family}` });
  };

  const changeTypeFilter = (value: ProgramTypeValue | "") => {
    setTypeFilter(value);
    setPage(1);
  };

  const changeLevel = (value: GenericProgramLevel | "") => {
    setLevel(value);
    setPage(1);
  };

  const confirmDelete = async () => {
    if (!deleteTarget) {
      return;
    }
    setDeleting(true);
    setDeleteError("");

    try {
      await deleteGenericProgram(deleteTarget.product_type, deleteTarget.id);
      setDeleteTarget(null);
      await load();
    } catch (error) {
      setDeleteError(
        error instanceof Error ? error.message : "Unable to delete the plan. Please try again."
      );
    } finally {
      setDeleting(false);
    }
  };

  const isFiltered =
    debouncedQuery.trim().length > 0 || typeFilter !== "" || level !== "";

  const emptyCopy = useMemo(() => {
    if (isFiltered) {
      return {
        title: "No plans match your filters",
        message: "Try a different search term or clear the filters to see every plan."
      };
    }
    return {
      title: `No ${familyPhrase(activeFamily)} plans yet`,
      message: `Create your first ${familyPhrase(activeFamily)} plan from the plan creation workspace.`
    };
  }, [activeFamily, isFiltered]);

  return (
    <>
      <Admin2PageHeader
        eyebrow="Programs"
        title="Plans"
        description={`Every platform-owned ${familyPhrase(activeFamily)} program. Search, open, edit or remove catalogue plans.`}
        actions={
          <Button to={`/admin2/plans/create?family=${activeFamily}`} size="small" icon={<Plus size={15} />}>
            New plan
          </Button>
        }
      />

      <div className={styles.toolbar}>
        <div className={styles.tabs} role="tablist" aria-label="Filter plans by product family">
          {PLANS_FAMILY_TABS.map((tab) => (
            <button
              key={tab.id}
              type="button"
              role="tab"
              aria-selected={activeFamily === tab.id}
              className={joinClassNames(styles.tab, activeFamily === tab.id && styles.tabActive)}
              onClick={() => changeFamily(tab.id)}
            >
              {tab.label}
            </button>
          ))}
        </div>

        <div className={styles.filters}>
          <div className={styles.searchBox}>
            <Search size={15} className={styles.searchIcon} aria-hidden="true" />
            <input
              className={styles.searchInput}
              type="search"
              value={query}
              placeholder="Search plans…"
              aria-label="Search plans by name or description"
              onChange={(event) => {
                setQuery(event.target.value);
                setPage(1);
              }}
            />
          </div>
          <label className={styles.levelField}>
            <span className={styles.levelFieldLabel}>Category</span>
            <select
              className={styles.levelSelect}
              value={typeFilter}
              aria-label="Filter plans by pricing category"
              onChange={(event) => changeTypeFilter(event.target.value as ProgramTypeValue | "")}
            >
              {TYPE_FILTERS.map((option) => (
                <option key={option.id || "all-types"} value={option.id}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
          <label className={styles.levelField}>
            <span className={styles.levelFieldLabel}>Level</span>
            <select
              className={styles.levelSelect}
              value={level}
              aria-label="Filter plans by level"
              onChange={(event) => changeLevel(event.target.value as GenericProgramLevel | "")}
            >
              {LEVEL_FILTERS.map((option) => (
                <option key={option.id || "all"} value={option.id}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
        </div>
      </div>

      <Admin2Section
        title={programFamilyLabel(activeFamily)}
        subtitle={
          loading
            ? "Loading…"
            : errorMessage
              ? ""
              : `${pagination?.total ?? 0} program${pagination && pagination.total === 1 ? "" : "s"}${
                  salesError ? " · Sales unavailable" : ""
                }`
        }
      >
        {loading ? (
          <div className={styles.pageState}>
            <p className={styles.tableMuted}>Loading plans…</p>
          </div>
        ) : errorMessage ? (
          <div className={styles.pageState}>
            <p className={styles.formError}>{errorMessage}</p>
            <div style={{ marginTop: "0.75rem" }}>
              <Button
                variant="secondary"
                size="small"
                onClick={() => void load()}
                icon={<RefreshCw size={15} />}
              >
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
                to={`/admin2/plans/create?family=${activeFamily}`}
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
                    <th>Level</th>
                    <th>Frequency</th>
                    <th>Duration</th>
                    <th className={styles.tableNumeric}>Price</th>
                    <th className={styles.tableNumeric}>Sales</th>
                    <th className={styles.tableNumeric}>Test</th>
                    <th className={styles.tableNumeric}>Revenue</th>
                    <th>Status</th>
                    <th>Updated</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {programs.map((program) => {
                    const sale = salesByProgram[program.id];
                    return (
                      <tr key={program.id}>
                        <td>
                          <Link to={`/admin2/plans/${program.id}?family=${program.product_type}`}>
                            {program.name}
                          </Link>
                        </td>
                        <td>
                          <Admin2StatusBadge
                            label={programTypeLabel(program.type)}
                            tone="success"
                            withDot={false}
                          />
                        </td>
                        <td className={styles.tableMuted}>{levelLabel(program.level)}</td>
                        <td className={styles.tableMuted}>
                          {frequencyLabel(program.frequency_per_week)}
                        </td>
                        <td className={styles.tableMuted}>{durationLabel(program.duration_weeks)}</td>
                        <td className={styles.tableNumeric}>
                          {formatAdminPrice(program.price_minor_units, program.currency)}
                        </td>
                        <td className={styles.tableNumeric}>{sale ? sale.real_sales : "—"}</td>
                        <td className={styles.tableNumeric}>{sale ? sale.test_purchases : "—"}</td>
                        <td className={styles.tableNumeric}>
                          {sale ? formatAdminPrice(sale.revenue_minor_units, program.currency) : "—"}
                        </td>
                        <td>
                          <Admin2StatusBadge
                            label={program.status === ProgramStatusEnum.PUBLISHED ? "Published" : "Draft"}
                            tone={program.status === ProgramStatusEnum.PUBLISHED ? "success" : "warning"}
                          />
                        </td>
                        <td className={styles.tableMuted}>{formatAdminDate(program.updated_at)}</td>
                        <td className={styles.tableAction}>
                          <Button
                            to={`/admin2/plans/create?edit=${program.id}&family=${program.product_type}`}
                            variant="ghost"
                            size="small"
                            icon={<Edit size={14} />}
                          >
                            Edit
                          </Button>
                          <Button
                            variant="ghost"
                            size="small"
                            icon={<Trash2 size={14} />}
                            onClick={() => {
                              setDeleteError("");
                              setDeleteTarget(program);
                            }}
                          >
                            Delete
                          </Button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {pagination && pagination.total_pages > 1 ? (
              <div className={styles.footer}>
                <p className={styles.footerInfo}>
                  Page {pagination.page} of {pagination.total_pages}
                </p>
                <div className={styles.pager}>
                  <Button
                    variant="secondary"
                    size="small"
                    disabled={pagination.page <= 1}
                    onClick={() => setPage((current) => Math.max(1, current - 1))}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="secondary"
                    size="small"
                    disabled={pagination.page >= pagination.total_pages}
                    onClick={() => setPage((current) => current + 1)}
                  >
                    Next
                  </Button>
                </div>
              </div>
            ) : null}
          </>
        )}
      </Admin2Section>

      {deleteTarget ? (
        <DeleteDialog
          program={deleteTarget}
          onClose={() => setDeleteTarget(null)}
          onConfirm={() => void confirmDelete()}
          submitting={deleting}
          errorMessage={deleteError}
        />
      ) : null}
    </>
  );
}
