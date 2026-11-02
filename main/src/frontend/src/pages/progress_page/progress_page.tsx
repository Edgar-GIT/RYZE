import { BarChart3, CalendarDays, CheckCircle2, Clock3, Dumbbell, Flame, RefreshCw, SkipForward, Trophy } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory } from "react-router-dom";

import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ApiError } from "@utils/http_client";
import { fetchClientStatistics, type ClientStatistics } from "@/services/statistics_api";

import styles from "./progress_page.module.css";

const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${minutes % 60}m`;
};

const formatDate = (value: string): string =>
  new Date(value).toLocaleDateString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric"
  });

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; stats: ClientStatistics };

export const ProgressPage = () => {
  const history = useHistory();
  const [state, setState] = useState<State>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      setState({ status: "ready", stats: await fetchClientStatistics() });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setState({ status: "error" });
    }
  }, [history]);

  useEffect(() => {
    void load();
  }, [load, retryCount]);

  const stats = state.status === "ready" ? state.stats : null;
  const percentage = Math.round(stats?.completion_percentage ?? 0);

  return (
    <PageWrapper className={styles.page}>
      <AnimatedBackground />
      <AccountNav />

      <main className={styles.main}>
        {state.status === "loading" ? <LoadingScreen /> : null}

        {state.status === "error" ? (
          <section className={styles.stateCard}>
            <p>Unable to load your progress.</p>
            <Button variant="secondary" size="small" onClick={() => setRetryCount((count) => count + 1)} icon={<RefreshCw size={15} />}>
              Try again
            </Button>
          </section>
        ) : null}

        {stats ? (
          <section className={styles.content}>
            <header className={styles.heading}>
              <span className={styles.icon}>
                <BarChart3 size={22} aria-hidden="true" />
              </span>
              <div>
                <h1 className={styles.title}>Progress</h1>
                <p className={styles.subtitle}>
                  Real numbers from the workouts and sets you have logged.
                </p>
              </div>
            </header>

            {!stats.has_active_program ? (
              <section className={styles.stateCard}>
                <p>You do not have an active programme yet.</p>
                <div className={styles.stateActions}>
                  <Button to="/services/my-programs" variant="primary" size="small">
                    My Programs
                  </Button>
                  <Button to="/services/generic-program" variant="secondary" size="small">
                    Browse training plans
                  </Button>
                </div>
              </section>
            ) : (
              <>
                <section className={styles.currentCard} aria-labelledby="current-program">
                  <div>
                    <p className={styles.eyebrow}>Current programme</p>
                    <h2 id="current-program" className={styles.currentTitle}>
                      {stats.current_program_name || "Training plan"}
                    </h2>
                    <p className={styles.currentMeta}>
                      {stats.has_active_assignment ? "Trainer-assigned" : "Purchased or activated"} ·{" "}
                      {stats.completed_sessions} completed session
                      {stats.completed_sessions === 1 ? "" : "s"}
                    </p>
                  </div>
                  <div className={styles.completion}>
                    <span className={styles.completionValue}>{percentage}%</span>
                    <span className={styles.completionLabel}>complete</span>
                  </div>
                </section>

                <div className={styles.progressTrack} aria-label={`Programme completion ${percentage}%`}>
                  <span style={{ width: `${percentage}%` }} />
                </div>

                <div className={styles.grid}>
                  <article className={styles.metric}>
                    <Dumbbell size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>Workouts completed</span>
                    <strong className={styles.metricValue}>{stats.unique_workouts_completed}</strong>
                    <em className={styles.metricNote}>of {stats.total_workouts_in_program} in this programme</em>
                  </article>
                  <article className={styles.metric}>
                    <Flame size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>Total executions</span>
                    <strong className={styles.metricValue}>{stats.total_executions}</strong>
                    <em className={styles.metricNote}>including repeats</em>
                  </article>
                  <article className={styles.metric}>
                    <CheckCircle2 size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>Sets completed</span>
                    <strong className={styles.metricValue}>{stats.completed_sets}</strong>
                    <em className={styles.metricNote}>{stats.skipped_sets} skipped</em>
                  </article>
                  <article className={styles.metric}>
                    <Clock3 size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>Training time</span>
                    <strong className={styles.metricValue}>{formatDuration(stats.total_active_seconds)}</strong>
                    <em className={styles.metricNote}>active time, pauses excluded</em>
                  </article>
                  <article className={styles.metric}>
                    <CalendarDays size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>Last workout</span>
                    <strong className={styles.metricValue}>
                      {stats.last_workout_date ? formatDate(stats.last_workout_date) : "—"}
                    </strong>
                    <em className={styles.metricNote}>most recent session</em>
                  </article>
                  <article className={styles.metric}>
                    <SkipForward size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>Abandoned sessions</span>
                    <strong className={styles.metricValue}>{stats.abandoned_sessions}</strong>
                    <em className={styles.metricNote}>incomplete workouts</em>
                  </article>
                </div>

                <div className={styles.actions}>
                  <Button to="/services/workout-history" variant="secondary" size="small" icon={<Trophy size={15} />} iconPosition="left">
                    Workout history
                  </Button>
                  <Button to="/services/my-programs" variant="ghost" size="small">
                    My Programs
                  </Button>
                </div>
              </>
            )}
          </section>
        ) : null}
      </main>
    </PageWrapper>
  );
};
