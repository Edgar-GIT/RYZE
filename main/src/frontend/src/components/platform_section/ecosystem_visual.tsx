import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
  type MouseEvent,
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

interface OrbitNode {
  id: NodeId;
  x: number;
  y: number;
  rotate: number;
  depth: number;
  delay: number;
  label: string;
  icon: ReactNode;
}

interface LinkLine {
  id: NodeId;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

/** Card centers in % of the stage. */
const ORBIT_NODES: OrbitNode[] = [
  {
    id: "goals",
    x: 50,
    y: 8,
    rotate: 1,
    depth: 16,
    delay: 0.5,
    label: "Weekly Goals",
    icon: <Zap aria-hidden="true" />
  },
  {
    id: "workout",
    x: 14,
    y: 24,
    rotate: -2,
    depth: 20,
    delay: 0.3,
    label: "Workout Plan",
    icon: <Activity aria-hidden="true" />
  },
  {
    id: "nutrition",
    x: 86,
    y: 24,
    rotate: 2,
    depth: 20,
    delay: 0.7,
    label: "Nutrition",
    icon: <Flame aria-hidden="true" />
  },
  {
    id: "ai",
    x: 12,
    y: 50,
    rotate: -1.5,
    depth: 18,
    delay: 1.0,
    label: "AI Analysis",
    icon: <Brain aria-hidden="true" />
  },
  {
    id: "progress",
    x: 88,
    y: 50,
    rotate: 1.5,
    depth: 18,
    delay: 0.6,
    label: "Progress",
    icon: <Trophy aria-hidden="true" />
  },
  {
    id: "recovery",
    x: 16,
    y: 76,
    rotate: -2,
    depth: 20,
    delay: 1.2,
    label: "Recovery",
    icon: <Moon aria-hidden="true" />
  },
  {
    id: "pr",
    x: 84,
    y: 76,
    rotate: 2,
    depth: 20,
    delay: 0.9,
    label: "Personal Records",
    icon: <Trophy aria-hidden="true" />
  },
  {
    id: "coach",
    x: 50,
    y: 93,
    rotate: 0,
    depth: 14,
    delay: 1.4,
    label: "Coach Feedback",
    icon: <CheckCircle2 aria-hidden="true" />
  }
];

const PHONE_RADIUS_RATIO = 0.14;
const CARD_INSET = 6;

const floatTransition = (delay: number) => ({
  duration: 6 + delay,
  repeat: Infinity,
  repeatType: "mirror" as const,
  ease: "easeInOut" as const
});

const intersectRectEdge = (
  originX: number,
  originY: number,
  targetX: number,
  targetY: number,
  left: number,
  top: number,
  right: number,
  bottom: number
) => {
  const dx = targetX - originX;
  const dy = targetY - originY;

  if (dx === 0 && dy === 0) {
    return { x: targetX, y: targetY };
  }

  let bestT = Number.POSITIVE_INFINITY;

  if (dx !== 0) {
    for (const edgeX of [left, right]) {
      const t = (edgeX - originX) / dx;
      if (t > 0.02 && t < bestT) {
        const y = originY + t * dy;
        if (y >= top - 0.5 && y <= bottom + 0.5) {
          bestT = t;
        }
      }
    }
  }

  if (dy !== 0) {
    for (const edgeY of [top, bottom]) {
      const t = (edgeY - originY) / dy;
      if (t > 0.02 && t < bestT) {
        const x = originX + t * dx;
        if (x >= left - 0.5 && x <= right + 0.5) {
          bestT = t;
        }
      }
    }
  }

  if (!Number.isFinite(bestT)) {
    return { x: targetX, y: targetY };
  }

  return {
    x: originX + bestT * dx,
    y: originY + bestT * dy
  };
};

export const EcosystemVisual = () => {
  const stageRef = useRef<HTMLDivElement>(null);
  const phoneRef = useRef<HTMLDivElement>(null);
  const cardRefs = useRef<Partial<Record<NodeId, HTMLElement | null>>>({});
  const reduceMotion = useReducedMotion();
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const [isVisible, setIsVisible] = useState(false);
  const [lines, setLines] = useState<LinkLine[]>([]);
  const [stageSize, setStageSize] = useState({ width: 0, height: 0 });

  const updateLines = useCallback(() => {
    const stage = stageRef.current;
    const phone = phoneRef.current;

    if (!stage || !phone) {
      return;
    }

    const stageRect = stage.getBoundingClientRect();
    const originX = stageRect.width / 2;
    const originY = stageRect.height / 2;
    const phoneRadius = Math.min(phone.offsetWidth, phone.offsetHeight) * PHONE_RADIUS_RATIO;

    setStageSize({ width: stageRect.width, height: stageRect.height });

    const nextLines: LinkLine[] = [];

    for (const node of ORBIT_NODES) {
      const card = cardRefs.current[node.id];
      if (!card) {
        continue;
      }

      // Resting center from layout %, size from layout box (ignores float transform).
      const targetX = (node.x / 100) * stageRect.width;
      const targetY = (node.y / 100) * stageRect.height;
      const halfW = card.offsetWidth / 2;
      const halfH = card.offsetHeight / 2;
      const distance = Math.hypot(targetX - originX, targetY - originY) || 1;
      const start = {
        x: originX + ((targetX - originX) / distance) * phoneRadius,
        y: originY + ((targetY - originY) / distance) * phoneRadius
      };
      const end = intersectRectEdge(
        originX,
        originY,
        targetX,
        targetY,
        targetX - halfW + CARD_INSET,
        targetY - halfH + CARD_INSET,
        targetX + halfW - CARD_INSET,
        targetY + halfH - CARD_INSET
      );

      nextLines.push({
        id: node.id,
        x1: start.x,
        y1: start.y,
        x2: end.x,
        y2: end.y
      });
    }

    setLines(nextLines);
  }, []);

  useLayoutEffect(() => {
    updateLines();
    const frame = window.requestAnimationFrame(() => updateLines());

    const stage = stageRef.current;
    if (!stage) {
      return () => window.cancelAnimationFrame(frame);
    }

    const observer = new ResizeObserver(() => {
      updateLines();
    });

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

  const handleMouseMove = (event: MouseEvent<HTMLDivElement>) => {
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
          x: offset.x * depth * 0.3,
          y: offset.y * depth * 0.3
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
        {Array.from({ length: 12 }, (_, index) => (
          <span key={index} className={styles.particle} style={{ "--i": index } as CSSProperties} />
        ))}
      </div>

      <svg
        className={styles.links}
        width={stageSize.width || "100%"}
        height={stageSize.height || "100%"}
        viewBox={`0 0 ${Math.max(stageSize.width, 1)} ${Math.max(stageSize.height, 1)}`}
      >
        {lines.map((line) => (
          <g key={line.id}>
            <line className={styles.linkPath} x1={line.x1} y1={line.y1} x2={line.x2} y2={line.y2} />
            <circle className={styles.linkDot} cx={line.x2} cy={line.y2} r="2.4" />
          </g>
        ))}
      </svg>

      <div className={styles.phoneAnchor}>
        <motion.div
          className={styles.phoneWrap}
          animate={layer(10)}
          transition={{ type: "spring", stiffness: 80, damping: 22 }}
        >
          <motion.div
            ref={phoneRef}
            className={styles.phone}
            animate={reduceMotion ? undefined : { y: [0, -4, 0] }}
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
      </div>

      {ORBIT_NODES.map((node) => (
        <div
          key={node.id}
          className={styles.cardAnchor}
          style={{ left: `${node.x}%`, top: `${node.y}%` }}
        >
          <motion.div
            className={styles.cardParallax}
            animate={layer(node.depth)}
            transition={{ type: "spring", stiffness: 70, damping: 22 }}
          >
            <motion.article
              ref={(element) => {
                cardRefs.current[node.id] = element;
              }}
              className={`${styles.card} ${styles[`card_${node.id}`]}`}
              animate={
                reduceMotion
                  ? undefined
                  : { y: [0, -3, 0], rotate: [node.rotate, node.rotate + 0.6, node.rotate] }
              }
              transition={floatTransition(node.delay)}
            >
              <header>
                {node.icon}
                <span>{node.label}</span>
              </header>
              <NodeBody id={node.id} isVisible={isVisible} />
            </motion.article>
          </motion.div>
        </div>
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
