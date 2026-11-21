import { CalendarCheck, Dumbbell, Flame, RefreshCw, Trophy } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";

import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { BarChart, LineChart, type ChartPoint } from "@/components/training/charts";
import styles from "@/components/training/training_page.module.css";
import { useI18n } from "@/i18n/i18n";
import { fetchClientStatistics, type ClientStatistics } from "@/services/statistics_api";
import {
  currentStreakWeeks,
  exerciseSeries,
  formatWeight,
  summarizeSession,
  trainedExercises,
  weeklyBuckets
} from "@/services/training_insights";
import { fetchWorkoutHistory, type WorkoutSession } from "@/services/workout_sessions_api";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";

import { MeasurementsPanel } from "./measurements_panel";
import { MuscleRecoveryPanel } from "./muscle_recovery_panel";
import tabStyles from "./progress_tabs.module.css";

// The progress page has three views (owner request, 2026-11-20): training
// charts, the muscle recovery body map and body measurements. The view is in
// the address (?tab=) so it survives a reload and can be linked to.
type Tab = "training" | "muscles" | "measurements";
const TABS: Tab[] = ["training", "muscles", "measurements"];
const TAB_LABELS = {
  en: { training: "Training", muscles: "Muscles", measurements: "Measurements" },
  pt: { training: "Treino", muscles: "Músculos", measurements: "Medições" }
} as const;

type Metric = "weight" | "e1rm" | "volume";

const DAY = 24 * 60 * 60 * 1000;

// ProgressPage turns saved workouts into a quick read of progress: how often
// you train, how much you lift, and how each exercise is moving.
export const ProgressPage = () => {
  const { t, intlLocale, locale } = useI18n();
  const history = useHistory();
  const location = useLocation();
  const requested = new URLSearchParams(location.search).get("tab") as Tab | null;
  const tab: Tab = requested && TABS.includes(requested) ? requested : "training";
  const selectTab = (next: Tab) => history.replace({ pathname: location.pathname, search: next === "training" ? "" : `?tab=${next}` });
  const [sessions, setSessions] = useState<WorkoutSession[] | null>(null);
  const [stats, setStats] = useState<ClientStatistics | null>(null);
  const [error, setError] = useState(false);
  const [exerciseId, setExerciseId] = useState("");
  const [metric, setMetric] = useState<Metric>("weight");

  const load = useCallback(async () => {
    setError(false);
    try {
      const [loaded, statistics] = await Promise.all([fetchWorkoutHistory(100), fetchClientStatistics().catch(() => null)]);
      setSessions(loaded);
      setStats(statistics);
    } catch (caught) {
      if (caught instanceof ApiError && caught.status === 401) {
        history.replace("/login");
        return;
      }
      setError(true);
    }
  }, [history]);

  useEffect(() => {
    void load();
  }, [load]);

  const completed = useMemo(() => (sessions ?? []).filter((session) => session.status === "completed"), [sessions]);
  const exercises = useMemo(() => trainedExercises(completed), [completed]);
  const selectedExercise = exerciseId || exercises[0]?.exerciseId || "";

  const kg = (value: number) => t("training.progress.kg", { value: formatWeight(value, intlLocale) });
  const shortDate = (iso: string | Date) => new Date(iso).toLocaleDateString(intlLocale, { day: "numeric", month: "short" });

  const weeks = useMemo(() => weeklyBuckets(completed, 8), [completed]);
  const weekPoints: ChartPoint[] = weeks.map((week) => ({
    label: shortDate(week.weekStart),
    value: week.workouts,
    description: t("training.progress.chartBars", { label: shortDate(week.weekStart), value: week.workouts })
  }));

  const recent = [...completed].sort((a, b) => a.started_at.localeCompare(b.started_at)).slice(-12);
  const volumePoints: ChartPoint[] = recent.map((session) => {
    const volume = summarizeSession(session).volumeKg;
    return {
      label: shortDate(session.started_at),
      value: Math.round(volume),
      description: t("training.progress.chartBars", { label: shortDate(session.started_at), value: kg(volume) })
    };
  });

  const series = selectedExercise ? exerciseSeries(completed, selectedExercise).slice(-16) : [];
  const exercisePoints: ChartPoint[] = series.map((point) => {
    const value = metric === "weight" ? point.bestWeight : metric === "e1rm" ? Math.round(point.e1rm * 10) / 10 : Math.round(point.volumeKg);
    return {
      label: shortDate(point.date),
      value,
      description: t("training.progress.chartBars", { label: shortDate(point.date), value: kg(value) })
    };
  });

  const now = Date.now();
  const thisWeek = weeks[weeks.length - 1]?.workouts ?? 0;
  const volume30 = completed
    .filter((session) => now - new Date(session.started_at).getTime() <= 30 * DAY)
    .reduce((sum, session) => sum + summarizeSession(session).volumeKg, 0);
  const streak = currentStreakWeeks(completed);

  return (
    <PageWrapper className={styles.page}>

      <div className={styles.main}>
        <header className={styles.header}>
          <h1 className={styles.title}>{t("training.progress.title")}</h1>
          <p className={styles.lead}>{t("training.progress.lead")}</p>
        </header>

        <div className={tabStyles.tabs} role="tablist" aria-label={t("training.progress.title")}>
          {TABS.map((item) => (
            <button
              key={item}
              type="button"
              role="tab"
              id={`progress-tab-${item}`}
              aria-selected={tab === item}
              aria-controls={`progress-panel-${item}`}
              className={joinClassNames(tabStyles.tab, tab === item && tabStyles.tabActive)}
              onClick={() => selectTab(item)}
            >
              {TAB_LABELS[locale][item]}
            </button>
          ))}
        </div>

        {tab === "muscles" ? (
          <div role="tabpanel" id="progress-panel-muscles" aria-labelledby="progress-tab-muscles">
            <MuscleRecoveryPanel />
          </div>
        ) : null}
        {tab === "measurements" ? (
          <div role="tabpanel" id="progress-panel-measurements" aria-labelledby="progress-tab-measurements">
            <MeasurementsPanel />
          </div>
        ) : null}
        {tab === "training" ? (
          <div role="tabpanel" id="progress-panel-training" aria-labelledby="progress-tab-training" className={tabStyles.panel}>

          {error ? (
            <div className={styles.state}>
              <p>{t("account.progress.loadError")}</p>
              <button type="button" className={joinClassNames(styles.button, styles.buttonQuiet)} onClick={() => void load()}>
                <RefreshCw aria-hidden="true" /> {t("common.retry")}
              </button>
            </div>
          ) : null}

          {!sessions && !error ? <p className={styles.lead}>{t("common.loading")}</p> : null}

          {sessions ? (
            <>
              <dl className={styles.tiles}>
                <div className={styles.tile}>
                  <dt>
                    <Dumbbell aria-hidden="true" /> {t("training.progress.workouts")}
                  </dt>
                  <dd>{completed.length}</dd>
                </div>
                <div className={styles.tile}>
                  <dt>
                    <CalendarCheck aria-hidden="true" /> {t("training.progress.thisWeek")}
                  </dt>
                  <dd>{thisWeek}</dd>
                </div>
                <div className={styles.tile}>
                  <dt>
                    <Trophy aria-hidden="true" /> {t("training.progress.volume30")}
                  </dt>
                  <dd>{kg(volume30)}</dd>
                </div>
                <div className={styles.tile}>
                  <dt>
                    <Flame aria-hidden="true" /> {t("training.progress.streak")}
                  </dt>
                  <dd>{streak}</dd>
                </div>
              </dl>

              {stats?.has_active_program ? (
                <section className={styles.card} aria-label={t("training.progress.currentProgram")}>
                  <div className={styles.programRow}>
                    <div>
                      <p className={styles.programMeta}>{t("training.progress.currentProgram")}</p>
                      <p className={styles.programName}>{stats.current_program_name || t("account.progress.fallbackName")}</p>
                      <p className={styles.programMeta}>
                        {t("training.progress.complete", { percent: Math.round(stats.completion_percentage) })} ·{" "}
                        {t("training.progress.workoutsDone", {
                          done: stats.unique_workouts_completed,
                          total: stats.total_workouts_in_program
                        })}
                      </p>
                    </div>
                    <Link className={joinClassNames(styles.button, styles.buttonQuiet)} to="/services/my-programs">
                      {t("training.progress.openPlan")}
                    </Link>
                  </div>
                  <div className={styles.meter} aria-hidden="true">
                    <span style={{ width: `${Math.min(100, Math.max(0, stats.completion_percentage))}%` }} />
                  </div>
                </section>
              ) : null}

              {completed.length === 0 ? (
                <div className={styles.state}>
                  <p>{t("training.progress.noData")}</p>
                  <Link className={styles.button} to="/services/my-programs">
                    {t("training.history.emptyAction")}
                  </Link>
                </div>
              ) : (
                <div className={styles.charts}>
                  <section className={styles.card}>
                    <h2 className={styles.cardTitle}>{t("training.progress.perWeek")}</h2>
                    <BarChart title={t("training.progress.perWeek")} points={weekPoints} />
                  </section>
                  <section className={styles.card}>
                    <h2 className={styles.cardTitle}>{t("training.progress.volumePerWorkout")}</h2>
                    <BarChart title={t("training.progress.volumePerWorkout")} points={volumePoints} format={(value) => formatWeight(value, intlLocale)} />
                  </section>
                  <section className={joinClassNames(styles.card, styles.wide)}>
                    <h2 className={styles.cardTitle}>
                      {t("training.progress.exercise")}
                      <select
                        className={styles.select}
                        value={selectedExercise}
                        onChange={(event) => setExerciseId(event.target.value)}
                        aria-label={t("training.progress.pickExercise")}
                      >
                        {exercises.map((exercise) => (
                          <option key={exercise.exerciseId} value={exercise.exerciseId}>
                            {exercise.name} ({t("training.progress.sessionsDone", { count: exercise.sessions })})
                          </option>
                        ))}
                      </select>
                    </h2>
                    <div className={styles.metricTabs} role="group" aria-label={t("training.progress.exercise")}>
                      {(["weight", "e1rm", "volume"] as const).map((option) => (
                        <button
                          key={option}
                          type="button"
                          aria-pressed={metric === option}
                          className={joinClassNames(styles.metricTab, metric === option && styles.metricTabActive)}
                          onClick={() => setMetric(option)}
                        >
                          {option === "weight"
                            ? t("training.progress.metricWeight")
                            : option === "e1rm"
                              ? t("training.progress.metricE1rm")
                              : t("training.progress.metricVolume")}
                        </button>
                      ))}
                    </div>
                    <LineChart title={t("training.progress.exercise")} points={exercisePoints} format={(value) => formatWeight(value, intlLocale)} />
                  </section>
                </div>
              )}
            </>
          ) : null}
          </div>
        ) : null}
      </div>
    </PageWrapper>
  );
};
