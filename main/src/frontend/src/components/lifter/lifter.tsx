import { ArrowRight, Medal, TrendingUp } from "lucide-react";
import { Link } from "react-router-dom";

import advancedBadge from "@resources/img/badges/advanced.webp";
import averageBadge from "@resources/img/badges/average.webp";
import beginnerBadge from "@resources/img/badges/beginner.webp";
import dedicatedBadge from "@resources/img/badges/dedicated.webp";
import eliteBadge from "@resources/img/badges/elite.webp";
import legendBadge from "@resources/img/badges/legend.webp";
import masterBadge from "@resources/img/badges/master.webp";
import rookieBadge from "@resources/img/badges/rookie.webp";

import { type Copy, useI18n } from "@/i18n/i18n";
import {
  levelProgress,
  type Achievements,
  type LifterLevel,
  type LifterLevelKey,
  type RecordMedal
} from "@/services/achievements_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./lifter.module.css";

// The lifter levels' names (the server sends the key and the thresholds).
export const LEVEL_NAMES: Copy<Record<LifterLevelKey, string>> = {
  en: {
    rookie: "Rookie Lifter",
    beginner: "Beginner Lifter",
    average: "Average Lifter",
    dedicated: "Dedicated Lifter",
    advanced: "Advanced Lifter",
    elite: "Elite Lifter",
    master: "Master Lifter",
    legend: "Legendary Lifter"
  },
  pt: {
    rookie: "Lifter Novato",
    beginner: "Lifter Iniciante",
    average: "Lifter Médio",
    dedicated: "Lifter Dedicado",
    advanced: "Lifter Avançado",
    elite: "Lifter de Elite",
    master: "Lifter Mestre",
    legend: "Lifter Lendário"
  }
};

const COPY: Copy<{
  kicker: string;
  levelOf: (rank: number, total: number) => string;
  toNext: (xp: string, name: string) => string;
  top: string;
  medals: (count: number) => string;
  noMedals: string;
  seeAll: string;
  recent: string;
  better: (kg: string) => string;
  progressLabel: string;
}> = {
  en: {
    kicker: "Your lifter level",
    levelOf: (rank, total) => `Level ${rank} of ${total}`,
    toNext: (xp, name) => `${xp} EXP to ${name}`,
    top: "Top level reached",
    medals: (count) => `${count} ${count === 1 ? "medal" : "medals"}`,
    noMedals: "Beat a weight record to earn your first medal.",
    seeAll: "See achievements",
    recent: "Latest medals",
    better: (kg) => `+${kg} kg`,
    progressLabel: "Progress to the next level"
  },
  pt: {
    kicker: "O teu nível de lifter",
    levelOf: (rank, total) => `Nível ${rank} de ${total}`,
    toNext: (xp, name) => `Faltam ${xp} EXP para ${name}`,
    top: "Nível máximo alcançado",
    medals: (count) => `${count} ${count === 1 ? "medalha" : "medalhas"}`,
    noMedals: "Bate um recorde de peso para ganhares a tua primeira medalha.",
    seeAll: "Ver conquistas",
    recent: "Últimas medalhas",
    better: (kg) => `+${kg} kg`,
    progressLabel: "Progresso para o próximo nível"
  }
};

// The owner's level artwork (main/src/resources/img/badges/badges.png), cut
// into one picture per level, in rank order.
const BADGE_ART: Record<LifterLevelKey, string> = {
  rookie: rookieBadge,
  beginner: beginnerBadge,
  average: averageBadge,
  dedicated: dedicatedBadge,
  advanced: advancedBadge,
  elite: eliteBadge,
  master: masterBadge,
  legend: legendBadge
};

// LevelBadge shows the level's badge with its number. The level name is always
// written next to it, so the picture itself is decorative.
export const LevelBadge = ({ level, size = "md" }: { level: Pick<LifterLevel, "rank" | "key">; size?: "sm" | "md" | "lg" }) => (
  <span className={joinClassNames(styles.badge, styles[`badge_${size}`])}>
    <img src={BADGE_ART[level.key]} alt="" aria-hidden="true" decoding="async" draggable={false} />
    {size === "sm" ? null : <b>{level.rank}</b>}
  </span>
);

// MedalBadge is the gold record medal.
export const MedalBadge = ({ small = false }: { small?: boolean }) => (
  <span className={joinClassNames(styles.medal, small && styles.medalSmall)} aria-hidden="true">
    <Medal />
  </span>
);

export const MedalItem = ({ medal, compact = false }: { medal: RecordMedal; compact?: boolean }) => {
  const { locale, formatDate, formatNumber } = useI18n();
  const text = COPY[locale];
  const kg = (value: number) => formatNumber(value, { maximumFractionDigits: 2 });
  return (
    <li className={joinClassNames(styles.medalItem, compact && styles.medalItemCompact)}>
      <MedalBadge small={compact} />
      <span className={styles.medalText}>
        <strong>{medal.exercise_name}</strong>
        <span>
          {kg(medal.weight_kg)} kg × {medal.reps} · {formatDate(medal.earned_at, { day: "numeric", month: "short" })}
        </span>
      </span>
      <em className={styles.medalGain}>{text.better(kg(medal.weight_kg - medal.previous_best_kg))}</em>
    </li>
  );
};

export const XpBar = ({ data }: { data: Pick<Achievements, "xp" | "level" | "next_level"> }) => {
  const { locale, formatNumber } = useI18n();
  const text = COPY[locale];
  const names = LEVEL_NAMES[locale];
  const progress = levelProgress(data.xp, data.level, data.next_level);
  return (
    <div className={styles.xp}>
      <div className={styles.xpTop}>
        <strong>{formatNumber(data.xp)} EXP</strong>
        <span>
          {data.next_level
            ? text.toNext(formatNumber(data.next_level.min_xp - data.xp), names[data.next_level.key])
            : text.top}
        </span>
      </div>
      <div
        className={styles.bar}
        role="progressbar"
        aria-label={text.progressLabel}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(progress * 100)}
      >
        <span style={{ width: `${Math.max(progress * 100, 2)}%` }} />
      </div>
    </div>
  );
};

// LifterCard opens the member home: level, EXP and the latest medals.
export const LifterCard = ({ data, totalLevels }: { data: Achievements | null; totalLevels: number }) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const names = LEVEL_NAMES[locale];
  if (!data) {
    return <section className={joinClassNames(styles.card, styles.skeleton)} aria-hidden="true" />;
  }
  const level: LifterLevel = data.level;
  return (
    <section className={styles.card} aria-labelledby="lifter-level">
      <div className={styles.identity}>
        <LevelBadge level={level} size="lg" />
        <div>
          <p className={styles.kicker}>
            <TrendingUp aria-hidden="true" /> {text.kicker}
          </p>
          <h2 id="lifter-level" className={styles.levelName}>
            {names[level.key]}
          </h2>
          <p className={styles.levelOf}>{text.levelOf(level.rank, totalLevels)}</p>
        </div>
      </div>
      <XpBar data={data} />
      <div className={styles.medals}>
        <div className={styles.medalsHead}>
          <span className={styles.medalCount}>
            <MedalBadge small /> {text.medals(data.medal_count)}
          </span>
          <Link className={styles.link} to="/achievements">
            {text.seeAll} <ArrowRight aria-hidden="true" />
          </Link>
        </div>
        {data.medals.length > 0 ? (
          <ul className={styles.medalRow} aria-label={text.recent}>
            {data.medals.slice(0, 3).map((medal) => (
              <MedalItem key={medal.id} medal={medal} compact />
            ))}
          </ul>
        ) : (
          <p className={styles.empty}>{text.noMedals}</p>
        )}
      </div>
    </section>
  );
};
