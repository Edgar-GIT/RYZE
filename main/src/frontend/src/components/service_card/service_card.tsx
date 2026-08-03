import { ArrowRight, Check } from "lucide-react";

import { Button } from "@/components/button/button";
import type { ServicePlan } from "@/constants/services";

import styles from "./service_card.module.css";

interface ServiceCardProps {
  service: ServicePlan;
}

export const ServiceCard = ({ service }: ServiceCardProps) => (
  <article className={styles.card}>
    <div className={styles.media}>
      <img src={service.imageSrc} alt={service.imageAlt} loading="lazy" />
      <span className={styles.badge}>{service.badge}</span>
    </div>

    <div className={styles.body}>
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

      <Button className={styles.button} to={service.to} variant="secondary" icon={<ArrowRight />}>
        {service.ctaLabel}
      </Button>
    </div>
  </article>
);
