import { Archive, ArchiveRestore, ArrowLeft, Dumbbell, PencilLine, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";

import { useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./program_overview.module.css";

export type ProgramOverviewState = "published" | "draft" | "archived";

const COPY = {
  en: {
    states: { published: "Published · on sale", draft: "Draft · not visible to anyone", archived: "Archived · off the market" },
    edit: "Edit programme",
    editHint: "Change the details, the weeks and days, the exercises and (for Premium) the meal plan.",
    continueDraft: "Continue editing",
    archive: "Archive",
    archiveHint: (count: number, limit: number) =>
      `Takes it off the market for a while. People who already have it keep using it. You can bring it back at any time. Archived: ${count} of ${limit}.`,
    archiveFull: (limit: number) => `You already have ${limit} archived programmes: restore or delete one before archiving another.`,
    restore: "Put back on the market",
    restoreHint: "Makes it visible and available to buy or start again.",
    remove: "Delete",
    removeHint: "Removes it for good. People who already bought it keep their access.",
    removeConfirm: (name: string) => `Delete “${name}”? This cannot be undone.`,
    archiveConfirm: (name: string) => `Archive “${name}”? It leaves the market until you put it back.`,
    working: "Working…",
    back: "Back to the programmes"
  },
  pt: {
    states: { published: "Publicado · à venda", draft: "Rascunho · ninguém o vê", archived: "Arquivado · fora do mercado" },
    edit: "Editar programa",
    editHint: "Muda os detalhes, as semanas e os dias, os exercícios e (no Premium) o plano de refeições.",
    continueDraft: "Continuar a editar",
    archive: "Arquivar",
    archiveHint: (count: number, limit: number) =>
      `Tira-o do mercado por uns tempos. Quem já o tem continua a usá-lo. Podes voltar a pô-lo à venda quando quiseres. Arquivados: ${count} de ${limit}.`,
    archiveFull: (limit: number) => `Já tens ${limit} programas arquivados: retoma ou apaga um antes de arquivares outro.`,
    restore: "Voltar a pôr no mercado",
    restoreHint: "Fica outra vez visível e disponível para comprar ou começar.",
    remove: "Apagar",
    removeHint: "Remove-o de vez. Quem já o comprou mantém o acesso.",
    removeConfirm: (name: string) => `Apagar “${name}”? Não é possível desfazer.`,
    archiveConfirm: (name: string) => `Arquivar “${name}”? Sai do mercado até o voltares a pôr.`,
    working: "A processar…",
    back: "Voltar aos programas"
  }
} as const;

// ProgramOverview is the first screen of a programme (owner request,
// 2026-11-27): its picture and name, what state it is in, and the three things
// you do with it — edit (green), archive or put back, delete — each explained.
export const ProgramOverview = ({
  name,
  coverUrl,
  kicker,
  facts,
  state,
  editTo,
  backTo,
  archive,
  onDelete
}: {
  name: string;
  coverUrl: string;
  kicker: string;
  facts: string[];
  state: ProgramOverviewState;
  editTo: string;
  backTo: string;
  archive?: { count: number; limit: number; onArchive: () => Promise<void>; onRestore: () => Promise<void> };
  onDelete?: () => Promise<void>;
}) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const [busy, setBusy] = useState("");

  const run = async (key: string, action: () => Promise<void>) => {
    setBusy(key);
    try {
      await action();
    } finally {
      setBusy("");
    }
  };

  const archiveFull = archive ? archive.count >= archive.limit : false;

  return (
    <section className={styles.overview} aria-labelledby="program-overview-title">
      <Link className={styles.back} to={backTo}>
        <ArrowLeft aria-hidden="true" /> {text.back}
      </Link>
      <div className={styles.card}>
        <div className={styles.cover}>{coverUrl ? <img src={coverUrl} alt="" /> : <Dumbbell aria-hidden="true" />}</div>
        <div className={styles.body}>
          <p className={styles.kicker}>{kicker}</p>
          <h1 id="program-overview-title" className={styles.title}>
            {name}
          </h1>
          <span className={joinClassNames(styles.state, styles[state])}>{text.states[state]}</span>
          {facts.length > 0 ? (
            <ul className={styles.facts}>
              {facts.map((fact) => (
                <li key={fact}>{fact}</li>
              ))}
            </ul>
          ) : null}
        </div>
      </div>

      <div className={styles.actions}>
        <div className={styles.action}>
          <Link className={styles.edit} to={editTo}>
            <PencilLine aria-hidden="true" /> {state === "draft" ? text.continueDraft : text.edit}
          </Link>
          <p>{text.editHint}</p>
        </div>

        {archive && state === "published" ? (
          <div className={styles.action}>
            <button
              type="button"
              className={styles.secondary}
              disabled={Boolean(busy) || archiveFull}
              onClick={() => {
                if (window.confirm(text.archiveConfirm(name))) void run("archive", archive.onArchive);
              }}
            >
              <Archive aria-hidden="true" /> {busy === "archive" ? text.working : text.archive}
            </button>
            <p>{archiveFull ? text.archiveFull(archive.limit) : text.archiveHint(archive.count, archive.limit)}</p>
          </div>
        ) : null}

        {archive && state === "archived" ? (
          <div className={styles.action}>
            <button type="button" className={styles.secondary} disabled={Boolean(busy)} onClick={() => void run("restore", archive.onRestore)}>
              <ArchiveRestore aria-hidden="true" /> {busy === "restore" ? text.working : text.restore}
            </button>
            <p>{text.restoreHint}</p>
          </div>
        ) : null}

        {onDelete ? (
          <div className={styles.action}>
            <button
              type="button"
              className={styles.danger}
              disabled={Boolean(busy)}
              onClick={() => {
                if (window.confirm(text.removeConfirm(name))) void run("delete", onDelete);
              }}
            >
              <Trash2 aria-hidden="true" /> {busy === "delete" ? text.working : text.remove}
            </button>
            <p>{text.removeHint}</p>
          </div>
        ) : null}
      </div>
    </section>
  );
};
