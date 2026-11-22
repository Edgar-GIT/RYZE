import { useEffect, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useI18n } from "@/i18n/i18n";
import { fetchPlanFocus, loadLibraryPlans, nextWorkoutOf, pickFocus } from "@/services/plan_focus";
import { fetchProgramAccess } from "@/services/purchases_api";
import { fetchActiveWorkoutSession, fetchWorkoutHistory, startWorkoutSession } from "@/services/workout_sessions_api";
import { ApiError } from "@utils/http_client";

import styles from "@/components/training/training_page.module.css";

const COPY = {
  en: { error: "We could not open your next workout.", library: "Open your plans" },
  pt: { error: "Não foi possível abrir o teu próximo treino.", library: "Abrir os teus planos" }
} as const;

// NextWorkoutPage is where "Começar treino" leads (owner request,
// 2026-11-22): the workout in progress, else the next workout of the plan in
// focus (started now), else the plan itself when it is complete, else the
// library when there is no plan.
export const NextWorkoutPage = () => {
  const { locale } = useI18n();
  const history = useHistory();
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const go = (path: string) => {
      if (!cancelled) history.replace(path);
    };
    (async () => {
      const active = await fetchActiveWorkoutSession();
      if (active) return go(`/services/workout-sessions/${encodeURIComponent(active.id)}`);
      const [plans, sessions, choice] = await Promise.all([loadLibraryPlans(), fetchWorkoutHistory(60), fetchPlanFocus().catch(() => null)]);
      const focus = pickFocus(plans, sessions, null, choice);
      if (!focus) return go("/services/my-programs");
      const plan = await fetchProgramAccess(focus);
      const next = nextWorkoutOf(plan, sessions);
      if (!next) return go(`/services/my-programs/${encodeURIComponent(focus)}`);
      const session = await startWorkoutSession(focus, next.workout.id);
      go(`/services/workout-sessions/${encodeURIComponent(session.id)}`);
    })().catch((error) => {
      if (cancelled) return;
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setFailed(true);
    });
    return () => {
      cancelled = true;
    };
  }, [history]);

  if (!failed) return <AppLoading />;
  return (
    <PageWrapper className={styles.page}>
      <div className={styles.main}>
        <div className={styles.state}>
          <p>{COPY[locale].error}</p>
          <Link className={styles.button} to="/services/my-programs">
            {COPY[locale].library}
          </Link>
        </div>
      </div>
    </PageWrapper>
  );
};
