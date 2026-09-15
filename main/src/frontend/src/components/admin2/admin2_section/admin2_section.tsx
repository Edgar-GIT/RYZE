import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./admin2_section.module.css";

interface Admin2SectionProps {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}

export const Admin2Section = ({
  title,
  subtitle,
  actions,
  children,
  className
}: Admin2SectionProps) => (
  <section className={joinClassNames(styles.section, className)}>
    <div className={styles.header}>
      <div className={styles.titleBlock}>
        <h2 className={styles.title}>{title}</h2>
        {subtitle ? <p className={styles.subtitle}>{subtitle}</p> : null}
      </div>
      {actions ? <div className={styles.actions}>{actions}</div> : null}
    </div>
    <div className={styles.body}>{children}</div>
  </section>
);