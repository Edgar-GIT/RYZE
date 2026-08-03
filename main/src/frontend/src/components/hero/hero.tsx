import {
  Activity,
  ArrowRight,
  BrainCircuit,
  CheckCircle2,
  Sparkles,
  TrendingUp,
  Utensils
} from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";

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
        <div className={styles.productStage} aria-label="RYZE product preview">
          <div className={styles.phoneMockup}>
            <div className={styles.phoneHeader}>
              <span>Today</span>
              <strong>Upper Body</strong>
            </div>
            <div className={styles.workoutProgress}>
              <span />
              <span />
              <span />
            </div>
            <div className={styles.exerciseList}>
              <div>
                <CheckCircle2 aria-hidden="true" />
                <span>Dumbbell press</span>
                <strong>4 x 10</strong>
              </div>
              <div>
                <CheckCircle2 aria-hidden="true" />
                <span>Row variation</span>
                <strong>3 x 12</strong>
              </div>
              <div>
                <Activity aria-hidden="true" />
                <span>Core finisher</span>
                <strong>8 min</strong>
              </div>
            </div>
          </div>

          <div className={styles.panelProgress}>
            <TrendingUp aria-hidden="true" />
            <span>Weekly progress</span>
            <strong>+18%</strong>
          </div>

          <div className={styles.panelNutrition}>
            <Utensils aria-hidden="true" />
            <span>Nutrition target</span>
            <strong>Ready for today</strong>
          </div>

          <div className={styles.panelAi}>
            <BrainCircuit aria-hidden="true" />
            <span>AI plan match</span>
            <strong>Goal aligned</strong>
          </div>
        </div>
      </div>
    </Container>
  </section>
);
