import styles from "./animated_background.module.css";

export const AnimatedBackground = () => (
  <div className={styles.background} aria-hidden="true">
    <span className={styles.glowOne} />
    <span className={styles.glowTwo} />
    <span className={styles.glowThree} />
    <span className={styles.grid} />
    <span className={styles.noise} />
  </div>
);
