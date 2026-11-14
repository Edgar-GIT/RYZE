import { CalendarDays, Dumbbell, Plus, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { type Copy, useI18n } from "@/i18n/i18n";
import {
  fetchTrainerPrograms,
  TrainerProgramStatus,
  type TrainerProgram,
  type TrainerProgramTypeValue
} from "@/services/trainer_api";
import { joinClassNames } from "@utils/class_names";
import { publicFileUrl } from "@utils/http_client";

import styles from "./trainer_programs_page.module.css";

const PAGE_SIZE = 24;

const COPY: Copy<{
  kicker: string;
  title: string;
  lead: string;
  create: string;
  createText: string;
  all: string;
  published: string;
  drafts: string;
  publishedOne: string;
  draftOne: string;
  empty: string;
  emptyText: string;
  error: string;
  retry: string;
  types: Record<TrainerProgramTypeValue, string>;
  free: string;
  weeks: (n: number) => string;
  daysPerWeek: (n: number) => string;
  updated: (date: string) => string;
  previous: string;
  next: string;
  page: (current: number, total: number) => string;
}> = {
  en: {
    kicker: "Trainer",
    title: "Programmes",
    lead: "Build plans for your clients and for the store. Prices follow the type: Free, Premium €14.99, Personalized €19.49.",
    create: "New programme",
    createText: "Cover, goal, weeks and days, then the exercises.",
    all: "All",
    published: "Published",
    drafts: "Drafts",
    publishedOne: "Published",
    draftOne: "Draft",
    empty: "No programmes yet",
    emptyText: "Create your first programme: choose the type and the cover, then fill the weeks day by day.",
    error: "Your programmes could not be loaded.",
    retry: "Try again",
    types: { free: "Free", premium: "Premium", personalized: "Personalized" },
    free: "Free",
    weeks: (n) => `${n} ${n === 1 ? "week" : "weeks"}`,
    daysPerWeek: (n) => `${n}×/week`,
    updated: (date) => `Updated ${date}`,
    previous: "Previous",
    next: "Next",
    page: (current, total) => `Page ${current} of ${total}`
  },
  pt: {
    kicker: "Treinador",
    title: "Programas",
    lead: "Cria planos para os teus clientes e para a loja. O preço segue o tipo: Grátis, Premium 14,99 €, Personalizado 19,49 €.",
    create: "Novo programa",
    createText: "Capa, objetivo, semanas e dias, depois os exercícios.",
    all: "Todos",
    published: "Publicados",
    drafts: "Rascunhos",
    publishedOne: "Publicado",
    draftOne: "Rascunho",
    empty: "Ainda sem programas",
    emptyText: "Cria o teu primeiro programa: escolhe o tipo e a capa e depois preenche as semanas dia a dia.",
    error: "Não foi possível carregar os teus programas.",
    retry: "Tentar de novo",
    types: { free: "Grátis", premium: "Premium", personalized: "Personalizado" },
    free: "Grátis",
    weeks: (n) => `${n} ${n === 1 ? "semana" : "semanas"}`,
    daysPerWeek: (n) => `${n}×/semana`,
    updated: (date) => `Atualizado a ${date}`,
    previous: "Anterior",
    next: "Seguinte",
    page: (current, total) => `Página ${current} de ${total}`
  }
};

type State = { status: "loading" } | { status: "error" } | { status: "ready" };

// TrainerProgramsPage lists the trainer's own programmes as cards (cover,
// type and fixed price, status) and opens the plan builder to create or edit.
export const TrainerProgramsPage = () => {
  const { locale, formatDate } = useI18n();
  const text = COPY[locale];
  const [state, setState] = useState<State>({ status: "loading" });
  const [programs, setPrograms] = useState<TrainerProgram[]>([]);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [filter, setFilter] = useState<"all" | "published" | "draft">("all");
  const money = useMemo(
    () => new Intl.NumberFormat(locale === "pt" ? "pt-PT" : "en-IE", { style: "currency", currency: "EUR" }),
    [locale]
  );

  const load = useCallback(async (target: number) => {
    setState({ status: "loading" });
    try {
      const result = await fetchTrainerPrograms(target, PAGE_SIZE);
      setPrograms(result.programs);
      setPage(result.pagination.page);
      setTotalPages(result.pagination.total_pages);
      setState({ status: "ready" });
    } catch {
      setState({ status: "error" });
    }
  }, []);

  useEffect(() => {
    void load(1);
  }, [load]);

  if (state.status === "loading") {
    return <AppLoading />;
  }

  if (state.status === "error") {
    return (
      <section className={styles.state}>
        <p>{text.error}</p>
        <button type="button" className={styles.ghost} onClick={() => void load(page)}>
          <RefreshCw aria-hidden="true" /> {text.retry}
        </button>
      </section>
    );
  }

  const visible = programs.filter((program) => filter === "all" || program.status === filter);
  const publishedCount = programs.filter((program) => program.status === TrainerProgramStatus.PUBLISHED).length;

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <p className={styles.kicker}>{text.kicker}</p>
          <h1 className={styles.title}>{text.title}</h1>
          <p className={styles.lead}>{text.lead}</p>
        </div>
        <div className={styles.filters} role="group" aria-label={text.title}>
          {(["all", "published", "draft"] as const).map((option) => (
            <button key={option} type="button" aria-pressed={filter === option} onClick={() => setFilter(option)}>
              {option === "all"
                ? `${text.all} · ${programs.length}`
                : option === "published"
                  ? `${text.published} · ${publishedCount}`
                  : `${text.drafts} · ${programs.length - publishedCount}`}
            </button>
          ))}
        </div>
      </header>

      <ul className={styles.grid}>
        <li>
          <Link className={styles.create} to="/trainer/programs/new">
            <span className={styles.createIcon} aria-hidden="true">
              <Plus />
            </span>
            <strong>{text.create}</strong>
            <span>{text.createText}</span>
          </Link>
        </li>
        {visible.map((program) => {
          const cover = publicFileUrl(program.cover_image_url);
          const published = program.status === TrainerProgramStatus.PUBLISHED;
          return (
            <li key={program.id}>
              <Link className={styles.card} to={`/trainer/programs/${program.id}`}>
                <span className={styles.cover}>
                  {cover ? <img src={cover} alt="" loading="lazy" /> : <Dumbbell aria-hidden="true" />}
                  <span className={styles.type}>{text.types[program.type] ?? program.type}</span>
                  <span className={joinClassNames(styles.status, published ? styles.statusLive : styles.statusDraft)}>
                    {published ? text.publishedOne : text.draftOne}
                  </span>
                </span>
                <span className={styles.body}>
                  <strong>{program.name}</strong>
                  <span className={styles.meta}>
                    {program.duration_weeks ? (
                      <span>
                        <CalendarDays aria-hidden="true" /> {text.weeks(program.duration_weeks)}
                      </span>
                    ) : null}
                    {program.frequency_per_week ? <span>{text.daysPerWeek(program.frequency_per_week)}</span> : null}
                  </span>
                  <span className={styles.foot}>
                    <b>{program.price_minor_units === 0 ? text.free : money.format(program.price_minor_units / 100)}</b>
                    <em>{text.updated(formatDate(program.updated_at, { day: "numeric", month: "short" }))}</em>
                  </span>
                </span>
              </Link>
            </li>
          );
        })}
      </ul>

      {programs.length === 0 ? (
        <section className={styles.state}>
          <h2>{text.empty}</h2>
          <p>{text.emptyText}</p>
        </section>
      ) : null}

      {totalPages > 1 ? (
        <div className={styles.pager}>
          <button type="button" className={styles.ghost} disabled={page <= 1} onClick={() => void load(page - 1)}>
            {text.previous}
          </button>
          <span>{text.page(page, totalPages)}</span>
          <button type="button" className={styles.ghost} disabled={page >= totalPages} onClick={() => void load(page + 1)}>
            {text.next}
          </button>
        </div>
      ) : null}
    </div>
  );
};
