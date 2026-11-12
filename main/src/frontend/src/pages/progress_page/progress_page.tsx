import { BarChart3, CalendarDays, CheckCircle2, Clock3, Dumbbell, Flame, RefreshCw, SkipForward, Trophy } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory } from "react-router-dom";

import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useI18n } from "@/i18n/i18n";
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


type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; stats: ClientStatistics };

export const ProgressPage = () => {
  const { t, formatDate } = useI18n();
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
            <p>{t("account.progress.loadError")}</p>
            <Button variant="secondary" size="small" onClick={() => setRetryCount((count) => count + 1)} icon={<RefreshCw size={15} />}>
              {t("common.retry")}
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
                <h1 className={styles.title}>{t("account.progress.title")}</h1>
                <p className={styles.subtitle}>{t("account.progress.lead")}</p>
              </div>
            </header>

            {!stats.has_active_program ? (
              <section className={styles.stateCard}>
                <p>{t("account.progress.noProgram")}</p>
                <div className={styles.stateActions}>
                  <Button to="/services/my-programs" variant="primary" size="small">
                    {t("account.progress.myPrograms")}
                  </Button>
                  <Button to="/services/generic-program" variant="secondary" size="small">
                    {t("account.progress.browse")}
                  </Button>
                </div>
              </section>
            ) : (
              <>
                <section className={styles.currentCard} aria-labelledby="current-program">
                  <div>
                    <p className={styles.eyebrow}>{t("account.progress.current")}</p>
                    <h2 id="current-program" className={styles.currentTitle}>
                      {stats.current_program_name || t("account.progress.fallbackName")}
                    </h2>
                    <p className={styles.currentMeta}>
                      {stats.has_active_assignment ? t("account.progress.assigned") : t("account.progress.purchased")} ·{" "}
                      {stats.completed_sessions === 1
                        ? t("account.progress.session")
                        : t("account.progress.sessions", { count: stats.completed_sessions })}
                    </p>
                  </div>
                  <div className={styles.completion}>
                    <span className={styles.completionValue}>{percentage}%</span>
                    <span className={styles.completionLabel}>{t("account.progress.complete")}</span>
                  </div>
                </section>

                <div className={styles.progressTrack} aria-label={t("account.progress.completion", { percent: percentage })}>
                  <span style={{ width: `${percentage}%` }} />
                </div>

                <div className={styles.grid}>
                  <article className={styles.metric}>
                    <Dumbbell size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>{t("account.progress.workouts")}</span>
                    <strong className={styles.metricValue}>{stats.unique_workouts_completed}</strong>
                    <em className={styles.metricNote}>
                      {t("account.progress.workoutsNote", { total: stats.total_workouts_in_program })}
                    </em>
                  </article>
                  <article className={styles.metric}>
                    <Flame size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>{t("account.progress.executions")}</span>
                    <strong className={styles.metricValue}>{stats.total_executions}</strong>
                    <em className={styles.metricNote}>{t("account.progress.executionsNote")}</em>
                  </article>
                  <article className={styles.metric}>
                    <CheckCircle2 size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>{t("account.progress.sets")}</span>
                    <strong className={styles.metricValue}>{stats.completed_sets}</strong>
                    <em className={styles.metricNote}>{t("account.progress.setsNote", { count: stats.skipped_sets })}</em>
                  </article>
                  <article className={styles.metric}>
                    <Clock3 size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>{t("account.progress.time")}</span>
                    <strong className={styles.metricValue}>{formatDuration(stats.total_active_seconds)}</strong>
                    <em className={styles.metricNote}>{t("account.progress.timeNote")}</em>
                  </article>
                  <article className={styles.metric}>
                    <CalendarDays size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>{t("account.progress.last")}</span>
                    <strong className={styles.metricValue}>
                      {stats.last_workout_date
                        ? formatDate(stats.last_workout_date, { day: "numeric", month: "short", year: "numeric" })
                        : "—"}
                    </strong>
                    <em className={styles.metricNote}>{t("account.progress.lastNote")}</em>
                  </article>
                  <article className={styles.metric}>
                    <SkipForward size={18} aria-hidden="true" />
                    <span className={styles.metricLabel}>{t("account.progress.abandoned")}</span>
                    <strong className={styles.metricValue}>{stats.abandoned_sessions}</strong>
                    <em className={styles.metricNote}>{t("account.progress.abandonedNote")}</em>
                  </article>
                </div>

                <div className={styles.actions}>
                  <Button to="/services/workout-history" variant="secondary" size="small" icon={<Trophy size={15} />} iconPosition="left">
                    {t("account.progress.history")}
                  </Button>
                  <Button to="/services/my-programs" variant="ghost" size="small">
                    {t("account.progress.myPrograms")}
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
