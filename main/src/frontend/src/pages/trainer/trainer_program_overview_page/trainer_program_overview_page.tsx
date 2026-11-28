import { RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Redirect, useParams } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { ProgramOverview, type ProgramOverviewState } from "@/components/program_overview/program_overview";
import { type Copy, useI18n } from "@/i18n/i18n";
import {
  archiveTrainerProgram,
  createTrainerProgramVersion,
  deleteTrainerProgram,
  fetchTrainerArchiveCount,
  fetchTrainerProgram,
  restoreTrainerProgram,
  TrainerProgramStatus,
  TrainerProgramType,
  type TrainerProgram
} from "@/services/trainer_api";
import { ApiError, publicFileUrl } from "@utils/http_client";

import styles from "./trainer_program_overview_page.module.css";

const COPY: Copy<{
  kicker: Record<string, string>;
  weeks: (n: number) => string;
  days: (n: number) => string;
  price: (value: string) => string;
  free: string;
  error: string;
  retry: string;
  errors: Record<string, string>;
  failed: string;
}> = {
  en: {
    kicker: {
      free: "Free programme",
      premium: "Premium package (Level 1) · €14.99",
      personalized: "Personalised plan (Level 2)"
    },
    weeks: (n) => `${n} ${n === 1 ? "week" : "weeks"}`,
    days: (n) => `${n} training ${n === 1 ? "day" : "days"} per week`,
    price: (value) => `Price ${value}`,
    free: "Free for everyone",
    error: "This programme could not be loaded.",
    retry: "Try again",
    errors: {
      ARCHIVE_LIMIT_REACHED: "You already have 15 archived programmes. Restore or delete one first.",
      PROGRAM_NOT_ARCHIVABLE: "Only a published programme can be archived.",
      PROGRAM_LOCKED: "This programme was delivered in a Premium Level 2 plan and cannot change.",
      PROGRAM_NOT_FOUND: "This programme no longer exists.",
      PROGRAM_SOLD: "This package has been sold: edit it through a new version."
    },
    failed: "That did not work. Try again."
  },
  pt: {
    kicker: {
      free: "Programa grátis",
      premium: "Pacote Premium (Nível 1) · 14,99 €",
      personalized: "Plano personalizado (Nível 2)"
    },
    weeks: (n) => `${n} ${n === 1 ? "semana" : "semanas"}`,
    days: (n) => `${n} ${n === 1 ? "dia" : "dias"} de treino por semana`,
    price: (value) => `Preço ${value}`,
    free: "Grátis para todos",
    error: "Não foi possível carregar este programa.",
    retry: "Tentar de novo",
    errors: {
      ARCHIVE_LIMIT_REACHED: "Já tens 15 programas arquivados. Retoma ou apaga um primeiro.",
      PROGRAM_NOT_ARCHIVABLE: "Só um programa publicado pode ser arquivado.",
      PROGRAM_LOCKED: "Este programa foi entregue num Premium Nível 2 e já não pode mudar.",
      PROGRAM_NOT_FOUND: "Este programa já não existe.",
      PROGRAM_SOLD: "Este pacote já foi vendido: edita-o através de uma nova versão."
    },
    failed: "Não resultou. Tenta de novo."
  }
};

// TrainerProgramOverviewPage is the first screen of one of the trainer's
// programmes (owner request, 2026-11-27): picture, name and state, then
// Edit (green), Archive / Put back on the market, and Delete.
export const TrainerProgramOverviewPage = () => {
  const { programId } = useParams<{ programId: string }>();
  const { locale } = useI18n();
  const text = COPY[locale];
  const [program, setProgram] = useState<TrainerProgram | null>(null);
  const [archived, setArchived] = useState({ archived: 0, limit: 15 });
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [message, setMessage] = useState("");
  const [redirect, setRedirect] = useState("");

  const load = useCallback(async () => {
    setStatus("loading");
    try {
      const [loaded, count] = await Promise.all([fetchTrainerProgram(programId), fetchTrainerArchiveCount()]);
      setProgram(loaded);
      setArchived(count);
      setStatus("ready");
    } catch {
      setStatus("error");
    }
  }, [programId]);

  useEffect(() => {
    void load();
  }, [load]);

  const describe = (error: unknown) => (error instanceof ApiError && text.errors[error.code]) || text.failed;

  if (redirect) return <Redirect to={redirect} />;
  if (status === "loading") return <AppLoading />;
  if (status === "error" || !program) {
    return (
      <section className={styles.state}>
        <p>{text.error}</p>
        <button type="button" className={styles.ghost} onClick={() => void load()}>
          <RefreshCw aria-hidden="true" /> {text.retry}
        </button>
      </section>
    );
  }

  const state: ProgramOverviewState = program.archived_at
    ? "archived"
    : program.status === TrainerProgramStatus.PUBLISHED
      ? "published"
      : "draft";
  const money = new Intl.NumberFormat(locale === "pt" ? "pt-PT" : "en-IE", { style: "currency", currency: "EUR" });
  const facts = [
    program.duration_weeks ? text.weeks(program.duration_weeks) : "",
    program.frequency_per_week ? text.days(program.frequency_per_week) : "",
    program.type === TrainerProgramType.FREE ? text.free : text.price(money.format(program.price_minor_units / 100))
  ].filter(Boolean);

  return (
    <div className={styles.page}>
      {message ? (
        <p className={styles.error} role="alert">
          {message}
        </p>
      ) : null}
      <ProgramOverview
        name={program.name}
        coverUrl={publicFileUrl(program.cover_image_url)}
        kicker={text.kicker[program.type] ?? program.type}
        facts={facts}
        state={state}
        editTo={`/trainer/programs/${program.id}/edit`}
        backTo="/trainer/programs"
        archive={
          program.type === TrainerProgramType.PERSONALIZED
            ? undefined
            : {
                count: archived.archived,
                limit: archived.limit,
                onArchive: async () => {
                  try {
                    await archiveTrainerProgram(program.id);
                    setRedirect(`/trainer/programs?archived=${encodeURIComponent(program.name)}`);
                  } catch (error) {
                    setMessage(describe(error));
                  }
                },
                onRestore: async () => {
                  try {
                    await restoreTrainerProgram(program.id);
                    setRedirect(`/trainer/programs?restored=${encodeURIComponent(program.name)}`);
                  } catch (error) {
                    setMessage(describe(error));
                  }
                }
              }
        }
        onDelete={
          program.sold
            ? undefined
            : async () => {
                try {
                  await deleteTrainerProgram(program.id);
                  setRedirect(`/trainer/programs?deleted=${encodeURIComponent(program.name)}`);
                } catch (error) {
                  setMessage(describe(error));
                }
              }
        }
        version={{
          sold: Boolean(program.sold),
          draftTo: program.draft_version_id ? `/trainer/programs/${program.draft_version_id}/edit` : undefined,
          onCreate: async () => {
            try {
              const created = await createTrainerProgramVersion(program.id);
              setRedirect(`/trainer/programs/${created.id}/edit`);
            } catch (error) {
              setMessage(describe(error));
            }
          }
        }}
      />
    </div>
  );
};
