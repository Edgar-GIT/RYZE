import { BarChart3, CheckCircle2, Clock3, Dumbbell, History, ListChecks, Repeat, Trophy, XCircle } from "lucide-react";
import { useState } from "react";
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
}

// WorkoutReport is the end-of-workout summary: time, volume, sets and reps,
// personal records and what was done on each exercise. In review mode it
// offers to save the workout (or go back, or abandon it).
export const WorkoutReport = ({ session, history, mode, busy = false, onSave, onKeepGoing, onAbandon }: WorkoutReportProps) => {
  const { t, intlLocale, formatDate } = useI18n();
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
      <header className={styles.head}>
        {mode === "saved" ? (
          abandoned ? (
            <XCircle className={joinClassNames(styles.headIcon, styles.headIconMuted)} aria-hidden="true" />
          ) : (
            <CheckCircle2 className={styles.headIcon} aria-hidden="true" />
          )
        ) : null}
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
            <Clock3 aria-hidden="true" /> {t("training.report.duration")}
          </dt>
          <dd>{formatClock(summary.durationSeconds)}</dd>
        </div>
        <div>
          <dt>
            <Dumbbell aria-hidden="true" /> {t("training.report.volume")}
          </dt>
          <dd>{t("training.progress.kg", { value: formatWeight(summary.volumeKg, intlLocale) })}</dd>
        </div>
        <div>
          <dt>
            <ListChecks aria-hidden="true" /> {t("training.report.sets")}
          </dt>
          <dd>
            {summary.completedSets}/{summary.totalSets}
          </dd>
        </div>
        <div>
          <dt>
            <Repeat aria-hidden="true" /> {t("training.report.reps")}
          </dt>
          <dd>{summary.reps}</dd>
        </div>
      </dl>

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
        <ul className={styles.exercises}>
          {summary.exercises.map((exercise) => (
            <li key={exercise.exerciseId + exercise.name}>
              <span className={styles.exerciseName}>{exercise.name}</span>
              <span className={styles.exerciseMeta}>
                {t("training.report.setsDone", { done: exercise.completedSets, total: exercise.totalSets })}
                {" · "}
                {exercise.best
                  ? exercise.best.weight > 0
                    ? t("training.report.bestSet", { weight: formatWeight(exercise.best.weight, intlLocale), reps: exercise.best.reps })
                    : t("training.report.bestReps", { reps: exercise.best.reps })
                  : t("training.report.noSets")}
              </span>
            </li>
          ))}
        </ul>
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
