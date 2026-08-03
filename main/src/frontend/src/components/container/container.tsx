import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./container.module.css";

interface ContainerProps {
  children: ReactNode;
  className?: string;
  size?: "default" | "narrow";
}

export const Container = ({
  children,
  className,
  size = "default"
}: ContainerProps) => (
  <div className={joinClassNames(styles.container, styles[size], className)}>
    {children}
  </div>
);
