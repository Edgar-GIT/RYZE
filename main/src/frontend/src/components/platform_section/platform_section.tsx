import { ArrowRight } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { EcosystemVisual } from "@/components/platform_section/ecosystem_visual";

import styles from "./platform_section.module.css";

export const PlatformSection = () => (
  <section className={styles.section} aria-labelledby="platform-heading">
    <Container className={styles.layout}>
      <div className={styles.copy}>
        <p className={styles.eyebrow}>Platform</p>
        <h2 id="platform-heading">Your fitness operating system.</h2>
        <p className={styles.description}>
          One ecosystem. Training, nutrition, recovery, AI and coach — connected so every signal
          shapes the next decision.
        </p>
        <Button to="/services" icon={<ArrowRight />}>
          See how it works
        </Button>
      </div>

      <div className={styles.visual}>
        <EcosystemVisual />
      </div>
    </Container>
  </section>
);
