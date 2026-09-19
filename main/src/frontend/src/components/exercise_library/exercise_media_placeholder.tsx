import { BRAND_ASSETS } from "@/constants/brand_assets";

import styles from "./exercise_media_placeholder.module.css";

// ExerciseMediaPlaceholder is the RYZE-branded empty state for catalogue media.
// Until a demonstration video or preview image is attached to an exercise the
// placeholder communicates that visual content is coming, never a blank box.
export const ExerciseMediaPlaceholder = ({ label = "Demonstration media" }: { label?: string }) => (
  <div className={styles.placeholder} role="img" aria-label={`${label} coming soon`}>
    <div className={styles.logoWrap}>
      <img src={BRAND_ASSETS.icon} alt="" className={styles.logo} />
    </div>
    <p className={styles.text}>RYZE media placeholder</p>
    <p className={styles.hint}>{label} coming soon</p>
  </div>
);