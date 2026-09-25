import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { ProgramStructure } from "@/components/program_structure/program_structure";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ApiError } from "@utils/http_client";
import { fetchProgramAccess, type ProgramAccessDetail } from "@/services/purchases_api";
import { ArrowLeft, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory, useParams } from "react-router-dom";

import styles from "./program_access_page.module.css";

export const ProgramAccessPage = () => {
  const { programId } = useParams<{ programId: string }>();
  const history = useHistory();
  const [detail, setDetail] = useState<ProgramAccessDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [retryCount, setRetryCount] = useState(0);

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

  useEffect(() => {
    void load();
  }, [load, retryCount]);

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
          </section>
        ) : null}
      </main>
    </PageWrapper>
  );
};