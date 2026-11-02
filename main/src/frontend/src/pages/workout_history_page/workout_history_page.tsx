import { ArrowLeft, ChevronRight, History, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory } from "react-router-dom";

import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";
import { fetchWorkoutHistory, type WorkoutSession } from "@/services/workout_sessions_api";

import styles from "./workout_history_page.module.css";

const formatDate = (value: string): string =>
  new Date(value).toLocaleDateString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric"
  });

const statusLabel: Record<WorkoutSession["status"], string> = {
  in_progress: "In progress",
  paused: "Paused",
  completed: "Completed",
  abandoned: "Abandoned"
};

const statusClass = (status: WorkoutSession["status"]): string => {
  if (status === "completed") {
    return styles.badgeDone;
  }
  if (status === "abandoned") {
    return styles.badgeAbandoned;
  }
  return styles.badgeOpen;
};

const formatDuration = (seconds: number): string => {
  const minutes = Math.floor(seconds / 60);
  const remainder = seconds % 60;
  return minutes > 0 ? `${minutes}m ${remainder}s` : `${remainder}s`;
};

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; sessions: WorkoutSession[] };

export const WorkoutHistoryPage = () => {
  const history = useHistory();
  const [state, setState] = useState<State>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      setState({ status: "ready", sessions: await fetchWorkoutHistory(50) });
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

  return (
    <PageWrapper className={styles.page}>
      <AnimatedBackground />
      <AccountNav />

      <main className={styles.main}>
        {state.status === "loading" ? <LoadingScreen /> : null}

        {state.status === "error" ? (
          <section className={styles.stateCard}>
            <p>Unable to load your workout history.</p>
            <Button variant="secondary" size="small" onClick={() => setRetryCount((count) => count + 1)} icon={<RefreshCw size={15} />}>
              Try again
            </Button>
          </section>
        ) : null}

        {state.status === "ready" ? (
          <section className={styles.content}>
            <div className={styles.back}>
              <Button to="/services/my-programs" variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
                My Programs
              </Button>
            </div>

            <header className={styles.heading}>
              <span className={styles.icon}>
                <History size={22} aria-hidden="true" />
              </span>
              <div>
                <h1 className={styles.title}>Workout history</h1>
                <p className={styles.subtitle}>
                  Every workout session, newest first.
                </p>
              </div>
            </header>

            {state.sessions.length === 0 ? (
              <section className={styles.stateCard}>
                <p>No workout sessions yet. Start one from your programme.</p>
                <Button to="/services/my-programs" variant="primary" size="small">
                  Open My Programs
                </Button>
              </section>
            ) : (
              <div className={styles.list}>
                {state.sessions.map((session) => (
                  <article key={session.id} className={styles.card}>
                    <div className={styles.cardMain}>
                      <p className={styles.cardEyebrow}>
                        Week {session.week_number} · Day {session.workout_position} ·{" "}
                        {formatDate(session.started_at)}
                      </p>
                      <h2 className={styles.cardTitle}>{session.program_name}</h2>
                      <div className={styles.meta}>
                        <span className={joinClassNames(styles.badge, statusClass(session.status))}>
                          {statusLabel[session.status]}
                        </span>
                        <span>{session.completed_sets}/{session.total_sets} sets</span>
                        <span>{formatDuration(session.elapsed_seconds)}</span>
                        {session.skipped_sets > 0 ? <span>{session.skipped_sets} skipped</span> : null}
                      </div>
                    </div>
                    <Button
                      to={`/services/workout-sessions/${session.id}`}
                      variant="secondary"
                      size="small"
                      icon={<ChevronRight size={15} />}
                    >
                      Details
                    </Button>
                  </article>
                ))}
              </div>
            )}
          </section>
        ) : null}
      </main>
    </PageWrapper>
  );
};
