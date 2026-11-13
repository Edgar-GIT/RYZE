import { Check, RotateCcw, SkipForward } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { useI18n } from "@/i18n/i18n";
import { formatWeight } from "@/services/training_insights";
import type { WorkoutSessionSet, WorkoutSetInput } from "@/services/workout_sessions_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./workout_session_page.module.css";

const TYPE_BADGE: Record<string, "warmup" | "drop" | "backoff" | "failure"> = {
  warmup: "warmup",
  drop: "drop",
  backoff: "backoff",
  failure: "failure"
};

interface SetRowProps {
  set: WorkoutSessionSet;
  // The same set number from the last time this exercise was done.
  previous?: WorkoutSessionSet;
  disabled: boolean;
  onSave: (input: WorkoutSetInput) => Promise<void>;
}

const toNumber = (value: string): number | null => {
  const parsed = Number(value.replace(",", "."));
  return value.trim() === "" || !Number.isFinite(parsed) || parsed < 0 ? null : parsed;
};

// SetRow is one line of the workout table. Inputs show the target (or what
// was done last time) as a placeholder; ticking the set saves the typed
// values, or the placeholders when nothing was typed.
export const SetRow = ({ set, previous, disabled, onSave }: SetRowProps) => {
  const { t, intlLocale } = useI18n();
  const [weight, setWeight] = useState(set.actual_weight_kg == null ? "" : String(set.actual_weight_kg));
  const [reps, setReps] = useState(set.actual_reps == null ? "" : String(set.actual_reps));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setWeight(set.actual_weight_kg == null ? "" : String(set.actual_weight_kg));
    setReps(set.actual_reps == null ? "" : String(set.actual_reps));
  }, [set.actual_weight_kg, set.actual_reps]);

  useEffect(() => {
    if (!menuOpen) {
      return;
    }
    const close = (event: PointerEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, [menuOpen]);

  const previousWeight = previous?.actual_weight_kg ?? null;
  const previousReps = previous?.actual_reps ?? null;
  const placeholderWeight = set.prescribed_weight_kg ?? previousWeight;
  const placeholderReps = set.prescribed_reps ?? previousReps;

  const save = async (status: WorkoutSetInput["status"], values?: { weight: number | null; reps: number | null }) => {
    setSaving(true);
    setError("");
    setMenuOpen(false);
    try {
      await onSave({
        status,
        actual_weight_kg: values ? values.weight : toNumber(weight),
        actual_reps: values ? values.reps : toNumber(reps)
      });
    } catch {
      setError(t("training.player.saveError"));
    } finally {
      setSaving(false);
    }
  };

  const toggleDone = () => {
    if (set.status === "completed") {
      void save("pending");
      return;
    }
    const finalReps = toNumber(reps) ?? placeholderReps;
    if (finalReps == null) {
      setError(t("account.training.enterReps"));
      return;
    }
    const finalWeight = toNumber(weight) ?? placeholderWeight;
    setReps(String(finalReps));
    if (finalWeight != null) {
      setWeight(String(finalWeight));
    }
    void save("completed", { weight: finalWeight, reps: finalReps });
  };

  const usePrevious = () => {
    if (previousWeight != null) setWeight(String(previousWeight));
    if (previousReps != null) setReps(String(previousReps));
  };

  const badge = TYPE_BADGE[set.set_type];
  const done = set.status === "completed";
  const skipped = set.status === "skipped";
  const locked = disabled || saving;

  return (
    <div className={joinClassNames(styles.setRow, done && styles.setDone, skipped && styles.setSkipped)}>
      <div className={styles.setNumberCell} ref={menuRef}>
        <button
          type="button"
          className={joinClassNames(styles.setNumber, badge && styles[`badge_${badge}`])}
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          disabled={disabled}
          onClick={() => setMenuOpen((open) => !open)}
        >
          {badge ? t(`training.player.${badge}`) : set.set_number}
        </button>
        {menuOpen ? (
          <div className={styles.setMenu} role="menu">
            {set.status === "pending" ? (
              <button type="button" role="menuitem" onClick={() => void save("skipped")}>
                <SkipForward aria-hidden="true" />
                {t("training.player.skip", { set: set.set_number })}
              </button>
            ) : (
              <button type="button" role="menuitem" onClick={() => void save("pending")}>
                <RotateCcw aria-hidden="true" />
                {t("training.player.undo", { set: set.set_number })}
              </button>
            )}
          </div>
        ) : null}
      </div>

      <button
        type="button"
        className={styles.previousCell}
        onClick={usePrevious}
        disabled={locked || previous == null || done}
        aria-label={
          previous == null
            ? undefined
            : t("training.player.useValue", {
                value: `${formatWeight(previousWeight ?? 0, intlLocale)} kg × ${previousReps ?? 0}`
              })
        }
      >
        {previous == null ? "—" : `${formatWeight(previousWeight ?? 0, intlLocale)} × ${previousReps ?? 0}`}
      </button>

      {skipped ? (
        <span className={styles.skippedLabel}>{t("training.player.skipped")}</span>
      ) : (
        <>
          <input
            className={styles.setInput}
            type="text"
            inputMode="decimal"
            value={weight}
            placeholder={placeholderWeight == null ? "–" : formatWeight(placeholderWeight, intlLocale)}
            disabled={locked || done}
            onChange={(event) => setWeight(event.target.value)}
            aria-label={t("account.training.actualWeight", { set: set.set_number })}
          />
          <input
            className={styles.setInput}
            type="text"
            inputMode="numeric"
            value={reps}
            placeholder={placeholderReps == null ? "–" : String(placeholderReps)}
            disabled={locked || done}
            onChange={(event) => setReps(event.target.value)}
            aria-label={t("account.training.actualReps", { set: set.set_number })}
          />
        </>
      )}

      {skipped ? (
        <span aria-hidden="true" />
      ) : (
        <button
          type="button"
          className={joinClassNames(styles.checkButton, done && styles.checkButtonDone)}
          onClick={toggleDone}
          disabled={locked}
          aria-pressed={done}
          aria-label={done ? t("training.player.undo", { set: set.set_number }) : t("training.player.markDone", { set: set.set_number })}
        >
          <Check aria-hidden="true" strokeWidth={3} />
        </button>
      )}

      {error ? (
        <p className={styles.setError} role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
};
