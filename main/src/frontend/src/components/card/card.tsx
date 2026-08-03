import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./card.module.css";

interface CardProps {
  children: ReactNode;
  className?: string;
}

export const Card = ({ children, className }: CardProps) => (
  <article className={joinClassNames(styles.card, className)}>{children}</article>
);
