import { Check, Dumbbell, Sparkles } from "lucide-react";
import type { CSSProperties } from "react";

import { BRAND_ASSETS } from "@/constants/brand_assets";
import { joinClassNames } from "@utils/class_names";

import styles from "./standout_visual.module.css";

// What sets RYZE apart today, and the next step on the roadmap.
const DIFFERENCES = [
  "Training and nutrition in one plan",
  "Plans prepared by real trainers",
  "Every set logged, every week tracked"
] as const;

const SAME_TILES = 12;

// StandoutVisual: a wall of identical, greyed-out fitness apps with RYZE lit up
// in the middle, listing what it does differently. Decorative (the copy and the
// comparison section below say the same in text).
export const StandoutVisual = ({ className }: { className?: string }) => (
  <div className={joinClassNames(styles.wall, className)} aria-hidden="true">
    {Array.from({ length: SAME_TILES }, (_, index) => (
      <span key={index} className={styles.same} style={{ "--i": index } as CSSProperties}>
        <Dumbbell strokeWidth={1.8} />
        <i />
        <i />
        <small>12-week plan</small>
      </span>
    ))}

    <div className={styles.ryze}>
      <div className={styles.brand}>
        <img src={BRAND_ASSETS.icon} alt="" width="600" height="658" />
        <span>RYZE</span>
      </div>
      <ul>
        {DIFFERENCES.map((item) => (
          <li key={item}>
            <Check strokeWidth={2.6} />
            {item}
          </li>
        ))}
        <li className={styles.next}>
          <Sparkles strokeWidth={2.2} />
          AI-generated programs
          <em>Next</em>
        </li>
      </ul>
    </div>
  </div>
);
