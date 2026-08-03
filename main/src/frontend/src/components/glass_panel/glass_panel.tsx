import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./glass_panel.module.css";

type GlassPanelElement = "div" | "article" | "section";

interface GlassPanelProps {
  children: ReactNode;
  as?: GlassPanelElement;
  className?: string;
}

export const GlassPanel = ({
  children,
  as: Component = "div",
  className
}: GlassPanelProps) => (
  <Component className={joinClassNames(styles.panel, className)}>{children}</Component>
);
