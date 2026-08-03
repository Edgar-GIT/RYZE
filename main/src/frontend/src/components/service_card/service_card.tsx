import { ArrowRight, Dumbbell, ShieldCheck, Utensils } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { Button } from "@/components/button/button";
import { Card } from "@/components/card/card";
import type { ServiceIcon, ServicePlan } from "@/constants/services";

import styles from "./service_card.module.css";

const serviceIcons: Record<ServiceIcon, LucideIcon> = {
  training: Dumbbell,
  nutrition: Utensils,
  personalized: ShieldCheck
};

interface ServiceCardProps {
  service: ServicePlan;
}

export const ServiceCard = ({ service }: ServiceCardProps) => {
  const Icon = serviceIcons[service.icon];

  return (
    <Card className={styles.card}>
      <div className={styles.iconWrap}>
        <Icon aria-hidden="true" />
      </div>
      <div className={styles.content}>
        <h3>{service.title}</h3>
        <p>{service.description}</p>
      </div>
      <ul className={styles.features} aria-label={`${service.title} features`}>
        {service.features.map((feature) => (
          <li key={feature}>{feature}</li>
        ))}
      </ul>
      <Button
        className={styles.button}
        to={service.to}
        variant="secondary"
        icon={<ArrowRight />}
      >
        {service.ctaLabel}
      </Button>
    </Card>
  );
};
