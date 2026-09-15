import type { LucideIcon } from "lucide-react";

import { joinClassNames } from "@utils/class_names";

import styles from "./admin2_metric_card.module.css";

interface Admin2MetricCardProps {
  label: string;
  value: string | number;
  hint?: string;
  icon: LucideIcon;
  className?: string;
}

export const Admin2MetricCard = ({
  label,
  value,
  hint,
  icon: Icon,
  className
}: Admin2MetricCardProps) => (
  <div className={joinClassNames(styles.card, className)}>
    <p className={styles.label}>
      <Icon size={15} className={styles.icon} aria-hidden="true" strokeWidth={1.8} />
      {label}
    </p>
    <p className={styles.value}>{value}</p>
    {hint ? <p className={styles.hint}>{hint}</p> : null}
  </div>
);