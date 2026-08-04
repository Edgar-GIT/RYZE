import { ArrowRight } from "lucide-react";
import { Link } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { EcosystemVisual } from "@/components/platform_section/ecosystem_visual";
import { Reveal } from "@/components/reveal/reveal";

import styles from "./platform_section.module.css";

export const PlatformSection = () => (
  <section className={styles.section} aria-labelledby="platform-heading">
    <Container className={styles.layout}>
      <Reveal className={styles.copy}>
        <p className={styles.eyebrow}>Start Faster</p>
        <h2 id="platform-heading">Get a plan that fits your week, without waiting.</h2>
        <p className={styles.description}>
          Answer a few questions. RYZE builds your plan. Everything lands on your phone.
        </p>
        <div className={styles.actions}>
          <Button to="/services" icon={<ArrowRight />}>
            Choose your plan
          </Button>
          <Link className={styles.secondaryLink} to="/our-vision">
            See how it works
            <ArrowRight aria-hidden="true" />
          </Link>
        </div>
      </Reveal>

      <Reveal className={styles.visual} delay={0.12} y={36}>
        <EcosystemVisual />
      </Reveal>
    </Container>
  </section>
);
