import { CheckCircle2, Dumbbell, Lock, Medal, Sparkles, Trophy } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { LEVEL_NAMES, LevelBadge, MedalItem, XpBar } from "@/components/lifter/lifter";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchAchievements, type Achievements } from "@/services/achievements_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./achievements_page.module.css";

const COPY: Copy<{
  kicker: string;
  title: string;
  lead: string;
  levelOf: (rank: number, total: number) => string;
  workouts: (count: number) => string;
  medalsStat: (count: number) => string;
  medalsTitle: string;
  medalsLead: string;
  noMedals: string;
  startTraining: string;
  levelsTitle: string;
  from: (xp: string) => string;
  reached: string;
  current: string;
  locked: string;
  howTitle: string;
  rules: (rules: Achievements["rules"]) => string[];
  loadError: string;
  more: (count: number) => string;
}> = {
  en: {
    kicker: "Achievements",
    title: "Your achievements",
    lead: "EXP, your lifter level and every weight record you have beaten.",
    levelOf: (rank, total) => `Level ${rank} of ${total}`,
    workouts: (count) => `${count} ${count === 1 ? "workout" : "workouts"} counted`,
    medalsStat: (count) => `${count} ${count === 1 ? "medal" : "medals"}`,
    medalsTitle: "Record medals",
    medalsLead:
      "A medal every time you lift more than ever on an exercise. The first time you do an exercise sets your starting record.",
    noMedals: "No medals yet. Finish a workout lifting more than your best on any exercise to earn the first one.",
    startTraining: "Go to training",
    levelsTitle: "Lifter levels",
    from: (xp) => `from ${xp} EXP`,
    reached: "Reached",
    current: "You are here",
    locked: "Locked",
    howTitle: "How to earn EXP",
    rules: (rules) => [
      `${rules.workout_xp} EXP for each workout you finish`,
      `${rules.set_xp} EXP per completed set (up to ${rules.max_sets_counted} per workout)`,
      `${rules.record_xp} EXP for each weight record`,
      `Up to ${rules.sessions_per_day} workouts a day earn EXP`
    ],
    loadError: "We could not load your achievements. Please try again later.",
    more: (count) => `and ${count} more`
  },
  pt: {
    kicker: "Conquistas",
    title: "As tuas conquistas",
    lead: "EXP, o teu nível de lifter e todos os recordes de peso que já bateste.",
    levelOf: (rank, total) => `Nível ${rank} de ${total}`,
    workouts: (count) => `${count} ${count === 1 ? "treino contado" : "treinos contados"}`,
    medalsStat: (count) => `${count} ${count === 1 ? "medalha" : "medalhas"}`,
    medalsTitle: "Medalhas de recorde",
    medalsLead:
      "Uma medalha sempre que levantas mais do que nunca num exercício. A primeira vez que fazes um exercício define o teu recorde inicial.",
    noMedals: "Ainda não tens medalhas. Termina um treino a levantar mais do que o teu melhor num exercício para ganhares a primeira.",
    startTraining: "Ir treinar",
    levelsTitle: "Níveis de lifter",
    from: (xp) => `a partir de ${xp} EXP`,
    reached: "Alcançado",
    current: "Estás aqui",
    locked: "Por alcançar",
    howTitle: "Como ganhar EXP",
    rules: (rules) => [
      `${rules.workout_xp} EXP por cada treino que terminas`,
      `${rules.set_xp} EXP por série concluída (até ${rules.max_sets_counted} por treino)`,
      `${rules.record_xp} EXP por cada recorde de peso`,
      `Contam até ${rules.sessions_per_day} treinos por dia`
    ],
    loadError: "Não foi possível carregar as tuas conquistas. Tenta novamente mais tarde.",
    more: (count) => `e mais ${count}`
  }
};

const MEDALS_SHOWN = 60;

// AchievementsPage lists the member's lifter level, EXP and record medals
// (owner request, 2026-11-14), all computed by the server from the workouts.
export const AchievementsPage = () => {
  const { locale, formatNumber } = useI18n();
  const text = COPY[locale];
  const names = LEVEL_NAMES[locale];
  const [data, setData] = useState<Achievements | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetchAchievements()
      .then((value) => {
        if (!cancelled) setData(value);
      })
      .catch(() => {
        if (!cancelled) setFailed(true);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <PageWrapper className={styles.page}>
      <header className={styles.header}>
        <p className={styles.kicker}>
          <Trophy aria-hidden="true" /> {text.kicker}
        </p>
        <h1 className={styles.title}>
          {text.title}
          <span className={styles.stop}>.</span>
        </h1>
        <p className={styles.lead}>{text.lead}</p>
      </header>

      {failed ? (
        <p className={styles.error} role="alert">
          {text.loadError}
        </p>
      ) : !data ? (
        <div className={joinClassNames(styles.hero, styles.skeleton)} aria-hidden="true" />
      ) : (
        <>
          <section className={styles.hero} aria-labelledby="achievements-level">
            <div className={styles.heroIdentity}>
              <LevelBadge level={data.level} size="lg" />
              <div>
                <h2 id="achievements-level" className={styles.levelName}>
                  {names[data.level.key]}
                </h2>
                <p className={styles.levelOf}>{text.levelOf(data.level.rank, data.levels.length)}</p>
              </div>
            </div>
            <XpBar data={data} />
            <ul className={styles.stats}>
              <li>
                <Dumbbell aria-hidden="true" /> {text.workouts(data.workouts_counted)}
              </li>
              <li>
                <Medal aria-hidden="true" /> {text.medalsStat(data.medal_count)}
              </li>
            </ul>
          </section>

          <div className={styles.columns}>
            <section className={styles.card} aria-labelledby="achievements-medals">
              <h2 id="achievements-medals" className={styles.cardTitle}>
                <Medal aria-hidden="true" /> {text.medalsTitle}
              </h2>
              <p className={styles.cardLead}>{text.medalsLead}</p>
              {data.medals.length > 0 ? (
                <>
                  <ul className={styles.medalList}>
                    {data.medals.slice(0, MEDALS_SHOWN).map((medal) => (
                      <MedalItem key={medal.id} medal={medal} />
                    ))}
                  </ul>
                  {data.medal_count > MEDALS_SHOWN ? (
                    <p className={styles.cardLead}>{text.more(data.medal_count - MEDALS_SHOWN)}</p>
                  ) : null}
                </>
              ) : (
                <div className={styles.empty}>
                  <p>{text.noMedals}</p>
                  <Link className={styles.action} to="/services/my-programs">
                    {text.startTraining}
                  </Link>
                </div>
              )}
            </section>

            <div className={styles.side}>
              <section className={styles.card} aria-labelledby="achievements-levels">
                <h2 id="achievements-levels" className={styles.cardTitle}>
                  <Trophy aria-hidden="true" /> {text.levelsTitle}
                </h2>
                <ol className={styles.ladder}>
                  {data.levels.map((level) => {
                    const state =
                      level.rank === data.level.rank ? "current" : level.rank < data.level.rank ? "reached" : "locked";
                    return (
                      <li key={level.key} className={styles.rung} data-state={state} aria-current={state === "current" ? "step" : undefined}>
                        <LevelBadge level={level} size="sm" />
                        <span className={styles.rungText}>
                          <strong>{names[level.key]}</strong>
                          <span>{text.from(formatNumber(level.min_xp))}</span>
                        </span>
                        <span className={styles.rungState}>
                          {state === "reached" ? (
                            <CheckCircle2 aria-hidden="true" />
                          ) : state === "locked" ? (
                            <Lock aria-hidden="true" />
                          ) : (
                            <Sparkles aria-hidden="true" />
                          )}
                          <span className={styles.srOnly}>
                            {state === "reached" ? text.reached : state === "locked" ? text.locked : text.current}
                          </span>
                        </span>
                      </li>
                    );
                  })}
                </ol>
              </section>

              <section className={styles.card} aria-labelledby="achievements-how">
                <h2 id="achievements-how" className={styles.cardTitle}>
                  <Sparkles aria-hidden="true" /> {text.howTitle}
                </h2>
                <ul className={styles.rules}>
                  {text.rules(data.rules).map((rule) => (
                    <li key={rule}>{rule}</li>
                  ))}
                </ul>
              </section>
            </div>
          </div>
        </>
      )}
    </PageWrapper>
  );
};

export default AchievementsPage;
