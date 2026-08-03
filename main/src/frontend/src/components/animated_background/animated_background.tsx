import styles from "./animated_background.module.css";

export const AnimatedBackground = () => (
  <div className={styles.background} aria-hidden="true">
    <span className={styles.nebulaOne} />
    <span className={styles.nebulaTwo} />
    <span className={styles.nebulaThree} />
    <span className={styles.nebulaFour} />
    <span className={styles.glowPulse} />
    <span className={styles.beam} />
    <span className={styles.beamTwo} />
    <span className={styles.vignette} />
    <span className={styles.stars} />
    <span className={styles.grid} />
    <span className={styles.noise} />
  </div>
);
