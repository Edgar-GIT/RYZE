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
    initial={{ opacity: 0 }}
    animate={{ opacity: 1 }}
    exit={{ opacity: 0 }}
    transition={{ duration: 0.22, ease: "easeOut" }}
  >
    {children}
  </motion.div>
);
