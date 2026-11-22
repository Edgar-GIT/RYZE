import { BarChart3, CheckCircle2, Clock3, Dumbbell, History, ListChecks, Repeat, Star, Trophy, XCircle } from "lucide-react";
import { useState, type ReactNode } from "react";
import { Link } from "react-router-dom";

import { useI18n } from "@/i18n/i18n";
import { formatClock, formatWeight, personalRecords, summarizeSession } from "@/services/training_insights";
import type { WorkoutSession } from "@/services/workout_sessions_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./workout_report.module.css";

interface WorkoutReportProps {
  session: WorkoutSession;
  // Earlier sessions, used to find personal records.
  history: WorkoutSession[];
  // "review": before saving (Finish was tapped); otherwise the saved outcome.
  mode: "review" | "saved";
  busy?: boolean;
  onSave?: () => void;
  onKeepGoing?: () => void;
  onAbandon?: () => void;
  // Shown after the numbers (the saved workout's EXP and medals).
  rewards?: ReactNode;
}

const SET_COPY = {
  en: { skipped: "Skipped", pending: "Not done", best: "Best set", sets: "Set" },
  pt: { skipped: "Saltada", pending: "Por fazer", best: "Melhor série", sets: "Série" }
} as const;

// WorkoutReport is the end-of-workout summary: time, volume, sets and reps,
// personal records and what was done on each exercise. In review mode it
// offers to save the workout (or go back, or abandon it).
export const WorkoutReport = ({ session, history, mode, busy = false, onSave, onKeepGoing, onAbandon, rewards }: WorkoutReportProps) => {
  const { t, intlLocale, formatDate, locale } = useI18n();
  const setText = SET_COPY[locale];
  const [confirmAbandon, setConfirmAbandon] = useState(false);
  const summary = summarizeSession(session);
  const records = personalRecords(session, history);
  const abandoned = session.status === "abandoned";

  const heading =
    mode === "review"
      ? t("training.report.title")
      : abandoned
        ? t("training.report.abandonedTitle")
        : t("training.report.savedTitle");

  return (
    <section className={styles.report} aria-labelledby="workout-report-title">
      <header className={joinClassNames(styles.head, abandoned && styles.headMuted)}>
        <span className={styles.headIcon} aria-hidden="true">
          {abandoned ? <XCircle /> : mode === "saved" ? <CheckCircle2 /> : <ListChecks />}
        </span>
        <p className={styles.kicker}>
          {session.program_name} · {t("account.training.weekDay", { week: session.week_number, day: session.workout_position })}
        </p>
        <h1 id="workout-report-title" className={styles.title}>
          {heading}
        </h1>
        <p className={styles.lead}>
          {mode === "saved"
            ? abandoned
              ? t("training.report.abandonedText")
              : t("training.report.savedText")
            : formatDate(session.started_at, { weekday: "long", day: "numeric", month: "long" })}
        </p>
      </header>

      <dl className={styles.stats}>
        <div>
          <dt>
            <span className={styles.statIcon} aria-hidden="true">
              <Clock3 />
            </span>
            {t("training.report.duration")}
          </dt>
          <dd>{formatClock(summary.durationSeconds)}</dd>
        </div>
        <div>
          <dt>
            <span className={styles.statIcon} aria-hidden="true">
              <Dumbbell />
            </span>
            {t("training.report.volume")}
          </dt>
          <dd>{t("training.progress.kg", { value: formatWeight(summary.volumeKg, intlLocale) })}</dd>
        </div>
        <div>
          <dt>
            <span className={styles.statIcon} aria-hidden="true">
              <ListChecks />
            </span>
            {t("training.report.sets")}
          </dt>
          <dd>
            {summary.completedSets}/{summary.totalSets}
          </dd>
        </div>
        <div>
          <dt>
            <span className={styles.statIcon} aria-hidden="true">
              <Repeat />
            </span>
            {t("training.report.reps")}
          </dt>
          <dd>{summary.reps}</dd>
        </div>
      </dl>

      {rewards}

      {mode === "review" && summary.pendingSets > 0 ? (
        <p className={styles.warning}>{t("training.report.unfinished", { count: summary.pendingSets })}</p>
      ) : null}

      {records.length > 0 ? (
        <section className={styles.block} aria-labelledby="workout-report-records">
          <h2 id="workout-report-records" className={styles.blockTitle}>
            <Trophy aria-hidden="true" /> {t("training.report.records")}
          </h2>
          <ul className={styles.records}>
            {records.map((record) => (
              <li key={record.exerciseId}>
                <strong>{record.name}</strong>
                <span>
                  {t(record.kind === "weight" ? "training.report.recordWeight" : "training.report.recordE1rm", {
                    weight: formatWeight(record.weight, intlLocale),
                    reps: record.reps
                  })}
                </span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <section className={styles.block} aria-labelledby="workout-report-exercises">
        <h2 id="workout-report-exercises" className={styles.blockTitle}>
          {t("training.report.exercises")}
        </h2>
        <ol className={styles.exercises}>
          {session.exercises.map((exercise, index) => {
            const done = summary.exercises[index];
            const isRecord = records.some((record) => record.exerciseId === exercise.exercise_id);
            const bestId = (() => {
              let best: { id: string; weight: number; reps: number } | null = null;
              for (const set of exercise.sets) {
                if (set.status !== "completed") continue;
                const weight = set.actual_weight_kg ?? 0;
                const reps = set.actual_reps ?? 0;
                if (!best || weight > best.weight || (weight === best.weight && reps > best.reps)) best = { id: set.id, weight, reps };
              }
              return best?.id ?? null;
            })();
            return (
              <li key={exercise.id} className={styles.exercise}>
                <div className={styles.exerciseTop}>
                  <span className={styles.exerciseIndex}>{index + 1}</span>
                  <span className={styles.exerciseName}>{exercise.name}</span>
                  {isRecord ? (
                    <span className={styles.prBadge}>
                      <Trophy aria-hidden="true" /> PR
                    </span>
                  ) : null}
                </div>
                <p className={styles.exerciseMeta}>
                  {t("training.report.setsDone", { done: done?.completedSets ?? 0, total: done?.totalSets ?? exercise.sets.length })}
                  {done && done.volumeKg > 0 ? ` · ${t("training.progress.kg", { value: formatWeight(done.volumeKg, intlLocale) })}` : ""}
                </p>
                <ol className={styles.setChips}>
                  {exercise.sets.map((set, setIndex) => {
                    const weight = set.actual_weight_kg ?? 0;
                    const reps = set.actual_reps ?? 0;
                    const label =
                      set.status === "completed"
                        ? weight > 0
                          ? `${formatWeight(weight, intlLocale)} kg × ${reps}`
                          : t("training.report.bestReps", { reps })
                        : set.status === "skipped"
                          ? setText.skipped
                          : setText.pending;
                    return (
                      <li
                        key={set.id}
                        className={joinClassNames(styles.setChip, set.status !== "completed" && styles.setChipOff, set.id === bestId && styles.setChipBest)}
                      >
                        <span className={styles.setChipIndex} aria-label={`${setText.sets} ${setIndex + 1}`}>
                          {setIndex + 1}
                        </span>
                        {label}
                        {set.id === bestId && exercise.sets.length > 1 ? <Star aria-label={setText.best} /> : null}
                      </li>
                    );
                  })}
                </ol>
              </li>
            );
          })}
        </ol>
      </section>

      {mode === "review" ? (
        <div className={styles.actions}>
          <button type="button" className={styles.primary} onClick={onSave} disabled={busy}>
            {busy ? t("training.report.saving") : t("training.report.save")}
          </button>
          <button type="button" className={styles.secondary} onClick={onKeepGoing} disabled={busy}>
            {t("training.report.keepGoing")}
          </button>
          <button
            type="button"
            className={styles.danger}
            disabled={busy}
            onClick={() => {
              if (confirmAbandon) {
                onAbandon?.();
              } else {
                setConfirmAbandon(true);
              }
            }}
          >
            {confirmAbandon ? t("training.report.abandonConfirm") : t("training.report.abandon")}
          </button>
        </div>
      ) : (
        <div className={styles.actions}>
          <Link className={styles.primary} to="/services/progress">
            <BarChart3 aria-hidden="true" /> {t("training.report.viewProgress")}
          </Link>
          <Link className={styles.secondary} to="/services/workout-history">
            <History aria-hidden="true" /> {t("training.report.viewHistory")}
          </Link>
          <Link className={styles.secondary} to={`/services/my-programs/${encodeURIComponent(session.program_id)}`}>
            {t("training.report.backToPlan")}
          </Link>
        </div>
      )}
    </section>
  );
};
