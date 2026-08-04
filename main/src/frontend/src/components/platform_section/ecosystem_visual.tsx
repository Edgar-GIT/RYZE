import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode
} from "react";
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

type NodeId =
  | "goals"
  | "workout"
  | "nutrition"
  | "ai"
  | "progress"
  | "recovery"
  | "pr"
  | "coach";

interface OrbitCard {
  id: NodeId;
  label: string;
  icon: ReactNode;
  slot: string;
  delay: number;
}

interface LinkLine {
  id: NodeId;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

const CARDS: OrbitCard[] = [
  {
    id: "goals",
    label: "Weekly Goals",
    icon: <Zap aria-hidden="true" />,
    slot: styles.slotGoals,
    delay: 0.4
  },
  {
    id: "workout",
    label: "Workout Plan",
    icon: <Activity aria-hidden="true" />,
    slot: styles.slotWorkout,
    delay: 0.2
  },
  {
    id: "nutrition",
    label: "Nutrition",
    icon: <Flame aria-hidden="true" />,
    slot: styles.slotNutrition,
    delay: 0.6
  },
  {
    id: "ai",
    label: "AI Analysis",
    icon: <Brain aria-hidden="true" />,
    slot: styles.slotAi,
    delay: 0.9
  },
  {
    id: "progress",
    label: "Progress",
    icon: <Trophy aria-hidden="true" />,
    slot: styles.slotProgress,
    delay: 0.5
  },
  {
    id: "recovery",
    label: "Recovery",
    icon: <Moon aria-hidden="true" />,
    slot: styles.slotRecovery,
    delay: 1.1
  },
  {
    id: "pr",
    label: "Personal Records",
    icon: <Trophy aria-hidden="true" />,
    slot: styles.slotPr,
    delay: 0.8
  },
  {
    id: "coach",
    label: "Coach Feedback",
    icon: <CheckCircle2 aria-hidden="true" />,
    slot: styles.slotCoach,
    delay: 1.3
  }
];

const floatTransition = (delay: number) => ({
  duration: 5.5 + delay,
  repeat: Infinity,
  repeatType: "mirror" as const,
  ease: "easeInOut" as const
});

const edgePoint = (
  ox: number,
  oy: number,
  cx: number,
  cy: number,
  halfW: number,
  halfH: number
) => {
  const dx = cx - ox;
  const dy = cy - oy;
  const distance = Math.hypot(dx, dy) || 1;
  const nx = dx / distance;
  const ny = dy / distance;
  const inset = 8;
  const tx = Math.abs(nx) < 0.001 ? Number.POSITIVE_INFINITY : (halfW - inset) / Math.abs(nx);
  const ty = Math.abs(ny) < 0.001 ? Number.POSITIVE_INFINITY : (halfH - inset) / Math.abs(ny);
  const t = Math.min(tx, ty);

  return {
    x: cx - nx * t,
    y: cy - ny * t
  };
};

export const EcosystemVisual = () => {
  const stageRef = useRef<HTMLDivElement>(null);
  const phoneRef = useRef<HTMLDivElement>(null);
  const cardRefs = useRef<Partial<Record<NodeId, HTMLElement | null>>>({});
  const reduceMotion = useReducedMotion();
  const [isVisible, setIsVisible] = useState(false);
  const [lines, setLines] = useState<LinkLine[]>([]);
  const [stageSize, setStageSize] = useState({ width: 1, height: 1 });

  const updateLines = useCallback(() => {
    const stage = stageRef.current;
    const phone = phoneRef.current;

    if (!stage || !phone || window.matchMedia("(max-width: 960px)").matches) {
      setLines([]);
      return;
    }

    const stageRect = stage.getBoundingClientRect();
    const phoneRect = phone.getBoundingClientRect();
    const originX = phoneRect.left + phoneRect.width / 2 - stageRect.left;
    const originY = phoneRect.top + phoneRect.height / 2 - stageRect.top;
    const phoneRadius = Math.min(phoneRect.width, phoneRect.height) * 0.42;

    setStageSize({ width: stageRect.width, height: stageRect.height });

    const next: LinkLine[] = [];

    for (const card of CARDS) {
      const node = cardRefs.current[card.id];
      if (!node) {
        continue;
      }

      const rect = node.getBoundingClientRect();
      const cx = rect.left + rect.width / 2 - stageRect.left;
      const cy = rect.top + rect.height / 2 - stageRect.top;
      const distance = Math.hypot(cx - originX, cy - originY) || 1;
      const start = {
        x: originX + ((cx - originX) / distance) * phoneRadius,
        y: originY + ((cy - originY) / distance) * phoneRadius
      };
      const end = edgePoint(originX, originY, cx, cy, rect.width / 2, rect.height / 2);

      next.push({ id: card.id, x1: start.x, y1: start.y, x2: end.x, y2: end.y });
    }

    setLines(next);
  }, []);

  useLayoutEffect(() => {
    updateLines();
    const frame = window.requestAnimationFrame(() => updateLines());
    const stage = stageRef.current;

    if (!stage) {
      return () => window.cancelAnimationFrame(frame);
    }

    const observer = new ResizeObserver(() => updateLines());
    observer.observe(stage);
    window.addEventListener("resize", updateLines);

    return () => {
      window.cancelAnimationFrame(frame);
      observer.disconnect();
      window.removeEventListener("resize", updateLines);
    };
  }, [updateLines]);

  useEffect(() => {
    const node = stageRef.current;
    if (!node) {
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) {
          setIsVisible(true);
          updateLines();
          observer.disconnect();
        }
      },
      { threshold: 0.2 }
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, [updateLines]);

  return (
    <div ref={stageRef} className={styles.stage} aria-hidden="true">
      <div className={styles.glowCore} />

      <svg
        className={styles.links}
        width={stageSize.width}
        height={stageSize.height}
        viewBox={`0 0 ${stageSize.width} ${stageSize.height}`}
      >
        {lines.map((line) => (
          <g key={line.id}>
            <line className={styles.linkPath} x1={line.x1} y1={line.y1} x2={line.x2} y2={line.y2} />
            <circle className={styles.linkDot} cx={line.x2} cy={line.y2} r="2.5" />
          </g>
        ))}
      </svg>

      <motion.div
        ref={phoneRef}
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

      {CARDS.map((card) => (
        <motion.article
          key={card.id}
          ref={(element) => {
            cardRefs.current[card.id] = element;
          }}
          className={`${styles.card} ${card.slot}`}
          animate={reduceMotion ? undefined : { y: [0, -5, 0] }}
          transition={floatTransition(card.delay)}
        >
          <header>
            {card.icon}
            <span>{card.label}</span>
          </header>
          <NodeBody id={card.id} isVisible={isVisible} />
        </motion.article>
      ))}
    </div>
  );
};

const NodeBody = ({ id, isVisible }: { id: NodeId; isVisible: boolean }) => {
  if (id === "workout") {
    return (
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
    );
  }

  if (id === "nutrition") {
    return (
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
                style={{ strokeDasharray: `${isVisible ? macro.value : 0} 100` }}
              />
            </svg>
            <strong>{macro.label}</strong>
          </div>
        ))}
      </div>
    );
  }

  if (id === "ai") {
    return <p>Volume rising. Keep intensity, add 1 rest day if sleep dips below 7h.</p>;
  }

  if (id === "progress") {
    return (
      <>
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
      </>
    );
  }

  if (id === "recovery") {
    return (
      <div className={styles.recoveryBody}>
        <div className={styles.scoreRing}>
          <svg viewBox="0 0 64 64">
            <circle cx="32" cy="32" r="24" className={styles.ringTrack} />
            <circle
              cx="32"
              cy="32"
              r="24"
              className={styles.scoreValue}
              style={{ strokeDasharray: `${isVisible ? 130 : 0} 151` }}
            />
          </svg>
          <strong>86</strong>
        </div>
        <ul>
          <li>
            <Droplets aria-hidden="true" />
            <span>Hydration OK</span>
          </li>
          <li>
            <Moon aria-hidden="true" />
            <span>Sleep 7.4h</span>
          </li>
        </ul>
      </div>
    );
  }

  if (id === "pr") {
    return (
      <>
        <p className={styles.prBadge}>New PR</p>
        <strong>Deadlift 232 kg</strong>
      </>
    );
  }

  if (id === "coach") {
    return (
      <p>
        <em>Approved</em> · Plan reviewed for week 6
      </p>
    );
  }

  return (
    <>
      <div className={styles.goalTrack}>
        <span className={isVisible ? styles.goalFill : undefined} />
      </div>
      <strong>4 of 5 sessions</strong>
    </>
  );
};
