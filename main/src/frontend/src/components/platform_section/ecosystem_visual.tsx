import { motion, useReducedMotion } from "framer-motion";
import { Activity, Droplets, Flame, Moon, Zap } from "lucide-react";

import styles from "./ecosystem_visual.module.css";

const floatTransition = (delay: number) => ({
  duration: 5.4 + delay,
  repeat: Infinity,
  repeatType: "mirror" as const,
  ease: "easeInOut" as const
});

type LinkVariant = "top" | "worm" | "bottom";

const LINK_PATHS: Record<LinkVariant, string> = {
  // Top → bottom descending arc into the phone
  top: "M156 6 C108 18, 58 42, 4 58",
  // Middle worm / minhoca
  worm: "M156 30 C128 6, 108 54, 84 28 C60 4, 36 52, 4 30",
  // Bottom descending arc into the phone
  bottom: "M156 8 C110 22, 62 48, 4 62"
};

const CardLink = ({ variant }: { variant: LinkVariant }) => {
  const linkClass =
    variant === "top"
      ? styles.cardLinkTop
      : variant === "worm"
        ? styles.cardLinkWorm
        : styles.cardLinkBottom;

  return (
    <svg
      className={`${styles.cardLink} ${linkClass}`}
      viewBox="0 0 160 68"
      preserveAspectRatio="none"
      aria-hidden="true"
    >
      <path className={styles.linkPath} d={LINK_PATHS[variant]} />
    </svg>
  );
};

export const EcosystemVisual = () => {
  const reduceMotion = useReducedMotion();

  return (
    <div className={styles.stage} aria-hidden="true">
      <div className={styles.glowCore} />
      <div className={styles.glowFloor} />

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

      <motion.article
        className={`${styles.card} ${styles.cardWeek}`}
        animate={reduceMotion ? undefined : { y: [0, -6, 0] }}
        transition={floatTransition(0.35)}
      >
        <CardLink variant="top" />
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
        <CardLink variant="worm" />
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
        <CardLink variant="bottom" />
        <header>
          <Moon aria-hidden="true" />
          <span>Recovery</span>
        </header>
        <div className={styles.recoveryBody}>
          <div className={styles.scoreRing}>
            <svg viewBox="0 0 64 64" aria-hidden="true">
              <circle cx="32" cy="32" r="24" className={styles.ringTrack} />
              <circle
                cx="32"
                cy="32"
                r="24"
                className={styles.scoreValue}
                style={{ strokeDasharray: "130 151" }}
              />
            </svg>
            <span className={styles.scoreValueText}>86</span>
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
