import { motion } from "framer-motion";

import { BRAND_ASSETS } from "@/constants/brand_assets";

import styles from "./loading_screen.module.css";

export const LoadingScreen = () => (
  <motion.div
    className={styles.overlay}
    role="status"
    aria-live="polite"
    initial={{ opacity: 1 }}
    animate={{ opacity: 1 }}
    exit={{ opacity: 0 }}
    transition={{ duration: 0.38, ease: "easeOut" }}
  >
    <div className={styles.loader}>
      <span className={styles.arrowOuter} aria-hidden="true" />
      <span className={styles.arrowInner} aria-hidden="true" />
      <img className={styles.logo} src={BRAND_ASSETS.icon} alt="RYZE" />
    </div>
    <p className={styles.label}>Preparing RYZE</p>
  </motion.div>
);
