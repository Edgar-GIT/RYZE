import { motion, useReducedMotion } from "framer-motion";
import {
  Activity,
  Brain,
  CheckCircle2,
  Droplets,
  Flame,
  Moon,
  Trophy,
  Zap
} from "lucide-react";

import styles from "./ecosystem_visual.module.css";

const floatTransition = (delay: number) => ({
  duration: 5.2 + delay,
  repeat: Infinity,
  repeatType: "mirror" as const,
  ease: "easeInOut" as const
});

export const EcosystemVisual = () => {
  const reduceMotion = useReducedMotion();

  return (
    <div className={styles.stage} aria-hidden="true">
      <div className={styles.glowCore} />
      <motion.div
        className={styles.deliveryBar}
        animate={reduceMotion ? undefined : { y: [0, -4, 0] }}
        transition={floatTransition(0.2)}
      >
        <header>
          <Zap aria-hidden="true" />
          <span>Ready faster</span>
        </header>
        <strong>Questionnaire to plan in minutes</strong>
      </motion.div>

      <motion.div
        className={styles.phone}
        animate={reduceMotion ? undefined : { y: [0, -6, 0] }}
        transition={floatTransition(0)}
      >
        <div className={styles.phoneNotch} />
        <div className={styles.phoneScreen}>
          <div className={styles.phoneHeader}>
            <span>RYZE</span>
            <em>Today</em>
          </div>
          <div className={styles.phoneSession}>
            <p>Today&apos;s workout</p>
            <strong>Upper strength · 4 exercises</strong>
            <div className={styles.phoneProgress}>
              <span className={styles.phoneProgressFill} />
            </div>
          </div>
          <div className={styles.phoneStats}>
            <div>
              <span>Week</span>
              <strong>4/5</strong>
            </div>
            <div>
              <span>Calories</span>
              <strong>1.8k</strong>
            </div>
            <div>
              <span>Recovery</span>
              <strong>86</strong>
            </div>
          </div>
          <div className={styles.phoneNext}>
            <Zap aria-hidden="true" />
            <span>Next: Legs tomorrow at 18:00</span>
          </div>
        </div>
      </motion.div>

      <div className={styles.railLeft}>
        <motion.article
          className={`${styles.card} ${styles.cardPlan}`}
          animate={reduceMotion ? undefined : { y: [0, -5, 0] }}
          transition={floatTransition(0.3)}
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
          className={`${styles.card} ${styles.cardCompact}`}
          animate={reduceMotion ? undefined : { y: [0, -4, 0] }}
          transition={floatTransition(0.8)}
        >
          <header>
            <Brain aria-hidden="true" />
            <span>Plan guidance</span>
          </header>
          <p>If sleep dips, RYZE shifts volume before the week falls apart.</p>
        </motion.article>

        <motion.article
          className={`${styles.card} ${styles.cardCompact}`}
          animate={reduceMotion ? undefined : { y: [0, -5, 0] }}
          transition={floatTransition(1.1)}
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

      <div className={styles.railRight}>
        <motion.article
          className={`${styles.card} ${styles.cardCompact}`}
          animate={reduceMotion ? undefined : { y: [0, -5, 0] }}
          transition={floatTransition(0.5)}
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
          className={`${styles.card} ${styles.cardProgress}`}
          animate={reduceMotion ? undefined : { y: [0, -4, 0] }}
          transition={floatTransition(0.7)}
        >
          <header>
            <Trophy aria-hidden="true" />
            <span>Progress</span>
          </header>
          <strong className={styles.cardLead}>See if the plan is working.</strong>
          <svg className={styles.curve} viewBox="0 0 140 48" preserveAspectRatio="none">
            <path
              className={styles.curveArea}
              d="M0 40 C20 36, 35 28, 50 30 C70 33, 85 18, 105 16 C120 14, 130 10, 140 8 L140 48 L0 48 Z"
            />
            <path
              className={styles.curveLine}
              d="M0 40 C20 36, 35 28, 50 30 C70 33, 85 18, 105 16 C120 14, 130 10, 140 8"
            />
          </svg>
          <div className={styles.metaRow}>
            <span>Strength index</span>
            <strong>+26%</strong>
          </div>
        </motion.article>

        <div className={styles.stackBottom}>
          <motion.article
            className={`${styles.card} ${styles.cardCompact}`}
            animate={reduceMotion ? undefined : { y: [0, -5, 0] }}
            transition={floatTransition(1)}
          >
            <header>
              <Trophy aria-hidden="true" />
              <span>Milestones</span>
            </header>
            <p className={styles.prBadge}>New PR</p>
            <strong>Deadlift 232 kg</strong>
          </motion.article>

          <motion.article
            className={`${styles.card} ${styles.cardCompact}`}
            animate={reduceMotion ? undefined : { y: [0, -4, 0] }}
            transition={floatTransition(1.3)}
          >
            <header>
              <CheckCircle2 aria-hidden="true" />
              <span>Support</span>
            </header>
            <p>
              <em>Coach-approved</em> when you move into Elite
            </p>
          </motion.article>
        </div>
      </div>
    </div>
  );
};
