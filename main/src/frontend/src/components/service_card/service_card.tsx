import { ArrowRight, Check } from "lucide-react";

import { Button } from "@/components/button/button";
import type { ServicePlan } from "@/constants/services";
import { joinClassNames } from "@utils/class_names";

import styles from "./service_card.module.css";

interface ServiceCardProps {
  service: ServicePlan;
}

export const ServiceCard = ({ service }: ServiceCardProps) => (
  <article className={joinClassNames(styles.card, service.featured && styles.featured)}>
    <div className={styles.top}>
      <div className={styles.logoWrap}>
        <img src={service.imageSrc} alt={service.imageAlt} loading="lazy" />
      </div>
      <span className={joinClassNames(styles.badge, service.featured && styles.badgeFeatured)}>
        {service.badge}
      </span>
    </div>

    <div className={styles.copy}>
      <h3>{service.title}</h3>
      <p>{service.description}</p>
    </div>

    <ul className={styles.features} aria-label={`${service.title} features`}>
      {service.features.map((feature) => (
        <li key={feature}>
          <Check aria-hidden="true" strokeWidth={2.4} />
          <span>{feature}</span>
        </li>
      ))}
    </ul>

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
