import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useI18n } from "@/i18n/i18n";
import { ApiError } from "@utils/http_client";
import { fetchEntitlements } from "@/services/purchases_api";
import { fetchFreeActivations } from "@/services/free_programs_api";
import { fetchMyPremiumLevel2Requests } from "@/services/premium_level_2_api";
import { ChevronRight, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import styles from "./my_programs_page.module.css";

// Premium Level 1 ships a complete package with the offer. Premium Level 2 is a
// trainer-delivered service, so its entitlement opens the service request where
// the intake, allocation and delivered plans live. The family comes from the
// backend, never from a hardcoded program id.
const isPremiumLevel1 = (productType: string): boolean => productType === "premium_level_1";
const isPremiumLevel2 = (productType: string): boolean => productType === "premium_level_2";

interface ProgramCard {
  key: string;
  programId: string;
  name: string;
  description: string;
  productType: string;
  to: string;
  trainingType: string | null;
  level: string | null;
  durationWeeks: number | null;
  frequencyPerWeek: number | null;
  free: boolean;
}

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "empty" }
  | { status: "ready"; programs: ProgramCard[] };

export const MyProgramsPage = () => {
  const { t } = useI18n();
  const history = useHistory();
  const [state, setState] = useState<State>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const [entitlements, activations, level2Requests] = await Promise.all([
        fetchEntitlements(),
        fetchFreeActivations(),
        fetchMyPremiumLevel2Requests().catch(() => [])
      ]);
      const programs: ProgramCard[] = [
        ...entitlements.map((entitlement) => {
          const level2Request = isPremiumLevel2(entitlement.program.product_type)
            ? level2Requests.find((request) => request.program_id === entitlement.program_id)
            : undefined;
          return {
            key: entitlement.id,
            programId: entitlement.program_id,
            name: entitlement.program.name,
            description: entitlement.program.description,
            productType: entitlement.program.product_type,
            to: level2Request
              ? `/services/premium-level-2/requests/${level2Request.id}`
              : isPremiumLevel2(entitlement.program.product_type)
                ? `/services/premium-level-2?program=${entitlement.program_id}`
                : `/services/my-programs/${entitlement.program_id}`,
            trainingType: entitlement.program.training_type ?? null,
            level: entitlement.program.level ?? null,
            durationWeeks: entitlement.program.duration_weeks ?? null,
            frequencyPerWeek: entitlement.program.frequency_per_week ?? null,
            free: entitlement.program.type === "free"
          };
        }),
        ...activations.map((activation) => ({
          key: activation.id,
          programId: activation.program.id,
          name: activation.program.name,
          description: activation.program.description,
          productType: "generic",
          to: `/services/my-programs/${activation.program.id}`,
          trainingType: activation.program.training_type,
          level: activation.program.level,
          durationWeeks: activation.program.duration_weeks,
          frequencyPerWeek: activation.program.frequency_per_week,
          free: true
        }))
      ];
      setState(programs.length === 0 ? { status: "empty" } : { status: "ready", programs });
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
            <p>{t("account.myPrograms.loadError")}</p>
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

        {state.status === "empty" ? (
          <section className={styles.stateCard}>
            <h1 className={styles.stateTitle}>{t("account.myPrograms.title")}</h1>
            <p className={styles.stateText}>{t("account.myPrograms.empty")}</p>
            <div className={styles.retry}>
              <Button to="/services/generic-program" variant="primary" size="small">
                {t("account.myPrograms.browse")}
              </Button>
              <Button to="/services/premium-level-1" variant="secondary" size="small">
                {t("account.myPrograms.explorePl1")}
              </Button>
              <Button to="/services/premium-level-2" variant="secondary" size="small">
                {t("account.myPrograms.explorePl2")}
              </Button>
            </div>
          </section>
        ) : null}

        {state.status === "ready" ? (
          <section className={styles.content}>
            <header className={styles.heading}>
              <h1 className={styles.title}>{t("account.myPrograms.title")}</h1>
              <p className={styles.subtitle}>{t("account.myPrograms.subtitle")}</p>
            </header>

            <div className={styles.grid}>
              {state.programs.map((program) => {
                const premiumLevel1 = isPremiumLevel1(program.productType);
                const premiumLevel2 = isPremiumLevel2(program.productType);
                return (
                  <Link key={program.key} className={styles.card} to={program.to}>
                    <span className={styles.cardTitle}>{program.name}</span>
                    {program.description ? (
                      <span className={styles.cardDescription}>{program.description}</span>
                    ) : null}

                    <span className={styles.chips}>
                      {program.free ? <span className={styles.chipFree}>{t("account.myPrograms.free")}</span> : null}
                      {premiumLevel1 ? (
                        <span className={styles.chipPremium}>{t("account.myPrograms.pl1")}</span>
                      ) : null}
                      {premiumLevel1 ? (
                        <span className={styles.chip}>{t("account.myPrograms.pl1Chip")}</span>
                      ) : null}
                      {premiumLevel2 ? (
                        <span className={styles.chipPremium}>{t("account.myPrograms.pl2")}</span>
                      ) : null}
                      {premiumLevel2 ? (
                        <span className={styles.chip}>{t("account.myPrograms.pl2Chip")}</span>
                      ) : null}
                      {program.trainingType ? <span className={styles.chip}>{program.trainingType}</span> : null}
                      {program.level ? <span className={styles.chip}>{program.level}</span> : null}
                      {program.durationWeeks ? (
                        <span className={styles.chip}>
                          {program.durationWeeks === 1
                            ? t("account.myPrograms.week")
                            : t("account.myPrograms.weeks", { count: program.durationWeeks })}
                        </span>
                      ) : null}
                      {program.frequencyPerWeek ? (
                        <span className={styles.chip}>
                          {program.frequencyPerWeek === 1
                            ? t("account.myPrograms.dayPerWeek")
                            : t("account.myPrograms.daysPerWeek", { count: program.frequencyPerWeek })}
                        </span>
                      ) : null}
                    </span>

                    <span className={styles.cardAction}>
                      {premiumLevel2
                        ? t("account.myPrograms.openService")
                        : premiumLevel1
                          ? t("account.myPrograms.openPackage")
                          : t("account.myPrograms.openProgram")}
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