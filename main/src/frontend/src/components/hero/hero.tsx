import { ArrowRight, Brain, Users } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import heroShowcase from "@resources/img/hero/hero.png";

import styles from "./hero.module.css";

const trustItems = [
  { label: "Trusted by athletes", icon: Users },
  { label: "AI adapts to you", icon: Brain }
] as const;

export const Hero = () => (
  <section className={styles.hero} aria-label="RYZE introduction">
    <img
      className={styles.heroImage}
      src={heroShowcase}
      alt="RYZE platform mark with training, nutrition and progress insights"
    />

    <Container className={styles.content}>
      <div className={styles.copy}>
        <p className={styles.eyebrow}>
          <span className={styles.eyebrowDot} aria-hidden="true" />
          AI-powered training and nutrition platform
        </p>
        <h1>
          Train smarter.
          <br />
          Fuel better.
          <br />
          <span className={styles.accentLine}>Evolve always.</span>
        </h1>
        <p className={styles.description}>
          Personalized training, nutrition and recovery — powered by AI and
          guided by real coaches.
        </p>
        <div className={styles.actions}>
          <Button to="/services/generic-program" size="large" icon={<ArrowRight />}>
            Start Free
          </Button>
          <Button to="/services" size="large" variant="secondary">
            Explore Platform
          </Button>
        </div>
        <ul className={styles.trust}>
          {trustItems.map((item) => {
            const Icon = item.icon;

            return (
              <li key={item.label}>
                <Icon aria-hidden="true" />
                <span>{item.label}</span>
              </li>
            );
          })}
        </ul>
      </div>
    </Container>
  </section>
);
