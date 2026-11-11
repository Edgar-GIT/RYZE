import { useId } from "react";
import type { CSSProperties } from "react";

import { BRAND_ASSETS } from "@/constants/brand_assets";
import { joinClassNames } from "@utils/class_names";

import { GLOBE_HOME, GLOBE_SVG_ARCS, GLOBE_SVG_CITIES, GLOBE_SVG_LAND } from "./globe_data";
import styles from "./brand_globe.module.css";

interface BrandGlobeProps {
  className?: string;
}

const home = GLOBE_SVG_CITIES.find((city) => city.id === GLOBE_HOME) ?? { x: 0, y: 0 };

// BrandGlobe is the lightweight (SVG only) version of the desktop 3D Earth: the
// real continents seen over the Atlantic, with RYZE routes leaving Lisbon.
// Decorative; used on phones, tablets, reduced motion and as the 3D placeholder.
export const BrandGlobe = ({ className }: BrandGlobeProps) => {
  const id = useId().replace(/:/g, "");
  const ref = (name: string) => `${name}-${id}`;
  const url = (name: string) => `url(#${ref(name)})`;

  return (
    <div className={joinClassNames(styles.globe, className)} aria-hidden="true">
      <svg viewBox="-132 -132 264 264" focusable="false">
        <defs>
          <radialGradient id={ref("atmosphere")} r="132" cx="0" cy="0" gradientUnits="userSpaceOnUse">
            <stop offset="0.72" stopColor="#1aa8ff" stopOpacity="0" />
            <stop offset="0.765" stopColor="#1aa8ff" stopOpacity="0.55" />
            <stop offset="0.86" stopColor="#1aa8ff" stopOpacity="0.12" />
            <stop offset="1" stopColor="#1aa8ff" stopOpacity="0" />
          </radialGradient>
          <radialGradient id={ref("ocean")} r="150" cx="-38" cy="-40" gradientUnits="userSpaceOnUse">
            <stop offset="0" stopColor="#0b3c9a" />
            <stop offset="0.55" stopColor="#05195a" />
            <stop offset="1" stopColor="#020a2e" />
          </radialGradient>
          <radialGradient id={ref("land")} r="160" cx="-38" cy="-40" gradientUnits="userSpaceOnUse">
            <stop offset="0" stopColor="#4aa6f2" />
            <stop offset="0.55" stopColor="#1a56c4" />
            <stop offset="1" stopColor="#0b2d7c" />
          </radialGradient>
          <radialGradient id={ref("shade")} r="150" cx="-30" cy="-34" gradientUnits="userSpaceOnUse">
            <stop offset="0.5" stopColor="#02041f" stopOpacity="0" />
            <stop offset="1" stopColor="#02041f" stopOpacity="0.78" />
          </radialGradient>
          <clipPath id={ref("clip")}>
            <circle r="100" />
          </clipPath>
        </defs>

        <circle r="132" fill={url("atmosphere")} />
        <circle r="100" fill={url("ocean")} />
        <path d={GLOBE_SVG_LAND} fill={url("land")} stroke="rgba(124, 235, 255, 0.4)" strokeWidth="0.4" clipPath={url("clip")} />
        <circle r="100" fill={url("shade")} />
        <circle r="99.6" fill="none" stroke="rgba(124, 235, 255, 0.6)" strokeWidth="0.9" />

        <g fill="none" strokeLinecap="round">
          {GLOBE_SVG_ARCS.map((arc, index) => (
            <g key={`${arc.from}-${arc.to}`}>
              <path d={arc.d} stroke="rgba(80, 226, 254, 0.32)" strokeWidth="0.7" />
              <path
                className={styles.pulse}
                d={arc.d}
                pathLength={100}
                stroke="#7cebff"
                strokeWidth="1.3"
                style={{ animationDelay: `${(index * 0.73) % 6}s` } as CSSProperties}
              />
            </g>
          ))}
        </g>

        {GLOBE_SVG_CITIES.map((city) => (
          <g key={city.id} transform={`translate(${city.x} ${city.y})`}>
            <circle r="3.6" fill="rgba(80, 226, 254, 0.22)" />
            <circle r="1.5" fill="#bdf4ff" />
          </g>
        ))}

        <g transform={`translate(${home.x} ${home.y})`}>
          <circle className={styles.homeRing} r="3" fill="none" stroke="#7cebff" strokeWidth="1" />
          <line y1="0" y2="-16" stroke="#7cebff" strokeWidth="0.8" />
          <g transform="translate(0 -27)">
            <circle r="11.5" fill="#02041f" stroke="#0481dc" strokeWidth="1.6" />
            <image href={BRAND_ASSETS.icon} x="-6.4" y="-7" width="12.8" height="14" />
          </g>
        </g>
      </svg>
    </div>
  );
};
