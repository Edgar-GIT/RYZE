import { motion, useReducedMotion } from "framer-motion";
import { Activity, Droplets, Flame, Moon, Zap } from "lucide-react";

import styles from "./ecosystem_visual.module.css";

const floatTransition = (delay: number) => ({
  duration: 5.4 + delay,
  repeat: Infinity,
  repeatType: "mirror" as const,
  ease: "easeInOut" as const
});

export const EcosystemVisual = () => {
  const reduceMotion = useReducedMotion();

  return (
    <div className={styles.stage} aria-hidden="true">
      <div className={styles.glowCore} />
      <div className={styles.glowFloor} />

      <svg className={styles.links} viewBox="0 0 640 520" preserveAspectRatio="none">
        <path className={styles.linkPath} d="M292 168 C340 150, 390 132, 458 118" />
        <path className={styles.linkPath} d="M300 250 C360 238, 420 232, 472 248" />
        <path className={styles.linkPath} d="M286 330 C350 350, 410 372, 464 388" />
      </svg>

      <motion.div
        className={styles.phoneScene}
        animate={reduceMotion ? undefined : { y: [0, -8, 0] }}
        transition={floatTransition(0)}
      >
        <div className={styles.phone}>
          <div className={styles.phoneBezel}>
            <div className={styles.phoneNotch} />
            <div className={styles.phoneScreen}>
              <div className={styles.phoneStatus}>
                <span>9:41</span>
                <em>RYZE</em>
              </div>

              <div className={styles.phoneGreeting}>
                <p>Today</p>
                <strong>Upper strength</strong>
              </div>

              <div className={styles.phoneSession}>
                <div className={styles.sessionMeta}>
                  <span>4 exercises</span>
                  <span>48 min</span>
                </div>
                <div className={styles.phoneProgress}>
                  <span className={styles.phoneProgressFill} />
                </div>
                <div className={styles.sessionList}>
                  <span>Bench press</span>
                  <span>Row · 4×8</span>
                  <span>Shoulder press</span>
                </div>
              </div>

              <div className={styles.phoneNext}>
                <Zap aria-hidden="true" />
                <span>Legs tomorrow · 18:00</span>
              </div>
            </div>
          </div>
          <div className={styles.phoneSide} />
          <div className={styles.phoneShadow} />
        </div>
      </motion.div>

      <motion.article
        className={`${styles.card} ${styles.cardWeek}`}
        animate={reduceMotion ? undefined : { y: [0, -6, 0] }}
        transition={floatTransition(0.35)}
      >
        <header>
          <Activity aria-hidden="true" />
          <span>This week</span>
        </header>
        <strong className={styles.cardLead}>Train around your real schedule.</strong>
        <div className={styles.heatmap}>
          {["M", "T", "W", "T", "F", "S", "S"].map((day, index) => (
            <div key={`${day}-${index}`} className={styles.heatCell}>
              <span
                className={
                  index < 4 ? styles.heatDone : index === 4 ? styles.heatActive : styles.heatIdle
                }
              />
              <em>{day}</em>
            </div>
          ))}
        </div>
        <div className={styles.metaRow}>
          <span>Current phase</span>
          <strong>Build block</strong>
        </div>
      </motion.article>

      <motion.article
        className={`${styles.card} ${styles.cardNutrition}`}
        animate={reduceMotion ? undefined : { y: [0, -5, 0] }}
        transition={floatTransition(0.7)}
      >
        <header>
          <Flame aria-hidden="true" />
          <span>Nutrition</span>
        </header>
        <div className={styles.macroRow}>
          {[
            { label: "P", value: "78" },
            { label: "C", value: "64" },
            { label: "F", value: "52" }
          ].map((macro) => (
            <div key={macro.label} className={styles.macroRing}>
              <svg viewBox="0 0 36 36">
                <circle cx="18" cy="18" r="14" className={styles.ringTrack} />
                <circle cx="18" cy="18" r="14" className={styles.ringValue} />
              </svg>
              <strong>{macro.label}</strong>
              <span>{macro.value}%</span>
            </div>
          ))}
        </div>
      </motion.article>

      <motion.article
        className={`${styles.card} ${styles.cardRecovery}`}
        animate={reduceMotion ? undefined : { y: [0, -4, 0] }}
        transition={floatTransition(1.05)}
      >
        <header>
          <Moon aria-hidden="true" />
          <span>Recovery</span>
        </header>
        <div className={styles.recoveryBody}>
          <div className={styles.scoreRing}>
            <svg viewBox="0 0 64 64">
              <circle cx="32" cy="32" r="24" className={styles.ringTrack} />
              <circle
                cx="32"
                cy="32"
                r="24"
                className={styles.scoreValue}
                style={{ strokeDasharray: "130 151" }}
              />
            </svg>
            <strong>86</strong>
          </div>
          <ul>
            <li>
              <Droplets aria-hidden="true" />
              <span>Hydration on track</span>
            </li>
            <li>
              <Moon aria-hidden="true" />
              <span>Sleep 7.4h avg</span>
            </li>
          </ul>
        </div>
      </motion.article>
    </div>
  );
};
