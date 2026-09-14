import { ChevronLeft, ChevronRight } from "lucide-react";

import { joinClassNames } from "@utils/class_names";

import styles from "./carousel_arrow.module.css";

interface CarouselArrowProps {
  onClick: () => void;
  disabled?: boolean;
  ariaLabel: string;
  direction?: "back" | "forward";
}

export const CarouselArrow = ({
  onClick,
  disabled = false,
  ariaLabel,
  direction = "forward"
}: CarouselArrowProps) => (
  <button
    type="button"
    className={joinClassNames(styles.arrow, disabled && styles.disabled)}
    onClick={onClick}
    disabled={disabled}
    aria-label={ariaLabel}
  >
    {direction === "back" ? (
      <ChevronLeft strokeWidth={2} aria-hidden="true" />
    ) : (
      <ChevronRight strokeWidth={2} aria-hidden="true" />
    )}
  </button>
);