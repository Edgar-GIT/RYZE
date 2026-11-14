import { ChevronRight, Clock3, Dumbbell, ListChecks, RefreshCw, Trophy } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import styles from "@/components/training/training_page.module.css";
import { useI18n } from "@/i18n/i18n";
import { formatClock, formatWeight, personalRecords, summarizeSession } from "@/services/training_insights";
import { fetchWorkoutHistory, type WorkoutSession } from "@/services/workout_sessions_api";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";

type State = { status: "loading" } | { status: "error" } | { status: "ready"; sessions: WorkoutSession[] };

// WorkoutHistoryPage lists every finished or stopped workout by month: when,
// which day of which plan, how long, how much volume and any records. Each
// entry opens its full report.
export const WorkoutHistoryPage = () => {
  const { t, intlLocale, formatDate } = useI18n();
  const history = useHistory();
  const [state, setState] = useState<State>({ status: "loading" });

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      setState({ status: "ready", sessions: await fetchWorkoutHistory(100) });
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
  }, [load]);

  const months = useMemo(() => {
    if (state.status !== "ready") return [];
    const groups: { key: string; label: string; sessions: WorkoutSession[] }[] = [];
    for (const session of state.sessions) {
      const date = new Date(session.started_at);
      const key = `${date.getFullYear()}-${date.getMonth()}`;
      let group = groups.find((entry) => entry.key === key);
      if (!group) {
        group = { key, label: date.toLocaleDateString(intlLocale, { month: "long", year: "numeric" }), sessions: [] };
        groups.push(group);
      }
      group.sessions.push(session);
    }
    return groups;
  }, [state, intlLocale]);

  return (
    <PageWrapper className={styles.page}>

      <div className={styles.main}>
        <header className={styles.header}>
          <h1 className={styles.title}>{t("training.history.title")}</h1>
          <p className={styles.lead}>{t("training.history.lead")}</p>
        </header>

        {state.status === "loading" ? <p className={styles.lead}>{t("common.loading")}</p> : null}

        {state.status === "error" ? (
          <div className={styles.state}>
            <p>{t("account.training.historyError")}</p>
            <button type="button" className={joinClassNames(styles.button, styles.buttonQuiet)} onClick={() => void load()}>
              <RefreshCw aria-hidden="true" /> {t("common.retry")}
            </button>
          </div>
        ) : null}

        {state.status === "ready" && state.sessions.length === 0 ? (
          <div className={styles.state}>
            <p>{t("training.history.empty")}</p>
            <Link className={styles.button} to="/services/my-programs">
              {t("training.history.emptyAction")}
            </Link>
          </div>
        ) : null}

        {state.status === "ready"
          ? months.map((month) => (
              <section key={month.key} className={styles.month} aria-label={month.label}>
                <h2 className={styles.monthTitle}>{month.label}</h2>
                <ul className={styles.sessionList}>
                  {month.sessions.map((session) => {
                    const summary = summarizeSession(session);
                    const records = session.status === "completed" ? personalRecords(session, state.sessions).length : 0;
                    const date = new Date(session.started_at);
                    return (
                      <li key={session.id}>
                        <Link className={styles.session} to={`/services/workout-sessions/${encodeURIComponent(session.id)}`}>
                          <span className={styles.date} aria-hidden="true">
                            <strong>{date.getDate()}</strong>
                            <span>{date.toLocaleDateString(intlLocale, { weekday: "short" })}</span>
                          </span>
                          <span className={styles.sessionBody}>
                            <span className={styles.sessionTitle}>
                              {t("account.training.weekDay", { week: session.week_number, day: session.workout_position })} ·{" "}
                              {session.program_name}
                            </span>
                            <span className={styles.sessionMeta}>
                              <span>
                                <Clock3 aria-hidden="true" />
                                {formatClock(summary.durationSeconds)}
                              </span>
                              <span>
                                <Dumbbell aria-hidden="true" />
                                {t("training.progress.kg", { value: formatWeight(summary.volumeKg, intlLocale) })}
                              </span>
                              <span>
                                <ListChecks aria-hidden="true" />
                                {summary.completedSets}/{summary.totalSets}
                              </span>
                              {records > 0 ? (
                                <span className={styles.record}>
                                  <Trophy aria-hidden="true" />
                                  {records === 1 ? t("training.history.record") : t("training.history.records", { count: records })}
                                </span>
                              ) : null}
                              {session.status === "abandoned" ? (
                                <span className={styles.abandoned}>{t("training.history.abandoned")}</span>
                              ) : null}
                            </span>
                            <span className={styles.srOnly}>{formatDate(session.started_at)}</span>
                          </span>
                          <ChevronRight className={styles.chevron} aria-label={t("training.history.open")} />
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </section>
            ))
          : null}
      </div>
    </PageWrapper>
  );
};
