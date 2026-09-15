import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./admin2_empty_state.module.css";

interface Admin2EmptyStateProps {
  icon: LucideIcon;
  title: string;
  message: string;
  action?: ReactNode;
  className?: string;
}

export const Admin2EmptyState = ({
  icon: Icon,
  title,
  message,
  action,
  className
}: Admin2EmptyStateProps) => (
  <div className={joinClassNames(styles.empty, className)}>
    <Icon size={34} className={styles.icon} aria-hidden="true" strokeWidth={1.5} />
    <p className={styles.title}>{title}</p>
    <p className={styles.message}>{message}</p>
    {action ? <div className={styles.action}>{action}</div> : null}
  </div>
);