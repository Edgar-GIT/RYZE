import { ArrowLeft, ChevronRight, History, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory } from "react-router-dom";

import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";
import { fetchWorkoutHistory, type WorkoutSession } from "@/services/workout_sessions_api";

import styles from "./workout_history_page.module.css";

const STATUS_KEY = {
  in_progress: "account.training.inProgress",
  paused: "account.training.paused",
  completed: "account.training.completed",
  abandoned: "account.training.abandoned"
} as const satisfies Record<WorkoutSession["status"], string>;

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
  const { t, formatDate } = useI18n();
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
            <p>{t("account.training.historyError")}</p>
            <Button variant="secondary" size="small" onClick={() => setRetryCount((count) => count + 1)} icon={<RefreshCw size={15} />}>
              {t("common.retry")}
            </Button>
          </section>
        ) : null}

        {state.status === "ready" ? (
          <section className={styles.content}>
            <div className={styles.back}>
              <Button to="/services/my-programs" variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
                {t("account.training.myPrograms")}
              </Button>
            </div>

            <header className={styles.heading}>
              <span className={styles.icon}>
                <History size={22} aria-hidden="true" />
              </span>
              <div>
                <h1 className={styles.title}>{t("account.training.history")}</h1>
                <p className={styles.subtitle}>{t("account.training.historyLead")}</p>
              </div>
            </header>

            {state.sessions.length === 0 ? (
              <section className={styles.stateCard}>
                <p>{t("account.training.historyEmpty")}</p>
                <Button to="/services/my-programs" variant="primary" size="small">
                  {t("account.training.openMyPrograms")}
                </Button>
              </section>
            ) : (
              <div className={styles.list}>
                {state.sessions.map((session) => (
                  <article key={session.id} className={styles.card}>
                    <div className={styles.cardMain}>
                      <p className={styles.cardEyebrow}>
                        {t("account.training.weekDay", { week: session.week_number, day: session.workout_position })} ·{" "}
                        {formatDate(session.started_at, { day: "numeric", month: "short", year: "numeric" })}
                      </p>
                      <h2 className={styles.cardTitle}>{session.program_name}</h2>
                      <div className={styles.meta}>
                        <span className={joinClassNames(styles.badge, statusClass(session.status))}>
                          {t(STATUS_KEY[session.status])}
                        </span>
                        <span>{t("account.training.setsShort", { done: session.completed_sets, total: session.total_sets })}</span>
                        <span>{formatDuration(session.elapsed_seconds)}</span>
                        {session.skipped_sets > 0 ? (
                          <span>{t("account.training.skipped", { count: session.skipped_sets })}</span>
                        ) : null}
                      </div>
                    </div>
                    <Button
                      to={`/services/workout-sessions/${session.id}`}
                      variant="secondary"
                      size="small"
                      icon={<ChevronRight size={15} />}
                    >
                      {t("account.training.details")}
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
