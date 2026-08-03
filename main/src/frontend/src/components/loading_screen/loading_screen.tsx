import { motion } from "framer-motion";

import { BrandMark } from "@/components/brand_mark/brand_mark";

import styles from "./loading_screen.module.css";

export const LoadingScreen = () => (
  <motion.div
    className={styles.overlay}
    role="status"
    aria-live="polite"
    initial={{ opacity: 0, scale: 0.98 }}
    animate={{ opacity: 1, scale: 1 }}
    exit={{ opacity: 0, scale: 1.02 }}
    transition={{ duration: 0.42, ease: "easeOut" }}
  >
    <div className={styles.loader}>
      <span className={styles.arrowOuter} aria-hidden="true" />
      <span className={styles.arrowInner} aria-hidden="true" />
      <BrandMark size="loading" />
    </div>
    <p className={styles.label}>Preparing RYZE</p>
  </motion.div>
);
