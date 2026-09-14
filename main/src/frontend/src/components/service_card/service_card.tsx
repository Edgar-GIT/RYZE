import { ArrowRight } from "lucide-react";

import { Button } from "@/components/button/button";
import type { ServiceProgram } from "@/constants/services";
import { joinClassNames } from "@utils/class_names";

import styles from "./service_card.module.css";

interface ServiceCardProps {
  service: ServiceProgram;
}

export const ServiceCard = ({ service }: ServiceCardProps) => (
  <article className={joinClassNames(styles.card, service.featured && styles.featured)}>
    <img
      className={styles.background}
      src={service.imageSrc}
      alt=""
      aria-hidden="true"
    />
    <span className={styles.overlay} aria-hidden="true" />

    {service.featured && <span className={styles.mostPopular}>Most Popular</span>}

    <div className={styles.footer}>
      <p className={styles.price}>{service.price}</p>
      <Button
        className={styles.button}
        to={service.to}
        variant={service.featured ? "primary" : "secondary"}
        icon={<ArrowRight />}
      >
        {service.ctaLabel}
      </Button>
    </div>
  </article>
);