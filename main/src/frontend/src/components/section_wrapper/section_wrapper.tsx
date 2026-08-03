import type { ReactNode } from "react";

import { Container } from "@/components/container/container";
import { joinClassNames } from "@utils/class_names";

import styles from "./section_wrapper.module.css";

interface SectionWrapperProps {
  children: ReactNode;
  id?: string;
  className?: string;
  containerClassName?: string;
  size?: "default" | "narrow";
}

export const SectionWrapper = ({
  children,
  id,
  className,
  containerClassName,
  size = "default"
}: SectionWrapperProps) => (
  <section id={id} className={joinClassNames(styles.section, className)}>
    <Container size={size} className={containerClassName}>
      {children}
    </Container>
  </section>
);
