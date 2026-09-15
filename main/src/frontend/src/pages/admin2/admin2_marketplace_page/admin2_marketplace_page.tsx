import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Info, Package, RefreshCw } from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Button } from "@/components/button/button";
import { Admin2Table } from "@/components/admin2/admin2_table/admin2_table";
import { MARKETPLACE_FILTER_GROUPS } from "@/constants/marketplace";
import {
  AdminProgram,
  fetchAdminPrograms,
  formatAdminPrice,
  ProgramType,
  programTypeLabel
} from "@/services/admin2_api";
import { formatAdminDate } from "@/services/admin_api";

import styles from "./admin2_marketplace_page.module.css";

export default function Admin2MarketplacePage() {
  const [programs, setPrograms] = useState<AdminProgram[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");
    try {
      const result = await fetchAdminPrograms(1, 100);
      setPrograms(result.programs);
      setTotal(result.pagination.total);
    } catch {
      setErrorMessage("Unable to load the marketplace. Please try again.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <>
      <Admin2PageHeader
        eyebrow="Programs"
        title="Marketplace"
        description="How members discover programs. Curated sections and filters are part of the marketplace model."
      />

      <div className={styles.notice}>
        <Info size={15} className={styles.noticeIcon} aria-hidden="true" />
        <span>
          <strong>Curated rows pending sales data.</strong> Sections like “Most Popular” and
          “Trending” are defined in the storefront constants but only populate once purchase
          analytics are available. The published catalog below is live data.
        </span>
      </div>

      <Admin2Section title="Filter groups" subtitle="Defined in the marketplace model">
        <div className={styles.chips}>
          {MARKETPLACE_FILTER_GROUPS.flatMap((group) =>
            group.options.map((option) => (
              <span key={`${group.id}-${option}`} className={styles.chip}>
                {option}
              </span>
            ))
          )}
        </div>
      </Admin2Section>

      <div style={{ height: "0.75rem" }} />

      <Admin2Section
        title="Published catalog"
        subtitle={`${total} program${total === 1 ? "" : "s"} available in the marketplace`}
      >
        {loading ? (
          <div className={styles.pageState}>
            <p className={styles.muted}>Loading marketplace…</p>
          </div>
        ) : errorMessage ? (
          <div className={styles.pageState}>
            <p className={styles.muted}>{errorMessage}</p>
            <div style={{ marginTop: "0.75rem" }}>
              <Button variant="secondary" size="small" onClick={() => void load()} icon={<RefreshCw size={15} />}>
                Retry
              </Button>
            </div>
          </div>
        ) : (
          <Admin2Table
            columns={[
              {
                key: "name",
                label: "Program",
                render: (program: AdminProgram) => (
                  <Link to={`/admin2/plans/${program.id}`}>{program.name}</Link>
                )
              },
              {
                key: "type",
                label: "Type",
                render: (program: AdminProgram) => (
                  <Admin2StatusBadge label={programTypeLabel(program.type)} tone="success" withDot={false} />
                )
              },
              {
                key: "price",
                label: "Price",
                render: (program: AdminProgram) =>
                  program.type === ProgramType.FREE
                    ? "Free"
                    : formatAdminPrice(program.price_minor_units, program.currency)
              },
              {
                key: "created",
                label: "Added",
                className: styles.muted,
                render: (program: AdminProgram) => formatAdminDate(program.created_at)
              }
            ]}
            rows={programs}
            rowKey={(program) => program.id}
            emptyRow={
              <span className={styles.muted}>
                <Package
                  size={28}
                  aria-hidden="true"
                  style={{ display: "block", margin: "0 auto 0.5rem", opacity: 0.5 }}
                />
                No published programs in the marketplace yet.
              </span>
            }
          />
        )}
      </Admin2Section>
    </>
  );
}