import { ArrowRight } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";

import styles from "./cta_band.module.css";

interface CtaBandProps {
  eyebrow: string;
  title: string;
  description: string;
  primaryLabel: string;
  primaryTo: string;
  secondaryLabel?: string;
  secondaryTo?: string;
}

export const CtaBand = ({
  eyebrow,
  title,
  description,
  primaryLabel,
  primaryTo,
  secondaryLabel,
  secondaryTo
}: CtaBandProps) => (
  <section className={styles.section}>
    <Container className={styles.band}>
      <div className={styles.copy}>
        <p>{eyebrow}</p>
        <h2>{title}</h2>
        <span>{description}</span>
      </div>
      <div className={styles.actions}>
        <Button to={primaryTo} variant="light" icon={<ArrowRight />}>
          {primaryLabel}
        </Button>
        {secondaryLabel && secondaryTo ? (
          <Button to={secondaryTo} variant="outlineLight">
            {secondaryLabel}
          </Button>
        ) : null}
      </div>
    </Container>
  </section>
);
