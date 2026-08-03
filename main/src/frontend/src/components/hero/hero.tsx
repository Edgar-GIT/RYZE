import { ArrowRight, Sparkles } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import heroShowcase from "@resources/img/testing/test4.png";

import styles from "./hero.module.css";

export const Hero = () => (
  <section className={styles.hero}>
    <Container className={styles.inner}>
      <div className={styles.copy}>
        <p className={styles.eyebrow}>
          <Sparkles aria-hidden="true" />
          AI powered coaching platform
        </p>
        <h1>Train smarter. Evolve every week.</h1>
        <p className={styles.description}>
          RYZE combines professional training, nutrition guidance and progress
          intelligence in one premium fitness experience built for real
          consistency.
        </p>
        <div className={styles.actions}>
          <Button to="/services/generic-plan" size="large" icon={<ArrowRight />}>
            Join for Free
          </Button>
          <Button to="/services" size="large" variant="secondary">
            Get Started
          </Button>
        </div>
        <div className={styles.signals} aria-label="RYZE platform strengths">
          <span>Training plans</span>
          <span>Nutrition guidance</span>
          <span>Progress tracking</span>
        </div>
      </div>

      <div className={styles.visual}>
        <div className={styles.imageStage} aria-label="RYZE visual training showcase">
          <div className={styles.imageHalo} aria-hidden="true" />
          <img
            className={styles.heroImage}
            src={heroShowcase}
            alt="RYZE training and fitness visual showcase"
          />
        </div>
      </div>
    </Container>
  </section>
);
