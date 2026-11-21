import type { MuscleName, MuscleRecovery } from "@/services/body_stats_api";
import { joinClassNames } from "@utils/class_names";

import { BACK, BACK_LINES, FRONT, FRONT_LINES, SILHOUETTE, type DetailLine, type MuscleShape } from "./muscle_map_shapes";
import styles from "./muscle_map.module.css";

const MIRROR = "matrix(-1 0 0 1 200 0)";

interface MuscleMapProps {
  muscles: MuscleRecovery[];
  /** Accessible summary of the picture; the list beside it carries the data. */
  label: string;
  frontLabel: string;
  backLabel: string;
  active: MuscleName | null;
  onSelect: (muscle: MuscleName) => void;
}

// MuscleMap paints the front and back of a body by each muscle group's
// recovery (owner request, 2026-11-20, as in Lyfta). Clicking a muscle
// selects it in the list; the list is the keyboard and screen reader route.
export const MuscleMap = ({ muscles, label, frontLabel, backLabel, active, onSelect }: MuscleMapProps) => {
  const byName = new Map(muscles.map((muscle) => [muscle.muscle, muscle]));

  const figure = (shapes: MuscleShape[], lines: DetailLine[], x: number, caption: string) => (
    <g transform={`translate(${x} 0)`}>
      <path className={styles.body} d={SILHOUETTE} />
      <path className={styles.body} d={SILHOUETTE} transform={MIRROR} />
      <ellipse className={styles.body} cx="100" cy="34" rx="20" ry="25" />
      {shapes.map((shape, index) => {
        const status = byName.get(shape.muscle)?.status ?? "ready";
        const className = joinClassNames(styles.muscle, styles[status], active === shape.muscle && styles.active);
        return (
          <g key={`${shape.muscle}-${index}`} className={styles.region} onClick={() => onSelect(shape.muscle)}>
            <path className={className} d={shape.d} />
            {shape.mirror ? <path className={className} d={shape.d} transform={MIRROR} /> : null}
          </g>
        );
      })}
      {lines.map((line, index) => (
        <g key={index} className={styles.detail}>
          <path d={line.d} />
          {line.mirror ? <path d={line.d} transform={MIRROR} /> : null}
        </g>
      ))}
      <text className={styles.caption} x="100" y="436" textAnchor="middle">
        {caption}
      </text>
    </g>
  );

  return (
    <svg className={styles.map} viewBox="0 0 420 445" role="img" aria-label={label}>
      {figure(FRONT, FRONT_LINES, 0, frontLabel)}
      {figure(BACK, BACK_LINES, 220, backLabel)}
    </svg>
  );
};
