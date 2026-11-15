import { ArrowRight, PartyPopper } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchSessionAward, type SessionAward } from "@/services/achievements_api";

import { LEVEL_NAMES, LevelBadge, MedalItem, XpBar } from "./lifter";
import styles from "./session_rewards.module.css";

const COPY: Copy<{
  title: string;
  workout: string;
  sets: string;
  records: string;
  notCounted: (perDay: number) => string;
  levelUp: string;
  nowYou: (name: string) => string;
  medals: (count: number) => string;
  seeAll: string;
}> = {
  en: {
    title: "EXP earned",
    workout: "Workout",
    sets: "Sets",
    records: "Records",
    notCounted: (perDay) => `This workout earned no EXP: only ${perDay} workouts a day count.`,
    levelUp: "Level up!",
    nowYou: (name) => `You are now ${name}.`,
    medals: (count) => `${count === 1 ? "New medal" : `${count} new medals`}`,
    seeAll: "See achievements"
  },
  pt: {
    title: "EXP ganho",
    workout: "Treino",
    sets: "Séries",
    records: "Recordes",
    notCounted: (perDay) => `Este treino não somou EXP: só contam ${perDay} treinos por dia.`,
    levelUp: "Subiste de nível!",
    nowYou: (name) => `Agora és ${name}.`,
    medals: (count) => `${count === 1 ? "Nova medalha" : `${count} novas medalhas`}`,
    seeAll: "Ver conquistas"
  }
};

// SessionRewards shows what a saved workout earned: EXP (and why), a level up
// and the record medals (owner request, 2026-11-14). The numbers come from
// the server; nothing is shown when they cannot be loaded.
export const SessionRewards = ({ sessionId }: { sessionId: string }) => {
  const { locale, formatNumber } = useI18n();
  const text = COPY[locale];
  const names = LEVEL_NAMES[locale];
  const [award, setAward] = useState<SessionAward | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetchSessionAward(sessionId)
      .then((value) => {
        if (!cancelled) setAward(value);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [sessionId]);

  if (!award) return null;
  const { award: earned, level_before: before, level_after: after } = award;
  const leveledUp = after.rank > before.rank;
  const parts = [
    { label: text.workout, value: earned.breakdown.workout },
    { label: text.sets, value: earned.breakdown.sets },
    { label: text.records, value: earned.breakdown.records }
  ].filter((part) => part.value > 0);

  return (
    <section className={styles.rewards} aria-labelledby="session-rewards">
      <div className={styles.top}>
        <div>
          <h2 id="session-rewards" className={styles.title}>
            {text.title}
          </h2>
          <p className={styles.xp}>+{formatNumber(earned.xp_earned)} EXP</p>
        </div>
        {parts.length > 0 ? (
          <ul className={styles.parts}>
            {parts.map((part) => (
              <li key={part.label}>
                {part.label} <b>+{formatNumber(part.value)}</b>
              </li>
            ))}
          </ul>
        ) : null}
      </div>

      {!earned.counted && earned.completed_sets > 0 ? (
        <p className={styles.note}>{text.notCounted(award.rules.sessions_per_day)}</p>
      ) : null}

      {leveledUp ? (
        <div className={styles.levelUp} role="status">
          <PartyPopper className={styles.party} aria-hidden="true" />
          <LevelBadge level={after} size="md" />
          <div>
            <strong>{text.levelUp}</strong>
            <span>{text.nowYou(names[after.key])}</span>
          </div>
        </div>
      ) : null}

      <XpBar data={{ xp: earned.xp_after, level: after, next_level: award.next_level }} />

      {earned.medals.length > 0 ? (
        <div className={styles.medals}>
          <h3>{text.medals(earned.medals.length)}</h3>
          <ul>
            {earned.medals.map((medal) => (
              <MedalItem key={medal.id} medal={medal} />
            ))}
          </ul>
        </div>
      ) : null}

      <Link className={styles.link} to="/achievements">
        {text.seeAll} <ArrowRight aria-hidden="true" />
      </Link>
    </section>
  );
};
