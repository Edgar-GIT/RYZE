// Client-safe rendering of an entitled program structure. Only the access API
// exposes structure; the public marketplace is metadata-only. Value fields are
// optional by nature: a set may lack reps, rest or weight depending on its
// type.
import { Play } from "lucide-react";

import { Button } from "@/components/button/button";

import styles from "./program_structure.module.css";

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

const SetTable = ({ sets }: { sets: ProgramStructureSet[] }) => (
  <div className={styles.setWrap}>
    <table className={styles.setTable}>
      <thead>
        <tr>
          <th>Set</th>
          <th>Type</th>
          <th>Reps</th>
          <th>Weight</th>
          <th>RIR</th>
          <th>RPE</th>
          <th>Rest</th>
          <th>Tempo</th>
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

export const ProgramStructure = ({ weeks, onStartWorkout, activeWorkoutId, startingWorkoutId }: ProgramStructureProps) => {
  if (weeks.length === 0) {
    return (
      <div className={styles.state}>
        <p>This training plan has no weeks yet.</p>
      </div>
    );
  }

  return (
    <div className={styles.structure}>
      {weeks.map((week) => (
        <section key={week.id} className={styles.weekCard}>
          <h2 className={styles.weekTitle}>Week {week.week_number}</h2>

          {week.workouts.length === 0 ? (
            <p className={styles.emptyNote}>No training days in this week.</p>
          ) : (
            week.workouts.map((workout) => (
              <div key={workout.id} className={styles.dayCard}>
                <div className={styles.dayHead}>
                  <p className={styles.dayTitle}>Day {workout.position}</p>
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
                        ? "Starting…"
                        : activeWorkoutId === workout.id
                          ? "Resume"
                          : "Start"}
                    </Button>
                  ) : null}
                </div>

                {workout.exercises.length === 0 ? (
                  <p className={styles.emptyNote}>No exercises assigned.</p>
                ) : (
                  workout.exercises.map((exercise) => (
                    <article key={exercise.id} className={styles.exercise}>
                      <header className={styles.exerciseHead}>
                        <span className={styles.exercisePos}>{exercise.position}</span>
                        <span className={styles.exerciseName}>{exercise.exercise.name}</span>
                        <span className={styles.exerciseMeta}>
                          {exercise.sets.length} set{exercise.sets.length === 1 ? "" : "s"}
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