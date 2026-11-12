import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { ProgramStructure } from "@/components/program_structure/program_structure";
import {
  fetchActiveWorkoutSession,
  startWorkoutSession,
  type WorkoutSession
} from "@/services/workout_sessions_api";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useI18n } from "@/i18n/i18n";
import { ApiError } from "@utils/http_client";
import { fetchProgramAccess, type ProgramAccessDetail } from "@/services/purchases_api";
import {
  fetchNutritionAssignment,
  generateNutritionAssignment,
  nutritionGenerationErrorMessage,
  type NutritionAssignmentStatus
} from "@/services/premium_level1_api";
import { PremiumNutritionPlan } from "@/pages/premium_level1_page/premium_nutrition_plan";
import { ArrowLeft, BarChart3, History, Play, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory, useParams } from "react-router-dom";

import styles from "./program_access_page.module.css";

// A Premium Level 1 entitlement carries a nutrition programme alongside the
// training plan. The family comes from the backend, so Program Access never
// needs to know a program id in advance.
const isPremiumPackage = (productType: string): boolean => productType === "premium_level_1";

// Nutrition is absent for a generic plan and for any request that the backend
// refuses. Both collapse to the same quiet state: no plan to show, no error to
// explain, because the refusal itself is the privacy guarantee.
const nutritionUnavailable: NutritionAssignmentStatus = {
  program_id: "",
  status: "pending",
  version: 0,
  questionnaire_version: 0,
  out_of_date: false,
  plan: null
};

export const ProgramAccessPage = () => {
  const { t } = useI18n();
  const { programId } = useParams<{ programId: string }>();
  const history = useHistory();
  const [detail, setDetail] = useState<ProgramAccessDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [retryCount, setRetryCount] = useState(0);

  const [nutrition, setNutrition] = useState<NutritionAssignmentStatus>(nutritionUnavailable);
  const [nutritionLoaded, setNutritionLoaded] = useState(false);
  const [nutritionError, setNutritionError] = useState("");
  const [generating, setGenerating] = useState(false);

  const [activeSession, setActiveSession] = useState<WorkoutSession | null>(null);
  const [startingWorkoutId, setStartingWorkoutId] = useState<string | null>(null);
  const [sessionError, setSessionError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");
    try {
      setDetail(await fetchProgramAccess(programId));
    } catch (error) {
      setDetail(null);
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setErrorMessage(
        error instanceof ApiError && error.status === 404 ? t("account.training.accessUnavailable") : t("account.training.accessError")
      );
    } finally {
      setLoading(false);
    }
  }, [programId, history, t]);

  const loadActiveSession = useCallback(async () => {
    try {
      setActiveSession(await fetchActiveWorkoutSession());
    } catch {
      // The banner is an enhancement, never the reason the plan fails to render.
      setActiveSession(null);
    }
  }, []);

  const handleStartWorkout = useCallback(
    async (workoutId: string) => {
      setStartingWorkoutId(workoutId);
      setSessionError("");
      try {
        const session = await startWorkoutSession(programId, workoutId);
        history.push(`/services/workout-sessions/${session.id}`);
      } catch {
        setSessionError(t("account.training.otherSessionActive"));
        await loadActiveSession();
        setStartingWorkoutId(null);
      }
    },
    [programId, history, loadActiveSession, t]
  );

  const loadNutrition = useCallback(async () => {
    try {
      setNutrition(await fetchNutritionAssignment(programId));
      setNutritionLoaded(true);
    } catch {
      setNutritionLoaded(false);
    }
  }, [programId]);

  const handleGenerate = useCallback(async () => {
    setGenerating(true);
    setNutritionError("");
    try {
      setNutrition(await generateNutritionAssignment(programId));
      setNutritionLoaded(true);
    } catch (error) {
      // Wording is translated here: a server reason token is never rendered.
      setNutritionError(nutritionGenerationErrorMessage(error));
      // Re-read the authoritative state so a failed update never hides a plan
      // that is still there. If even the read fails, the view stands as it is.
      try {
        setNutrition(await fetchNutritionAssignment(programId));
        setNutritionLoaded(true);
      } catch {
        // Nothing further to do: the previously rendered state remains correct.
      }
    } finally {
      setGenerating(false);
    }
  }, [programId]);

  useEffect(() => {
    void load();
  }, [load, retryCount]);

  useEffect(() => {
    if (detail) {
      void loadActiveSession();
    }
  }, [detail, loadActiveSession]);

  // Nutrition is fetched only for a Premium Level 1 entitlement, and only after
  // the entitled program itself resolved. A generic plan never issues the
  // request at all.
  const premium = detail !== null && isPremiumPackage(detail.product_type);

  useEffect(() => {
    if (!premium) {
      setNutritionLoaded(false);
      return;
    }
    void loadNutrition();
  }, [premium, loadNutrition]);

  return (
    <PageWrapper className={styles.page}>
      <AnimatedBackground />

      <AccountNav />

      <main className={styles.main}>
        {loading ? <LoadingScreen /> : null}

        {!loading && (errorMessage || !detail) ? (
          <section className={styles.stateCard}>
            <p>{errorMessage}</p>
            <div className={styles.retry}>
              <Button
                variant="secondary"
                size="small"
                onClick={() => setRetryCount((count) => count + 1)}
                icon={<RefreshCw size={15} />}
              >
                {t("common.retry")}
              </Button>
            </div>
          </section>
        ) : null}

        {!loading && detail && !errorMessage ? (
          <section className={styles.content}>
            <div className={styles.back}>
              <Button to="/services/my-programs" variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
                {t("account.training.myPrograms")}
              </Button>
            </div>

            {activeSession ? (
              <aside className={styles.activeBanner}>
                <div>
                  <p className={styles.activeEyebrow}>{t("account.training.workoutInProgress")}</p>
                  <p className={styles.activeTitle}>
                    {activeSession.program_name} ·{" "}
                    {t("account.training.weekDay", { week: activeSession.week_number, day: activeSession.workout_position })}
                  </p>
                </div>
                <Button
                  to={`/services/workout-sessions/${activeSession.id}`}
                  variant="primary"
                  size="small"
                  icon={<Play size={14} />}
                  iconPosition="left"
                >
                  {activeSession.status === "paused" ? t("account.training.resumeWorkout") : t("account.training.continueWorkout")}
                </Button>
              </aside>
            ) : null}

            <header className={styles.heading}>
              <h1 className={styles.title}>{detail.name}</h1>
              {detail.description ? <p className={styles.subtitle}>{detail.description}</p> : null}

              <div className={styles.chips}>
                {premium ? <span className={styles.chipPremium}>{t("account.myPrograms.pl1")}</span> : null}
                {premium ? <span className={styles.chip}>{t("account.myPrograms.pl1Chip")}</span> : null}
                {detail.training_type ? <span className={styles.chip}>{detail.training_type}</span> : null}
                {detail.level ? <span className={styles.chip}>{detail.level}</span> : null}
                {detail.duration_weeks ? (
                  <span className={styles.chip}>
                    {detail.duration_weeks === 1
                      ? t("account.myPrograms.week")
                      : t("account.myPrograms.weeks", { count: detail.duration_weeks })}
                  </span>
                ) : null}
                {detail.frequency_per_week ? (
                  <span className={styles.chip}>
                    {detail.frequency_per_week === 1
                      ? t("account.myPrograms.dayPerWeek")
                      : t("account.myPrograms.daysPerWeek", { count: detail.frequency_per_week })}
                  </span>
                ) : null}
              </div>
            </header>

            {sessionError ? (
              <p className={styles.sessionError} role="alert">
                {sessionError}
              </p>
            ) : null}

            <div className={styles.actions}>
              <Button to="/services/workout-history" variant="secondary" size="small" icon={<History size={15} />} iconPosition="left">
                {t("account.training.history")}
              </Button>
              <Button to="/services/progress" variant="secondary" size="small" icon={<BarChart3 size={15} />} iconPosition="left">
                {t("account.training.viewProgress")}
              </Button>
            </div>

            <ProgramStructure
              weeks={detail.weeks}
              onStartWorkout={(workoutId) => void handleStartWorkout(workoutId)}
              activeWorkoutId={
                activeSession?.program_id === programId
                  ? activeSession.program_workout_id
                  : null
              }
              startingWorkoutId={startingWorkoutId}
            />

            {premium && nutritionLoaded ? (
              <PremiumNutritionPlan
                status={nutrition}
                generating={generating}
                errorMessage={nutritionError}
                onGenerate={() => void handleGenerate()}
                onRefresh={() => void loadNutrition()}
              />
            ) : null}
          </section>
        ) : null}
      </main>
    </PageWrapper>
  );
};