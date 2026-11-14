import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  ChevronRight,
  CircleSlash,
  Database,
  Dumbbell,
  FileCheck2,
  FlaskConical,
  Mail,
  RefreshCw,
  ScrollText,
  ServerCog,
  ShieldAlert,
  UserPlus,
  Users,
  UserX
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { TrendChart } from "@/components/admin_charts/trend_chart";
import board from "@/components/admin_board/admin_board.module.css";
import { Waves } from "@/components/public_ui/public_ui";
import {
  describeAuditAction,
  fetchAdminAuditLog,
  fetchAdminSystemStatus,
  formatAdminDateTime,
  formatDuration,
  type AdminAuditEntry,
  type AdminSystemStatus
} from "@/services/admin_api";
import { fetchPlatformOverview, OVERVIEW_RANGES, type PlatformOverview } from "@/services/admin_overview_api";
import { joinClassNames } from "@utils/class_names";

const count = new Intl.NumberFormat("en-IE");

const dayLabel = (iso: string) =>
  new Date(`${iso}T00:00:00Z`).toLocaleDateString("en-IE", { day: "numeric", month: "short", timeZone: "UTC" });

const METHOD_LABEL: Record<string, string> = { card: "Card", mbway: "MB WAY", paypal: "PayPal" };

interface Data {
  overview: PlatformOverview;
  status: AdminSystemStatus | null;
  audit: AdminAuditEntry[];
}

type State = { status: "loading" } | { status: "error" } | { status: "ready"; data: Data };

// AdminDashboardPage is the first screen of the Technical Administrator: the
// health of the platform (API, database, email queue, configuration), the
// accounts and their activity, and the latest administrative actions.
export const AdminDashboardPage = () => {
  const [days, setDays] = useState(30);
  const [state, setState] = useState<State>({ status: "loading" });

  const load = useCallback(async (range: number) => {
    try {
      const [overview, status, audit] = await Promise.all([
        fetchPlatformOverview(range),
        fetchAdminSystemStatus().catch(() => null),
        fetchAdminAuditLog(1, 6).catch(() => null)
      ]);
      setState({ status: "ready", data: { overview, status, audit: audit?.entries ?? [] } });
    } catch {
      setState({ status: "error" });
    }
  }, []);

  useEffect(() => {
    void load(days);
  }, [days, load]);

  if (state.status === "loading") {
    return <AppLoading />;
  }

  if (state.status === "error") {
    return (
      <section className={board.stateCard}>
        <h1 className={board.cardTitle}>Platform overview</h1>
        <p>The overview could not be loaded right now.</p>
        <button type="button" className={board.secondary} onClick={() => void load(days)}>
          <RefreshCw aria-hidden="true" /> Try again
        </button>
      </section>
    );
  }

  const { overview, status, audit } = state.data;
  const members = overview.members;
  const verifiedShare = members.total > 0 ? Math.round((members.verified / members.total) * 100) : 0;
  const emailFailed = status?.email_queue.find((row) => row.status === "failed")?.count ?? 0;
  const emailPending = status?.email_queue.find((row) => row.status === "pending")?.count ?? 0;
  const databaseOk = Boolean(status?.database.reachable) && !status?.database.schema_dirty;
  const healthy = databaseOk && emailFailed === 0 && overview.audit.failed_24h === 0;

  const attention = [
    {
      to: "/admin/trainer-applications",
      icon: FileCheck2,
      title: "Trainer applications",
      text: "Pending review",
      value: overview.trainers.pending_applications,
      hot: overview.trainers.pending_applications > 0
    },
    {
      to: "/admin/audit",
      icon: ShieldAlert,
      title: "Denied admin requests",
      text: "Last 24 hours",
      value: overview.audit.denied_24h,
      hot: overview.audit.denied_24h > 0
    },
    {
      to: "/admin/audit",
      icon: AlertTriangle,
      title: "Failed admin requests",
      text: "Last 24 hours",
      value: overview.audit.failed_24h,
      alert: overview.audit.failed_24h > 0
    },
    {
      to: "/admin/system",
      icon: Mail,
      title: "Email queue",
      text: `${count.format(emailPending)} waiting · ${count.format(emailFailed)} failed`,
      value: emailFailed,
      alert: emailFailed > 0
    },
    {
      to: "/admin/users",
      icon: UserX,
      title: "Disabled accounts",
      text: "Can be reactivated from Users",
      value: members.disabled
    }
  ];

  const features = status?.features;
  const configuration: { label: string; on: boolean; detail: string }[] = features
    ? [
        { label: "Payments", on: features.payment_methods.length > 0, detail: features.payment_methods.map((method) => METHOD_LABEL[method] ?? method).join(", ") || "No provider" },
        { label: "Transactional email", on: features.email_delivery, detail: features.email_delivery ? "Relay configured" : "Skipped" },
        { label: "Google Sign-In", on: features.google_sign_in, detail: features.google_sign_in ? "Available" : "Not configured" },
        { label: "Exercise media", on: features.exercise_media, detail: features.exercise_media ? "Served" : "Disabled (licence)" },
        { label: "Rate limiting", on: features.rate_limiting, detail: features.rate_limiting ? "On" : "Off" },
        { label: "Secure cookies", on: features.secure_cookies, detail: features.secure_cookies ? "On" : "Off (local only)" },
        { label: "Test Mode", on: features.test_mode, detail: features.test_mode ? "Available" : "Off" }
      ]
    : [];

  return (
    <div className={board.page}>
      <header className={board.header}>
        <div className={board.headText}>
          <p className={board.kicker}>Technical · ADMIN_1</p>
          <h1 className={board.title}>Platform overview</h1>
          <p className={board.lead}>Accounts, activity and the health of the API for the last {overview.range_days} days.</p>
        </div>
        <div className={board.headActions}>
          <div className={board.range} role="group" aria-label="Period">
            {OVERVIEW_RANGES.map((range) => (
              <button key={range} type="button" aria-pressed={days === range} onClick={() => setDays(range)}>
                {range}d
              </button>
            ))}
          </div>
          <Link className={board.secondary} to="/admin/test-mode">
            <FlaskConical aria-hidden="true" /> Test Mode
          </Link>
        </div>
      </header>

      <div className={board.grid}>
        <section className={joinClassNames(board.hero, board.span7)} aria-labelledby="health-title">
          <Waves className={board.heroWaves} />
          <p className={board.heroLabel} id="health-title">
            <ServerCog aria-hidden="true" /> System health
          </p>
          <p className={board.heroValue}>{status ? (healthy ? "All systems normal" : "Needs a look") : "Status unavailable"}</p>
          {status ? (
            <dl className={board.heroStats}>
              <div>
                <dt>API version</dt>
                <dd>{status.version}</dd>
              </div>
              <div>
                <dt>Uptime</dt>
                <dd>{formatDuration(status.uptime_seconds)}</dd>
              </div>
              <div>
                <dt>Database</dt>
                <dd>
                  {status.database.reachable ? "Reachable" : "Unreachable"}
                  {status.database.schema_version ? ` · schema ${status.database.schema_version}` : ""}
                  {status.database.schema_dirty ? " (dirty)" : ""}
                </dd>
              </div>
              <div>
                <dt>Connections</dt>
                <dd>{count.format(status.database.open_connections ?? 0)}</dd>
              </div>
            </dl>
          ) : null}
          <p className={board.heroNote}>
            {count.format(overview.audit.events_24h)} admin actions recorded in the last 24 hours.
          </p>
        </section>

        <section className={joinClassNames(board.card, board.span5)} aria-labelledby="attention-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="attention-title">
              <AlertTriangle aria-hidden="true" /> Needs attention
            </h2>
          </div>
          <ul className={board.queues}>
            {attention.map((row) => {
              const Icon = row.icon;
              return (
                <li key={row.title}>
                  <Link className={board.queue} to={row.to}>
                    <span className={board.queueIcon} aria-hidden="true">
                      <Icon />
                    </span>
                    <span className={board.queueText}>
                      <strong>{row.title}</strong>
                      <span>{row.text}</span>
                    </span>
                    <span className={joinClassNames(board.count, row.alert ? board.countAlert : row.hot ? board.countHot : undefined)}>
                      {count.format(row.value)}
                    </span>
                    <ChevronRight className={board.chevron} aria-hidden="true" />
                  </Link>
                </li>
              );
            })}
          </ul>
        </section>

        <div className={board.tiles}>
          <Link className={board.tile} to="/admin/users">
            <span className={board.tileIcon} aria-hidden="true">
              <Users />
            </span>
            <span className={board.tileLabel}>Member accounts</span>
            <span className={board.tileValue}>{count.format(members.total)}</span>
            <span className={board.tileHint}>
              <b>+{count.format(members.new)}</b> in {overview.range_days} days
            </span>
          </Link>
          <div className={board.tile}>
            <span className={board.tileIcon} aria-hidden="true">
              <CheckCircle2 />
            </span>
            <span className={board.tileLabel}>Verified emails</span>
            <span className={board.tileValue}>{verifiedShare}%</span>
            <span className={board.tileHint}>{count.format(members.verified)} accounts</span>
          </div>
          <div className={board.tile}>
            <span className={board.tileIcon} aria-hidden="true">
              <Activity />
            </span>
            <span className={board.tileLabel}>Active members</span>
            <span className={board.tileValue}>{count.format(members.active_7d)}</span>
            <span className={board.tileHint}>trained in the last 7 days</span>
          </div>
          <Link className={board.tile} to="/admin/trainers">
            <span className={board.tileIcon} aria-hidden="true">
              <Dumbbell />
            </span>
            <span className={board.tileLabel}>Trainers</span>
            <span className={board.tileValue}>{count.format(overview.trainers.active)}</span>
            <span className={board.tileHint}>{count.format(overview.trainers.pending_applications)} applications pending</span>
          </Link>
        </div>

        <section className={joinClassNames(board.card, board.span6)} aria-labelledby="signups-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="signups-title">
              <UserPlus aria-hidden="true" /> New accounts per day
            </h2>
            <span className={board.cardMeta}>{count.format(members.new)} in total</span>
          </div>
          <TrendChart
            title="New accounts per day"
            tone="violet"
            points={members.signups_daily.map((day) => ({ date: day.date, value: day.count }))}
            format={(value) => count.format(value)}
            describe={(point) => `${dayLabel(point.date)}: ${count.format(point.value)} new accounts`}
          />
        </section>

        <section className={joinClassNames(board.card, board.span6)} aria-labelledby="sessions-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="sessions-title">
              <Dumbbell aria-hidden="true" /> Workouts completed per day
            </h2>
            <span className={board.cardMeta}>{count.format(overview.training.sessions_completed)} in total</span>
          </div>
          <TrendChart
            title="Workouts completed per day"
            tone="mint"
            points={overview.training.sessions_daily.map((day) => ({ date: day.date, value: day.count }))}
            format={(value) => count.format(value)}
            describe={(point) => `${dayLabel(point.date)}: ${count.format(point.value)} workouts`}
          />
        </section>

        <section className={joinClassNames(board.card, board.span6)} aria-labelledby="config-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="config-title">
              <Database aria-hidden="true" /> Configuration
            </h2>
            <Link className={board.cardLink} to="/admin/configuration">
              Details <ChevronRight aria-hidden="true" />
            </Link>
          </div>
          {configuration.length === 0 ? (
            <p className={board.muted}>The configuration could not be read.</p>
          ) : (
            <ul className={board.queues}>
              {configuration.map((row) => (
                <li key={row.label} className={board.queue}>
                  <span className={board.queueIcon} aria-hidden="true">
                    {row.on ? <CheckCircle2 /> : <CircleSlash />}
                  </span>
                  <span className={board.queueText}>
                    <strong>{row.label}</strong>
                    <span>{row.detail}</span>
                  </span>
                  <span className={joinClassNames(board.status, row.on ? board.statusOk : board.statusWarn)}>
                    {row.on ? "On" : "Off"}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className={joinClassNames(board.card, board.span6)} aria-labelledby="audit-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="audit-title">
              <ScrollText aria-hidden="true" /> Latest admin actions
            </h2>
            <Link className={board.cardLink} to="/admin/audit">
              Audit log <ChevronRight aria-hidden="true" />
            </Link>
          </div>
          {audit.length === 0 ? (
            <p className={board.muted}>No administrative action recorded yet.</p>
          ) : (
            <ul className={board.queues}>
              {audit.map((entry) => (
                <li key={entry.id} className={board.queue}>
                  <span className={board.queueIcon} aria-hidden="true">
                    <ScrollText />
                  </span>
                  <span className={board.queueText}>
                    <strong>{describeAuditAction(entry.action)}</strong>
                    <span>
                      {entry.admin_id} · {formatAdminDateTime(entry.created_at)}
                    </span>
                  </span>
                  <span
                    className={joinClassNames(
                      board.status,
                      entry.outcome === "success" ? board.statusOk : entry.outcome === "denied" ? board.statusWarn : board.statusBad
                    )}
                  >
                    {entry.outcome}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </div>
  );
};
