import type { CSSProperties, ReactNode } from "react";

import { PhoneMockup } from "@/components/showcase/showcase";
import { joinClassNames } from "@utils/class_names";

import styles from "./hero_showcase.module.css";

interface HeroShowcaseProps {
  className?: string;
  screen: string; // the RYZE training screen
}

// The dumbbell from the side, angled: hex heads in gunmetal with a cyan
// inlay, chrome collars and a knurled handle. `ghost` draws its hologram.
const Dumbbell = ({ ghost }: { ghost?: boolean }) => (
  <svg viewBox="0 0 240 132" focusable="false">
    <defs>
      <linearGradient id={ghost ? "hs-chrome-g" : "hs-chrome"} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stopColor="#f4f8fd" />
        <stop offset="0.5" stopColor="#97a6bc" />
        <stop offset="1" stopColor="#e3eaf4" />
      </linearGradient>
      <linearGradient id={ghost ? "hs-metal-g" : "hs-metal"} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stopColor="#47546f" />
        <stop offset="0.28" stopColor="#222c44" />
        <stop offset="1" stopColor="#0d1324" />
      </linearGradient>
      <pattern id={ghost ? "hs-knurl-g" : "hs-knurl"} width="5" height="5" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
        <path d="M0 0V5" stroke="rgba(30,40,60,0.35)" strokeWidth="1.2" />
      </pattern>
    </defs>
    <g className={ghost ? styles.ghostStroke : undefined}>
      <rect x="66" y="59" width="108" height="14" rx="4" fill={ghost ? "rgba(54,214,255,0.12)" : "url(#hs-chrome)"} />
      {ghost ? null : <rect x="78" y="59" width="84" height="14" fill="url(#hs-knurl)" />}
      <rect x="58" y="52" width="12" height="28" rx="3" fill={ghost ? "rgba(54,214,255,0.12)" : "url(#hs-chrome)"} />
      <rect x="170" y="52" width="12" height="28" rx="3" fill={ghost ? "rgba(54,214,255,0.12)" : "url(#hs-chrome)"} />
      {[16, 182].map((x) => (
        <g key={x}>
          <rect x={x} y="18" width="42" height="96" rx="9" fill={ghost ? "rgba(54,214,255,0.14)" : "url(#hs-metal)"} />
          {ghost ? null : <rect x={x + 3} y="20" width="36" height="9" rx="4" fill="rgba(255,255,255,0.14)" />}
          <path d={`M${x + 8} 26 V106`} stroke="#50e2fe" strokeWidth="2.2" strokeLinecap="round" opacity={ghost ? 0.9 : 0.95} />
        </g>
      ))}
      {/* The near head's end face, a hexagon with the cyan inlay. */}
      <polygon points="16,40 4,52 4,80 16,92 28,80 28,52" fill={ghost ? "rgba(54,214,255,0.16)" : "#2b3651"} />
      <polygon points="16,50 10,56 10,76 16,82 22,76 22,56" fill="none" stroke="#50e2fe" strokeWidth="1.6" />
    </g>
  </svg>
);

// A served plate in a three-quarter view: grilled salmon on quinoa, avocado,
// asparagus, cherry tomatoes and lemon, with a fork and a knife beside it.
const Meal = ({ ghost }: { ghost?: boolean }) => {
  const fill = (color: string) => (ghost ? "rgba(54,214,255,0.14)" : color);
  return (
    <svg viewBox="0 0 250 120" focusable="false">
      <g className={ghost ? styles.ghostStroke : undefined}>
        <ellipse cx="104" cy="78" rx="100" ry="34" fill={fill("#dfe6ef")} />
        <ellipse cx="104" cy="72" rx="100" ry="34" fill={fill("#f6f8fc")} />
        <ellipse cx="104" cy="72" rx="96" ry="31.5" fill="none" stroke={ghost ? undefined : "#0b2a6a"} strokeWidth="1.3" />
        <ellipse cx="104" cy="74" rx="72" ry="23" fill={fill("#eef2f8")} />
        {/* Quinoa and the salmon fillet with grill marks. */}
        <ellipse cx="88" cy="68" rx="40" ry="14" fill={fill("#efe5cf")} />
        <path d="M58 64 Q84 48 116 56 Q124 64 112 74 Q86 82 60 74 Q52 69 58 64 Z" fill={fill("#f47a45")} />
        {ghost ? null : <path d="M72 56 L66 76 M86 54 L80 78 M100 55 L94 78" stroke="#7a3412" strokeWidth="3.4" strokeLinecap="round" opacity="0.75" />}
        {/* Avocado fan. */}
        {[0, 1, 2, 3].map((index) => (
          <path
            key={index}
            d={`M${122 + index * 7} ${60 + index * 2} q16 6 9 22 q-12 -8 -9 -22 Z`}
            fill={fill("#b9da5e")}
            stroke={ghost ? undefined : "#24461a"}
            strokeWidth="1.5"
          />
        ))}
        {/* Asparagus, tomatoes and lemon. */}
        {[0, 1, 2].map((index) => (
          <path key={`a${index}`} d={`M${70 + index * 4} ${84 + index * 4} L${132 + index * 4} ${80 + index * 4}`} stroke={fill("#7fae45")} strokeWidth="5" strokeLinecap="round" />
        ))}
        {[
          [146, 72],
          [156, 80],
          [140, 84]
        ].map(([x, y]) => (
          <g key={`t${x}`}>
            <circle cx={x} cy={y} r="6.5" fill={fill("#d8261f")} />
            {ghost ? null : <circle cx={x} cy={y} r="3.6" fill="#ff8a7a" />}
          </g>
        ))}
        <path d="M44 80 L60 86 A16 16 0 0 1 44 92 Z" fill={fill("#ffd23a")} />
        {/* Fork and knife beside the plate. */}
        <g transform="rotate(-62 222 78)">
          <rect x="182" y="73" width="80" height="5" rx="2.5" fill={fill("#dfe6ef")} />
          <rect x="182" y="83" width="54" height="5" rx="2.5" fill={fill("#dfe6ef")} />
          <path d="M236 81 h22 M236 84 h22 M236 87 h22" stroke={fill("#dfe6ef")} strokeWidth="1.6" strokeLinecap="round" />
        </g>
      </g>
    </svg>
  );
};

// The scanner drone, as in the desktop scene: a lens-shaped pod with a light
// ring, ducted rotors and a sensor underneath.
const Drone = () => (
  <svg viewBox="0 0 200 52" focusable="false">
    <line x1="34" y1="22" x2="166" y2="22" stroke="#1c2740" strokeWidth="4" strokeLinecap="round" />
    {[30, 170].map((x) => (
      <g key={x}>
        <ellipse cx={x} cy="21" rx="24" ry="6" fill="none" stroke="#3a4a6e" strokeWidth="3.4" />
        <ellipse className={styles.rotor} cx={x} cy="21" rx="18" ry="3.6" fill="#bff6ff" fillOpacity="0.3" />
      </g>
    ))}
    <path d="M58 26 Q100 8 142 26 Z" fill="#1a2438" />
    <ellipse cx="100" cy="27" rx="42" ry="8" fill="#151c2e" />
    <ellipse cx="100" cy="26" rx="42" ry="8" fill="none" stroke="#7cebff" strokeWidth="1.4" />
    <ellipse cx="100" cy="33" rx="20" ry="4.4" fill="#040914" stroke="#7cebff" strokeWidth="1.1" />
    <ellipse cx="100" cy="34" rx="5" ry="2.2" fill="#e8fdff" />
  </svg>
);

// One object of the composition: its hologram builds from the bottom up, then
// the solid object takes over behind a bright line.
const Item = ({ className, build, solid, children }: { className: string; build: number; solid: number; children: (ghost: boolean) => ReactNode }) => (
  <div className={joinClassNames(styles.item, className)} style={{ "--build": `${build}s`, "--solid": `${solid}s` } as CSSProperties}>
    <div className={styles.ghost}>{children(true)}</div>
    <div className={styles.solid}>{children(false)}</div>
    <span className={styles.line} />
    <span className={styles.burst} />
  </div>
);

// HeroShowcase is the light version (HTML, SVG and CSS, no WebGL) of the
// desktop hero's opening, for phones, reduced motion and no-WebGL: the drone
// flies in, projects the phone (the real RYZE training screen), a dumbbell and
// a meal as holograms, closes its beam and leaves as they turn solid. With
// reduced motion it is the final composition. Decorative: the copy says what
// RYZE is.
export const HeroShowcase = ({ className, screen }: HeroShowcaseProps) => (
  <div className={joinClassNames(styles.showcase, className)} aria-hidden="true">
    <div className={styles.droneTrack}>
      <div className={styles.drone}>
        <Drone />
      </div>
    </div>
    <span className={styles.lensFlash} />
    <div className={styles.beam} />
    <span className={styles.surface} />
    <Item className={styles.phone} build={2.3} solid={5.2}>
      {(ghost) => (ghost ? <div className={styles.phoneGhost}>{<img src={screen} alt="" />}</div> : <PhoneMockup src={screen} alt="" />)}
    </Item>
    <Item className={styles.dumbbell} build={2.6} solid={5.35}>
      {(ghost) => <Dumbbell ghost={ghost} />}
    </Item>
    <Item className={styles.meal} build={2.9} solid={5.5}>
      {(ghost) => <Meal ghost={ghost} />}
    </Item>
    <span className={styles.steam}>
      <i />
      <i />
      <i />
    </span>
  </div>
);
