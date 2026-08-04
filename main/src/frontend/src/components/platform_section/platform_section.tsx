import { ArrowRight } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { EcosystemVisual } from "@/components/platform_section/ecosystem_visual";

import styles from "./platform_section.module.css";

export const PlatformSection = () => (
  <section className={styles.section} aria-labelledby="platform-heading">
    <Container className={styles.layout}>
      <div className={styles.copy}>
        <p className={styles.eyebrow}>Start Faster</p>
        <h2 id="platform-heading">Get a plan that fits your week, without waiting.</h2>
        <p className={styles.description}>
          RYZE helps you start with structure right away. Answer a few questions, get a clear
          training direction, follow it on your phone, and keep nutrition and recovery in sync
          without depending on gym schedules or trainer availability.
        </p>
        <Button to="/services" icon={<ArrowRight />}>
          Choose your plan
        </Button>
      </div>

      <div className={styles.visual}>
        <EcosystemVisual />
      </div>
    </Container>
  </section>
);
