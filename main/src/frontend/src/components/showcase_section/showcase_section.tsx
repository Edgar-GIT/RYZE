import type { ReactNode } from "react";

import { SectionWrapper } from "@/components/section_wrapper/section_wrapper";
import { joinClassNames } from "@utils/class_names";

import styles from "./showcase_section.module.css";

interface ShowcaseSectionProps {
  eyebrow: string;
  title: string;
  description: string;
  children: ReactNode;
  reverse?: boolean;
  className?: string;
  tone?: "deep" | "surface" | "card";
}

export const ShowcaseSection = ({
  eyebrow,
  title,
  description,
  children,
  reverse = false,
  className,
  tone = "deep"
}: ShowcaseSectionProps) => (
  <SectionWrapper
    tone={tone}
    className={joinClassNames(styles.section, className)}
    containerClassName={joinClassNames(styles.inner, reverse && styles.reverse)}
  >
    <div className={styles.copy}>
      <p className={styles.eyebrow}>{eyebrow}</p>
      <h2>{title}</h2>
      <p className={styles.description}>{description}</p>
    </div>
    <div className={styles.visual}>{children}</div>
  </SectionWrapper>
);
