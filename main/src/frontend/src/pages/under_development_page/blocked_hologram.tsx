import { useId } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./blocked_hologram.module.css";

// BlockedHologram: a "no entry" sign on a post, projected as a cyan hologram
// from a disc on the floor (scan lines, a climbing light band, a faint
// flicker). SVG and CSS only; still with reduced motion. Decorative.
export const BlockedHologram = ({ className }: { className?: string }) => {
  const id = useId().replace(/:/g, "");
  const fill = `${id}-fill`;
  const beam = `${id}-beam`;
  const lines = `${id}-lines`;
  const glow = `${id}-glow`;
  const clip = `${id}-clip`;
  return (
    <div className={joinClassNames(styles.hologram, className)} aria-hidden="true">
      <svg viewBox="0 0 240 300" focusable="false">
        <defs>
          <linearGradient id={fill} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#9af3ff" stopOpacity="0.38" />
            <stop offset="1" stopColor="#0aa8f2" stopOpacity="0.12" />
          </linearGradient>
          <linearGradient id={beam} x1="0" y1="1" x2="0" y2="0">
            <stop offset="0" stopColor="#bff6ff" stopOpacity="0.5" />
            <stop offset="1" stopColor="#00cbff" stopOpacity="0" />
          </linearGradient>
          <pattern id={lines} width="4" height="4" patternUnits="userSpaceOnUse">
            <rect width="4" height="1" fill="#e8fdff" fillOpacity="0.22" />
          </pattern>
          <filter id={glow} x="-30%" y="-30%" width="160%" height="160%">
            <feGaussianBlur stdDeviation="2.4" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
          <clipPath id={clip}>
            <path d="M120 22a62 62 0 1 1 0 124a62 62 0 1 1 0-124ZM113 146h14v118h-14Z" />
          </clipPath>
        </defs>

        {/* The projector: a disc on the floor and the light rising from it. */}
        <polygon className={styles.beam} points="70,270 170,270 196,20 44,20" fill={`url(#${beam})`} />
        <ellipse cx="120" cy="272" rx="78" ry="14" fill="none" stroke="#9af3ff" strokeOpacity="0.7" strokeWidth="1.4" />
        <ellipse className={styles.pulse} cx="120" cy="272" rx="78" ry="14" fill="none" stroke="#e8fdff" strokeWidth="1.4" />
        <ellipse cx="120" cy="272" rx="50" ry="8" fill="#7cebff" fillOpacity="0.25" />

        {/* The sign on its post. */}
        <g className={styles.sign} filter={`url(#${glow})`} stroke="#8ff0ff" strokeWidth="2" strokeLinejoin="round">
          <rect x="113" y="146" width="14" height="118" rx="3" fill={`url(#${fill})`} />
          <rect x="106" y="258" width="28" height="8" rx="3" fill={`url(#${fill})`} />
          <circle cx="120" cy="84" r="62" fill={`url(#${fill})`} />
          <circle cx="120" cy="84" r="50" fill="none" strokeOpacity="0.55" />
          <rect x="84" y="76" width="72" height="16" rx="4" fill="#e8fdff" fillOpacity="0.85" stroke="#e8fdff" />
          <g clipPath={`url(#${clip})`}>
            <rect x="40" y="20" width="160" height="250" fill={`url(#${lines})`} />
            <rect className={styles.band} x="40" y="0" width="160" height="12" fill="#e8fdff" fillOpacity="0.55" />
          </g>
        </g>

        {/* Particles drifting up through the projection. */}
        {[78, 102, 138, 162, 118].map((x, index) => (
          <circle key={x} className={styles.particle} cx={x} cy="262" r="1.6" fill="#e8fdff" style={{ animationDelay: `${index * -0.7}s` }} />
        ))}
      </svg>
    </div>
  );
};
