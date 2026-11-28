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
    soldEdit: "Edit · new version",
    soldEditHint:
      "This package has been sold. Editing creates a new version: the people who bought it keep the version they bought, and the new one replaces it on the market when you publish it.",
    soldContinue: "Continue the new version",
    soldContinueHint: "A new version is already being prepared. Publish it to replace the current one on the market.",
    archive: "Archive",
    archiveHint: (count: number, limit: number) =>
      `Takes it off the market for a while. People who already have it keep using it. You can bring it back at any time. Archived: ${count} of ${limit}.`,
    archiveFull: (limit: number) => `You already have ${limit} archived programmes: restore or delete one before archiving another.`,
    restore: "Put back on the market",
    restoreHint: "Makes it visible and available to buy or start again.",
    remove: "Delete",
    removeHint: "Removes it for good.",
    removeSold: "A sold package cannot be deleted (its buyers keep it). Archive it to take it off the market.",
    removeConfirm: (name: string) => `Delete “${name}”? This cannot be undone.`,
    archiveConfirm: (name: string) => `Archive “${name}”? It leaves the market until you put it back.`,
    working: "Working…",
    back: "Back to the programmes",
    coverNote: "This is how the cover appears in the store."
  },
  pt: {
    states: { published: "Publicado · à venda", draft: "Rascunho · ninguém o vê", archived: "Arquivado · fora do mercado" },
    edit: "Editar programa",
    editHint: "Muda os detalhes, as semanas e os dias, os exercícios e (no Premium) o plano de refeições.",
    continueDraft: "Continuar a editar",
    soldEdit: "Editar · nova versão",
    soldEditHint:
      "Este pacote já foi vendido. Editar cria uma nova versão: quem comprou fica com a versão que comprou, e a nova substitui-a no mercado quando a publicares.",
    soldContinue: "Continuar a nova versão",
    soldContinueHint: "Já há uma nova versão em preparação. Publica-a para substituir a atual no mercado.",
    archive: "Arquivar",
    archiveHint: (count: number, limit: number) =>
      `Tira-o do mercado por uns tempos. Quem já o tem continua a usá-lo. Podes voltar a pô-lo à venda quando quiseres. Arquivados: ${count} de ${limit}.`,
    archiveFull: (limit: number) => `Já tens ${limit} programas arquivados: retoma ou apaga um antes de arquivares outro.`,
    restore: "Voltar a pôr no mercado",
    restoreHint: "Fica outra vez visível e disponível para comprar ou começar.",
    remove: "Apagar",
    removeHint: "Remove-o de vez.",
    removeSold: "Um pacote vendido não pode ser apagado (quem o comprou mantém-no). Arquiva-o para o tirares do mercado.",
    removeConfirm: (name: string) => `Apagar “${name}”? Não é possível desfazer.`,
    archiveConfirm: (name: string) => `Arquivar “${name}”? Sai do mercado até o voltares a pôr.`,
    working: "A processar…",
    back: "Voltar aos programas",
    coverNote: "É assim que a capa aparece na loja."
  }
} as const;

// ProgramOverview is the first screen of a programme (owner request,
// 2026-11-27): its picture (in the store's card format) and name, what state it
// is in, and the three things you do with it — edit (green), archive or put
// back, delete — each explained. A sold package is edited through a new
// version (v2.133.0).
export const ProgramOverview = ({
  name,
  coverUrl,
  kicker,
  facts,
  state,
  editTo,
  backTo,
  archive,
  onDelete,
  version
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
  // A sold package: edit through a new version (open one, or create it).
  version?: { sold: boolean; draftTo?: string; onCreate: () => Promise<void> };
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
  const sold = Boolean(version?.sold);

  const editAction = sold ? (
    version?.draftTo ? (
      <>
        <Link className={styles.edit} to={version.draftTo}>
          <PencilLine aria-hidden="true" /> {text.soldContinue}
        </Link>
        <p>{text.soldContinueHint}</p>
      </>
    ) : (
      <>
        <button
          type="button"
          className={styles.edit}
          disabled={Boolean(busy)}
          onClick={() => version && void run("version", version.onCreate)}
        >
          <PencilLine aria-hidden="true" /> {busy === "version" ? text.working : text.soldEdit}
        </button>
        <p>{text.soldEditHint}</p>
      </>
    )
  ) : (
    <>
      <Link className={styles.edit} to={editTo}>
        <PencilLine aria-hidden="true" /> {state === "draft" ? text.continueDraft : text.edit}
      </Link>
      <p>{text.editHint}</p>
    </>
  );

  return (
    <section className={styles.overview} aria-labelledby="program-overview-title">
      <Link className={styles.back} to={backTo}>
        <ArrowLeft aria-hidden="true" /> {text.back}
      </Link>
      <div className={styles.card}>
        <div className={styles.coverPanel}>
          <div className={styles.cover}>{coverUrl ? <img src={coverUrl} alt="" /> : <Dumbbell aria-hidden="true" />}</div>
          <p className={styles.coverNote}>{text.coverNote}</p>
        </div>
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

          <div className={styles.actions}>
            <div className={styles.action}>{editAction}</div>

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

            {onDelete || sold ? (
              <div className={styles.action}>
                <button
                  type="button"
                  className={styles.danger}
                  disabled={Boolean(busy) || sold || !onDelete}
                  onClick={() => {
                    if (onDelete && window.confirm(text.removeConfirm(name))) void run("delete", onDelete);
                  }}
                >
                  <Trash2 aria-hidden="true" /> {busy === "delete" ? text.working : text.remove}
                </button>
                <p>{sold ? text.removeSold : text.removeHint}</p>
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </section>
  );
};
