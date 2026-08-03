import { lazy, Suspense } from "react";
import { ArrowRight, Sparkles } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";

import styles from "./hero.module.css";

const HeroScene = lazy(() =>
  import("@/components/hero_scene/hero_scene").then((module) => ({
    default: module.HeroScene
  }))
);

export const Hero = () => (
  <section className={styles.hero}>
    <Container className={styles.inner}>
      <div className={styles.copy}>
        <p className={styles.eyebrow}>
          <Sparkles aria-hidden="true" />
          Premium online fitness guidance
        </p>
        <h1>Fitness guidance with the precision of modern software.</h1>
        <p className={styles.description}>
          RYZE turns training and nutrition into a structured digital
          experience, built for people who want clarity, momentum and a premium
          path into professional fitness.
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
          <span>Mobile-first plans</span>
          <span>Trainer-ready platform</span>
          <span>Automation foundation</span>
        </div>
      </div>

      <div className={styles.visual}>
        <Suspense fallback={<div className={styles.sceneFallback} />}>
          <HeroScene />
        </Suspense>
      </div>
    </Container>
  </section>
);
