import { ArrowRight, Crown, Dumbbell, UserCheck } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import heroShowcase from "@resources/img/hero/hero.png";

import styles from "./hero.module.css";

const features = [
  { label: "Free Programs", icon: Dumbbell },
  { label: "Premium Plans", icon: Crown },
  { label: "Personalized\nCoaching", icon: UserCheck }
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
          <span className={styles.eyebrowLine} aria-hidden="true" />
          THE FUTURE OF FITNESS
        </p>
        <h1>
          Train <span className={styles.accent}>smarter.</span>
          <br />
          Progress <span className={styles.accent}>further.</span>
        </h1>
        <p className={styles.description}>
          Free training plans, premium programs and personalized coaching — all
          in one platform.
        </p>
        <div className={styles.actions}>
          <Button
            to="/services/generic-program"
            size="large"
            icon={<ArrowRight />}
          >
            START TRAINING FREE
          </Button>
          <Button
            to="/services"
            size="large"
            variant="secondary"
            icon={<ArrowRight />}
            className={styles.heroSecondary}
          >
            EXPLORE PLANS
          </Button>
        </div>
        <ul className={styles.features}>
          {features.map((item) => {
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
