import { ArrowRight, Brain, CalendarDays, HelpCircle, TrendingUp } from "lucide-react";
import { Link } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { EcosystemVisual } from "@/components/platform_section/ecosystem_visual";
import { Reveal } from "@/components/reveal/reveal";

import styles from "./platform_section.module.css";

const processSteps = [
  { icon: HelpCircle, label: "Answer a few questions" },
  { icon: Brain, label: "AI builds your plan" },
  { icon: CalendarDays, label: "Training, nutrition & recovery synced" },
  { icon: TrendingUp, label: "Track progress every week" }
] as const;

export const PlatformSection = () => (
  <section className={styles.section} aria-labelledby="platform-heading">
    <div className={styles.arc} aria-hidden="true">
      <svg viewBox="0 0 1200 520" preserveAspectRatio="none">
        <path
          className={styles.arcPath}
          d="M 80 430 C 320 120, 760 40, 1120 210"
          fill="none"
        />
      </svg>
    </div>

    <Container className={styles.layout}>
      <Reveal className={styles.copy}>
        <p className={styles.eyebrow}>Start Faster</p>
        <h2 id="platform-heading">
          Get a plan that fits your week,{" "}
          <span className={styles.accentPhrase}>without waiting.</span>
        </h2>
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
                  <ArrowRight className={styles.processArrow} aria-hidden="true" />
                ) : null}
              </li>
            );
          })}
        </ol>
      </Reveal>

      <Reveal className={styles.visual} delay={0.12} y={36}>
        <EcosystemVisual />
      </Reveal>
    </Container>
  </section>
);
