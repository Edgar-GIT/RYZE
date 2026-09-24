import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { BrandMark } from "@/components/brand_mark/brand_mark";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ApiError } from "@utils/http_client";
import { fetchEntitlements, type Entitlement } from "@/services/purchases_api";
import { ChevronRight, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import styles from "./my_programs_page.module.css";

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "empty" }
  | { status: "ready"; entitlements: Entitlement[] };

export const MyProgramsPage = () => {
  const history = useHistory();
  const [state, setState] = useState<State>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const entitlements = await fetchEntitlements();
      setState(
        entitlements.length === 0
          ? { status: "empty" }
          : { status: "ready", entitlements }
      );
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

      <header className={styles.navbar}>
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <nav className={styles.centerNav} aria-label="Account">
          <Link className={styles.navLink} to="/profile">
            Profile
          </Link>
        </nav>

        <div className={styles.spacer} aria-hidden="true" />
      </header>

      <main className={styles.main}>
        {state.status === "loading" ? <LoadingScreen /> : null}

        {state.status === "error" ? (
          <section className={styles.stateCard}>
            <p>Unable to load your programs.</p>
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

        {state.status === "empty" ? (
          <section className={styles.stateCard}>
            <h1 className={styles.stateTitle}>My Programs</h1>
            <p className={styles.stateText}>
              You have no programs yet. Browse the marketplace to find your first training plan.
            </p>
            <div className={styles.retry}>
              <Button to="/services/generic-program" variant="primary" size="small">
                Browse training plans
              </Button>
            </div>
          </section>
        ) : null}

        {state.status === "ready" ? (
          <section className={styles.content}>
            <header className={styles.heading}>
              <h1 className={styles.title}>My Programs</h1>
              <p className={styles.subtitle}>Your purchased training plans.</p>
            </header>

            <div className={styles.grid}>
              {state.entitlements.map((entitlement) => {
                const program = entitlement.program;
                return (
                  <Link
                    key={entitlement.id}
                    className={styles.card}
                    to={`/services/my-programs/${entitlement.program_id}`}
                  >
                    <span className={styles.cardTitle}>{program.name}</span>
                    {program.description ? (
                      <span className={styles.cardDescription}>{program.description}</span>
                    ) : null}

                    <span className={styles.chips}>
                      {program.training_type ? <span className={styles.chip}>{program.training_type}</span> : null}
                      {program.level ? <span className={styles.chip}>{program.level}</span> : null}
                      {program.duration_weeks ? (
                        <span className={styles.chip}>
                          {program.duration_weeks} week{program.duration_weeks === 1 ? "" : "s"}
                        </span>
                      ) : null}
                      {program.frequency_per_week ? (
                        <span className={styles.chip}>
                          {program.frequency_per_week} day{program.frequency_per_week === 1 ? "" : "s"}/week
                        </span>
                      ) : null}
                    </span>

                    <span className={styles.cardAction}>
                      Open program
                      <ChevronRight size={15} aria-hidden="true" />
                    </span>
                  </Link>
                );
              })}
            </div>
          </section>
        ) : null}
      </main>
    </PageWrapper>
  );
};