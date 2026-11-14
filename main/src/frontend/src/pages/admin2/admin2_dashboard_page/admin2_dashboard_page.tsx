import {
  Activity,
  AlertTriangle,
  ArrowDownRight,
  ArrowUpRight,
  BadgeEuro,
  ChevronRight,
  ClipboardList,
  CreditCard,
  Dumbbell,
  Eye,
  FileCheck2,
  LifeBuoy,
  MessageSquare,
  Package,
  Plus,
  RefreshCw,
  Trophy,
  UserPlus,
  Users
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { ShareBars, TrendChart } from "@/components/admin_charts/trend_chart";
import board from "@/components/admin_board/admin_board.module.css";
import { ADMIN2_PATHS } from "@/components/admin2/admin2_navigation";
import { Waves } from "@/components/public_ui/public_ui";
import {
  FAMILY_LABEL,
  fetchBusinessOverview,
  OVERVIEW_RANGES,
  type BusinessOverview
} from "@/services/admin_overview_api";
import { joinClassNames } from "@utils/class_names";

const money = new Intl.NumberFormat("en-IE", { style: "currency", currency: "EUR" });
const moneyShort = new Intl.NumberFormat("en-IE", { style: "currency", currency: "EUR", maximumFractionDigits: 0 });
const count = new Intl.NumberFormat("en-IE");

const euros = (minor: number) => money.format(minor / 100);
const eurosShort = (minor: number) => moneyShort.format(minor / 100);

const dayLabel = (iso: string) =>
  new Date(`${iso}T00:00:00Z`).toLocaleDateString("en-IE", { day: "numeric", month: "short", timeZone: "UTC" });

// delta compares the period with the one before it.
const delta = (current: number, previous: number): { text: string; up: boolean } | null => {
  if (previous === 0) {
    return current > 0 ? { text: "New", up: true } : null;
  }
  const change = ((current - previous) / previous) * 100;
  return { text: `${change >= 0 ? "+" : ""}${change.toFixed(0)}%`, up: change >= 0 };
};

type State = { status: "loading" } | { status: "error" } | { status: "ready"; data: BusinessOverview };

// Admin2DashboardPage is the first screen of the Management Administrator: a
// live picture of the business — revenue and its trend, who is training, what
// sells, and the queues that need a decision — for the last 7, 30 or 90 days.
// Every figure comes from GET /admin/overview/business (completed, real
// purchases only; Test Mode is counted apart).
export default function Admin2DashboardPage() {
  const [days, setDays] = useState(30);
  const [state, setState] = useState<State>({ status: "loading" });

  const load = useCallback(async (range: number) => {
    setState((current) => (current.status === "ready" ? current : { status: "loading" }));
    try {
      setState({ status: "ready", data: await fetchBusinessOverview(range) });
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
        <h1 className={board.cardTitle}>Overview</h1>
        <p>The overview could not be loaded right now.</p>
        <button type="button" className={board.secondary} onClick={() => void load(days)}>
          <RefreshCw aria-hidden="true" /> Try again
        </button>
      </section>
    );
  }

  const data = state.data;
  const revenueDelta = delta(data.revenue.minor_units, data.revenue.previous_minor_units);
  const updated = new Date(data.generated_at).toLocaleTimeString("en-IE", { hour: "2-digit", minute: "2-digit" });
  const queues = data.queues;
  const premiumWaiting = queues.premium_level_2_awaiting_trainer;
  const premiumActive = queues.premium_level_2_in_progress + queues.premium_level_2_ready;

  const queueRows = [
    {
      to: ADMIN2_PATHS.PREMIUM_LEVEL_2,
      icon: ClipboardList,
      title: "Premium L2 waiting for a trainer",
      text: `${premiumActive} in progress · ${queues.premium_level_2_awaiting_intake} waiting for the intake`,
      value: premiumWaiting,
      hot: premiumWaiting > 0
    },
    {
      to: ADMIN2_PATHS.APPLICATIONS,
      icon: FileCheck2,
      title: "Trainer applications",
      text: "Pending review",
      value: data.trainers.pending_applications,
      hot: data.trainers.pending_applications > 0
    },
    {
      to: ADMIN2_PATHS.SUPPORT,
      icon: LifeBuoy,
      title: "Support conversations",
      text: "Open",
      value: queues.support_open,
      hot: queues.support_open > 0
    },
    {
      to: ADMIN2_PATHS.FEEDBACK,
      icon: MessageSquare,
      title: "Member feedback",
      text: "Waiting for approval",
      value: queues.feedback_pending,
      hot: queues.feedback_pending > 0
    },
    {
      to: ADMIN2_PATHS.SALES,
      icon: CreditCard,
      title: "Payments",
      text: `${data.purchases.pending} pending now · ${data.purchases.failed} failed in the period`,
      value: data.purchases.pending + data.purchases.failed,
      alert: data.purchases.failed > 0
    }
  ];

  const familyOrder = ["premium_level_1", "premium_level_2", "trainer", "generic"];
  const byFamily = familyOrder
    .map((family) => data.revenue.by_family.find((share) => share.label === family))
    .filter((share): share is NonNullable<typeof share> => Boolean(share))
    .map((share) => ({
      label: FAMILY_LABEL[share.label] ?? share.label,
      value: share.minor_units,
      detail: `${count.format(share.count)} ${share.count === 1 ? "sale" : "sales"}`
    }));

  const published = (family: string) =>
    data.catalogue.published_by_family.find((share) => share.label === family)?.count ?? 0;

  return (
    <div className={board.page}>
      <header className={board.header}>
        <div className={board.headText}>
          <p className={board.kicker}>Management · ADMIN_2</p>
          <h1 className={board.title}>Business overview</h1>
          <p className={board.lead}>
            Last {data.range_days} days · updated at {updated}. Real purchases only; Test Mode is counted apart.
          </p>
        </div>
        <div className={board.headActions}>
          <div className={board.range} role="group" aria-label="Period">
            {OVERVIEW_RANGES.map((range) => (
              <button key={range} type="button" aria-pressed={days === range} onClick={() => setDays(range)}>
                {range}d
              </button>
            ))}
          </div>
          <Link className={board.primary} to={ADMIN2_PATHS.PLAN_CREATE}>
            <Plus aria-hidden="true" /> New plan
          </Link>
        </div>
      </header>

      <div className={board.grid}>
        <section className={joinClassNames(board.hero, board.span7)} aria-labelledby="revenue-title">
          <Waves className={board.heroWaves} />
          <p className={board.heroLabel} id="revenue-title">
            <BadgeEuro aria-hidden="true" /> Revenue · last {data.range_days} days
          </p>
          <p className={board.heroValue}>
            {euros(data.revenue.minor_units)}
            {revenueDelta ? (
              <span className={joinClassNames(board.delta, revenueDelta.up ? board.deltaUp : board.deltaDown)}>
                {revenueDelta.up ? <ArrowUpRight aria-hidden="true" /> : <ArrowDownRight aria-hidden="true" />}
                {revenueDelta.text}
                <span className={board.srOnly}> versus the previous {data.range_days} days</span>
              </span>
            ) : null}
          </p>
          <dl className={board.heroStats}>
            <div>
              <dt>Sales</dt>
              <dd>{count.format(data.revenue.sales)}</dd>
            </div>
            <div>
              <dt>Platform share</dt>
              <dd>{euros(data.revenue.platform_minor_units)}</dd>
            </div>
            <div>
              <dt>Trainer payouts</dt>
              <dd>{euros(data.revenue.trainer_minor_units)}</dd>
            </div>
          </dl>
          <p className={board.heroNote}>
            {euros(data.revenue.all_time_minor_units)} from {count.format(data.revenue.all_time_sales)} sales since launch ·
            previous period {euros(data.revenue.previous_minor_units)}
          </p>
        </section>

        <section className={joinClassNames(board.card, board.span5)} aria-labelledby="queues-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="queues-title">
              <AlertTriangle aria-hidden="true" /> Needs attention
            </h2>
          </div>
          <ul className={board.queues}>
            {queueRows.map((row) => {
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
                    <span
                      className={joinClassNames(
                        board.count,
                        row.alert ? board.countAlert : row.hot ? board.countHot : undefined
                      )}
                    >
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
          <Link className={board.tile} to={ADMIN2_PATHS.CLIENTS}>
            <span className={board.tileIcon} aria-hidden="true">
              <Users />
            </span>
            <span className={board.tileLabel}>Members</span>
            <span className={board.tileValue}>{count.format(data.members.total)}</span>
            <span className={board.tileHint}>
              <b>+{count.format(data.members.new)}</b> in {data.range_days} days
            </span>
          </Link>
          <div className={board.tile}>
            <span className={board.tileIcon} aria-hidden="true">
              <Activity />
            </span>
            <span className={board.tileLabel}>Active members</span>
            <span className={board.tileValue}>{count.format(data.members.active_7d)}</span>
            <span className={board.tileHint}>
              trained in 7 days · {count.format(data.members.active_in_range)} in {data.range_days}
            </span>
          </div>
          <div className={board.tile}>
            <span className={board.tileIcon} aria-hidden="true">
              <Dumbbell />
            </span>
            <span className={board.tileLabel}>Workouts completed</span>
            <span className={board.tileValue}>{count.format(data.training.sessions_completed)}</span>
            <span className={board.tileHint}>in {data.range_days} days</span>
          </div>
          <Link className={board.tile} to={ADMIN2_PATHS.TRAINERS}>
            <span className={board.tileIcon} aria-hidden="true">
              <Trophy />
            </span>
            <span className={board.tileLabel}>Active trainers</span>
            <span className={board.tileValue}>{count.format(data.trainers.active)}</span>
            <span className={board.tileHint}>{count.format(data.trainers.pending_applications)} applications pending</span>
          </Link>
        </div>

        <section className={joinClassNames(board.card, board.span8)} aria-labelledby="trend-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="trend-title">
              <BadgeEuro aria-hidden="true" /> Revenue per day
            </h2>
            <span className={board.cardMeta}>{euros(data.revenue.minor_units)} in total</span>
          </div>
          <TrendChart
            title="Revenue per day"
            points={data.revenue.daily.map((day) => ({ date: day.date, value: day.minor_units }))}
            format={eurosShort}
            describe={(point) => `${dayLabel(point.date)}: ${euros(point.value)}`}
          />
        </section>

        <section className={joinClassNames(board.card, board.span4)} aria-labelledby="family-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="family-title">
              <Package aria-hidden="true" /> Revenue by category
            </h2>
          </div>
          <ShareBars items={byFamily} format={euros} empty="No sales in this period yet." />
          <p className={board.muted}>
            Prices are fixed by category: Generic free, Level 1 €14.99, Level 2 €19.49.
          </p>
        </section>

        <section className={joinClassNames(board.card, board.span6)} aria-labelledby="sessions-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="sessions-title">
              <Dumbbell aria-hidden="true" /> Workouts completed per day
            </h2>
            <span className={board.cardMeta}>
              {count.format(data.training.free_activations)} free plans started · {count.format(data.training.active_entitlements)} active entitlements
            </span>
          </div>
          <TrendChart
            title="Workouts completed per day"
            tone="mint"
            points={data.training.sessions_daily.map((day) => ({ date: day.date, value: day.count }))}
            format={(value) => count.format(value)}
            describe={(point) => `${dayLabel(point.date)}: ${count.format(point.value)} workouts`}
          />
        </section>

        <section className={joinClassNames(board.card, board.span6)} aria-labelledby="signups-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="signups-title">
              <UserPlus aria-hidden="true" /> New members per day
            </h2>
            <span className={board.cardMeta}>{count.format(data.members.new)} in {data.range_days} days</span>
          </div>
          <TrendChart
            title="New members per day"
            tone="violet"
            points={data.members.signups_daily.map((day) => ({ date: day.date, value: day.count }))}
            format={(value) => count.format(value)}
            describe={(point) => `${dayLabel(point.date)}: ${count.format(point.value)} new members`}
          />
        </section>

        <section className={joinClassNames(board.card, board.span6)} aria-labelledby="top-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="top-title">
              <Trophy aria-hidden="true" /> Best sellers
            </h2>
            <Link className={board.cardLink} to={ADMIN2_PATHS.SALES}>
              All sales <ChevronRight aria-hidden="true" />
            </Link>
          </div>
          {data.top_selling.length === 0 ? (
            <p className={board.muted}>No sales in the last {data.range_days} days.</p>
          ) : (
            <ol className={board.ranks}>
              {data.top_selling.map((program) => (
                <li key={program.id} className={board.rank}>
                  <span className={board.rankText}>
                    <strong>{program.name}</strong>
                    <span>
                      {FAMILY_LABEL[program.family] ?? program.family} · {count.format(program.count)}{" "}
                      {program.count === 1 ? "sale" : "sales"}
                    </span>
                  </span>
                  <span className={board.rankValue}>{euros(program.minor_units)}</span>
                </li>
              ))}
            </ol>
          )}
        </section>

        <section className={joinClassNames(board.card, board.span6)} aria-labelledby="viewed-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="viewed-title">
              <Eye aria-hidden="true" /> Most viewed plans
            </h2>
            <span className={board.cardMeta}>All time</span>
          </div>
          {data.top_viewed.length === 0 ? (
            <p className={board.muted}>No plan has been viewed yet.</p>
          ) : (
            <ol className={board.ranks}>
              {data.top_viewed.map((program) => (
                <li key={program.id} className={board.rank}>
                  <span className={board.rankText}>
                    <strong>{program.name}</strong>
                    <span>{FAMILY_LABEL[program.family] ?? program.family}</span>
                  </span>
                  <span className={board.rankValue}>{count.format(program.count)} views</span>
                </li>
              ))}
            </ol>
          )}
        </section>

        <section className={joinClassNames(board.card, board.span12)} aria-labelledby="catalogue-title">
          <div className={board.cardHead}>
            <h2 className={board.cardTitle} id="catalogue-title">
              <Package aria-hidden="true" /> Catalogue
            </h2>
            <Link className={board.cardLink} to={ADMIN2_PATHS.PLANS}>
              Manage plans <ChevronRight aria-hidden="true" />
            </Link>
          </div>
          <dl className={board.facts}>
            <div>
              <dt>Generic plans live</dt>
              <dd>{count.format(published("generic"))}</dd>
            </div>
            <div>
              <dt>Level 1 packages live</dt>
              <dd>{count.format(published("premium_level_1"))}</dd>
            </div>
            <div>
              <dt>Level 2 offers live</dt>
              <dd>{count.format(published("premium_level_2"))}</dd>
            </div>
            <div>
              <dt>Trainer plans live</dt>
              <dd>{count.format(published("trainer"))}</dd>
            </div>
            <div>
              <dt>Platform drafts</dt>
              <dd>{count.format(data.catalogue.platform_drafts)}</dd>
            </div>
            <div>
              <dt>Test purchases</dt>
              <dd>{count.format(data.purchases.test)}</dd>
            </div>
          </dl>
        </section>
      </div>
    </div>
  );
}
