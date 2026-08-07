import { ArrowRight, Check } from "lucide-react";

import { Button } from "@/components/button/button";
import type { ServiceProgram } from "@/constants/services";
import { joinClassNames } from "@utils/class_names";

import styles from "./service_card.module.css";

interface ServiceCardProps {
  service: ServiceProgram;
}

export const ServiceCard = ({ service }: ServiceCardProps) => (
  <article className={joinClassNames(styles.card, service.featured && styles.featured)}>
    <header className={styles.header}>
      <img
        className={styles.logo}
        src={service.logoSrc}
        alt={service.logoAlt}
        loading="lazy"
      />

      <div className={styles.copy}>
        <span className={joinClassNames(styles.badge, service.featured && styles.badgeFeatured)}>
          {service.badge}
        </span>
        <h3>{service.title}</h3>
        <p>{service.description}</p>
      </div>
    </header>

    <ul className={styles.features} aria-label={`${service.title} features`}>
      {service.features.map((feature) => (
        <li key={feature}>
          <span className={styles.check}>
            <Check aria-hidden="true" strokeWidth={2.6} />
          </span>
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
