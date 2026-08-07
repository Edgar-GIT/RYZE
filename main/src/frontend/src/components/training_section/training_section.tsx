import { Activity, CalendarDays, CircleCheck, TrendingUp } from "lucide-react";

import { Container } from "@/components/container/container";
import { Reveal } from "@/components/reveal/reveal";
import trainingSectionBackground from "@resources/img/hero/training_sec_bg.png";

import styles from "./training_section.module.css";

const featureItems = [
  {
    icon: CalendarDays,
    title: "Weekly structure",
    description: "Your training organized by day."
  },
  {
    icon: Activity,
    title: "Live sessions",
    description: "Exercises, sets, reps and weights."
  },
  {
    icon: TrendingUp,
    title: "Track progress",
    description: "Consistency that drives results."
  },
  {
    icon: CircleCheck,
    title: "Always clear",
    description: "You know exactly what to do."
  }
] as const;

export const TrainingSection = () => (
  <section className={styles.section} aria-labelledby="training-heading">
    <img
      className={styles.backgroundImage}
      src={trainingSectionBackground}
      alt=""
      aria-hidden="true"
    />

    <Container className={styles.layout}>
      <Reveal className={styles.copy}>
        <p className={styles.eyebrow}>Training</p>
        <h2 id="training-heading">
          Sessions you can
          <br />
          actually <span className={styles.accentPhrase}>follow.</span>
        </h2>
        <p className={styles.description}>
          Structured weekly program, live exercises and clear progression without the chaos of
          spreadsheets.
        </p>

        <ul className={styles.features}>
          {featureItems.map((item, index) => {
            const Icon = item.icon;

            return (
              <Reveal key={item.title} className={styles.feature} delay={0.08 + index * 0.06} y={18}>
                <span className={styles.featureIcon}>
                  <Icon aria-hidden="true" />
                </span>
                <span className={styles.featureCopy}>
                  <strong>{item.title}</strong>
                  <span>{item.description}</span>
                </span>
              </Reveal>
            );
          })}
        </ul>
      </Reveal>
    </Container>
  </section>
);
