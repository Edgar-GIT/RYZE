import { motion } from "framer-motion";
import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./page_wrapper.module.css";

interface PageWrapperProps {
  children: ReactNode;
  className?: string;
}

export const PageWrapper = ({ children, className }: PageWrapperProps) => (
  <motion.div
    className={joinClassNames(styles.page, className)}
    initial={{ opacity: 0, y: 14 }}
    animate={{ opacity: 1, y: 0 }}
    exit={{ opacity: 0, y: -10 }}
    transition={{ duration: 0.26, ease: "easeOut" }}
  >
    {children}
  </motion.div>
);
