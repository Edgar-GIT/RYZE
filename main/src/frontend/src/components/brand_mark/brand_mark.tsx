import { BRAND_ASSETS } from "@/constants/brand_assets";
import { joinClassNames } from "@utils/class_names";

import styles from "./brand_mark.module.css";

type BrandMarkSize = "navigation" | "footer" | "loading";

interface BrandMarkProps {
  size?: BrandMarkSize;
  className?: string;
}

export const BrandMark = ({ size = "navigation", className }: BrandMarkProps) => (
  <span className={joinClassNames(styles.mark, styles[size], className)} aria-hidden="true">
    <img src={BRAND_ASSETS.icon} alt="" />
  </span>
);
