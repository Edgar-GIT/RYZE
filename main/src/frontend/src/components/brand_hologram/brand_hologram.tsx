import { useId } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./brand_hologram.module.css";

// The scanner's readout on phones: the set being logged in each phase of the
// cycle (curls, then squats), in the workout player's terms. Sample data.
export interface BrandReadout {
  status: string;
  logged: string;
  sample: string;
  curl: { name: string; load: string };
  squat: { name: string; load: string };
}

interface BrandHologramProps {
  className?: string;
  readout?: BrandReadout;
}

// A seamless tile of the light network drawn over the body (jittered
// triangles and nodes), the same idea as the 3D athlete's skin network.
const NET =
  "M-3.9 -17.4L0.6 1.2M7.5 -13.4L0.6 1.2M7.5 -13.4L13.2 2.0M15.6 -15.7L13.2 2.0M15.6 -15.7L26.3 0.7M32.1 -17.4L26.3 0.7M32.1 -17.4L36.6 1.2M43.5 -13.4L36.6 1.2M-9.7 0.7L0.6 1.2M0.6 1.2L13.2 2.0M0.6 1.2L-3.9 13.6M0.6 1.2L7.5 17.6M13.2 2.0L26.3 0.7M13.2 2.0L7.5 17.6M13.2 2.0L15.6 15.3M26.3 0.7L36.6 1.2M26.3 0.7L15.6 15.3M26.3 0.7L32.1 13.6M36.6 1.2L49.2 2.0M36.6 1.2L32.1 13.6M36.6 1.2L43.5 17.6M-3.9 13.6L7.5 17.6M-3.9 13.6L0.6 32.2M7.5 17.6L15.6 15.3M7.5 17.6L0.6 32.2M7.5 17.6L13.2 33.0M15.6 15.3L32.1 13.6M15.6 15.3L13.2 33.0M15.6 15.3L26.3 31.7M32.1 13.6L43.5 17.6M32.1 13.6L26.3 31.7M32.1 13.6L36.6 32.2M43.5 17.6L36.6 32.2M-9.7 31.7L0.6 32.2M0.6 32.2L13.2 33.0M0.6 32.2L-3.9 44.6M0.6 32.2L7.5 48.6M13.2 33.0L26.3 31.7M26.3 31.7L36.6 32.2M26.3 31.7L15.6 46.3M26.3 31.7L32.1 44.6M36.6 32.2L49.2 33.0M36.6 32.2L32.1 44.6M36.6 32.2L43.5 48.6";
const NODES =
  "M-0.4 1.2a1 1 0 1 0 2 0a1 1 0 1 0 -2 0M12.2 2a1 1 0 1 0 2 0a1 1 0 1 0 -2 0M25.3 0.7a1 1 0 1 0 2 0a1 1 0 1 0 -2 0M35.6 1.2a1 1 0 1 0 2 0a1 1 0 1 0 -2 0M6.5 17.6a1 1 0 1 0 2 0a1 1 0 1 0 -2 0M14.1 15.3a1.5 1.5 0 1 0 3 0a1.5 1.5 0 1 0 -3 0M31.1 13.6a1 1 0 1 0 2 0a1 1 0 1 0 -2 0M-0.4 32.2a1 1 0 1 0 2 0a1 1 0 1 0 -2 0M25.3 31.7a1 1 0 1 0 2 0a1 1 0 1 0 -2 0M35.6 32.2a1 1 0 1 0 2 0a1 1 0 1 0 -2 0";

// The mannequin in profile, facing the copy (left). Joints, in view-box units:
// ankle 182,362 · knee 181,302 · hip 184,252 · shoulder 184,182 · elbow 184,221.
const SHIN = "M173 300C172 318 175 340 177 360L187 360C186 348 194 330 192 316C191 306 189 300 189 300Z";
const THIGH = "M171 250C166 266 168 286 173 304L189 304C191 290 198 270 197 252Z";
const UPPER_ARM = "M174 184C171 198 174 212 177 223L191 223C192 212 195 196 193 184Z";
const FOREARM = "M177 220C175 232 178 244 180 254L189 254C190 244 193 232 192 220Z";
const FOOT = "M176 356L188 356L189 367Q190 373 184 373L160 373Q154 372 156 366Q160 362 168 361Z";

// BrandHologram is the lightweight (SVG and CSS only) version of the desktop
// 3D hologram: a scanning drone reads a faceless, genderless training
// mannequin on a platform while it alternates dumbbell curls and squats, the
// worked muscles pulsing gold to red; with `readout`, a card logs each set.
// Phones, reduced motion and no-WebGL get it instead of Three.js, and the
// desktop story shows it while the 3D scene loads. Decorative: the story's
// copy says what it means.
export const BrandHologram = ({ className, readout }: BrandHologramProps) => {
  const id = useId().replace(/:/g, "");
  const beam = `${id}-beam`;
  const net = `${id}-net`;
  const top = `${id}-top`;
  const scan = `${id}-scan`;
  const glow = `${id}-glow`;
  const heat = `${id}-heat`;
  const clip = `${id}-clip`;
  const skin = { fill: `url(#${net})`, stroke: "#8ff0ff", strokeWidth: 1.3, strokeLinejoin: "round" as const };

  const arm = (far: boolean) => (
    <g className={joinClassNames(styles.upperArm, far ? styles.far : undefined)}>
      <ellipse cx="184" cy="183" rx="11" ry="12" {...skin} />
      <path d={UPPER_ARM} {...skin} />
      {!far && <ellipse className={styles.biceps} cx="177" cy="202" rx="6" ry="13" filter={`url(#${heat})`} />}
      <g className={styles.forearm}>
        <path d={FOREARM} {...skin} />
        <ellipse cx="185" cy="259" rx="6" ry="7" {...skin} />
        {/* Dumbbell, seen end-on across the fist. */}
        <polygon points="185,247 195,253 195,265 185,271 175,265 175,253" fill="rgba(154,243,255,0.16)" stroke="#bff6ff" strokeWidth="1.3" />
        <polygon points="185,253 190,256 190,262 185,265 180,262 180,256" fill="none" stroke="#bff6ff" strokeWidth="1" />
      </g>
    </g>
  );

  return (
    <div className={joinClassNames(styles.frame, className)}>
    <svg className={styles.hologram} viewBox="0 0 360 420" aria-hidden="true" focusable="false">
      <defs>
        <linearGradient id={beam} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#bff6ff" stopOpacity="0.6" />
          <stop offset="55%" stopColor="#36d6ff" stopOpacity="0.18" />
          <stop offset="100%" stopColor="#00cbff" stopOpacity="0.04" />
        </linearGradient>
        <pattern id={net} width="36" height="31" patternUnits="userSpaceOnUse" patternTransform="scale(0.5)">
          <rect width="36" height="31" fill="rgba(54,214,255,0.2)" />
          <path d={NET} fill="none" stroke="#7cebff" strokeOpacity="0.75" strokeWidth="1.1" />
          <path d={NODES} fill="#e8fdff" />
        </pattern>
        <radialGradient id={top} cx="50%" cy="45%" r="55%">
          <stop offset="0%" stopColor="#0c2a66" />
          <stop offset="100%" stopColor="#030a22" />
        </radialGradient>
        <linearGradient id={scan} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#e8fdff" stopOpacity="0" />
          <stop offset="50%" stopColor="#e8fdff" stopOpacity="0.95" />
          <stop offset="100%" stopColor="#e8fdff" stopOpacity="0" />
        </linearGradient>
        <filter id={glow} x="-30%" y="-30%" width="160%" height="160%">
          <feGaussianBlur stdDeviation="2.2" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
        <filter id={heat} x="-100%" y="-60%" width="300%" height="220%">
          <feGaussianBlur stdDeviation="3" />
        </filter>
        <clipPath id={clip}>
          <polygon points="166,62 194,62 318,378 42,378" />
        </clipPath>
      </defs>

      {/* The platform: a dark disc with a metal side, a light rim and rings. */}
      <path d="M30 378 L30 390 A150 26 0 0 0 330 390 L330 378 Z" fill="#0a1534" />
      <path d="M30 390 A150 26 0 0 0 330 390" fill="none" stroke="#7cebff" strokeOpacity="0.55" strokeWidth="1" strokeDasharray="5 4" />
      <ellipse cx="180" cy="378" rx="150" ry="26" fill={`url(#${top})`} stroke="#7cebff" strokeWidth="1.4" />
      <ellipse className={styles.pulse} cx="180" cy="378" rx="136" ry="22" fill="none" stroke="#e8fdff" strokeWidth="1.6" />
      <g className={styles.floorRings} fill="none" stroke="#9af3ff">
        <ellipse cx="180" cy="378" rx="138" ry="22" strokeOpacity="0.7" strokeWidth="1.4" />
        <ellipse cx="180" cy="378" rx="104" ry="16" strokeOpacity="0.45" strokeWidth="1" strokeDasharray="6 7" />
        <ellipse cx="180" cy="378" rx="70" ry="11" strokeOpacity="0.4" strokeWidth="1" />
      </g>

      {/* The beam from the sensor to the platform. */}
      <polygon className={styles.beam} points="166,62 194,62 318,378 42,378" fill={`url(#${beam})`} />
      <g fill="none" stroke="#bff6ff" strokeOpacity="0.55" strokeWidth="1">
        <line x1="166" y1="62" x2="42" y2="378" />
        <line x1="194" y1="62" x2="318" y2="378" />
      </g>

      {/* The athlete: feet planted, then the chain ankle → knee → hip → torso,
          so the squat folds it the way a body folds. */}
      <g className={styles.athlete} filter={`url(#${glow})`}>
        <path className={styles.far} d={FOOT} transform="translate(6 -2)" {...skin} />
        <path d={FOOT} {...skin} />
        <g className={styles.shin}>
          <path className={styles.far} d={SHIN} transform="translate(6 -2)" {...skin} />
          <path d={SHIN} {...skin} />
          <g className={styles.thigh}>
            <path className={styles.far} d={THIGH} transform="translate(6 -2)" {...skin} />
            <path d={THIGH} {...skin} />
            <ellipse className={styles.quads} cx="172" cy="276" rx="6" ry="17" filter={`url(#${heat})`} />
            <g className={styles.torso}>
              <g transform="translate(5 -1)">{arm(true)}</g>
              {/* Torso with the glutes, a neck and a faceless head. */}
              <path
                d="M171 168C163 176 161 186 163 196C164 206 167 214 169 226C170 236 166 246 169 256C172 264 177 268 184 268C192 268 199 263 199 255C199 247 193 240 190 232C188 222 195 210 196 196C197 182 191 172 187 168Z"
                {...skin}
              />
              <ellipse className={styles.glutes} cx="194" cy="255" rx="7" ry="10" filter={`url(#${heat})`} />
              <path d="M172 157L185 157L187 172L171 172Z" {...skin} />
              <path d="M178 129C190 129 192 140 191 146C190 154 184 160 176 161C170 161 165 157 165 150C164 142 168 129 178 129Z" {...skin} />
              {arm(false)}
            </g>
          </g>
        </g>
      </g>

      {/* The scanner's line runs up the body and back down. */}
      <g clipPath={`url(#${clip})`}>
        <rect className={styles.scan} x="40" y="-5" width="280" height="10" fill={`url(#${scan})`} />
      </g>

      {/* The scanner: a lens-shaped pod with a light ring, ducted rotors and a sensor underneath. */}
      <g className={styles.drone}>
        <line x1="104" y1="45" x2="256" y2="45" stroke="#1c2740" strokeWidth="4" strokeLinecap="round" />
        {[104, 256].map((x) => (
          <g key={x}>
            <ellipse cx={x} cy="44" rx="26" ry="6.5" fill="none" stroke="#3a4a6e" strokeWidth="3.4" />
            <ellipse className={styles.rotor} cx={x} cy="44" rx="20" ry="4" fill="#bff6ff" fillOpacity="0.28" />
            <ellipse cx={x} cy="46" rx="22" ry="5" fill="none" stroke="#7cebff" strokeOpacity="0.5" strokeWidth="0.8" />
          </g>
        ))}
        <path d="M136 50 Q180 33 224 50 Z" fill="#1a2438" />
        <path d="M156 44 Q180 38 204 44" fill="none" stroke="#7cebff" strokeOpacity="0.4" strokeWidth="0.8" />
        <ellipse cx="180" cy="51" rx="44" ry="8" fill="#151c2e" />
        <ellipse cx="180" cy="50" rx="44" ry="8" fill="none" stroke="#7cebff" strokeWidth="1.4" />
        <ellipse cx="180" cy="57" rx="22" ry="4.5" fill="#040914" stroke="#7cebff" strokeWidth="1.1" />
        <circle cx="180" cy="61" r="12" fill="#00cbff" fillOpacity="0.35" filter={`url(#${glow})`} />
        <ellipse cx="180" cy="58" rx="5" ry="2.2" fill="#e8fdff" />
      </g>
    </svg>
      {readout ? (
        <div className={styles.readout} aria-hidden="true">
          {[readout.curl, readout.squat].map((entry, index) => (
            <div key={entry.name} className={joinClassNames(styles.card, index === 0 ? styles.cardCurl : styles.cardSquat)}>
              <p className={styles.eyebrow}>
                <span className={styles.live} />
                {readout.status}
              </p>
              <p className={styles.title}>{entry.name}</p>
              <p className={styles.loadRow}>
                <strong>{entry.load}</strong>
                <span className={styles.pill}>✓ {readout.logged}</span>
              </p>
              <p className={styles.sample}>{readout.sample}</p>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
};
