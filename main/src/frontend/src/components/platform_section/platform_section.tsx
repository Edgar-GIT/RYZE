import { ArrowRight, Brain, CalendarDays, HelpCircle, TrendingUp } from "lucide-react";
import { Link } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { Reveal } from "@/components/reveal/reveal";
import startSectionBackground from "@resources/img/hero/start_sec_bg.png";

import styles from "./platform_section.module.css";

const processSteps = [
  { icon: HelpCircle, label: "Answer a few questions" },
  { icon: Brain, label: "AI builds your plan" },
  { icon: CalendarDays, label: "Training, nutrition & recovery" },
  { icon: TrendingUp, label: "Track progress" }
] as const;

export const PlatformSection = () => (
  <section className={styles.section} aria-labelledby="platform-heading">
    <img
      className={styles.backgroundImage}
      src={startSectionBackground}
      alt=""
      aria-hidden="true"
    />

    <Container className={styles.layout}>
      <Reveal className={styles.copy}>
        <p className={styles.eyebrow}>Start Faster</p>
        <h2 id="platform-heading">
          Get a plan
          <br />
          that fits your week,
          <br />
          <span className={styles.accentPhrase}>without waiting.</span>
        </h2>
        <p className={styles.description}>
          Answer a few questions.
          <br />
          RYZE builds your plan.
          <br />
          Everything lands on your phone.
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

        <ol className={styles.process}>
          {processSteps.map((step, index) => {
            const Icon = step.icon;

            return (
              <li key={step.label} className={styles.processStep}>
                <span className={styles.processIcon}>
                  <Icon aria-hidden="true" />
                </span>
                <span className={styles.processLabel}>{step.label}</span>
                {index < processSteps.length - 1 ? (
                  <span className={styles.processConnector} aria-hidden="true">
                    <i />
                    <ArrowRight />
                  </span>
                ) : null}
              </li>
            );
          })}
        </ol>
      </Reveal>
    </Container>
  </section>
);
