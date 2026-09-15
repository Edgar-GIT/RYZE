import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./admin2_page_header.module.css";

interface Admin2PageHeaderProps {
  eyebrow?: string;
  title: string;
  description?: string;
  actions?: ReactNode;
  className?: string;
}

export const Admin2PageHeader = ({
  eyebrow,
  title,
  description,
  actions,
  className
}: Admin2PageHeaderProps) => (
  <div className={joinClassNames(styles.header, className)}>
    <div className={styles.heading}>
      {eyebrow ? <p className={styles.eyebrow}>{eyebrow}</p> : null}
      <h1 className={styles.title}>{title}</h1>
      {description ? <p className={styles.description}>{description}</p> : null}
    </div>
    {actions ? <div className={styles.actions}>{actions}</div> : null}
  </div>
);