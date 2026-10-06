import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { ProgramStructure } from "@/components/program_structure/program_structure";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ApiError } from "@utils/http_client";
import { fetchProgramAccess, type ProgramAccessDetail } from "@/services/purchases_api";
import {
  fetchNutritionAssignment,
  generateNutritionAssignment,
  nutritionGenerationErrorMessage,
  type NutritionAssignmentStatus
} from "@/services/premium_level1_api";
import { PremiumNutritionPlan } from "@/pages/premium_level1_page/premium_nutrition_plan";
import { ArrowLeft, RefreshCw } from "lucide-react";
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
      setErrorMessage(error instanceof ApiError && error.status === 404 ? "This training plan is no longer available." : "Unable to load this training plan.");
    } finally {
      setLoading(false);
    }
  }, [programId, history]);

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
                Try again
              </Button>
            </div>
          </section>
        ) : null}

        {!loading && detail && !errorMessage ? (
          <section className={styles.content}>
            <div className={styles.back}>
              <Button to="/services/my-programs" variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
                My Programs
              </Button>
            </div>

            <header className={styles.heading}>
              <h1 className={styles.title}>{detail.name}</h1>
              {detail.description ? <p className={styles.subtitle}>{detail.description}</p> : null}

              <div className={styles.chips}>
                {premium ? <span className={styles.chipPremium}>Premium Level 1</span> : null}
                {premium ? <span className={styles.chip}>Training + nutrition</span> : null}
                {detail.training_type ? <span className={styles.chip}>{detail.training_type}</span> : null}
                {detail.level ? <span className={styles.chip}>{detail.level}</span> : null}
                {detail.duration_weeks ? (
                  <span className={styles.chip}>
                    {detail.duration_weeks} week{detail.duration_weeks === 1 ? "" : "s"}
                  </span>
                ) : null}
                {detail.frequency_per_week ? (
                  <span className={styles.chip}>
                    {detail.frequency_per_week} day{detail.frequency_per_week === 1 ? "" : "s"}/week
                  </span>
                ) : null}
              </div>
            </header>

            <ProgramStructure weeks={detail.weeks} />

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