import { CalendarDays, Dumbbell, Pencil, Plus, RefreshCw, Search, Trash2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";

import board from "@/components/admin_board/admin_board.module.css";
import people from "@/components/admin_board/people.module.css";
import { AppLoading } from "@/components/app_shell/app_loading";
import {
  deleteGenericProgram,
  fetchAdminSalesByProgramIds,
  fetchGenericPrograms,
  isProgramProductType,
  PROGRAM_PRODUCT_TYPE,
  ProgramStatusEnum,
  type AdminProgramSale,
  type GenericProgramLevel,
  type GenericProgramSummary,
  type ProgramProductTypeValue
} from "@/services/admin2_api";
import { joinClassNames } from "@utils/class_names";
import { publicFileUrl } from "@utils/http_client";

import styles from "./admin2_plans_page.module.css";

const PAGE_SIZE = 24;

const FAMILIES: Array<{ id: ProgramProductTypeValue; label: string; price: string; text: string }> = [
  { id: PROGRAM_PRODUCT_TYPE.GENERIC, label: "Generic", price: "Free", text: "Free training plans of the public catalogue." },
  { id: PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1, label: "Premium Level 1", price: "€14.99", text: "Training and nutrition packages, delivered immediately." },
  { id: PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_2, label: "Premium Level 2", price: "€19.49", text: "Personalised offers: a trainer writes each client's plan." }
];

const LEVELS: Array<GenericProgramLevel | ""> = ["", "Beginner", "Intermediate", "Advanced"];

const money = new Intl.NumberFormat("en-IE", { style: "currency", currency: "EUR" });

const familyFromQuery = (search: string): ProgramProductTypeValue => {
  const family = new URLSearchParams(search).get("family");
  return isProgramProductType(family) ? family : PROGRAM_PRODUCT_TYPE.GENERIC;
};

type State = { status: "loading" } | { status: "error" } | { status: "ready" };

// Admin2PlansPage lists the platform plans of one category as cards (cover,
// status, schedule, sales) and opens the plan builder to create or edit one.
// Prices are fixed by category, so the list filters by category, search,
// difficulty and status only.
export default function Admin2PlansPage() {
  const history = useHistory();
  const location = useLocation();
  const family = familyFromQuery(location.search);
  const familyInfo = FAMILIES.find((entry) => entry.id === family) ?? FAMILIES[0];

  const [state, setState] = useState<State>({ status: "loading" });
  const [programs, setPrograms] = useState<GenericProgramSummary[]>([]);
  const [sales, setSales] = useState<Record<string, AdminProgramSale>>({});
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  const [level, setLevel] = useState<GenericProgramLevel | "">("");
  const [status, setStatus] = useState<"all" | "published" | "draft">("all");
  const [deleting, setDeleting] = useState<string | null>(null);
  const [message, setMessage] = useState("");

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(query), 300);
    return () => window.clearTimeout(timer);
  }, [query]);

  const load = useCallback(async () => {
    setState((current) => (current.status === "ready" ? current : { status: "loading" }));
    try {
      const result = await fetchGenericPrograms(family, {
        page,
        limit: PAGE_SIZE,
        search: debounced.trim() || undefined,
        level: level || undefined
      });
      setPrograms(result.programs);
      setTotal(result.pagination.total);
      setTotalPages(result.pagination.total_pages);
      setState({ status: "ready" });
      if (result.programs.length > 0) {
        fetchAdminSalesByProgramIds(result.programs.map((program) => program.id))
          .then((rows) => setSales(Object.fromEntries(rows.map((row) => [row.program.id, row]))))
          .catch(() => setSales({}));
      }
    } catch {
      setState({ status: "error" });
    }
  }, [family, page, debounced, level]);

  useEffect(() => {
    void load();
  }, [load]);

  const changeFamily = (next: ProgramProductTypeValue) => {
    setPage(1);
    history.push({ pathname: "/admin2/plans", search: `?family=${next}` });
  };

  const remove = async (program: GenericProgramSummary) => {
    if (!window.confirm(`Delete "${program.name}"? It disappears from the catalogue (its data is kept).`)) return;
    setDeleting(program.id);
    setMessage("");
    try {
      await deleteGenericProgram(program.product_type, program.id);
      await load();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "The plan could not be deleted.");
    } finally {
      setDeleting(null);
    }
  };

  const visible = programs.filter((program) => status === "all" || program.status === status);

  return (
    <div className={board.page}>
      <header className={board.header}>
        <div className={board.headText}>
          <p className={board.kicker}>Catalogue</p>
          <h1 className={board.title}>Plans</h1>
          <p className={board.lead}>
            {familyInfo.text} Price: {familyInfo.price}, fixed by the category.
          </p>
        </div>
        <div className={board.headActions}>
          <Link className={board.primary} to={`/admin2/plans/create?family=${family}`}>
            <Plus aria-hidden="true" /> New {familyInfo.label} plan
          </Link>
        </div>
      </header>

      <div className={styles.families} role="tablist" aria-label="Category">
        {FAMILIES.map((entry) => (
          <button
            key={entry.id}
            type="button"
            role="tab"
            aria-selected={entry.id === family}
            className={styles.family}
            onClick={() => changeFamily(entry.id)}
          >
            <strong>{entry.label}</strong>
            <span>{entry.price}</span>
          </button>
        ))}
      </div>

      <div className={people.toolbar}>
        <label className={people.search}>
          <Search aria-hidden="true" />
          <span className={board.srOnly}>Search plans</span>
          <input type="search" placeholder="Search by name or description" value={query} onChange={(event) => {
            setQuery(event.target.value);
            setPage(1);
          }} />
        </label>
        {family !== PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_2 ? (
          <div className={board.range} role="group" aria-label="Difficulty">
            {LEVELS.map((option) => (
              <button key={option || "all"} type="button" aria-pressed={level === option} onClick={() => {
                setLevel(option);
                setPage(1);
              }}>
                {option || "Any level"}
              </button>
            ))}
          </div>
        ) : null}
        <div className={board.range} role="group" aria-label="Status">
          {(["all", "published", "draft"] as const).map((option) => (
            <button key={option} type="button" aria-pressed={status === option} onClick={() => setStatus(option)}>
              {option === "all" ? "All" : option === "published" ? "Live" : "Drafts"}
            </button>
          ))}
        </div>
      </div>

      {message ? <p className={people.error} role="alert">{message}</p> : null}

      {state.status === "loading" ? (
        <AppLoading />
      ) : state.status === "error" ? (
        <section className={board.stateCard}>
          <p>The plans could not be loaded.</p>
          <button type="button" className={board.secondary} onClick={() => void load()}>
            <RefreshCw aria-hidden="true" /> Try again
          </button>
        </section>
      ) : (
        <>
          <p className={board.muted}>
            {total} {total === 1 ? "plan" : "plans"} in {familyInfo.label}
          </p>
          <ul className={styles.grid}>
            <li>
              <Link className={styles.create} to={`/admin2/plans/create?family=${family}`}>
                <span className={styles.createIcon} aria-hidden="true">
                  <Plus />
                </span>
                <strong>New {familyInfo.label} plan</strong>
                <span>Cover, goal, weeks and days first, then the exercises.</span>
              </Link>
            </li>
            {visible.map((program) => {
              const cover = publicFileUrl(program.cover_image_url);
              const sale = sales[program.id];
              const live = program.status === ProgramStatusEnum.PUBLISHED;
              const editTo = `/admin2/plans/create?edit=${program.id}&family=${program.product_type}`;
              return (
                <li key={program.id} className={styles.card}>
                  <Link className={styles.cover} to={`/admin2/plans/${program.id}?family=${program.product_type}`} aria-label={`Open ${program.name}`}>
                    {cover ? <img src={cover} alt="" loading="lazy" /> : <Dumbbell aria-hidden="true" />}
                    <span className={joinClassNames(styles.status, live ? styles.statusLive : styles.statusDraft)}>
                      {live ? "Live" : "Draft"}
                    </span>
                  </Link>
                  <div className={styles.body}>
                    <Link className={styles.name} to={`/admin2/plans/${program.id}?family=${program.product_type}`}>
                      {program.name}
                    </Link>
                    <span className={styles.meta}>
                      {program.level ? <span>{program.level}</span> : null}
                      {program.duration_weeks ? (
                        <span>
                          <CalendarDays aria-hidden="true" /> {program.duration_weeks} wk
                        </span>
                      ) : null}
                      {program.frequency_per_week ? <span>{program.frequency_per_week}×/week</span> : null}
                    </span>
                    <dl className={styles.figures}>
                      <div>
                        <dt>Price</dt>
                        <dd>{program.price_minor_units === 0 ? "Free" : money.format(program.price_minor_units / 100)}</dd>
                      </div>
                      <div>
                        <dt>Sales</dt>
                        <dd>{sale ? sale.real_sales : "—"}</dd>
                      </div>
                      <div>
                        <dt>Revenue</dt>
                        <dd>{sale ? money.format(sale.revenue_minor_units / 100) : "—"}</dd>
                      </div>
                    </dl>
                    <div className={styles.actions}>
                      <Link className={board.secondary} to={editTo}>
                        <Pencil aria-hidden="true" /> Edit
                      </Link>
                      <button
                        type="button"
                        className={styles.delete}
                        disabled={deleting === program.id}
                        aria-label={`Delete ${program.name}`}
                        onClick={() => void remove(program)}
                      >
                        <Trash2 aria-hidden="true" />
                      </button>
                    </div>
                  </div>
                </li>
              );
            })}
          </ul>
          {totalPages > 1 ? (
            <div className={people.pager}>
              <button type="button" className={board.secondary} disabled={page <= 1} onClick={() => setPage(page - 1)}>
                Previous
              </button>
              <span>
                Page {page} of {totalPages}
              </span>
              <button type="button" className={board.secondary} disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
                Next
              </button>
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}
