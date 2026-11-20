import { motion, useReducedMotion } from "framer-motion";
import type { ReactNode } from "react";

import { useMediaQuery } from "@/hooks/use_media_query";
import { joinClassNames } from "@utils/class_names";

// Phones and tablets show content as it is, with no entrance animation (faster, and
// nothing waits for a scroll position to appear).
const PHONE_QUERY = "(max-width: 1023px)";

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
  const phone = useMediaQuery(PHONE_QUERY);

  return (
    <motion.div
      className={joinClassNames(className)}
      initial={reduceMotion || phone ? false : "hidden"}
      whileInView="visible"
      viewport={{ once: true, amount: 0.22 }}
      variants={fadeUp(y)}
      transition={{ duration: 0.55, ease: "easeOut", delay }}
    >
      {children}
    </motion.div>
  );
};
