import { motion, useReducedMotion } from "framer-motion";
import { Activity, Brain, Flame, HelpCircle, Leaf, Sparkles, Zap } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import styles from "./ecosystem_visual.module.css";

const floatTransition = (delay: number) => ({
  duration: 5.6 + delay,
  repeat: Infinity,
  repeatType: "mirror" as const,
  ease: "easeInOut" as const
});

export const EcosystemVisual = () => {
  const reduceMotion = useReducedMotion();
  const stageRef = useRef<HTMLDivElement>(null);
  const [tilt, setTilt] = useState({ x: 0, y: 0 });

  useEffect(() => {
    if (reduceMotion) {
      return;
    }

    const stage = stageRef.current;
    if (!stage) {
      return;
    }

    const onMove = (event: MouseEvent) => {
      const rect = stage.getBoundingClientRect();
      const px = (event.clientX - rect.left) / rect.width - 0.5;
      const py = (event.clientY - rect.top) / rect.height - 0.5;
      setTilt({ x: py * -4, y: px * 5 });
    };

    const onLeave = () => setTilt({ x: 0, y: 0 });

    stage.addEventListener("mousemove", onMove);
    stage.addEventListener("mouseleave", onLeave);
    return () => {
      stage.removeEventListener("mousemove", onMove);
      stage.removeEventListener("mouseleave", onLeave);
    };
  }, [reduceMotion]);

  return (
    <div className={styles.stage} ref={stageRef} aria-hidden="true">
      <div className={styles.atmosphere}>
        <div className={styles.grid} />
        <div className={styles.glowLarge} />
        <div className={styles.glowSoft} />
        <div className={styles.beam} />
        <div className={styles.beamTwo} />
        <div className={styles.ring} />
        <div className={styles.ringOuter} />
        <div className={styles.particles}>
          {Array.from({ length: 14 }, (_, index) => (
            <span key={index} style={{ ["--i" as string]: index }} />
          ))}
        </div>
        <svg className={styles.dataTrails} viewBox="0 0 640 520" preserveAspectRatio="none">
          <path d="M40 120 C140 90, 220 140, 300 180 S460 210, 560 150" />
          <path d="M60 280 C160 250, 240 300, 330 320 S470 340, 590 290" />
          <path d="M80 400 C180 360, 260 390, 360 410 S500 430, 600 380" />
        </svg>
        <svg className={styles.storyFlow} viewBox="0 0 640 520" preserveAspectRatio="none">
          <path d="M90 180 C150 200, 200 240, 250 280" />
          <path d="M250 280 C300 310, 340 330, 390 300" />
          <path d="M470 250 C520 220, 560 200, 600 180" />
        </svg>
      </div>

      <div className={styles.storyNodes}>
        <span className={`${styles.storyNode} ${styles.storyQ}`}>
          <HelpCircle />
        </span>
        <span className={`${styles.storyNode} ${styles.storyAi}`}>
          <Brain />
        </span>
        <span className={`${styles.storyNode} ${styles.storyPlan}`}>
          <Sparkles />
        </span>
        <span className={`${styles.storyNode} ${styles.storyProgress}`}>
          <Activity />
        </span>
      </div>

      <div className={styles.fragments}>
        <span className={`${styles.fragment} ${styles.fragRing}`}>
          <svg viewBox="0 0 40 40">
            <circle cx="20" cy="20" r="14" />
            <circle cx="20" cy="20" r="14" className={styles.fragRingValue} />
          </svg>
        </span>
        <span className={`${styles.fragment} ${styles.fragChart}`}>
          <i />
          <i />
          <i />
          <i />
        </span>
        <span className={`${styles.fragment} ${styles.fragTrain}`}>
          <Flame />
        </span>
        <span className={`${styles.fragment} ${styles.fragFuel}`}>
          <Leaf />
        </span>
      </div>

      <motion.div
        className={styles.phoneScene}
        animate={reduceMotion ? undefined : { y: [0, -10, 0] }}
        transition={floatTransition(0)}
      >
        <div
          className={styles.phoneParallax}
          style={{
            transform: `perspective(1200px) rotateX(${tilt.x}deg) rotateY(${tilt.y}deg)`
          }}
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
        </div>
      </motion.div>
    </div>
  );
};
