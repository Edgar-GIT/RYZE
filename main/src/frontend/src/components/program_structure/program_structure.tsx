// Client-safe rendering of an entitled program structure. Only the access API
// exposes structure; the public marketplace is metadata-only. Value fields are
// optional by nature: a set may lack reps, rest or weight depending on its
// type.
import { Play } from "lucide-react";

import { Button } from "@/components/button/button";

import styles from "./program_structure.module.css";
import { useI18n } from "@/i18n/i18n";

interface ProgramStructureSet {
  set_number: number;
  set_type: string;
  reps?: number | null;
  weight_kg?: number | null;
  rir?: number | null;
  rpe?: number | null;
  rest_seconds?: number | null;
  tempo?: string | null;
}

interface ProgramStructureExercise {
  id: string;
  position: number;
  instructions?: string | null;
  notes?: string | null;
  exercise: { name: string };
  sets: ProgramStructureSet[];
}

interface ProgramStructureWorkout {
  id: string;
  position: number;
  exercises: ProgramStructureExercise[];
}

interface ProgramStructureWeek {
  id: string;
  week_number: number;
  workouts: ProgramStructureWorkout[];
}

interface ProgramStructureProps {
  weeks: ProgramStructureWeek[];
  onStartWorkout?: (workoutId: string) => void;
  activeWorkoutId?: string | null;
  startingWorkoutId?: string | null;
}

const optionalValue = (value: number | null | undefined): string => (value == null ? "—" : String(value));
const textCell = (value: string | null | undefined): string =>
  !value || value.trim() === "" ? "—" : value;

const SetTable = ({ sets }: { sets: ProgramStructureSet[] }) => {
  const { t } = useI18n();
  return (
  <div className={styles.setWrap}>
    <table className={styles.setTable}>
      <thead>
        <tr>
          <th>{t("account.training.colSet")}</th>
          <th>{t("account.training.colType")}</th>
          <th>{t("account.training.colReps")}</th>
          <th>{t("account.training.colWeight")}</th>
          <th>RIR</th>
          <th>RPE</th>
          <th>{t("account.training.colRest")}</th>
          <th>{t("account.training.colTempo")}</th>
        </tr>
      </thead>
      <tbody>
        {sets.map((set) => (
          <tr key={set.set_number}>
            <td>{set.set_number}</td>
            <td>{textCell(set.set_type)}</td>
            <td>{optionalValue(set.reps)}</td>
            <td>{optionalValue(set.weight_kg)}</td>
            <td>{optionalValue(set.rir)}</td>
            <td>{optionalValue(set.rpe)}</td>
            <td>{optionalValue(set.rest_seconds)}</td>
            <td>{textCell(set.tempo)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
  );
};

export const ProgramStructure = ({ weeks, onStartWorkout, activeWorkoutId, startingWorkoutId }: ProgramStructureProps) => {
  const { t } = useI18n();
  if (weeks.length === 0) {
    return (
      <div className={styles.state}>
        <p>{t("account.training.noWeeks")}</p>
      </div>
    );
  }

  return (
    <div className={styles.structure}>
      {weeks.map((week) => (
        <section key={week.id} className={styles.weekCard}>
          <h2 className={styles.weekTitle}>{t("account.training.week", { week: week.week_number })}</h2>

          {week.workouts.length === 0 ? (
            <p className={styles.emptyNote}>{t("account.training.noDays")}</p>
          ) : (
            week.workouts.map((workout) => (
              <div key={workout.id} className={styles.dayCard}>
                <div className={styles.dayHead}>
                  <p className={styles.dayTitle}>{t("account.training.day", { day: workout.position })}</p>
                  {onStartWorkout ? (
                    <Button
                      variant={activeWorkoutId === workout.id ? "primary" : "secondary"}
                      size="small"
                      disabled={Boolean(startingWorkoutId) || workout.exercises.length === 0}
                      onClick={() => onStartWorkout(workout.id)}
                      icon={<Play size={14} />}
                      iconPosition="left"
                    >
                      {startingWorkoutId === workout.id
                        ? t("account.training.starting")
                        : activeWorkoutId === workout.id
                          ? t("account.training.resume")
                          : t("account.training.start")}
                    </Button>
                  ) : null}
                </div>

                {workout.exercises.length === 0 ? (
                  <p className={styles.emptyNote}>{t("account.training.noExercises")}</p>
                ) : (
                  workout.exercises.map((exercise) => (
                    <article key={exercise.id} className={styles.exercise}>
                      <header className={styles.exerciseHead}>
                        <span className={styles.exercisePos}>{exercise.position}</span>
                        <span className={styles.exerciseName}>{exercise.exercise.name}</span>
                        <span className={styles.exerciseMeta}>
                          {exercise.sets.length === 1
                            ? t("account.training.setCount")
                            : t("account.training.setsCount", { count: exercise.sets.length })}
                        </span>
                      </header>

                      {exercise.instructions ? (
                        <p className={styles.exerciseNote}>{exercise.instructions}</p>
                      ) : null}
                      {exercise.notes ? <p className={styles.exerciseNote}>{exercise.notes}</p> : null}

                      <SetTable sets={exercise.sets} />
                    </article>
                  ))
                )}
              </div>
            ))
          )}
        </section>
      ))}
    </div>
  );
};