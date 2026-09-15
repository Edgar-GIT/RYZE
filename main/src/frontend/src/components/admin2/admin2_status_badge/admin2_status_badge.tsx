import { joinClassNames } from "@utils/class_names";

import styles from "./admin2_status_badge.module.css";

export type Admin2BadgeTone = "success" | "neutral" | "warning" | "danger";

interface Admin2StatusBadgeProps {
  label: string;
  tone?: Admin2BadgeTone;
  withDot?: boolean;
  className?: string;
}

export const Admin2StatusBadge = ({
  label,
  tone = "neutral",
  withDot = true,
  className
}: Admin2StatusBadgeProps) => (
  <span className={joinClassNames(styles.badge, styles[tone], className)}>
    {withDot ? <span className={styles.dot} aria-hidden="true" /> : null}
    {label}
  </span>
);