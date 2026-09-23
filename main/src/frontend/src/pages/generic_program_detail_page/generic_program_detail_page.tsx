import { ArrowLeft, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import {
  fetchMarketplaceProgram,
  type MarketplaceProgramDetail,
  type MarketplaceSet
} from "@/services/marketplace_api";

import styles from "./generic_program_detail_page.module.css";

const optionalValue = (value: number | null): string => (value === null ? "—" : String(value));
const textCell = (value: string): string => (value.trim() === "" ? "—" : value);

const SetTable = ({ sets }: { sets: MarketplaceSet[] }) => (
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

export const GenericProgramDetailPage = () => {
  const { programId } = useParams<{ programId: string }>();
  const [detail, setDetail] = useState<MarketplaceProgramDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");
    try {
      setDetail(await fetchMarketplaceProgram(programId));
    } catch {
      setDetail(null);
      setErrorMessage("This training plan is not available through the marketplace.");
    } finally {
      setLoading(false);
    }
  }, [programId]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <PageWrapper className={styles.page}>
      <section className={styles.detail}>
        <Container size="wide" className={styles.container}>
          <div className={styles.back}>
            <Button
              to="/services/generic-program"
              variant="ghost"
              size="small"
              icon={<ArrowLeft size={15} />}
              iconPosition="left"
            >
              Back to Training plans
            </Button>
          </div>

          {loading ? (
            <div className={styles.state}>
              <p>Loading training plan…</p>
            </div>
          ) : errorMessage || !detail ? (
            <div className={styles.state}>
              <p>{errorMessage}</p>
              <div className={styles.retry}>
                <Button
                  variant="secondary"
                  size="small"
                  onClick={() => void load()}
                  icon={<RefreshCw size={15} />}
                >
                  Retry
                </Button>
              </div>
            </div>
          ) : (
            <>
              <header className={styles.heading}>
                <h1 className={styles.title}>{detail.name}</h1>
                {detail.description ? (
                  <p className={styles.subtitle}>{detail.description}</p>
                ) : null}

                <div className={styles.chips}>
                  {detail.training_type ? (
                    <span className={styles.chip}>{detail.training_type}</span>
                  ) : null}
                  {detail.level ? <span className={styles.chip}>{detail.level}</span> : null}
                  {detail.duration_weeks !== null ? (
                    <span className={styles.chip}>
                      {detail.duration_weeks} week{detail.duration_weeks === 1 ? "" : "s"}
                    </span>
                  ) : null}
                  {detail.frequency_per_week !== null ? (
                    <span className={styles.chip}>
                      {detail.frequency_per_week} day{detail.frequency_per_week === 1 ? "" : "s"}/week
                    </span>
                  ) : null}
                </div>
              </header>

              {detail.weeks.length === 0 ? (
                <div className={styles.state}>
                  <p>This training plan has no weeks yet.</p>
                </div>
              ) : (
                <div className={styles.structure}>
                  {detail.weeks.map((week) => (
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
                                      {exercise.sets.length} set
                                      {exercise.sets.length === 1 ? "" : "s"}
                                    </span>
                                  </header>

                                  {exercise.instructions ? (
                                    <p className={styles.exerciseNote}>{exercise.instructions}</p>
                                  ) : null}
                                  {exercise.notes ? (
                                    <p className={styles.exerciseNote}>{exercise.notes}</p>
                                  ) : null}

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
              )}
            </>
          )}
        </Container>
      </section>
    </PageWrapper>
  );
};