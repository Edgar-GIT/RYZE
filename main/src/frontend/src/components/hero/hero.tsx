import { ArrowRight, Sparkles } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { BRAND_ASSETS } from "@/constants/brand_assets";

import styles from "./hero.module.css";

export const Hero = () => (
  <section className={styles.hero}>
    <Container className={styles.inner}>
      <div className={styles.copy}>
        <p className={styles.eyebrow}>
          <Sparkles aria-hidden="true" />
          Online fitness guidance
        </p>
        <h1>Professional training starts with a clear plan.</h1>
        <p className={styles.description}>
          RYZE brings structured training and nutrition guidance into a clean
          digital experience built for beginners, intermediate users and future
          trainer-led growth.
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
          <span>Mobile first</span>
          <span>Structured plans</span>
          <span>Nutrition ready</span>
        </div>
      </div>

      <div className={styles.visual} aria-label="RYZE brand preview">
        <div className={styles.visualFrame}>
          <img src={BRAND_ASSETS.set} alt="RYZE brand set" />
        </div>
        <div className={styles.visualRail} aria-hidden="true">
          <span />
          <span />
          <span />
        </div>
      </div>
    </Container>
  </section>
);
