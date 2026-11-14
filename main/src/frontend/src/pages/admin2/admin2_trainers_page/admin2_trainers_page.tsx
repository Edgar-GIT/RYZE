import { Check, ExternalLink, Percent, RefreshCw, RotateCcw, Search, UserRoundCheck } from "lucide-react";
import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import board from "@/components/admin_board/admin_board.module.css";
import people from "@/components/admin_board/people.module.css";
import { ADMIN2_PATHS } from "@/components/admin2/admin2_navigation";
import { fetchAdminTrainers, fullName, type AdminTrainer } from "@/services/admin_api";
import {
  deleteCommissionRule,
  fetchCommissionResolution,
  updateCommissionRule,
  type AdminCommissionResolution
} from "@/services/admin2_api";
import { apiGet } from "@utils/http_client";
import { joinClassNames } from "@utils/class_names";

interface CommerceSettings {
  default_platform_commission_bps: number;
  max_commission_bps: number;
}

const PAGE_SIZE = 50;
const QUICK_PERCENTAGES = [20, 25, 30, 35];

// Commission is shown and typed as a plain percentage ("30" = 30 %); the API
// keeps basis points (1 % = 100 bps).
export const percentOf = (bps: number) => {
  const value = bps / 100;
  return Number.isInteger(value) ? String(value) : value.toFixed(2).replace(/0+$/, "").replace(/\.$/, "");
};

export const parsePercent = (raw: string): number | null => {
  const cleaned = raw.replace("%", "").replace(",", ".").trim();
  if (cleaned === "") return null;
  const value = Number(cleaned);
  if (!Number.isFinite(value)) return null;
  return Math.round(value * 100);
};

const initials = (trainer: AdminTrainer) =>
  `${trainer.first_name[0] ?? ""}${trainer.last_name[0] ?? ""}`.toUpperCase() || trainer.email[0]?.toUpperCase() || "T";

const CommissionEditor = ({
  trainer,
  resolution,
  settings,
  onChanged
}: {
  trainer: AdminTrainer;
  resolution: AdminCommissionResolution | undefined;
  settings: CommerceSettings | null;
  onChanged: (trainerId: string, next: AdminCommissionResolution) => void;
}) => {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const maxPercent = settings ? settings.max_commission_bps / 100 : 100;

  const open = () => {
    setValue(resolution ? percentOf(resolution.commission_bps) : "");
    setError("");
    setEditing(true);
  };

  const save = async (event: FormEvent) => {
    event.preventDefault();
    const bps = parsePercent(value);
    if (bps === null || bps < 0 || bps > maxPercent * 100) {
      setError(`Enter a percentage between 0 and ${maxPercent}.`);
      return;
    }
    setSaving(true);
    setError("");
    try {
      await updateCommissionRule(trainer.id, bps);
      onChanged(trainer.id, await fetchCommissionResolution(trainer.id));
      setEditing(false);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The commission could not be saved.");
    } finally {
      setSaving(false);
    }
  };

  const useDefault = async () => {
    setSaving(true);
    setError("");
    try {
      await deleteCommissionRule(trainer.id);
      onChanged(trainer.id, await fetchCommissionResolution(trainer.id));
      setEditing(false);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The default could not be restored.");
    } finally {
      setSaving(false);
    }
  };

  if (!editing) {
    return (
      <div className={people.commission}>
        <span className={people.commissionValue}>
          {resolution ? `${percentOf(resolution.commission_bps)}%` : "—"}
        </span>
        <span className={people.commissionNote}>
          {resolution ? (resolution.is_override ? "Custom rate" : "Platform default") : "Loading"}
        </span>
        <button type="button" className={people.linkButton} onClick={open}>
          Change
        </button>
      </div>
    );
  }

  return (
    <form className={people.commissionForm} onSubmit={save} noValidate>
      <label className={people.percentField}>
        <span className={board.srOnly}>Commission for {fullName(trainer)} in percent</span>
        <input
          inputMode="decimal"
          value={value}
          autoFocus
          onChange={(event) => setValue(event.target.value)}
          aria-describedby={error ? `commission-error-${trainer.id}` : undefined}
        />
        <Percent aria-hidden="true" />
      </label>
      <div className={people.quick} role="group" aria-label="Quick values">
        {QUICK_PERCENTAGES.map((percent) => (
          <button
            key={percent}
            type="button"
            aria-pressed={value === String(percent)}
            onClick={() => setValue(String(percent))}
          >
            {percent}%
          </button>
        ))}
      </div>
      <div className={people.formActions}>
        <button type="submit" className={board.primary} disabled={saving}>
          <Check aria-hidden="true" /> {saving ? "Saving…" : "Save"}
        </button>
        {resolution?.is_override ? (
          <button type="button" className={board.secondary} disabled={saving} onClick={() => void useDefault()}>
            <RotateCcw aria-hidden="true" /> Use default
            {settings ? ` (${percentOf(settings.default_platform_commission_bps)}%)` : ""}
          </button>
        ) : null}
        <button type="button" className={people.linkButton} onClick={() => setEditing(false)}>
          Cancel
        </button>
      </div>
      {error ? (
        <p className={people.error} id={`commission-error-${trainer.id}`} role="alert">
          {error}
        </p>
      ) : null}
    </form>
  );
};

type State = { status: "loading" } | { status: "error" } | { status: "ready" };

// Admin2TrainersPage lists the trainer network with each trainer's platform
// commission. The commission is the share RYZE keeps when one of the trainer's
// plans is sold; prices themselves are fixed by category and never set here.
export default function Admin2TrainersPage() {
  const [state, setState] = useState<State>({ status: "loading" });
  const [trainers, setTrainers] = useState<AdminTrainer[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [resolutions, setResolutions] = useState<Record<string, AdminCommissionResolution>>({});
  const [settings, setSettings] = useState<CommerceSettings | null>(null);
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | "active" | "disabled">("all");

  const load = useCallback(async (targetPage: number) => {
    setState({ status: "loading" });
    try {
      const result = await fetchAdminTrainers(targetPage, PAGE_SIZE);
      setTrainers(result.trainers);
      setTotal(result.pagination.total);
      setTotalPages(result.pagination.total_pages);
      setPage(result.pagination.page);
      setState({ status: "ready" });
      const entries = await Promise.all(
        result.trainers.map(async (trainer) => {
          try {
            return [trainer.id, await fetchCommissionResolution(trainer.id)] as const;
          } catch {
            return null;
          }
        })
      );
      setResolutions(Object.fromEntries(entries.filter((entry): entry is NonNullable<typeof entry> => entry !== null)));
    } catch {
      setState({ status: "error" });
    }
  }, []);

  useEffect(() => {
    void load(1);
    apiGet<CommerceSettings>("/admin/settings/commerce")
      .then(setSettings)
      .catch(() => undefined);
  }, [load]);

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return trainers.filter((trainer) => {
      if (statusFilter !== "all" && (trainer.status === "active" ? "active" : "disabled") !== statusFilter) {
        return false;
      }
      if (!needle) return true;
      return `${fullName(trainer)} ${trainer.email}`.toLowerCase().includes(needle);
    });
  }, [trainers, query, statusFilter]);

  const customRates = Object.values(resolutions).filter((resolution) => resolution.is_override).length;

  if (state.status === "loading") {
    return <AppLoading />;
  }

  if (state.status === "error") {
    return (
      <section className={board.stateCard}>
        <h1 className={board.cardTitle}>Trainers</h1>
        <p>The trainers could not be loaded right now.</p>
        <button type="button" className={board.secondary} onClick={() => void load(page)}>
          <RefreshCw aria-hidden="true" /> Try again
        </button>
      </section>
    );
  }

  return (
    <div className={board.page}>
      <header className={board.header}>
        <div className={board.headText}>
          <p className={board.kicker}>Business</p>
          <h1 className={board.title}>Trainers & commissions</h1>
          <p className={board.lead}>
            The commission is the share RYZE keeps when a trainer's plan is sold. Type it as a percentage: 30 means
            30%. Default {settings ? `${percentOf(settings.default_platform_commission_bps)}%` : "30%"}.
          </p>
        </div>
        <div className={board.headActions}>
          <Link className={board.secondary} to={ADMIN2_PATHS.APPLICATIONS}>
            <UserRoundCheck aria-hidden="true" /> Applications
          </Link>
        </div>
      </header>

      <dl className={board.facts}>
        <div>
          <dt>Trainers</dt>
          <dd>{total}</dd>
        </div>
        <div>
          <dt>Custom rates (this page)</dt>
          <dd>{customRates}</dd>
        </div>
        <div>
          <dt>Default commission</dt>
          <dd>{settings ? `${percentOf(settings.default_platform_commission_bps)}%` : "—"}</dd>
        </div>
      </dl>

      <div className={people.toolbar}>
        <label className={people.search}>
          <Search aria-hidden="true" />
          <span className={board.srOnly}>Search trainers</span>
          <input
            type="search"
            placeholder="Search by name or email"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
        </label>
        <div className={board.range} role="group" aria-label="Status">
          {(["all", "active", "disabled"] as const).map((option) => (
            <button key={option} type="button" aria-pressed={statusFilter === option} onClick={() => setStatusFilter(option)}>
              {option === "all" ? "All" : option === "active" ? "Active" : "Disabled"}
            </button>
          ))}
        </div>
      </div>

      {visible.length === 0 ? (
        <section className={board.card}>
          <p className={board.muted}>{trainers.length === 0 ? "No trainers yet. Approved applications appear here." : "No trainer matches this search."}</p>
        </section>
      ) : (
        <ul className={people.list}>
          {visible.map((trainer) => (
            <li key={trainer.id} className={people.row}>
              <span className={people.avatar} aria-hidden="true">
                {initials(trainer)}
              </span>
              <span className={people.identity}>
                <strong>{fullName(trainer)}</strong>
                <span>{trainer.email}</span>
                <span className={people.meta}>
                  <span className={joinClassNames(board.status, trainer.status === "active" ? board.statusOk : board.statusBad)}>
                    {trainer.status === "active" ? "Active" : "Disabled"}
                  </span>
                  Trainer since {new Date(trainer.created_at).toLocaleDateString("en-IE", { month: "short", year: "numeric" })}
                  <Link className={people.inlineLink} to={`/coaches/${trainer.id}`} target="_blank" rel="noreferrer">
                    Public profile <ExternalLink aria-hidden="true" />
                  </Link>
                </span>
              </span>
              <CommissionEditor
                trainer={trainer}
                resolution={resolutions[trainer.id]}
                settings={settings}
                onChanged={(id, next) => setResolutions((current) => ({ ...current, [id]: next }))}
              />
            </li>
          ))}
        </ul>
      )}

      {totalPages > 1 ? (
        <div className={people.pager}>
          <button type="button" className={board.secondary} disabled={page <= 1} onClick={() => void load(page - 1)}>
            Previous
          </button>
          <span>
            Page {page} of {totalPages}
          </span>
          <button type="button" className={board.secondary} disabled={page >= totalPages} onClick={() => void load(page + 1)}>
            Next
          </button>
        </div>
      ) : null}
    </div>
  );
}
