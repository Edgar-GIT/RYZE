import type { LucideIcon } from "lucide-react";

import { joinClassNames } from "@utils/class_names";

import styles from "./admin_placeholder.module.css";

interface AdminPlaceholderPageProps {
  eyebrow: string;
  title: string;
  description: string;
  icon: LucideIcon;
  className?: string;
}

export const AdminPlaceholderPage = ({
  eyebrow,
  title,
  description,
  icon: Icon,
  className
}: AdminPlaceholderPageProps) => (
  <div className={joinClassNames(styles.placeholderPage, className)}>
    <header className={styles.header}>
      <p className={styles.eyebrow}>{eyebrow}</p>
      <h1 className={styles.title}>{title}</h1>
      <p className={styles.description}>{description}</p>
    </header>
    <div className={styles.card} role="status">
      <Icon className={styles.cardIcon} aria-hidden="true" strokeWidth={1.4} />
      <h2 className={styles.cardTitle}>Not yet implemented</h2>
      <p className={styles.cardDescription}>
        This section will receive live data and controls once the corresponding
        backend services are available.
      </p>
    </div>
  </div>
);