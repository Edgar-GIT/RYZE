import { joinClassNames } from "@utils/class_names";

import styles from "./showcase.module.css";

interface ScreenProps {
  src: string;
  alt: string;
  className?: string;
}

// PhoneMockup frames a RYZE app screen in a titanium phone with the same blue
// edge light as the desktop 3D phone. Used wherever the 3D scene is not loaded.
export const PhoneMockup = ({ src, alt, className }: ScreenProps) => (
  <figure className={joinClassNames(styles.phone, className)}>
    <div className={styles.phoneScreen}>
      <img src={src} alt={alt} width="780" height="1688" loading="lazy" decoding="async" />
      <span className={styles.island} aria-hidden="true" />
    </div>
  </figure>
);

export const LaptopMockup = ({ src, alt, className }: ScreenProps) => (
  <figure className={joinClassNames(styles.laptop, className)}>
    <div className={styles.lid}>
      <img src={src} alt={alt} width="1600" height="1000" loading="lazy" decoding="async" />
    </div>
    <div className={styles.base} aria-hidden="true" />
  </figure>
);
