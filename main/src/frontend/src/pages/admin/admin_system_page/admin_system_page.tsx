import { Activity, Database, Mail, MessageSquare, RefreshCw, ServerCog } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/button/button";
import styles from "@/pages/admin/admin_dashboard_page/admin_dashboard_page.module.css";
import { fetchAdminSystemStatus, formatDuration, type AdminSystemStatus } from "@/services/admin_api";
import { joinClassNames } from "@utils/class_names";

const QUEUE_LABEL: Record<string, string> = {
  pending: "Waiting to send",
  sending: "Sending now",
  sent: "Sent",
  failed: "Failed (needs attention)",
  skipped: "Not sent (delivery off)",
  expired: "Expired before sending"
};

// AdminSystemPage is the Technical Administrator's live view of this
// deployment: API version and uptime, database reachability and schema, the
// email queue and the live product queues.
export const AdminSystemPage = () => {
  const [status, setStatus] = useState<AdminSystemStatus | null>(null);
  const [error, setError] = useState(false);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError(false);
    try {
      setStatus(await fetchAdminSystemStatus());
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const failed = status?.email_queue.find((row) => row.status === "failed")?.count ?? 0;

  return (
    <div className={styles.dashboardPage}>
      <header className={styles.header}>
        <p className={styles.eyebrow}>System</p>
        <h1 className={styles.title}>System &amp; Infrastructure</h1>
        <p className={styles.description}>Live status of this API instance, its database and its queues.</p>
      </header>

      <div>
        <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} disabled={loading} onClick={() => void load()}>
          {loading ? "Refreshing…" : "Refresh"}
        </Button>
      </div>

      {error ? (
        <p className={joinClassNames(styles.description, styles.overviewWarning)} role="alert">
          The system status could not be loaded. Only the Technical Administrator can read it.
        </p>
      ) : null}

      {status ? (
        <>
          <div className={styles.statsRow}>
            <div className={styles.statCard}>
              <ServerCog className={styles.statIcon} aria-hidden="true" strokeWidth={1.6} />
              <div className={styles.statContent}>
                <p className={styles.statLabel}>API version · uptime</p>
                <p className={styles.statValue}>
                  {status.version} · {formatDuration(status.uptime_seconds)}
                </p>
              </div>
            </div>
            <div className={styles.statCard}>
              <Database className={styles.statIcon} aria-hidden="true" strokeWidth={1.6} />
              <div className={styles.statContent}>
                <p className={styles.statLabel}>Database</p>
                <p className={joinClassNames(styles.statValue, !status.database.reachable && styles.statValueError)}>
                  {status.database.reachable ? `Schema v${status.database.schema_version ?? "?"}` : "Unreachable"}
                </p>
              </div>
            </div>
            <div className={styles.statCard}>
              <Mail className={styles.statIcon} aria-hidden="true" strokeWidth={1.6} />
              <div className={styles.statContent}>
                <p className={styles.statLabel}>Failed emails</p>
                <p className={joinClassNames(styles.statValue, failed > 0 && styles.statValueError)}>{failed}</p>
              </div>
            </div>
          </div>

          <section className={styles.overviewSection} aria-label="Database">
            <h2 className={styles.sectionTitle}>Database</h2>
            <dl className={styles.overviewCard}>
              <div className={styles.overviewRow}>
                <dt className={styles.overviewLabel}>Reachable</dt>
                <dd className={joinClassNames(styles.overviewValue, status.database.reachable ? styles.overviewOk : styles.overviewWarning)}>
                  {status.database.reachable ? "Yes" : "No"}
                </dd>
              </div>
              <div className={styles.overviewRow}>
                <dt className={styles.overviewLabel}>Schema version</dt>
                <dd className={joinClassNames(styles.overviewValue, status.database.schema_dirty && styles.overviewWarning)}>
                  {status.database.schema_version ?? "—"}
                  {status.database.schema_dirty ? " (dirty: a migration failed half-way)" : ""}
                </dd>
              </div>
              <div className={styles.overviewRow}>
                <dt className={styles.overviewLabel}>Open connections</dt>
                <dd className={styles.overviewValue}>{status.database.open_connections ?? "—"}</dd>
              </div>
              <div className={styles.overviewRow}>
                <dt className={styles.overviewLabel}>Running since</dt>
                <dd className={styles.overviewValue}>{new Date(status.started_at).toLocaleString("en-GB")}</dd>
              </div>
            </dl>
          </section>

          <section className={styles.overviewSection} aria-label="Email queue">
            <h2 className={styles.sectionTitle}>Email queue</h2>
            <dl className={styles.overviewCard}>
              {status.email_queue.length === 0 ? (
                <div className={styles.overviewRow}>
                  <dt className={styles.overviewLabel}>Emails recorded</dt>
                  <dd className={joinClassNames(styles.overviewValue, styles.overviewMuted)}>None yet</dd>
                </div>
              ) : (
                status.email_queue.map((row) => (
                  <div key={row.status} className={styles.overviewRow}>
                    <dt className={styles.overviewLabel}>{QUEUE_LABEL[row.status] ?? row.status}</dt>
                    <dd className={joinClassNames(styles.overviewValue, row.status === "failed" && row.count > 0 && styles.overviewWarning)}>
                      {row.count}
                    </dd>
                  </div>
                ))
              )}
            </dl>
          </section>

          <section className={styles.overviewSection} aria-label="Live queues">
            <h2 className={styles.sectionTitle}>Live queues</h2>
            <dl className={styles.overviewCard}>
              <div className={styles.overviewRow}>
                <dt className={styles.overviewLabel}>
                  <Activity className={styles.overviewIcon} aria-hidden="true" strokeWidth={1.6} /> Notifications visible (24 h)
                </dt>
                <dd className={styles.overviewValue}>{status.visible_notifications}</dd>
              </div>
              <div className={styles.overviewRow}>
                <dt className={styles.overviewLabel}>
                  <MessageSquare className={styles.overviewIcon} aria-hidden="true" strokeWidth={1.6} /> Open support conversations
                </dt>
                <dd className={styles.overviewValue}>{status.open_support}</dd>
              </div>
            </dl>
          </section>
        </>
      ) : null}
    </div>
  );
};
