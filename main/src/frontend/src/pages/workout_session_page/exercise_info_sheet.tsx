import { ExternalLink, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";

import { useI18n } from "@/i18n/i18n";
import { exerciseMuscles, fetchExerciseDetail, type ExerciseCatalogEntry } from "@/services/exercise_library";

import styles from "./workout_session_page.module.css";

interface ExerciseInfoSheetProps {
  exerciseId: string;
  name: string;
  // Instructions and notes written for this workout by the trainer.
  instructions: string;
  notes: string;
  onClose: () => void;
}

// ExerciseInfoSheet shows what an exercise works and how to do it without
// leaving the workout. It reads the public catalogue entry; the full page
// stays one tap away in the Exercise Library.
export const ExerciseInfoSheet = ({ exerciseId, name, instructions, notes, onClose }: ExerciseInfoSheetProps) => {
  const { t } = useI18n();
  const [entry, setEntry] = useState<ExerciseCatalogEntry | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const closeRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    let cancelled = false;
    fetchExerciseDetail(exerciseId)
      .then((detail) => {
        if (!cancelled) {
          setEntry(detail.exercise);
          setState("ready");
        }
      })
      .catch(() => {
        if (!cancelled) setState("error");
      });
    return () => {
      cancelled = true;
    };
  }, [exerciseId]);

  useEffect(() => {
    closeRef.current?.focus();
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  const steps = entry?.instruction_steps?.filter(Boolean) ?? [];

  return (
    <div className={styles.sheetBackdrop} onClick={onClose}>
      <div
        className={styles.sheet}
        role="dialog"
        aria-modal="true"
        aria-label={t("training.player.info", { name })}
        onClick={(event) => event.stopPropagation()}
      >
        <header className={styles.sheetHead}>
          <h2>{name}</h2>
          <button ref={closeRef} type="button" className={styles.iconButton} onClick={onClose} aria-label={t("common.close")}>
            <X aria-hidden="true" />
          </button>
        </header>

        {notes || instructions ? (
          <div className={styles.sheetNote}>
            {instructions ? <p>{instructions}</p> : null}
            {notes ? <p>{notes}</p> : null}
          </div>
        ) : null}

        {state === "loading" ? <p className={styles.muted}>{t("training.player.infoLoading")}</p> : null}
        {state === "error" ? <p className={styles.muted}>{t("training.player.infoError")}</p> : null}
        {entry ? (
          <dl className={styles.sheetFacts}>
            {exerciseMuscles(entry) ? (
              <div>
                <dt>{t("training.player.infoMuscles")}</dt>
                <dd>{exerciseMuscles(entry)}</dd>
              </div>
            ) : null}
            {entry.equipment ? (
              <div>
                <dt>{t("training.player.infoEquipment")}</dt>
                <dd>{entry.equipment}</dd>
              </div>
            ) : null}
            {steps.length > 0 || entry.instructions ? (
              <div>
                <dt>{t("training.player.infoHowTo")}</dt>
                <dd>
                  {steps.length > 0 ? (
                    <ol className={styles.sheetSteps}>
                      {steps.map((step) => (
                        <li key={step}>{step}</li>
                      ))}
                    </ol>
                  ) : (
                    entry.instructions
                  )}
                </dd>
              </div>
            ) : null}
          </dl>
        ) : null}

        <Link className={styles.sheetLink} to={`/exercises/${encodeURIComponent(exerciseId)}`} target="_blank" rel="noopener">
          {t("training.player.infoOpenLibrary")}
          <ExternalLink aria-hidden="true" />
        </Link>
      </div>
    </div>
  );
};
