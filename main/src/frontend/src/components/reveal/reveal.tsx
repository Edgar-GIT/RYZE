import { motion, useReducedMotion } from "framer-motion";
import type { ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

interface RevealProps {
  children: ReactNode;
  className?: string;
  delay?: number;
  y?: number;
}

const fadeUp = (y: number) => ({
  hidden: { opacity: 0, y },
  visible: { opacity: 1, y: 0 }
});

export const Reveal = ({ children, className, delay = 0, y = 28 }: RevealProps) => {
  const reduceMotion = useReducedMotion();

  return (
    <motion.div
      className={joinClassNames(className)}
      initial={reduceMotion ? false : "hidden"}
      whileInView="visible"
      viewport={{ once: true, amount: 0.22 }}
      variants={fadeUp(y)}
      transition={{ duration: 0.55, ease: "easeOut", delay }}
    >
      {children}
    </motion.div>
  );
};
