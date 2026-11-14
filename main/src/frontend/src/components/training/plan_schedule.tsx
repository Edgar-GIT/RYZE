import { CheckCircle2, ChevronDown, Play, Timer } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { useI18n } from "@/i18n/i18n";
import type { ProgramAccessWeek, ProgramAccessWorkout } from "@/services/purchases_api";
import type { WorkoutSession } from "@/services/workout_sessions_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./plan_schedule.module.css";

interface PlanScheduleProps {
  weeks: ProgramAccessWeek[];
  // Completed sessions of this programme (to show what was done and what is
  // next) and the workout in progress, if any.
  sessions: WorkoutSession[];
  activeWorkoutId: string | null;
  startingWorkoutId: string | null;
  onStart: (workoutId: string) => void;
}

// Rough duration: ~45 s per set plus its rest, rounded to 5 minutes.
const estimateMinutes = (workout: ProgramAccessWorkout): number => {
  const seconds = workout.exercises.reduce(
    (sum, exercise) => sum + exercise.sets.reduce((setSum, set) => setSum + 45 + (set.rest_seconds ?? 60), 0),
    0
  );
  return Math.max(5, Math.round(seconds / 60 / 5) * 5);
};

const repsRange = (workout: ProgramAccessWorkout["exercises"][number]): string | null => {
  const reps = workout.sets.map((set) => set.reps).filter((value): value is number => value != null);
  if (reps.length === 0) return null;
  const min = Math.min(...reps);
  const max = Math.max(...reps);
  return min === max ? String(min) : `${min}–${max}`;
};

// PlanSchedule is "my plan": weeks as tabs, each training day as a card with
// what it holds, when it was last done and a one-tap start. Opening a card
// previews the exercises.
export const PlanSchedule = ({ weeks, sessions, activeWorkoutId, startingWorkoutId, onStart }: PlanScheduleProps) => {
  const { t, formatDate } = useI18n();

  const lastDone = useMemo(() => {
    const map = new Map<string, string>();
    for (const session of sessions) {
      if (session.status !== "completed") continue;
      const previous = map.get(session.program_workout_id);
      if (!previous || session.started_at > previous) map.set(session.program_workout_id, session.started_at);
    }
    return map;
  }, [sessions]);

  // The next workout: the first one, in plan order, never completed.
  const nextWorkoutId = useMemo(() => {
    for (const week of weeks) {
      for (const workout of [...week.workouts].sort((a, b) => a.position - b.position)) {
        if (workout.exercises.length > 0 && !lastDone.has(workout.id)) return workout.id;
      }
    }
    return null;
  }, [weeks, lastDone]);

  const sortedWeeks = useMemo(() => [...weeks].sort((a, b) => a.week_number - b.week_number), [weeks]);
  const focusWeek =
    sortedWeeks.find((week) => week.workouts.some((workout) => workout.id === (activeWorkoutId ?? nextWorkoutId)))?.id ??
    sortedWeeks[0]?.id ??
    "";
  const [weekId, setWeekId] = useState(focusWeek);
  const [openId, setOpenId] = useState<string | null>(null);

  useEffect(() => {
    setWeekId(focusWeek);
  }, [focusWeek]);

  if (sortedWeeks.length === 0) {
    return <p className={styles.empty}>{t("training.plan.empty")}</p>;
  }

  const week = sortedWeeks.find((entry) => entry.id === weekId) ?? sortedWeeks[0];
  const workouts = [...week.workouts].sort((a, b) => a.position - b.position);

  return (
    <section className={styles.schedule}>
      {sortedWeeks.length > 1 ? (
        <div className={styles.weekTabs} role="tablist" aria-label={t("training.plan.weeks")}>
          {sortedWeeks.map((entry) => (
            <button
              key={entry.id}
              type="button"
              role="tab"
              aria-selected={entry.id === week.id}
              className={joinClassNames(styles.weekTab, entry.id === week.id && styles.weekTabActive)}
              onClick={() => setWeekId(entry.id)}
            >
              {t("training.plan.week", { week: entry.week_number })}
            </button>
          ))}
        </div>
      ) : null}

      {workouts.length === 0 ? <p className={styles.empty}>{t("training.plan.empty")}</p> : null}

      <ul className={styles.days}>
        {workouts.map((workout) => {
          const exerciseCount = workout.exercises.length;
          const setCount = workout.exercises.reduce((sum, exercise) => sum + exercise.sets.length, 0);
          const done = lastDone.get(workout.id);
          const active = workout.id === activeWorkoutId;
          const next = !active && workout.id === nextWorkoutId;
          const open = openId === workout.id;
          const names = workout.exercises.map((exercise) => exercise.exercise.name);
          return (
            <li key={workout.id} className={joinClassNames(styles.day, (active || next) && styles.dayHighlight)}>
              <button
                type="button"
                className={styles.dayHead}
                aria-expanded={open}
                onClick={() => setOpenId(open ? null : workout.id)}
              >
                <span className={styles.dayTitleRow}>
                  <span className={styles.dayTitle}>
                    {t("training.plan.day", { day: workout.position })}
                    {workout.name?.trim() ? ` · ${workout.name.trim()}` : ""}
                  </span>
                  {active ? <span className={styles.badgeActive}>{t("account.training.inProgress")}</span> : null}
                  {next ? <span className={styles.badgeNext}>{t("training.plan.upNext")}</span> : null}
                  {done && !active ? (
                    <span className={styles.badgeDone}>
                      <CheckCircle2 aria-hidden="true" />
                      {t("training.plan.doneOn", { date: formatDate(done, { day: "numeric", month: "short" }) })}
                    </span>
                  ) : null}
                </span>
                <span className={styles.dayMeta}>
                  {exerciseCount === 1 ? t("training.plan.exercise") : t("training.plan.exercises", { count: exerciseCount })}
                  {" · "}
                  {t("training.plan.sets", { count: setCount })}
                  {exerciseCount > 0 ? (
                    <>
                      {" · "}
                      <Timer aria-hidden="true" />
                      {t("training.plan.minutes", { count: estimateMinutes(workout) })}
                    </>
                  ) : null}
                </span>
                {!open && names.length > 0 ? (
                  <span className={styles.dayNames}>
                    {names.slice(0, 3).join(", ")}
                    {names.length > 3 ? ` +${names.length - 3}` : ""}
                  </span>
                ) : null}
                <ChevronDown className={joinClassNames(styles.chevron, open && styles.chevronOpen)} aria-hidden="true" />
              </button>

              {open && workout.description?.trim() ? <p className={styles.dayNames}>{workout.description.trim()}</p> : null}
              {open ? (
                <ol className={styles.preview} aria-label={t("training.plan.preview")}>
                  {workout.exercises.length === 0 ? <li className={styles.previewEmpty}>{t("training.plan.noExercises")}</li> : null}
                  {[...workout.exercises]
                    .sort((a, b) => a.position - b.position)
                    .map((exercise) => {
                      const range = repsRange(exercise);
                      return (
                        <li key={exercise.id}>
                          <span className={styles.previewName}>{exercise.exercise.name}</span>
                          <span className={styles.previewSets}>
                            {range
                              ? t("training.plan.setsReps", { sets: exercise.sets.length, reps: range })
                              : t("training.plan.sets1", { sets: exercise.sets.length })}
                          </span>
                        </li>
                      );
                    })}
                </ol>
              ) : null}

              {exerciseCount > 0 ? (
                <button
                  type="button"
                  className={joinClassNames(styles.start, !(active || next || open) && styles.startQuiet)}
                  disabled={Boolean(startingWorkoutId)}
                  onClick={() => onStart(workout.id)}
                >
                  <Play aria-hidden="true" />
                  {startingWorkoutId === workout.id
                    ? t("training.plan.starting")
                    : active
                      ? t("training.plan.resume")
                      : t("training.plan.start")}
                </button>
              ) : null}
            </li>
          );
        })}
      </ul>
    </section>
  );
};
