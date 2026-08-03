import { ArrowRight, CheckCircle2 } from "lucide-react";

import { Button } from "@/components/button/button";
import { Card } from "@/components/card/card";
import type { ServicePlan } from "@/constants/services";

import styles from "./service_card.module.css";

interface ServiceCardProps {
  service: ServicePlan;
}

export const ServiceCard = ({ service }: ServiceCardProps) => (
  <Card className={styles.card}>
    <div className={styles.media}>
      <img src={service.imageSrc} alt={service.imageAlt} loading="lazy" />
      <span>{service.badge}</span>
    </div>
    <div className={styles.content}>
      <h3>{service.title}</h3>
      <p>{service.description}</p>
    </div>
    <ul className={styles.features} aria-label={`${service.title} features`}>
      {service.features.map((feature) => (
        <li key={feature}>
          <CheckCircle2 aria-hidden="true" />
          <span>{feature}</span>
        </li>
      ))}
    </ul>
    <Button className={styles.button} to={service.to} variant="secondary" icon={<ArrowRight />}>
      {service.ctaLabel}
    </Button>
  </Card>
);
