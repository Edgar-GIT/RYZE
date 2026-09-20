import { useCallback, useEffect, useMemo, useState } from "react";
import { BarChart3, Info, RefreshCw, TrendingUp, Users } from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2EmptyState } from "@/components/admin2/admin2_empty_state/admin2_empty_state";
import { Admin2MetricCard } from "@/components/admin2/admin2_metric_card/admin2_metric_card";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Button } from "@/components/button/button";
import { fetchAdminTrainers, fetchAdminUsers, formatAdminDate } from "@/services/admin_api";
import {
  fetchAdminPrograms,
  formatAdminPrice,
  ProgramType,
  programTypeLabel,
  type ProgramTypeValue
} from "@/services/admin2_api";

import styles from "./admin2_analytics_page.module.css";

interface AnalyticsSnapshot {
  clientsTotal: number;
  trainersTotal: number;
  programsTotal: number;
  programsByType: Array<{ type: ProgramTypeValue; count: number; percent: number }>;
  averagePrice: number | null;
  programs: Array<{ name: string; type: ProgramTypeValue; price: number; currency: string; created: string }>;
}

export default function Admin2AnalyticsPage() {
  const [snapshot, setSnapshot] = useState<AnalyticsSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");

    try {
      const [users, trainers, programsResult] = await Promise.all([
        fetchAdminUsers(1, 1),
        fetchAdminTrainers(1, 1),
        fetchAdminPrograms(1, 100)
      ]);

      const programs = programsResult.programs.map((p) => ({
        name: p.name,
        type: p.type as ProgramTypeValue,
        price: p.price_minor_units,
        currency: p.currency,
        created: p.created_at
      }));

      const types = [ProgramType.FREE, ProgramType.PREMIUM, ProgramType.PERSONALIZED] as const;
      const byType = types.map((type) => ({
        type,
        count: programs.filter((p) => p.type === type).length
      }));
      const denominator = byType.reduce((sum, entry) => sum + entry.count, 0) || 1;
      const programsByType = byType.map((entry) => ({
        ...entry,
        percent: Math.round((entry.count / denominator) * 100)
      }));

      const priced = programs.filter((p) => p.type !== ProgramType.FREE);
      const averagePrice = priced.length
        ? Math.round(priced.reduce((sum, p) => sum + p.price, 0) / priced.length)
        : null;

      setSnapshot({
        clientsTotal: users.pagination.total,
        trainersTotal: trainers.pagination.total,
        programsTotal: programsResult.pagination.total,
        programsByType,
        averagePrice,
        programs
      });
    } catch {
      setErrorMessage("Unable to load analytics. Please try again.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) {
    return (
      <>
        <Admin2PageHeader eyebrow="Business" title="Analytics" />
        <div className={styles.pageState}>
          <p className={styles.muted}>Loading analytics…</p>
        </div>
      </>
    );
  }

  if (errorMessage && !snapshot) {
    return (
      <>
        <Admin2PageHeader eyebrow="Business" title="Analytics" />
        <div className={styles.pageState}>
          <p className={styles.muted}>{errorMessage}</p>
          <div style={{ marginTop: "0.75rem" }}>
            <Button variant="secondary" size="small" onClick={() => void load()} icon={<RefreshCw size={15} />}>
              Retry
            </Button>
          </div>
        </div>
      </>
    );
  }

  return (
    <>
      <Admin2PageHeader
        eyebrow="Business"
        title="Analytics"
        description="A signal-based view of growth, product adoption and performance."
      />

      <div className={styles.notice}>
        <Info size={15} className={styles.noticeIcon} aria-hidden="true" />
        <span>
          <strong>Charts pending sales data.</strong> Revenue and conversion graphs are built
          once the purchase analytics pipeline is implemented. The metrics below are live.
        </span>
      </div>

      <div className={styles.metrics}>
        <Admin2MetricCard label="Clients" value={snapshot?.clientsTotal ?? 0} hint="Registered accounts" icon={Users} />
        <Admin2MetricCard label="Trainers" value={snapshot?.trainersTotal ?? 0} hint="Active trainer profiles" icon={Users} />
        <Admin2MetricCard label="Published programs" value={snapshot?.programsTotal ?? 0} hint="Visible in the marketplace" icon={BarChart3} />
        <Admin2MetricCard
          label="Average paid price"
          value={
            snapshot?.averagePrice != null
              ? formatAdminPrice(snapshot.averagePrice, "EUR")
              : "—"
          }
          hint="Across published paid programs"
          icon={TrendingUp}
        />
      </div>

      <div className={styles.split}>
        <Admin2Section
          title="Business performance"
          subtitle="Revenue and purchases over time"
        >
          <Admin2EmptyState
            icon={BarChart3}
            title="Purchase analytics are not implemented yet."
            message="This chart will visualize completed transactions and revenue once the admin sales API exists."
          />
        </Admin2Section>

        <Admin2Section title="Product mix" subtitle="Published programs by product type">
          <div className={styles.productMixBody}>
            {(snapshot?.programsByType ?? []).map((entry) => (
              <div key={entry.type} className={styles.mixRow}>
                <div className={styles.mixRowHead}>
                  <p className={styles.mixLabel}>{programTypeLabel(entry.type)}</p>
                  <p className={styles.mixValue}>{entry.count} · {entry.percent}%</p>
                </div>
                <div className={styles.mixTrack}>
                  <div className={styles.mixFill} style={{ width: `${entry.percent}%` }} />
                </div>
              </div>
            ))}
          </div>
        </Admin2Section>
      </div>

      <Admin2Section
        title="Most recently published"
        subtitle="Programs ordered by publication date"
      >
        <div style={{ padding: "0.9rem 1.15rem", display: "flex", flexDirection: "column", gap: "0.4rem" }}>
          {(snapshot?.programs ?? []).slice(0, 5).map((program) => (
            <div key={program.name} style={{ display: "flex", justifyContent: "space-between", gap: "0.75rem", borderBottom: "var(--border)", paddingBottom: "0.5rem" }}>
              <span style={{ fontWeight: 600, color: "var(--text-primary)" }}>{program.name}</span>
              <span className={styles.muted}>{formatAdminDate(program.created)}</span>
            </div>
          ))}
        </div>
      </Admin2Section>
    </>
  );
}