import { Activity, Dumbbell, FileCheck2, ShieldCheck, Users } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";

import { useAdminSession } from "@/components/admin/admin_session_context";
import {
  adminRoleName,
  fetchAdminTrainerApplications,
  fetchAdminTrainers,
  fetchAdminUsers,
  ApplicationStatus
} from "@/services/admin_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./admin_dashboard_page.module.css";

type MetricState =
  | { status: "loading" }
  | { status: "ready"; value: string }
  | { status: "error" };

const STAT_INTERVAL = 1;

const StatCard = ({ icon: Icon, label, metric }: { icon: typeof Users; label: string; metric: MetricState }) => (
  <div className={styles.statCard}>
    <Icon className={styles.statIcon} aria-hidden="true" strokeWidth={1.6} />
    <div className={styles.statContent}>
      <p className={styles.statLabel}>{label}</p>
      <p className={joinClassNames(styles.statValue, metric.status === "error" && styles.statValueError)}>
        {metric.status === "loading" ? "…" : metric.status === "error" ? "—" : metric.value}
      </p>
    </div>
  </div>
);

interface OverviewRowProps {
  label: string;
  value: ReactNode;
  valueClass?: string;
}

const OverviewRow = ({ label, value, valueClass }: OverviewRowProps) => (
  <div className={styles.overviewRow}>
    <dt className={styles.overviewLabel}>{label}</dt>
    <dd className={joinClassNames(styles.overviewValue, valueClass)}>
      {value}
    </dd>
  </div>
);

export const AdminDashboardPage = () => {
  const { identity } = useAdminSession();
  const [usersTotal, setUsersTotal] = useState<MetricState>({ status: "loading" });
  const [trainersTotal, setTrainersTotal] = useState<MetricState>({ status: "loading" });
  const [pendingApplications, setPendingApplications] = useState<MetricState>({ status: "loading" });
  const [apiReachable, setApiReachable] = useState(true);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      const [usersResult, trainersResult, applicationsResult] = await Promise.allSettled([
        fetchAdminUsers(STAT_INTERVAL, STAT_INTERVAL),
        fetchAdminTrainers(STAT_INTERVAL, STAT_INTERVAL),
        fetchAdminTrainerApplications(STAT_INTERVAL, STAT_INTERVAL, ApplicationStatus.PENDING)
      ]);

      if (cancelled) {
        return;
      }

      setUsersTotal(
        usersResult.status === "fulfilled"
          ? { status: "ready", value: String(usersResult.value.pagination.total) }
          : { status: "error" }
      );
      setTrainersTotal(
        trainersResult.status === "fulfilled"
          ? { status: "ready", value: String(trainersResult.value.pagination.total) }
          : { status: "error" }
      );
      setPendingApplications(
        applicationsResult.status === "fulfilled"
          ? { status: "ready", value: String(applicationsResult.value.pagination.total) }
          : { status: "error" }
      );

      setApiReachable(
        usersResult.status === "fulfilled" ||
        trainersResult.status === "fulfilled" ||
        applicationsResult.status === "fulfilled"
      );
    };

    load();
    return () => { cancelled = true; };
  }, []);

  return (
    <div className={styles.dashboardPage}>
      <header className={styles.header}>
        <p className={styles.eyebrow}>Overview</p>
        <h1 className={styles.title}>Dashboard</h1>
        <p className={styles.description}>
          Platform overview for the {adminRoleName(identity.role)}.
        </p>
      </header>

      <div className={styles.statsRow}>
        <StatCard icon={Users} label="Total users" metric={usersTotal} />
        <StatCard icon={Dumbbell} label="Total trainers" metric={trainersTotal} />
        <StatCard icon={FileCheck2} label="Pending applications" metric={pendingApplications} />
      </div>

      <section className={styles.overviewSection} aria-label="System overview">
        <h2 className={styles.sectionTitle}>System overview</h2>
        <dl className={styles.overviewCard}>
          <OverviewRow
            label="Current administrator"
            value={(
              <span className={styles.overviewStrong}>
                <ShieldCheck className={styles.overviewIcon} aria-hidden="true" strokeWidth={1.6} />
                {identity.id} · {adminRoleName(identity.role)}
              </span>
            )}
          />
          <OverviewRow
            label="API &amp; database"
            value={apiReachable ? "Reachable" : "Unreachable"}
            valueClass={apiReachable ? styles.overviewOk : styles.overviewWarning}
          />
          <OverviewRow label="Server uptime" value="Not collected" valueClass={styles.overviewMuted} />
          <OverviewRow label="Session lifetime" value="15 minutes" valueClass={styles.overviewMuted} />
        </dl>
      </section>

      <section className={styles.activitySection} aria-label="Recent activity">
        <h2 className={styles.sectionTitle}>Recent activity</h2>
        <div className={styles.activityCard}>
          <Activity className={styles.activityIcon} aria-hidden="true" strokeWidth={1.4} />
          <p className={styles.activityTitle}>Activity tracking is not available yet</p>
          <p className={styles.activityDescription}>
            The audit log will surface recent administrative events here once the
            activity service is implemented.
          </p>
        </div>
      </section>
    </div>
  );
};