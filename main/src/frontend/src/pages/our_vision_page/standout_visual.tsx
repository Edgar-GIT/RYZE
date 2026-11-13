import { Check, Dumbbell, Sparkles } from "lucide-react";
import type { CSSProperties } from "react";

import { Waves } from "@/components/public_ui/public_ui";
import { BRAND_ASSETS } from "@/constants/brand_assets";
import { useGettext } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import { OUR_VISION_PT } from "./our_vision_page.pt";
import styles from "./standout_visual.module.css";

// What sets RYZE apart today, and the next step on the roadmap.
const DIFFERENCES = [
  "Training and nutrition in one plan",
  "Plans prepared by real trainers",
  "Every set logged, every week tracked"
] as const;

const SAME_TILES = 12;

// RyzeCard: RYZE as the light-blue wave panel (the login page's bright half),
// listing what it does differently. Shared by the flat wall and the 3D stage.
export const RyzeCard = ({ className }: { className?: string }) => {
  const tr = useGettext(OUR_VISION_PT);
  return (
    <div className={joinClassNames(styles.ryze, className)}>
      <div className={styles.brand}>
        <img src={BRAND_ASSETS.icon} alt="" width="600" height="658" />
        <span>RYZE</span>
      </div>
      <ul>
        {DIFFERENCES.map((item) => (
          <li key={item}>
            <Check strokeWidth={2.6} />
            {tr(item)}
          </li>
        ))}
        <li className={styles.next}>
          <Sparkles strokeWidth={2.2} />
          {tr("AI-generated programs")}
          <em>{tr("Next")}</em>
        </li>
      </ul>
      <Waves className={styles.waves} />
    </div>
  );
};

// StandoutVisual: a wall of identical, greyed-out fitness apps with RYZE in the
// middle. Decorative (the copy and the comparison section below say the same in
// text). This is the phone, reduced-motion and no-WebGL version of the hero;
// tablets and desktops get the 3D tunnel (VisionHeroVisual).
export const StandoutVisual = ({ className }: { className?: string }) => {
  const tr = useGettext(OUR_VISION_PT);
  return (
  <div className={joinClassNames(styles.wall, className)} aria-hidden="true">
    {Array.from({ length: SAME_TILES }, (_, index) => (
      <span key={index} className={styles.same} style={{ "--i": index } as CSSProperties}>
        <Dumbbell strokeWidth={1.8} />
        <i />
        <i />
        <small>{tr("12-week plan")}</small>
      </span>
    ))}

    <RyzeCard />
  </div>
  );
};
