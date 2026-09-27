import { useCallback, useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import {
  Banknote,
  CheckCircle2,
  Clock,
  Eye,
  FlaskConical,
  Layers,
  RefreshCw,
  ShoppingBag,
  XCircle
} from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2MetricCard } from "@/components/admin2/admin2_metric_card/admin2_metric_card";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2Table, type Admin2Column } from "@/components/admin2/admin2_table/admin2_table";
import { Admin2Pagination } from "@/components/admin2/admin2_pagination/admin2_pagination";
import { Admin2EmptyState } from "@/components/admin2/admin2_empty_state/admin2_empty_state";
import {
  Admin2StatusBadge,
  type Admin2BadgeTone
} from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Admin2Modal } from "@/components/admin2/admin2_modal/admin2_modal";
import { Button } from "@/components/button/button";
import {
  fetchAdminProgramSales,
  fetchAdminPurchases,
  fetchAdminSalesByProgramIds,
  formatAdminPrice,
  programTypeLabel,
  programTypeShortLabel,
  PurchaseStatusEnum,
  type AdminProgramSale,
  type AdminPurchase,
  type AdminSalesSummary,
  type PurchaseStatus
} from "@/services/admin2_api";
import { formatAdminDate, type AdminPagination } from "@/services/admin_api";

import styles from "./admin2_sales_page.module.css";

const PAGE_SIZE = 20;
// The program filter draws its options from the per-program sales endpoint.
// Capped so a large catalogue stays responsive.
const PROGRAM_OPTIONS_LIMIT = 100;

const STATUS_OPTIONS: Array<{ id: PurchaseStatus | ""; label: string }> = [
  { id: "", label: "All statuses" },
  { id: PurchaseStatusEnum.PENDING, label: "Pending" },
  { id: PurchaseStatusEnum.COMPLETED, label: "Completed" },
  { id: PurchaseStatusEnum.FAILED, label: "Failed" }
];

const TEST_TYPE_OPTIONS: Array<{ id: "all" | "real" | "test"; label: string }> = [
  { id: "all", label: "Real & test" },
  { id: "real", label: "Real only" },
  { id: "test", label: "Test only" }
];

const statusMeta = (status: PurchaseStatus): { label: string; tone: Admin2BadgeTone } => {
  if (status === PurchaseStatusEnum.COMPLETED) {
    return { label: "Completed", tone: "success" };
  }
  if (status === PurchaseStatusEnum.PENDING) {
    return { label: "Pending", tone: "warning" };
  }
  return { label: "Failed", tone: "danger" };
};

export default function Admin2SalesPage() {
  const location = useLocation();
  const [programId, setProgramId] = useState(
    () => new URLSearchParams(location.search).get("program_id") ?? ""
  );

  const [purchases, setPurchases] = useState<AdminPurchase[]>([]);
  const [summary, setSummary] = useState<AdminSalesSummary | null>(null);
  const [pagination, setPagination] = useState<AdminPagination | null>(null);
  const [page, setPage] = useState(1);

  const [programOptions, setProgramOptions] = useState<AdminProgramSale[]>([]);
  const [status, setStatus] = useState<PurchaseStatus | "">("");
  const [testMode, setTestMode] = useState<"all" | "real" | "test">("all");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  const [programSales, setProgramSales] = useState<AdminProgramSale[]>([]);
  const [programSalesPagination, setProgramSalesPagination] =
    useState<AdminPagination | null>(null);
  const [programSalesPage, setProgramSalesPage] = useState(1);

  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [programSalesLoading, setProgramSalesLoading] = useState(true);
  const [selected, setSelected] = useState<AdminPurchase | null>(null);

  const testFilter = testMode === "all" ? undefined : testMode === "real" ? false : true;
  const isFiltered = programId !== "" || status !== "" || testMode !== "all" || from !== "" || to !== "";

  const loadPurchases = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");

    try {
      const result = await fetchAdminPurchases({
        page,
        limit: PAGE_SIZE,
        program_id: programId || undefined,
        status,
        test: testFilter,
        from: from || undefined,
        to: to || undefined
      });
      setPurchases(result.purchases);
      setSummary(result.summary);
      setPagination(result.pagination);
    } catch {
      setErrorMessage("Unable to load sales data. Please try again.");
    } finally {
      setLoading(false);
    }
  }, [page, programId, status, testFilter, from, to]);

  const loadProgramSales = useCallback(async () => {
    setProgramSalesLoading(true);

    try {
      const result = await fetchAdminProgramSales({
        page: programSalesPage,
        limit: PAGE_SIZE,
        program_id: programId || undefined,
        status,
        test: testFilter,
        from: from || undefined,
        to: to || undefined
      });
      setProgramSales(result.sales);
      setProgramSalesPagination(result.pagination);
    } catch {
      setProgramSales([]);
      setProgramSalesPagination(null);
    } finally {
      setProgramSalesLoading(false);
    }
  }, [programSalesPage, programId, status, testFilter, from, to]);

  useEffect(() => {
    void loadPurchases();
  }, [loadPurchases]);

  useEffect(() => {
    void loadProgramSales();
  }, [loadProgramSales]);

  useEffect(() => {
    let cancelled = false;
    const loadOptions = async () => {
      const [top, linked] = await Promise.all([
        fetchAdminProgramSales({ page: 1, limit: PROGRAM_OPTIONS_LIMIT }),
        programId ? fetchAdminSalesByProgramIds([programId]) : Promise.resolve([])
      ]);
      if (cancelled) {
        return;
      }
      const options = new Map<string, AdminProgramSale>();
      if (linked[0]) {
        options.set(linked[0].program.id, linked[0]);
      }
      for (const sale of top.sales) {
        options.set(sale.program.id, sale);
      }
      setProgramOptions(Array.from(options.values()));
    };
    loadOptions().catch(() => {
      if (!cancelled) {
        setProgramOptions([]);
      }
    });
    return () => {
      cancelled = true;
    };
    // programId is a one-shot deep link read from the URL on mount: the filter
    // state is the source of truth afterwards, so the effect intentionally runs
    // once and does not track the live filter value.
  }, []);

  const onProgramChange = (value: string) => {
    setProgramId(value);
    setPage(1);
    setProgramSalesPage(1);
  };

  const onStatusChange = (value: PurchaseStatus | "") => {
    setStatus(value);
    setPage(1);
    setProgramSalesPage(1);
  };

  const onTestChange = (value: "all" | "real" | "test") => {
    setTestMode(value);
    setPage(1);
    setProgramSalesPage(1);
  };

  const resetFilters = () => {
    setProgramId("");
    setStatus("");
    setTestMode("all");
    setFrom("");
    setTo("");
    setPage(1);
    setProgramSalesPage(1);
  };

  const metrics = [
    {
      label: "Revenue",
      value: summary ? formatAdminPrice(summary.real_revenue_minor_units, "EUR") : "—",
      hint: "Completed sales · Test excluded",
      icon: Banknote
    },
    {
      label: "Completed",
      value: summary ? summary.completed_count : "—",
      hint: "Including test purchases",
      icon: CheckCircle2
    },
    {
      label: "Pending",
      value: summary ? summary.pending_count : "—",
      hint: "Awaiting payment confirmation",
      icon: Clock
    },
    {
      label: "Failed",
      value: summary ? summary.failed_count : "—",
      hint: "Rejected or abandoned payments",
      icon: XCircle
    },
    {
      label: "Test purchases",
      value: summary ? summary.test_purchase_count : "—",
      hint: "Test Mode completions",
      icon: FlaskConical
    },
    {
      label: "Programs with sales",
      value: summary ? summary.programs_with_sales : "—",
      hint: "At least one real completed sale",
      icon: Layers
    }
  ];

  const purchaseColumns: Array<Admin2Column<AdminPurchase>> = [
    {
      key: "program",
      label: "Program",
      render: (purchase) => (
        <>
          <span className={styles.cellTitle}>{purchase.program.name}</span>
          <span className={styles.cellMeta}>
            {programTypeShortLabel(purchase.program.type)}
            {purchase.program.deleted ? (
              <Admin2StatusBadge label="Retired" tone="danger" withDot={false} className={styles.inlineBadge} />
            ) : null}
            {purchase.test ? (
              <Admin2StatusBadge label="Test" tone="neutral" withDot={false} className={styles.inlineBadge} />
            ) : null}
          </span>
        </>
      )
    },
    {
      key: "customer",
      label: "Customer",
      render: (purchase) => (
        <>
          <span className={styles.cellTitle}>{purchase.customer.name || "—"}</span>
          <span className={styles.cellMeta}>{purchase.customer.email}</span>
        </>
      )
    },
    {
      key: "amount",
      label: "Amount",
      render: (purchase) => (
        <span className={styles.amount}>{formatAdminPrice(purchase.price_minor_units, purchase.currency)}</span>
      )
    },
    {
      key: "status",
      label: "Status",
      render: (purchase) => {
        const meta = statusMeta(purchase.status);
        return <Admin2StatusBadge label={meta.label} tone={meta.tone} />;
      }
    },
    {
      key: "access",
      label: "Access",
      render: (purchase) => (
        <Admin2StatusBadge label={purchase.access ? "Active" : "No access"} tone={purchase.access ? "success" : "neutral"} />
      )
    },
    {
      key: "date",
      label: "Date",
      render: (purchase) => <span className={styles.tableMuted}>{formatAdminDate(purchase.created_at)}</span>
    },
    {
      key: "actions",
      label: "",
      render: (purchase) => (
        <Button variant="ghost" size="small" icon={<Eye size={14} />} onClick={() => setSelected(purchase)}>
          View
        </Button>
      )
    }
  ];

  const programSalesColumns: Array<Admin2Column<AdminProgramSale>> = [
    {
      key: "program",
      label: "Program",
      render: (sale) => (
        <>
          <span className={styles.cellTitle}>{sale.program.name}</span>
          <span className={styles.cellMeta}>
            {programTypeShortLabel(sale.program.type)}
            {sale.program.deleted ? (
              <Admin2StatusBadge label="Retired" tone="danger" withDot={false} className={styles.inlineBadge} />
            ) : null}
          </span>
        </>
      )
    },
    {
      key: "completed",
      label: "Completed",
      render: (sale) => <span className={styles.amount}>{sale.completed_sales}</span>
    },
    {
      key: "revenue",
      label: "Revenue",
      render: (sale) => (
        <span className={styles.amount}>{formatAdminPrice(sale.revenue_minor_units, sale.program.currency)}</span>
      )
    },
    {
      key: "pending",
      label: "Pending",
      render: (sale) => <span className={styles.tableMuted}>{sale.pending_purchases > 0 ? sale.pending_purchases : "—"}</span>
    },
    {
      key: "failed",
      label: "Failed",
      render: (sale) => <span className={styles.tableMuted}>{sale.failed_purchases > 0 ? sale.failed_purchases : "—"}</span>
    },
    {
      key: "test",
      label: "Test",
      render: (sale) => <span className={styles.tableMuted}>{sale.test_purchases > 0 ? sale.test_purchases : "—"}</span>
    }
  ];

  return (
    <>
      <Admin2PageHeader
        eyebrow="Business"
        title="Sales"
        description="Every purchase on the platform, compiled from completed customer transactions."
      />

      <div className={styles.metrics}>
        {metrics.map((metric) => (
          <Admin2MetricCard
            key={metric.label}
            label={metric.label}
            value={metric.value}
            hint={metric.hint}
            icon={metric.icon}
          />
        ))}
      </div>

      <div className={styles.toolbar}>
        <div className={styles.filters}>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Program</span>
            <select
              className={styles.select}
              value={programId}
              aria-label="Filter purchases by program"
              onChange={(event) => onProgramChange(event.target.value)}
            >
              <option value="">All programs</option>
              {programOptions.map((sale) => (
                <option key={sale.program.id} value={sale.program.id}>
                  {sale.program.name}
                </option>
              ))}
            </select>
          </label>

          <label className={styles.field}>
            <span className={styles.fieldLabel}>Status</span>
            <select
              className={styles.select}
              value={status}
              aria-label="Filter purchases by status"
              onChange={(event) => onStatusChange(event.target.value as PurchaseStatus | "")}
            >
              {STATUS_OPTIONS.map((option) => (
                <option key={option.id || "all"} value={option.id}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>

          <label className={styles.field}>
            <span className={styles.fieldLabel}>Mode</span>
            <select
              className={styles.select}
              value={testMode}
              aria-label="Filter purchases by test mode"
              onChange={(event) => onTestChange(event.target.value as "all" | "real" | "test")}
            >
              {TEST_TYPE_OPTIONS.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>

          <label className={styles.field}>
            <span className={styles.fieldLabel}>From</span>
            <input
              className={styles.dateInput}
              type="date"
              value={from}
              aria-label="Filter purchases from this date"
              onChange={(event) => {
                setFrom(event.target.value);
                setPage(1);
                setProgramSalesPage(1);
              }}
            />
          </label>

          <label className={styles.field}>
            <span className={styles.fieldLabel}>To</span>
            <input
              className={styles.dateInput}
              type="date"
              value={to}
              aria-label="Filter purchases until this date"
              onChange={(event) => {
                setTo(event.target.value);
                setPage(1);
                setProgramSalesPage(1);
              }}
            />
          </label>

          {isFiltered ? (
            <Button variant="ghost" size="small" onClick={resetFilters}>
              Reset filters
            </Button>
          ) : null}
        </div>
      </div>

      <Admin2Section
        title="Transactions"
        subtitle={
          loading
            ? "Loading…"
            : errorMessage
              ? ""
              : `${pagination?.total ?? 0} purchase${pagination && pagination.total === 1 ? "" : "s"}`
        }
      >
        {loading ? (
          <div className={styles.pageState}>
            <p className={styles.tableMuted}>Loading transactions…</p>
          </div>
        ) : errorMessage ? (
          <div className={styles.pageState}>
            <p className={styles.pageError}>{errorMessage}</p>
            <div style={{ marginTop: "0.75rem" }}>
              <Button variant="secondary" size="small" onClick={() => void loadPurchases()} icon={<RefreshCw size={15} />}>
                Retry
              </Button>
            </div>
          </div>
        ) : purchases.length === 0 ? (
          <Admin2EmptyState
            icon={ShoppingBag}
            title={isFiltered ? "No purchases match the filters" : "No purchases yet"}
            message={
              isFiltered
                ? "Try a different program, status, mode or date range to widen the view."
                : "Customer purchases will appear here once the first transaction completes."
            }
          />
        ) : (
          <>
            <Admin2Table
              columns={purchaseColumns}
              rows={purchases}
              rowKey={(purchase) => purchase.id}
            />
            {pagination && pagination.total > 1 ? (
              <Admin2Pagination
                page={pagination.page}
                totalPages={pagination.total_pages}
                pageSize={pagination.limit}
                totalItems={pagination.total}
                onChange={(next) => setPage(next)}
              />
            ) : null}
          </>
        )}
      </Admin2Section>

      <Admin2Section
        title="Sales by program"
        subtitle={
          programSalesLoading
            ? "Loading…"
            : programSales.length === 0
              ? ""
              : `${programSalesPagination?.total ?? 0} program${programSalesPagination && programSalesPagination.total === 1 ? "" : "s"}`
        }
      >
        {programSalesLoading ? (
          <div className={styles.pageState}>
            <p className={styles.tableMuted}>Loading program sales…</p>
          </div>
        ) : programSales.length === 0 ? (
          <Admin2EmptyState
            icon={Layers}
            title={isFiltered ? "No programs match the filters" : "No sales yet"}
            message="Per-program revenue and counts appear once purchases complete."
          />
        ) : (
          <>
            <Admin2Table
              columns={programSalesColumns}
              rows={programSales}
              rowKey={(sale) => sale.program.id}
            />
            {programSalesPagination && programSalesPagination.total > 1 ? (
              <Admin2Pagination
                page={programSalesPagination.page}
                totalPages={programSalesPagination.total_pages}
                pageSize={programSalesPagination.limit}
                totalItems={programSalesPagination.total}
                onChange={(next) => setProgramSalesPage(next)}
              />
            ) : null}
          </>
        )}
      </Admin2Section>

      {selected ? (
        <Admin2Modal
          title={selected.program.name}
          description={`Purchase ${selected.id.slice(0, 8)}… · ${formatAdminDate(selected.created_at)}`}
          onClose={() => setSelected(null)}
        >
          <div className={styles.detailList}>
            <div className={styles.detailRow}>
              <span className={styles.detailLabel}>Customer</span>
              <span className={styles.detailValue}>
                {selected.customer.name || "—"}
                <span className={styles.detailMuted}> · {selected.customer.email}</span>
              </span>
            </div>
            <div className={styles.detailRow}>
              <span className={styles.detailLabel}>Program</span>
              <span className={styles.detailValue}>
                {selected.program.name}
                {selected.program.deleted ? (
                  <Admin2StatusBadge label="Retired" tone="danger" withDot={false} className={styles.inlineBadge} />
                ) : null}
              </span>
            </div>
            <div className={styles.detailRow}>
              <span className={styles.detailLabel}>Type</span>
              <span className={styles.detailValue}>{programTypeLabel(selected.program.type)}</span>
            </div>
            <div className={styles.detailRow}>
              <span className={styles.detailLabel}>Amount</span>
              <span className={styles.detailAmount}>
                {formatAdminPrice(selected.price_minor_units, selected.currency)}
              </span>
            </div>
            <div className={styles.detailRow}>
              <span className={styles.detailLabel}>Status</span>
              <span className={styles.detailValue}>
                <Admin2StatusBadge label={statusMeta(selected.status).label} tone={statusMeta(selected.status).tone} />
              </span>
            </div>
            <div className={styles.detailRow}>
              <span className={styles.detailLabel}>Access</span>
              <span className={styles.detailValue}>
                <Admin2StatusBadge
                  label={selected.access ? "Active" : "No access"}
                  tone={selected.access ? "success" : "neutral"}
                />
              </span>
            </div>
            <div className={styles.detailRow}>
              <span className={styles.detailLabel}>Mode</span>
              <span className={styles.detailValue}>{selected.test ? "Test purchase" : "Real purchase"}</span>
            </div>
            <div className={styles.detailRow}>
              <span className={styles.detailLabel}>Transaction</span>
              <span className={styles.detailMono}>{selected.id}</span>
            </div>
          </div>
          <div className={styles.modalActions}>
            <Button variant="secondary" size="small" onClick={() => setSelected(null)}>
              Close
            </Button>
          </div>
        </Admin2Modal>
      ) : null}
    </>
  );
}