// Shared client-safe rendering of a program structure for entitled programs
// and the public marketplace. Both sources serve the identical backend shape
// (weeks -> workouts -> exercises -> prescription sets), so a single component
// keeps the visual contract consistent. Value fields are optional by nature:
// a set may lack reps, rest or weight depending on its type.
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
  name: string;
  instructions?: string | null;
  notes?: string | null;
  position: number;
  sets: ProgramStructureSet[];
}

interface ProgramStructureWorkout {
  position: number;
  exercises: ProgramStructureExercise[];
}

interface ProgramStructureWeek {
  week_number: number;
  workouts: ProgramStructureWorkout[];
}

interface ProgramStructureProps {
  weeks: ProgramStructureWeek[];
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

export const ProgramStructure = ({ weeks }: ProgramStructureProps) => {
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
        <section key={week.week_number} className={styles.weekCard}>
          <h2 className={styles.weekTitle}>Week {week.week_number}</h2>

          {week.workouts.length === 0 ? (
            <p className={styles.emptyNote}>No training days in this week.</p>
          ) : (
            week.workouts.map((workout) => (
              <div key={workout.position} className={styles.dayCard}>
                <p className={styles.dayTitle}>Day {workout.position}</p>

                {workout.exercises.length === 0 ? (
                  <p className={styles.emptyNote}>No exercises assigned.</p>
                ) : (
                  workout.exercises.map((exercise) => (
                    <article key={exercise.position} className={styles.exercise}>
                      <header className={styles.exerciseHead}>
                        <span className={styles.exercisePos}>{exercise.position}</span>
                        <span className={styles.exerciseName}>{exercise.name}</span>
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