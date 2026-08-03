import { useEffect, useRef, useState } from "react";
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
  duration: 5.5 + delay,
  repeat: Infinity,
  repeatType: "mirror" as const,
  ease: "easeInOut" as const
});

export const EcosystemVisual = () => {
  const stageRef = useRef<HTMLDivElement>(null);
  const reduceMotion = useReducedMotion();
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    const node = stageRef.current;
    if (!node) {
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) {
          setIsVisible(true);
          observer.disconnect();
        }
      },
      { threshold: 0.25 }
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  const handleMouseMove = (event: React.MouseEvent<HTMLDivElement>) => {
    if (reduceMotion || !stageRef.current) {
      return;
    }

    const rect = stageRef.current.getBoundingClientRect();
    const x = (event.clientX - rect.left) / rect.width - 0.5;
    const y = (event.clientY - rect.top) / rect.height - 0.5;
    setOffset({ x, y });
  };

  const handleMouseLeave = () => {
    setOffset({ x: 0, y: 0 });
  };

  const layer = (depth: number) =>
    reduceMotion
      ? undefined
      : {
          x: offset.x * depth,
          y: offset.y * depth
        };

  return (
    <div
      ref={stageRef}
      className={styles.stage}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      aria-hidden="true"
    >
      <div className={styles.glowCore} />
      <div className={styles.particles}>
        {Array.from({ length: 14 }, (_, index) => (
          <span key={index} className={styles.particle} style={{ "--i": index } as React.CSSProperties} />
        ))}
      </div>

      <svg className={styles.links} viewBox="0 0 640 560" fill="none">
        <path className={styles.linkPath} d="M320 280 C260 220, 170 180, 110 150" />
        <path className={styles.linkPath} d="M320 280 C380 210, 470 170, 540 140" />
        <path className={styles.linkPath} d="M320 280 C250 300, 150 320, 90 360" />
        <path className={styles.linkPath} d="M320 280 C390 310, 490 340, 560 380" />
        <path className={styles.linkPath} d="M320 280 C300 200, 290 120, 270 70" />
        <path className={styles.linkPath} d="M320 280 C340 360, 360 450, 380 510" />
        <path className={styles.linkPath} d="M320 280 C220 250, 140 240, 70 230" />
        <path className={styles.linkPath} d="M320 280 C420 260, 510 250, 590 240" />
      </svg>

      <motion.div
        className={styles.phoneWrap}
        animate={layer(18)}
        transition={{ type: "spring", stiffness: 60, damping: 18 }}
      >
        <motion.div
          className={styles.phone}
          animate={reduceMotion ? undefined : { y: [0, -8, 0] }}
          transition={floatTransition(0)}
        >
          <div className={styles.phoneNotch} />
          <div className={styles.phoneScreen}>
            <div className={styles.phoneHeader}>
              <span>RYZE</span>
              <em>Today</em>
            </div>
            <div className={styles.phoneSession}>
              <p>Upper strength</p>
              <strong>4 exercises · 62 min</strong>
              <div className={styles.phoneProgress}>
                <span className={isVisible ? styles.phoneProgressFill : undefined} />
              </div>
            </div>
            <div className={styles.phoneStats}>
              <div>
                <span>Streak</span>
                <strong>12</strong>
              </div>
              <div>
                <span>Ready</span>
                <strong>86%</strong>
              </div>
              <div>
                <span>Phase</span>
                <strong>Build</strong>
              </div>
            </div>
            <div className={styles.phoneNext}>
              <Zap aria-hidden="true" />
              <span>Next · Legs · tomorrow 18:00</span>
            </div>
          </div>
        </motion.div>
      </motion.div>

      <motion.article
        className={`${styles.card} ${styles.cardWorkout}`}
        animate={layer(28)}
        transition={{ type: "spring", stiffness: 55, damping: 18 }}
      >
        <motion.div
          animate={reduceMotion ? undefined : { y: [0, -10, 0], rotate: [-4, -2, -4] }}
          transition={floatTransition(0.4)}
        >
          <header>
            <Activity aria-hidden="true" />
            <span>Workout Plan</span>
          </header>
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
        </motion.div>
      </motion.article>

      <motion.article
        className={`${styles.card} ${styles.cardNutrition}`}
        animate={layer(34)}
        transition={{ type: "spring", stiffness: 55, damping: 18 }}
      >
        <motion.div
          animate={reduceMotion ? undefined : { y: [0, -12, 0], rotate: [5, 3, 5] }}
          transition={floatTransition(0.8)}
        >
          <header>
            <Flame aria-hidden="true" />
            <span>Nutrition</span>
          </header>
          <div className={styles.macroRow}>
            {[
              { label: "P", value: 78 },
              { label: "C", value: 62 },
              { label: "F", value: 54 }
            ].map((macro) => (
              <div key={macro.label} className={styles.macroRing}>
                <svg viewBox="0 0 36 36">
                  <circle cx="18" cy="18" r="14" className={styles.ringTrack} />
                  <circle
                    cx="18"
                    cy="18"
                    r="14"
                    className={styles.ringValue}
                    style={{
                      strokeDasharray: `${isVisible ? macro.value : 0} 100`
                    }}
                  />
                </svg>
                <strong>{macro.label}</strong>
              </div>
            ))}
          </div>
        </motion.div>
      </motion.article>

      <motion.article
        className={`${styles.card} ${styles.cardAi}`}
        animate={layer(24)}
        transition={{ type: "spring", stiffness: 55, damping: 18 }}
      >
        <motion.div
          animate={reduceMotion ? undefined : { y: [0, -9, 0], rotate: [-3, -5, -3] }}
          transition={floatTransition(1.1)}
        >
          <header>
            <Brain aria-hidden="true" />
            <span>AI Analysis</span>
          </header>
          <p>Volume rising. Keep intensity, add 1 rest day if sleep dips below 7h.</p>
        </motion.div>
      </motion.article>

      <motion.article
        className={`${styles.card} ${styles.cardProgress}`}
        animate={layer(30)}
        transition={{ type: "spring", stiffness: 55, damping: 18 }}
      >
        <motion.div
          animate={reduceMotion ? undefined : { y: [0, -11, 0], rotate: [3, 5, 3] }}
          transition={floatTransition(0.6)}
        >
          <header>
            <Trophy aria-hidden="true" />
            <span>Progress</span>
          </header>
          <svg className={styles.curve} viewBox="0 0 140 48" preserveAspectRatio="none">
            <path
              className={styles.curveArea}
              d="M0 40 C20 36, 35 28, 50 30 C70 33, 85 18, 105 16 C120 14, 130 10, 140 8 L140 48 L0 48 Z"
            />
            <path
              className={`${styles.curveLine} ${isVisible ? styles.curveLineActive : ""}`}
              d="M0 40 C20 36, 35 28, 50 30 C70 33, 85 18, 105 16 C120 14, 130 10, 140 8"
            />
          </svg>
          <strong>+8.4 kg strength</strong>
        </motion.div>
      </motion.article>

      <motion.article
        className={`${styles.card} ${styles.cardRecovery}`}
        animate={layer(26)}
        transition={{ type: "spring", stiffness: 55, damping: 18 }}
      >
        <motion.div
          animate={reduceMotion ? undefined : { y: [0, -8, 0], rotate: [-6, -4, -6] }}
          transition={floatTransition(1.4)}
        >
          <header>
            <Moon aria-hidden="true" />
            <span>Recovery</span>
          </header>
          <div className={styles.scoreRow}>
            <div className={styles.scoreRing}>
              <svg viewBox="0 0 64 64">
                <circle cx="32" cy="32" r="26" className={styles.ringTrack} />
                <circle
                  cx="32"
                  cy="32"
                  r="26"
                  className={styles.scoreValue}
                  style={{
                    strokeDasharray: `${isVisible ? 86 : 0} 163`
                  }}
                />
              </svg>
              <strong>86</strong>
            </div>
            <ul>
              <li>
                <Droplets aria-hidden="true" /> Hydration OK
              </li>
              <li>
                <Moon aria-hidden="true" /> Sleep 7.4h
              </li>
            </ul>
          </div>
        </motion.div>
      </motion.article>

      <motion.article
        className={`${styles.card} ${styles.cardPr}`}
        animate={layer(32)}
        transition={{ type: "spring", stiffness: 55, damping: 18 }}
      >
        <motion.div
          animate={reduceMotion ? undefined : { y: [0, -10, 0], rotate: [4, 2, 4] }}
          transition={floatTransition(0.9)}
        >
          <header>
            <Trophy aria-hidden="true" />
            <span>Personal Records</span>
          </header>
          <p className={styles.prBadge}>New PR</p>
          <strong>Deadlift 232 kg</strong>
        </motion.div>
      </motion.article>

      <motion.article
        className={`${styles.card} ${styles.cardCoach}`}
        animate={layer(22)}
        transition={{ type: "spring", stiffness: 55, damping: 18 }}
      >
        <motion.div
          animate={reduceMotion ? undefined : { y: [0, -7, 0], rotate: [-2, 0, -2] }}
          transition={floatTransition(1.6)}
        >
          <header>
            <CheckCircle2 aria-hidden="true" />
            <span>Coach Feedback</span>
          </header>
          <p>
            <em>Approved</em> · Plan reviewed for week 6
          </p>
        </motion.div>
      </motion.article>

      <motion.article
        className={`${styles.card} ${styles.cardGoals}`}
        animate={layer(36)}
        transition={{ type: "spring", stiffness: 55, damping: 18 }}
      >
        <motion.div
          animate={reduceMotion ? undefined : { y: [0, -9, 0], rotate: [2, 4, 2] }}
          transition={floatTransition(1.2)}
        >
          <header>
            <Zap aria-hidden="true" />
            <span>Weekly Goals</span>
          </header>
          <div className={styles.goalTrack}>
            <span className={isVisible ? styles.goalFill : undefined} />
          </div>
          <strong>4 of 5 sessions</strong>
        </motion.div>
      </motion.article>
    </div>
  );
};
