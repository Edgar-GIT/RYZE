import { CalendarCheck, Camera, ChevronRight, Clock3, Dumbbell, ListChecks, RefreshCw, Trophy } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import styles from "@/components/training/training_page.module.css";
import cards from "./workout_history_page.module.css";
import { useI18n } from "@/i18n/i18n";
import { formatClock, formatWeight, personalRecords, summarizeSession } from "@/services/training_insights";
import { fetchWorkoutHistory, type WorkoutSession } from "@/services/workout_sessions_api";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";

const COPY = {
  en: {
    period: "Last 30 days",
    workouts: "Workouts",
    time: "Time",
    volume: "Volume",
    records: "Records",
    count: (count: number) => `${count} ${count === 1 ? "workout" : "workouts"}`,
    more: (count: number) => `+${count} more`
  },
  pt: {
    period: "Últimos 30 dias",
    workouts: "Treinos",
    time: "Tempo",
    volume: "Volume",
    records: "Recordes",
    count: (count: number) => `${count} ${count === 1 ? "treino" : "treinos"}`,
    more: (count: number) => `+${count}`
  }
} as const;

const DAY = 24 * 60 * 60 * 1000;

type State = { status: "loading" } | { status: "error" } | { status: "ready"; sessions: WorkoutSession[] };

// WorkoutHistoryPage lists every finished or stopped workout by month: when,
// which day of which plan, how long, how much volume and any records. Each
// entry opens its full report.
export const WorkoutHistoryPage = () => {
  const { t, intlLocale, formatDate, locale } = useI18n();
  const text = COPY[locale];
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
        const label = date.toLocaleDateString(intlLocale, { month: "long", year: "numeric" });
        group = { key, label: label.charAt(0).toUpperCase() + label.slice(1), sessions: [] };
        groups.push(group);
      }
      group.sessions.push(session);
    }
    return groups;
  }, [state, intlLocale]);

  const recent = useMemo(() => {
    if (state.status !== "ready") return null;
    const now = Date.now();
    const last = state.sessions.filter((session) => session.status === "completed" && now - new Date(session.started_at).getTime() <= 30 * DAY);
    return {
      workouts: last.length,
      seconds: last.reduce((sum, session) => sum + summarizeSession(session).durationSeconds, 0),
      volume: last.reduce((sum, session) => sum + summarizeSession(session).volumeKg, 0),
      records: last.reduce((sum, session) => sum + personalRecords(session, state.sessions).length, 0)
    };
  }, [state]);

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

        {recent && state.status === "ready" && state.sessions.length > 0 ? (
          <section className={cards.kpis} aria-label={text.period}>
            <h2 className={cards.kpiTitle}>{text.period}</h2>
            <dl>
              {[
                { icon: CalendarCheck, label: text.workouts, value: String(recent.workouts) },
                { icon: Clock3, label: text.time, value: formatClock(recent.seconds) },
                { icon: Dumbbell, label: text.volume, value: t("training.progress.kg", { value: formatWeight(recent.volume, intlLocale) }) },
                { icon: Trophy, label: text.records, value: String(recent.records) }
              ].map(({ icon: Icon, label, value }) => (
                <div key={label}>
                  <dt>
                    <Icon aria-hidden="true" /> {label}
                  </dt>
                  <dd>{value}</dd>
                </div>
              ))}
            </dl>
          </section>
        ) : null}

        {state.status === "ready"
          ? months.map((month) => (
              <section key={month.key} className={styles.month} aria-label={month.label}>
                <h2 className={cards.monthTitle}>
                  {month.label} <span>{text.count(month.sessions.length)}</span>
                </h2>
                <ul className={cards.list}>
                  {month.sessions.map((session) => {
                    const summary = summarizeSession(session);
                    const records = session.status === "completed" ? personalRecords(session, state.sessions).length : 0;
                    const date = new Date(session.started_at);
                    return (
                      <li key={session.id}>
                        <Link
                          className={joinClassNames(cards.card, session.status === "abandoned" && cards.cardAbandoned)}
                          to={`/services/workout-sessions/${encodeURIComponent(session.id)}`}
                        >
                          <span className={cards.date} aria-hidden="true">
                            <strong>{date.getDate()}</strong>
                            <span>{date.toLocaleDateString(intlLocale, { weekday: "short" })}</span>
                          </span>
                          <span className={cards.body}>
                            <span className={cards.title}>
                              {t("account.training.weekDay", { week: session.week_number, day: session.workout_position })} · {session.program_name}
                            </span>
                            {session.exercises.length > 0 ? (
                              <span className={cards.exercises}>
                                {session.exercises
                                  .slice(0, 3)
                                  .map((exercise) => exercise.name)
                                  .join(" · ")}
                                {session.exercises.length > 3 ? ` ${text.more(session.exercises.length - 3)}` : ""}
                              </span>
                            ) : null}
                            <span className={cards.meta}>
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
                                <span className={cards.record}>
                                  <Trophy aria-hidden="true" />
                                  {records === 1 ? t("training.history.record") : t("training.history.records", { count: records })}
                                </span>
                              ) : null}
                              {session.status === "abandoned" ? <span className={cards.abandoned}>{t("training.history.abandoned")}</span> : null}
                              {session.has_photo ? (
                                <span className={cards.photo} aria-label={locale === "pt" ? "Com foto" : "With photo"}>
                                  <Camera aria-hidden="true" />
                                </span>
                              ) : null}
                            </span>
                            <span className={cards.bar} aria-hidden="true">
                              <span style={{ width: `${summary.totalSets > 0 ? Math.round((summary.completedSets / summary.totalSets) * 100) : 0}%` }} />
                            </span>
                            <span className={styles.srOnly}>{formatDate(session.started_at)}</span>
                          </span>
                          <ChevronRight className={cards.chevron} aria-label={t("training.history.open")} />
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
