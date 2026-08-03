import { joinClassNames } from "@utils/class_names";

import styles from "./section_title.module.css";

interface SectionTitleProps {
  eyebrow?: string;
  title: string;
  description?: string;
  align?: "left" | "center";
  className?: string;
}

export const SectionTitle = ({
  eyebrow,
  title,
  description,
  align = "left",
  className
}: SectionTitleProps) => (
  <header className={joinClassNames(styles.header, styles[align], className)}>
    {eyebrow ? <p className={styles.eyebrow}>{eyebrow}</p> : null}
    <h2>{title}</h2>
    {description ? <p className={styles.description}>{description}</p> : null}
  </header>
);
