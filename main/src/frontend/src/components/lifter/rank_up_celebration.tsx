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

const BAND = 520;
const NOTCH = 26;

// A restraint of light: a segmented energy band with bracket caps. It looks
// whole until it snaps at `breakAt` (a share of its length): the two parts
// retract and fade, and a spark marks the break.
const Band = ({ className, breakAt }: { className: string; breakAt: number }) => {
  const cut = Math.round(BAND * breakAt);
  const part = (from: number, to: number) => (
    <>
      <rect x={from} y="13" width={to - from} height="10" rx="5" fill="url(#rank-band)" />
      {Array.from({ length: Math.floor((to - from) / NOTCH) }, (_, index) => (
        <rect key={index} x={from + 12 + index * NOTCH} y="13" width="3" height="10" fill="#02041f" opacity="0.55" />
      ))}
    </>
  );
  return (
    <svg className={className} viewBox={`0 0 ${BAND} 36`} aria-hidden="true">
      <g className={styles.chainLeft}>
        <rect x="1" y="6" width="14" height="24" rx="4" fill="none" stroke="#ffffff" strokeWidth="2" opacity="0.85" />
        {part(8, cut - 3)}
      </g>
      <g className={styles.chainRight}>
        <rect x={BAND - 15} y="6" width="14" height="24" rx="4" fill="none" stroke="#ffffff" strokeWidth="2" opacity="0.85" />
        {part(cut + 3, BAND - 8)}
      </g>
      <circle className={styles.snap} cx={cut} cy="18" r="16" />
    </svg>
  );
};

const hexagon = (radius: number) =>
  Array.from({ length: 6 }, (_, index) => {
    const angle = (Math.PI / 3) * index - Math.PI / 2;
    return `${(80 + radius * Math.cos(angle)).toFixed(1)},${(80 + radius * Math.sin(angle)).toFixed(1)}`;
  }).join(" ");

// The seal in the middle: a hexagon with a line lock (the app's icon style),
// a turning ring of light and cracks that spread before it bursts.
const Seal = () => (
  <svg className={styles.lock} viewBox="0 0 160 160" aria-hidden="true">
    <circle className={styles.sealRing} cx="80" cy="80" r="74" fill="none" stroke="var(--rank)" strokeWidth="2" strokeDasharray="2 8" />
    <circle cx="80" cy="80" r="65" fill="none" stroke="#ffffff" strokeOpacity="0.25" strokeWidth="1" />
    <polygon points={hexagon(54)} fill="#02041f" fillOpacity="0.88" stroke="var(--rank)" strokeWidth="3" strokeLinejoin="round" />
    <polygon points={hexagon(45)} fill="none" stroke="#ffffff" strokeOpacity="0.22" strokeWidth="1.2" />
    <g className={styles.glyph} fill="none" stroke="#ffffff" strokeWidth="5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M66 76 V67 a14 14 0 0 1 28 0 V76" />
      <rect x="58" y="76" width="44" height="32" rx="8" />
      <path d="M80 88 V96" />
    </g>
    <g className={styles.cracks} fill="none" stroke="#ffffff" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path pathLength={1} d="M80 80 L92 70 L97 72 L112 52 L118 50" />
      <path pathLength={1} d="M80 80 L68 92 L70 99 L55 112" />
      <path pathLength={1} d="M80 80 L95 92 L104 90 L121 103" />
    </g>
  </svg>
);

/**
 * RankUpCelebration plays the level-up moment (owner request, 2026-11-21):
 * the screen darkens, the new badge appears blurred and locked under two
 * crossed bands of light and a hexagonal seal, everything shakes harder and
 * harder, the bands snap one after the other, the seal cracks and bursts and
 * the new level is revealed, glowing (v2.130.1: light restraints instead of
 * realistic chains and a padlock, owner feedback). Reduced motion shows the reveal directly. "Skip" jumps
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
          <linearGradient id="rank-band" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" style={{ stopColor: "var(--rank-2)" }} />
            <stop offset="0.38" style={{ stopColor: "var(--rank)" }} />
            <stop offset="0.5" stopColor="#ffffff" />
            <stop offset="0.62" style={{ stopColor: "var(--rank)" }} />
            <stop offset="1" style={{ stopColor: "var(--rank-2)" }} />
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
          <Band className={joinClassNames(styles.chain, styles.chainA)} breakAt={0.3} />
          <Band className={joinClassNames(styles.chain, styles.chainB)} breakAt={0.7} />
          <Seal />
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
