import { motion, useReducedMotion } from "framer-motion";
import { Flame, Zap } from "lucide-react";

import styles from "./ecosystem_visual.module.css";

const floatTransition = {
  duration: 5.8,
  repeat: Infinity,
  repeatType: "mirror" as const,
  ease: "easeInOut" as const
};

export const EcosystemVisual = () => {
  const reduceMotion = useReducedMotion();

  return (
    <div className={styles.stage} aria-hidden="true">
      <div className={styles.ambient}>
        <div className={styles.glowCore} />
        <div className={styles.glowSoft} />
        <div className={styles.particles}>
          {Array.from({ length: 10 }, (_, index) => (
            <span key={index} style={{ ["--i" as string]: index }} />
          ))}
        </div>
      </div>

      <motion.div
        className={styles.phoneScene}
        animate={reduceMotion ? undefined : { y: [0, -8, 0] }}
        transition={floatTransition}
      >
        <div className={styles.phoneGlow} />
        <div className={styles.phone}>
          <div className={styles.phoneBezel}>
            <div className={styles.phoneNotch} />
            <div className={styles.phoneScreen}>
              <div className={styles.phoneStatus}>
                <span>9:41</span>
                <em>RYZE</em>
              </div>

              <div className={styles.phoneBanner}>
                <div>
                  <p>Fri 4 · Evening</p>
                  <strong>Ready to train</strong>
                </div>
                <span className={styles.readyPill}>86</span>
              </div>

              <div className={styles.phoneProfile}>
                <div>
                  <p>Welcome back</p>
                  <strong>Alex</strong>
                </div>
                <span className={styles.streakChip}>
                  <Flame aria-hidden="true" />
                  12 day streak
                </span>
              </div>

              <div className={styles.phoneTabs}>
                <span className={styles.tabActive}>Train</span>
                <span>Fuel</span>
                <span>Recover</span>
              </div>

              <div className={styles.phoneGreeting}>
                <p>Today&apos;s session</p>
                <strong>Upper strength</strong>
                <span className={styles.phaseChip}>Build block · Week 4</span>
              </div>

              <div className={styles.miniWeek}>
                {["M", "T", "W", "T", "F", "S", "S"].map((day, index) => (
                  <span
                    key={`${day}-${index}`}
                    className={
                      index < 4
                        ? styles.miniDone
                        : index === 4
                          ? styles.miniActive
                          : styles.miniIdle
                    }
                  >
                    {day}
                  </span>
                ))}
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
                  <div>
                    <span>Bench press</span>
                    <em>4×8</em>
                  </div>
                  <div>
                    <span>Barbell row</span>
                    <em>4×8</em>
                  </div>
                  <div>
                    <span>Shoulder press</span>
                    <em>3×10</em>
                  </div>
                  <div>
                    <span>Face pulls</span>
                    <em>3×12</em>
                  </div>
                </div>
              </div>

              <div className={styles.phoneStats}>
                <div>
                  <span>Week</span>
                  <strong>4/5</strong>
                </div>
                <div>
                  <span>Load</span>
                  <strong>+6%</strong>
                </div>
                <div>
                  <span>Ready</span>
                  <strong>86</strong>
                </div>
              </div>

              <div className={styles.phoneInsight}>
                <span>Volume trimmed after late sleep</span>
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
    </div>
  );
};
