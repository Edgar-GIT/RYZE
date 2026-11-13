import { ArrowLeft, BadgeCheck, Eye, LayoutGrid, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ProgramCard } from "@/components/program_card/program_card";
import { useTestMode } from "@/components/test_mode/test_mode_context";
import { type Copy, useI18n } from "@/i18n/i18n";
import { coachInitials, fetchCoachProfile, type CoachProfile } from "@/services/coaches_api";
import { ApiError } from "@utils/http_client";

import styles from "./coach_profile_page.module.css";

interface CoachText {
  back: string;
  loading: string;
  notFound: string;
  loadError: string;
  retry: string;
  verified: string;
  since: (date: string) => string;
  programs: (count: number) => string;
  views: (count: number, formatted: string) => string;
  programsTitle: string;
  programsLead: string;
}

const COPY: Copy<CoachText> = {
  en: {
    back: "Training plans",
    loading: "Loading coach…",
    notFound: "This coach has no public profile.",
    loadError: "Unable to load this coach. Please try again.",
    retry: "Retry",
    verified: "Verified RYZE coach",
    since: (date) => `Coach since ${date}`,
    programs: (count) => (count === 1 ? "1 program" : `${count} programs`),
    views: (count, formatted) => (count === 1 ? "1 view" : `${formatted} views`),
    programsTitle: "Programs",
    programsLead: "Every plan this coach has published on RYZE."
  },
  pt: {
    back: "Planos de treino",
    loading: "A carregar o treinador…",
    notFound: "Este treinador não tem um perfil público.",
    loadError: "Não foi possível carregar este treinador. Tenta novamente.",
    retry: "Tentar novamente",
    verified: "Treinador RYZE verificado",
    since: (date) => `Treinador desde ${date}`,
    programs: (count) => (count === 1 ? "1 programa" : `${count} programas`),
    views: (count, formatted) => (count === 1 ? "1 visualização" : `${formatted} visualizações`),
    programsTitle: "Programas",
    programsLead: "Todos os planos que este treinador publicou na RYZE."
  }
};

// CoachProfilePage is the public profile of a RYZE coach: who they are and
// every program they published. Only the coach's registered name is shown;
// contact details are never public.
export const CoachProfilePage = () => {
  const { trainerId } = useParams<{ trainerId: string }>();
  const { locale, formatDate, formatNumber } = useI18n();
  const text = COPY[locale];
  const { isActive: testModeActive } = useTestMode();
  const [profile, setProfile] = useState<CoachProfile | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "missing" | "error">("loading");

  const load = useCallback(async () => {
    setState("loading");
    try {
      setProfile(await fetchCoachProfile(trainerId));
      setState("ready");
    } catch (error) {
      setProfile(null);
      setState(error instanceof ApiError && (error.status === 404 || error.status === 400) ? "missing" : "error");
    }
  }, [trainerId]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <PageWrapper className={styles.page}>
      <Container size="wide" className={styles.container}>
        <div className={styles.back}>
          <Button to="/services/generic-program" variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
            {text.back}
          </Button>
        </div>

        {state === "loading" ? (
          <div className={styles.state}>
            <p>{text.loading}</p>
          </div>
        ) : state !== "ready" || !profile ? (
          <div className={styles.state}>
            <p>{state === "missing" ? text.notFound : text.loadError}</p>
            {state === "error" ? (
              <Button variant="secondary" size="small" onClick={() => void load()} icon={<RefreshCw size={15} />}>
                {text.retry}
              </Button>
            ) : null}
          </div>
        ) : (
          <>
            <header className={styles.hero}>
              <span className={styles.avatar} aria-hidden="true">
                {coachInitials(profile.coach.name)}
              </span>
              <div className={styles.identity}>
                <span className={styles.verified}>
                  <BadgeCheck aria-hidden="true" /> {text.verified}
                </span>
                <h1 className={styles.name}>{profile.coach.name}</h1>
                <p className={styles.since}>
                  {text.since(formatDate(profile.coach.coach_since, { month: "long", year: "numeric" }))}
                </p>
              </div>
              <ul className={styles.stats}>
                <li>
                  <LayoutGrid aria-hidden="true" />
                  {text.programs(profile.programs.length)}
                </li>
                <li>
                  <Eye aria-hidden="true" />
                  {text.views(profile.coach.total_views, formatNumber(profile.coach.total_views))}
                </li>
              </ul>
            </header>

            <section className={styles.programs} aria-labelledby="coach-programs-title">
              <div className={styles.programsHead}>
                <h2 id="coach-programs-title" className={styles.programsTitle}>
                  {text.programsTitle}
                </h2>
                <p className={styles.programsLead}>{text.programsLead}</p>
              </div>
              <ul className={styles.grid}>
                {profile.programs.map((program) => (
                  <li key={program.id}>
                    <ProgramCard program={program} forceFree={testModeActive} />
                  </li>
                ))}
              </ul>
            </section>
          </>
        )}
      </Container>
    </PageWrapper>
  );
};
