import { CalendarDays, CheckCircle2, Dumbbell, Plus, RefreshCw, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { type Copy, useI18n } from "@/i18n/i18n";
import {
  fetchTrainerArchiveCount,
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
  archived: (count: number, limit: number) => string;
  archivedText: string;
  publishedOne: string;
  draftOne: string;
  archivedOne: string;
  newVersion: string;
  open: string;
  notices: Record<"published" | "saved" | "archived" | "restored" | "deleted", (name: string) => string>;
  dismiss: string;
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
    kicker: "Trainer · my programmes",
    title: "My programmes",
    lead: "Every plan you created. Free plans are open to everyone; Premium packages (Level 1, €14.99) include a meal plan and are recommended by the Level 1 questionnaire. Open a programme to edit, archive or delete it. Personalised Level 2 plans are made inside each client's request.",
    create: "Create a new programme",
    createText: "Step by step: details and type, weeks and training days, exercises, and (for Premium) the meal plan.",
    all: "All",
    published: "On sale",
    drafts: "Drafts",
    archived: (count, limit) => `Archived · ${count}/${limit}`,
    archivedText: "Archived programmes are off the market. People who have them keep using them. Open one to put it back on the market.",
    publishedOne: "On sale",
    draftOne: "Draft",
    archivedOne: "Archived",
    newVersion: "New version · draft",
    open: "Open · edit, archive or delete",
    notices: {
      published: (name) => `Programme published: “${name}” is now on the market.`,
      saved: (name) => `Changes saved: “${name}” is up to date.`,
      archived: (name) => `Programme archived: “${name}” is off the market.`,
      restored: (name) => `“${name}” is back on the market.`,
      deleted: (name) => `Programme deleted: “${name}”.`
    },
    dismiss: "Close the message",
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
    kicker: "Treinador · os meus programas",
    title: "Os meus programas",
    lead: "Todos os planos que criaste. Os planos grátis estão abertos a todos; os pacotes Premium (Nível 1, 14,99 €) incluem plano de refeições e são recomendados pelo questionário do Nível 1. Abre um programa para o editar, arquivar ou apagar. Os planos personalizados do Nível 2 fazem-se dentro do pedido de cada cliente.",
    create: "Criar um programa novo",
    createText: "Passo a passo: detalhes e tipo, semanas e dias de treino, exercícios e (no Premium) o plano de refeições.",
    all: "Todos",
    published: "À venda",
    drafts: "Rascunhos",
    archived: (count, limit) => `Arquivados · ${count}/${limit}`,
    archivedText: "Os programas arquivados estão fora do mercado. Quem já os tem continua a usá-los. Abre um para o voltares a pôr no mercado.",
    publishedOne: "À venda",
    draftOne: "Rascunho",
    archivedOne: "Arquivado",
    newVersion: "Nova versão · rascunho",
    open: "Abrir · editar, arquivar ou apagar",
    notices: {
      published: (name) => `Programa publicado: “${name}” já está no mercado.`,
      saved: (name) => `Alterações guardadas: “${name}” está atualizado.`,
      archived: (name) => `Programa arquivado: “${name}” saiu do mercado.`,
      restored: (name) => `“${name}” voltou ao mercado.`,
      deleted: (name) => `Programa apagado: “${name}”.`
    },
    dismiss: "Fechar a mensagem",
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
  const [filter, setFilter] = useState<"all" | "published" | "draft" | "archived">(() =>
    new URLSearchParams(window.location.search).get("view") === "archived" ? "archived" : "all"
  );
  const [archiveLimit, setArchiveLimit] = useState(15);
  // The green confirmation after publishing, archiving and so on arrives in
  // the address (?published=<name>) and is removed from it once shown.
  const location = useLocation();
  const history = useHistory();
  const [notice] = useState(() => {
    const params = new URLSearchParams(location.search);
    for (const kind of ["published", "saved", "archived", "restored", "deleted"] as const) {
      const name = params.get(kind);
      if (name) return { kind, name };
    }
    return null;
  });
  const [showNotice, setShowNotice] = useState(Boolean(notice));
  useEffect(() => {
    if (notice) history.replace({ search: "" });
  }, [history, notice]);
  // The sidebar's "Archived programmes" opens this list on that filter.
  useEffect(() => {
    const archivedView = new URLSearchParams(location.search).get("view") === "archived";
    setFilter((current) => (archivedView ? "archived" : current === "archived" ? "all" : current));
  }, [location.search]);
  const money = useMemo(
    () => new Intl.NumberFormat(locale === "pt" ? "pt-PT" : "en-IE", { style: "currency", currency: "EUR" }),
    [locale]
  );

  const load = useCallback(async (target: number) => {
    setState({ status: "loading" });
    try {
      const [result, archive] = await Promise.all([
        fetchTrainerPrograms(target, PAGE_SIZE),
        fetchTrainerArchiveCount().catch(() => null)
      ]);
      if (archive) setArchiveLimit(archive.limit);
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

  const stateOf = (program: TrainerProgram) =>
    program.archived_at ? "archived" : program.status === TrainerProgramStatus.PUBLISHED ? "published" : "draft";
  const visible = programs.filter((program) => filter === "all" || stateOf(program) === filter);
  const counts = {
    published: programs.filter((program) => stateOf(program) === "published").length,
    draft: programs.filter((program) => stateOf(program) === "draft").length,
    archived: programs.filter((program) => stateOf(program) === "archived").length
  };

  return (
    <div className={styles.page}>
      {notice && showNotice ? (
        <div className={styles.notice} role="status">
          <CheckCircle2 aria-hidden="true" />
          <p>{text.notices[notice.kind](notice.name)}</p>
          <button type="button" aria-label={text.dismiss} onClick={() => setShowNotice(false)}>
            <X aria-hidden="true" />
          </button>
        </div>
      ) : null}
      <header className={styles.header}>
        <div>
          <p className={styles.kicker}>{text.kicker}</p>
          <h1 className={styles.title}>{text.title}</h1>
          <p className={styles.lead}>{text.lead}</p>
        </div>
        <div className={styles.filters} role="group" aria-label={text.title}>
          {(["all", "published", "draft", "archived"] as const).map((option) => (
            <button key={option} type="button" aria-pressed={filter === option} onClick={() => setFilter(option)}>
              {option === "all"
                ? `${text.all} · ${programs.length}`
                : option === "published"
                  ? `${text.published} · ${counts.published}`
                  : option === "draft"
                    ? `${text.drafts} · ${counts.draft}`
                    : text.archived(counts.archived, archiveLimit)}
            </button>
          ))}
        </div>
      </header>
      {filter === "archived" ? <p className={styles.lead}>{text.archivedText}</p> : null}

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
          const state = stateOf(program);
          return (
            <li key={program.id}>
              <Link className={styles.card} to={`/trainer/programs/${program.id}`}>
                <span className={styles.cover}>
                  {cover ? <img src={cover} alt="" loading="lazy" /> : <Dumbbell aria-hidden="true" />}
                  <span className={styles.type}>{text.types[program.type] ?? program.type}</span>
                  <span
                    className={joinClassNames(
                      styles.status,
                      state === "published" ? styles.statusLive : state === "draft" ? styles.statusDraft : styles.statusArchived
                    )}
                  >
                    {state === "published"
                      ? text.publishedOne
                      : state === "draft"
                        ? program.previous_version_id
                          ? text.newVersion
                          : text.draftOne
                        : text.archivedOne}
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
                  <span className={styles.openHint}>{text.open}</span>
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
