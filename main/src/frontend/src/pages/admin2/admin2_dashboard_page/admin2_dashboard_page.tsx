import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  ArrowRight,
  BadgeCheck,
  BarChart3,
  Clock,
  Package,
  RefreshCw,
  Users
} from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2MetricCard } from "@/components/admin2/admin2_metric_card/admin2_metric_card";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Button } from "@/components/button/button";
import {
  fetchAdminTrainerApplications,
  fetchAdminTrainers,
  fetchAdminUsers,
  formatAdminDate
} from "@/services/admin_api";
import {
  fetchAdminPrograms,
  formatAdminPrice,
  programTypeLabel,
  programTypeShortLabel,
  ProgramType,
  type ProgramTypeValue
} from "@/services/admin2_api";

import styles from "./admin2_dashboard_page.module.css";

interface DashboardData {
  clientsTotal: number;
  trainersTotal: number;
  programsTotal: number;
  pendingApplications: number;
  programs: Array<{
    id: string;
    name: string;
    type: string;
    price_minor_units: number;
    currency: string;
    created_at: string;
  }>;
}

const DashboardLoading = () => (
  <div className={styles.pageState}>
    <p className={styles.tableMuted}>Loading overview…</p>
  </div>
);

export default function Admin2DashboardPage() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");

    try {
      const [users, trainers, pending, programs] = await Promise.all([
        fetchAdminUsers(1, 1),
        fetchAdminTrainers(1, 1),
        fetchAdminTrainerApplications(1, 1, "PENDING"),
        fetchAdminPrograms(1, 100)
      ]);

      setData({
        clientsTotal: users.pagination.total,
        trainersTotal: trainers.pagination.total,
        programsTotal: programs.pagination.total,
        pendingApplications: pending.pagination.total,
        programs: programs.programs.map((program) => ({
          id: program.id,
          name: program.name,
          type: program.type,
          price_minor_units: program.price_minor_units,
          currency: program.currency,
          created_at: program.created_at
        }))
      });
    } catch {
      setErrorMessage("Unable to load the overview. Please try again.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const mix = useMemo(() => {
    if (!data) {
      return [];
    }

    const types = [ProgramType.FREE, ProgramType.PREMIUM, ProgramType.PERSONALIZED];
    const counts = types.map((type) => ({
      type,
      count: data.programs.filter((program) => program.type === type).length
    }));

    const denominator = counts.reduce((sum, entry) => sum + entry.count, 0) || 1;
    return counts.map((entry) => ({
      ...entry,
      percent: Math.round((entry.count / denominator) * 100)
    }));
  }, [data]);

  const averagePrice = useMemo(() => {
    if (!data) {
      return null;
    }

    const priced = data.programs.filter((program) => program.type !== ProgramType.FREE);
    if (priced.length === 0) {
      return null;
    }
    const sum = priced.reduce((acc, program) => acc + program.price_minor_units, 0);
    return Math.round(sum / priced.length);
  }, [data]);

  if (loading && !data) {
    return (
      <>
        <Admin2PageHeader eyebrow="Business console" title="Overview" />
        <DashboardLoading />
      </>
    );
  }

  if (errorMessage && !data) {
    return (
      <>
        <Admin2PageHeader eyebrow="Business console" title="Overview" />
        <div className={styles.pageState}>
          <p className={styles.tableMuted}>{errorMessage}</p>
          <Button variant="secondary" size="small" onClick={() => void load()} icon={<RefreshCw size={15} />}>
            Retry
          </Button>
        </div>
      </>
    );
  }

  return (
    <>
      <Admin2PageHeader
        eyebrow="Business console"
        title="Overview"
        description="The state of the RYZE business: platform growth, product mix and content health."
        actions={
          <Button to="/admin2/plans" variant="secondary" size="small" icon={<ArrowRight size={15} />}>
            Manage plans
          </Button>
        }
      />

      <div className={styles.metrics}>
        <Admin2MetricCard
          label="Clients"
          value={data?.clientsTotal ?? 0}
          hint="Registered user accounts"
          icon={Users}
        />
        <Admin2MetricCard
          label="Trainers"
          value={data?.trainersTotal ?? 0}
          hint="Active trainer profiles"
          icon={BadgeCheck}
        />
        <Admin2MetricCard
          label="Published programs"
          value={data?.programsTotal ?? 0}
          hint="Live in the marketplace"
          icon={Package}
        />
        <Admin2MetricCard
          label="Pending approvals"
          value={data?.pendingApplications ?? 0}
          hint="Trainer applications to review"
          icon={Clock}
        />
      </div>

      <div className={styles.split}>
        <Admin2Section
          title="Product mix"
          subtitle="Published programs by product type"
        >
          <div className={styles.productMixBody}>
            {mix.map((entry) => (
              <div key={entry.type} className={styles.mixRow}>
                <div className={styles.mixRowHead}>
                  <p className={styles.mixLabel}>{programTypeLabel(entry.type as ProgramTypeValue)}</p>
                  <p className={styles.mixValue}>
                    {entry.count} · {entry.percent}%
                  </p>
                </div>
                <div className={styles.mixTrack}>
                  <div className={styles.mixFill} style={{ width: `${entry.percent}%` }} />
                </div>
              </div>
            ))}
          </div>
        </Admin2Section>

        <Admin2Section title="Platform summary" subtitle="Current business facts">
          <div className={styles.summaryList}>
            <div className={styles.summaryRow}>
              <p className={styles.summaryLabel}>
                <Package size={16} className={styles.summaryIcon} aria-hidden="true" />
                Free programs
              </p>
              <p className={styles.summaryValue}>
                {data?.programs.filter((p) => p.type === ProgramType.FREE).length ?? 0}
              </p>
            </div>
            <div className={styles.summaryRow}>
              <p className={styles.summaryLabel}>
                <Package size={16} className={styles.summaryIcon} aria-hidden="true" />
                Personalized programs
              </p>
              <p className={styles.summaryValue}>
                {data?.programs.filter((p) => p.type === ProgramType.PERSONALIZED).length ?? 0}
              </p>
            </div>
            <div className={styles.summaryRow}>
              <p className={styles.summaryLabel}>
                <BarChart3 size={16} className={styles.summaryIcon} aria-hidden="true" />
                Average paid price
              </p>
              <p className={styles.summaryValue}>
                {averagePrice
                  ? formatAdminPrice(averagePrice, data?.programs[0]?.currency ?? "EUR")
                  : "—"}
              </p>
            </div>
          </div>
        </Admin2Section>
      </div>

      <Admin2Section
        title="Recently published"
        subtitle="The newest programs in the marketplace"
        actions={
          <Link className={styles.linkButton} to="/admin2/plans">
            View all plans
          </Link>
        }
      >
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Program</th>
                <th>Type</th>
                <th>Price</th>
                <th>Published</th>
              </tr>
            </thead>
            <tbody>
              {(data?.programs ?? []).slice(0, 6).map((program) => (
                <tr key={program.id}>
                  <td>
                    <Link to={`/admin2/plans/${program.id}`}>{program.name}</Link>
                  </td>
                  <td>
                    <Admin2StatusBadge label={programTypeShortLabel(program.type as ProgramTypeValue)} tone="success" withDot={false} />
                  </td>
                  <td>{formatAdminPrice(program.price_minor_units, program.currency)}</td>
                  <td className={styles.tableMuted}>{formatAdminDate(program.created_at)}</td>
                </tr>
              ))}
              {(data?.programs ?? []).length === 0 ? (
                <tr>
                  <td colSpan={4} className={styles.tableMuted}>
                    No published programs yet.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </Admin2Section>
    </>
  );
}