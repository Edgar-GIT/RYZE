import { type CSSProperties, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

import { type Copy, useI18n } from "@/i18n/i18n";
import type { LifterLevel, LifterLevelKey } from "@/services/achievements_api";
import { joinClassNames } from "@utils/class_names";

import { BADGE_ART, LEVEL_NAMES } from "./lifter";
import styles from "./rank_up_celebration.module.css";

// Each level has its own light: the glow, the rays, the sparks and the name.
export const RANK_COLORS: Record<LifterLevelKey, [string, string]> = {
  rookie: ["#c9d5ec", "#6b7a99"],
  beginner: ["#34d399", "#0ea472"],
  average: ["#18d8ff", "#0481dc"],
  dedicated: ["#5aa8ff", "#2a5bff"],
  advanced: ["#b69bff", "#7c3aed"],
  elite: ["#ff7ac6", "#db2777"],
  master: ["#ffc24a", "#f97316"],
  legend: ["#ffe08a", "#ff4d6d"]
};

const TOTAL_LEVELS = 8;

const COPY: Copy<{ kicker: string; levelOf: (rank: number) => string; next: string; skip: string; label: (name: string) => string }> = {
  en: {
    kicker: "New level unlocked",
    levelOf: (rank) => `Level ${rank} of ${TOTAL_LEVELS}`,
    next: "Continue",
    skip: "Skip",
    label: (name) => `New level: ${name}`
  },
  pt: {
    kicker: "Novo nível desbloqueado",
    levelOf: (rank) => `Nível ${rank} de ${TOTAL_LEVELS}`,
    next: "Continuar",
    skip: "Saltar",
    label: (name) => `Novo nível: ${name}`
  }
};

// Deterministic spreads for the sparks and the padlock shards.
const SPARKS = Array.from({ length: 28 }, (_, index) => ({
  angle: (index * 360) / 28 + (index % 3) * 7,
  distance: 150 + ((index * 37) % 5) * 38,
  size: 4 + (index % 4) * 2,
  delay: (index % 5) * 40
}));
const SHARDS = Array.from({ length: 12 }, (_, index) => ({
  angle: (index * 360) / 12 + 12,
  distance: 170 + (index % 3) * 60,
  spin: (index % 2 ? 1 : -1) * (220 + index * 25)
}));

const LINKS = 9;
const STEP = 52;

// A chain of steel links. It looks whole until it snaps after `breakAt`
// links: the two parts fly apart and a spark marks the break.
const Chain = ({ className, breakAt }: { className: string; breakAt: number }) => {
  const link = (index: number) => {
    const x = index * STEP;
    return (
      <g key={index}>
        <rect x={x + 4} y="16" width="46" height="28" rx="14" fill="none" stroke="url(#rank-steel)" strokeWidth="7" />
        {index < LINKS - 1 ? <rect x={x + 34} y="26" width="38" height="8" rx="4" fill="url(#rank-steel)" /> : null}
      </g>
    );
  };
  const indexes = Array.from({ length: LINKS }, (_, index) => index);
  return (
    <svg className={className} viewBox={`0 0 ${LINKS * STEP + 6} 60`} aria-hidden="true">
      <g className={styles.chainLeft}>{indexes.slice(0, breakAt).map(link)}</g>
      <g className={styles.chainRight}>{indexes.slice(breakAt).map(link)}</g>
      <circle className={styles.snap} cx={breakAt * STEP + 2} cy="30" r="22" />
    </svg>
  );
};

/**
 * RankUpCelebration plays the level-up moment (owner request, 2026-11-21):
 * the screen darkens, the new badge appears blurred and locked under two
 * crossed chains and a padlock, everything shakes harder and harder, the
 * chains snap one after the other, the padlock explodes and the new level is
 * revealed, glowing. Reduced motion shows the reveal directly. "Skip" jumps
 * to the reveal; Escape or "Continue" closes it.
 */
export const RankUpCelebration = ({ level, onClose }: { level: Pick<LifterLevel, "rank" | "key">; onClose: () => void }) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const name = LEVEL_NAMES[locale][level.key];
  const [skipped, setSkipped] = useState(false);
  const dialogRef = useRef<HTMLDivElement>(null);
  const [primary, secondary] = RANK_COLORS[level.key];

  useEffect(() => {
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    dialogRef.current?.focus();
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = previous;
      window.removeEventListener("keydown", onKey);
    };
  }, [onClose]);

  return createPortal(
    <div
      ref={dialogRef}
      className={joinClassNames(styles.overlay, skipped && styles.skipped)}
      role="dialog"
      aria-modal="true"
      aria-label={text.label(name)}
      tabIndex={-1}
      style={{ "--rank": primary, "--rank-2": secondary } as CSSProperties}
    >
      <svg width="0" height="0" aria-hidden="true" className={styles.defs}>
        <defs>
          <linearGradient id="rank-steel" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#f2f6fc" />
            <stop offset="0.45" stopColor="#8d9bb5" />
            <stop offset="1" stopColor="#2b3550" />
          </linearGradient>
          <linearGradient id="rank-gold" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" stopColor="#fff1b8" />
            <stop offset="0.5" stopColor="#e7b33c" />
            <stop offset="1" stopColor="#8a5a12" />
          </linearGradient>
        </defs>
      </svg>

      {skipped ? null : (
        <button type="button" className={styles.skip} onClick={() => setSkipped(true)}>
          {text.skip}
        </button>
      )}

      <div className={styles.stage} aria-hidden="true">
        <div className={styles.rays} />
        <div className={styles.glow} />
        <div className={styles.shockwave} />
        <div className={styles.shaker}>
          <div className={styles.float}>
            <img className={styles.badge} src={BADGE_ART[level.key]} alt="" draggable={false} />
          </div>
          <Chain className={joinClassNames(styles.chain, styles.chainA)} breakAt={3} />
          <Chain className={joinClassNames(styles.chain, styles.chainB)} breakAt={6} />
          <svg className={styles.lock} viewBox="0 0 120 150">
            <path d="M32 72 V46 a28 28 0 0 1 56 0 V72" fill="none" stroke="url(#rank-steel)" strokeWidth="13" strokeLinecap="round" />
            <rect x="12" y="66" width="96" height="78" rx="16" fill="url(#rank-gold)" stroke="#5c3b08" strokeWidth="3" />
            <circle cx="60" cy="98" r="10" fill="#2a1a03" />
            <rect x="55" y="102" width="10" height="24" rx="4" fill="#2a1a03" />
          </svg>
        </div>
        <div className={styles.shards}>
          {SHARDS.map((shard, index) => (
            <span
              key={index}
              style={
                {
                  "--dx": `${Math.cos((shard.angle * Math.PI) / 180) * shard.distance}px`,
                  "--dy": `${Math.sin((shard.angle * Math.PI) / 180) * shard.distance}px`,
                  "--spin": `${shard.spin}deg`
                } as CSSProperties
              }
            />
          ))}
        </div>
        <div className={styles.sparks}>
          {SPARKS.map((spark, index) => (
            <span
              key={index}
              style={
                {
                  "--dx": `${Math.cos((spark.angle * Math.PI) / 180) * spark.distance}px`,
                  "--dy": `${Math.sin((spark.angle * Math.PI) / 180) * spark.distance}px`,
                  "--size": `${spark.size}px`,
                  "--delay": `${spark.delay}ms`
                } as CSSProperties
              }
            />
          ))}
        </div>
        <div className={styles.flash} />
      </div>

      <div className={styles.reveal}>
        <p className={styles.kicker}>{text.kicker}</p>
        <h2 className={styles.name}>{name}</h2>
        <p className={styles.rank}>{text.levelOf(level.rank)}</p>
        <button type="button" className={styles.next} onClick={onClose}>
          {text.next}
        </button>
      </div>
    </div>,
    document.body
  );
};
