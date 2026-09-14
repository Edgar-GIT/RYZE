import { Download, Heart } from "lucide-react";

import { BRAND_ASSETS } from "@/constants/brand_assets";

import styles from "./program_card.module.css";

interface ProgramCardProps {
  title: string;
}

export const ProgramCard = ({ title }: ProgramCardProps) => (
  <article className={styles.card}>
    <div className={styles.image} aria-hidden="true">
      <img className={styles.logo} src={BRAND_ASSETS.icon} alt="" />
    </div>
    <div className={styles.body}>
      <h3 className={styles.title}>{title}</h3>
      <div className={styles.meta}>
        <span className={styles.metaItem}>
          <Download className={styles.metaIcon} strokeWidth={1.8} aria-hidden="true" />
          <span>0 downloads</span>
        </span>
        <span className={styles.metaItem}>
          <Heart className={styles.metaIcon} strokeWidth={1.8} aria-hidden="true" />
          <span>0/5</span>
        </span>
      </div>
    </div>
  </article>
);