import { CheckCircle2, CircleSlash, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/button/button";
import styles from "@/pages/admin/admin_dashboard_page/admin_dashboard_page.module.css";
import { fetchAdminSystemStatus, formatDuration, type AdminSystemStatus } from "@/services/admin_api";
import { joinClassNames } from "@utils/class_names";

const METHOD_LABEL: Record<string, string> = { card: "Card", mbway: "MB WAY", paypal: "PayPal" };

// AdminConfigurationPage lists what this deployment has switched on. Values
// come from the server's environment (never secrets); they change through the
// deployment settings, not from this page.
export const AdminConfigurationPage = () => {
  const [status, setStatus] = useState<AdminSystemStatus | null>(null);
  const [error, setError] = useState(false);

  const load = useCallback(async () => {
    setError(false);
    try {
      setStatus(await fetchAdminSystemStatus());
    } catch {
      setError(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const features = status?.features;
  const rows: { label: string; on: boolean; detail: string }[] = features
    ? [
        { label: "Transactional email", on: features.email_delivery, detail: features.email_delivery ? "Sent through the configured relay" : "Recorded as skipped (MAIL_* not set)" },
        { label: "Google Sign-In", on: features.google_sign_in, detail: features.google_sign_in ? "Login and account linking available" : "Button explains it is not available (GOOGLE_* not set)" },
        {
          label: "Payment methods",
          on: features.payment_methods.length > 0,
          detail: features.payment_methods.length > 0 ? features.payment_methods.map((method) => METHOD_LABEL[method] ?? method).join(", ") : "No provider configured: paid checkout is unavailable"
        },
        { label: "Contact form delivery", on: features.contact_delivery, detail: features.contact_delivery ? "Messages are emailed to RYZE" : "The form answers that contact is unavailable" },
        { label: "Exercise media", on: features.exercise_media, detail: features.exercise_media ? "Media references are served" : "Off until the dataset licence is confirmed" },
        { label: "Rate limiting", on: features.rate_limiting, detail: features.rate_limiting ? "Sensitive endpoints are limited per IP" : "Off — never run production like this" },
        { label: "Secure cookies", on: features.secure_cookies, detail: features.secure_cookies ? "Session cookies require HTTPS" : "Off — local development only" },
        { label: "Admin Test Mode", on: features.test_mode, detail: features.test_mode ? "Personas available to the Technical Administrator" : "Disabled" }
      ]
    : [];

  return (
    <div className={styles.dashboardPage}>
      <header className={styles.header}>
        <p className={styles.eyebrow}>Technical configuration</p>
        <h1 className={styles.title}>Technical configuration</h1>
        <p className={styles.description}>
          What this deployment has switched on. Values come from the server environment and change through
          the deployment settings; no secret is ever shown.
        </p>
      </header>

      <div>
        <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => void load()}>
          Refresh
        </Button>
      </div>

      {error ? (
        <p className={joinClassNames(styles.description, styles.overviewWarning)} role="alert">
          The configuration could not be loaded. Only the Technical Administrator can read it.
        </p>
      ) : null}

      {features ? (
        <section className={styles.overviewSection} aria-label="Features">
          <h2 className={styles.sectionTitle}>Features</h2>
          <dl className={styles.overviewCard}>
            {rows.map((row) => (
              <div key={row.label} className={styles.overviewRow}>
                <dt className={styles.overviewLabel}>
                  {row.on ? (
                    <CheckCircle2 className={joinClassNames(styles.overviewIcon, styles.overviewOk)} aria-label="On" strokeWidth={1.8} />
                  ) : (
                    <CircleSlash className={joinClassNames(styles.overviewIcon, styles.overviewMuted)} aria-label="Off" strokeWidth={1.8} />
                  )}{" "}
                  {row.label}
                </dt>
                <dd className={joinClassNames(styles.overviewValue, !row.on && styles.overviewMuted)}>{row.detail}</dd>
              </div>
            ))}
            <div className={styles.overviewRow}>
              <dt className={styles.overviewLabel}>Session lifetime</dt>
              <dd className={styles.overviewValue}>{formatDuration(features.session_lifetime_seconds)}</dd>
            </div>
            <div className={styles.overviewRow}>
              <dt className={styles.overviewLabel}>Default platform commission</dt>
              <dd className={styles.overviewValue}>{features.default_platform_commission_bps / 100}%</dd>
            </div>
          </dl>
        </section>
      ) : null}
    </div>
  );
};
