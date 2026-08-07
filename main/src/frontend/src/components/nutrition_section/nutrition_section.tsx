import { motion, useReducedMotion } from "framer-motion";
import { Apple, Clock3, Droplets, Leaf, Sparkles } from "lucide-react";

import { Container } from "@/components/container/container";
import { Reveal } from "@/components/reveal/reveal";

import styles from "./nutrition_section.module.css";

const featureItems = [
  {
    icon: Apple,
    title: "Daily macros",
    description: "Protein, carbs and fats tracked clearly."
  },
  {
    icon: Clock3,
    title: "Meal timing",
    description: "Fuel timed around your training."
  },
  {
    icon: Sparkles,
    title: "Smart guidance",
    description: "Targets that adapt with your week."
  },
  {
    icon: Leaf,
    title: "Simple targets",
    description: "Clear numbers — no diet noise."
  }
] as const;

const macros = [
  { label: "Protein", value: "145g", target: "180g", width: "80%" },
  { label: "Carbs", value: "210g", target: "240g", width: "86%" },
  { label: "Fat", value: "52g", target: "70g", width: "64%" }
] as const;

const floatTransition = {
  duration: 5.8,
  repeat: Infinity,
  repeatType: "mirror" as const,
  ease: "easeInOut" as const
};

export const NutritionSection = () => {
  const reduceMotion = useReducedMotion();

  return (
    <section className={styles.section} aria-labelledby="nutrition-heading">
      <div className={styles.atmosphere} aria-hidden="true">
        <div className={styles.nebula} />
        <div className={styles.grid} />
        <div className={styles.stars}>
          {Array.from({ length: 16 }, (_, index) => (
            <span key={index} style={{ ["--i" as string]: index }} />
          ))}
        </div>
      </div>

      <Container className={styles.layout}>
        <Reveal className={styles.copy}>
          <p className={styles.eyebrow}>Nutrition</p>
          <h2 id="nutrition-heading">
            Fuel that moves
            <br />
            with your <span className={styles.accentPhrase}>program.</span>
          </h2>
          <p className={styles.description}>
            Macros, timing and guidance that adapt to the training week — not a static PDF.
          </p>

          <ul className={styles.features}>
            {featureItems.map((item, index) => {
              const Icon = item.icon;

              return (
                <Reveal
                  key={item.title}
                  className={styles.feature}
                  delay={0.1 + index * 0.07}
                  y={18}
                >
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

        <Reveal className={styles.visual} delay={0.14} y={32}>
          <motion.article
            className={styles.card}
            animate={reduceMotion ? undefined : { y: [0, -8, 0] }}
            transition={floatTransition}
          >
            <header className={styles.cardHeader}>
              <Droplets aria-hidden="true" />
              <span>Nutrition · Today</span>
            </header>

            <div className={styles.cardBody}>
              <div className={styles.calorieRing} aria-hidden="true">
                <svg viewBox="0 0 120 120">
                  <circle cx="60" cy="60" r="48" className={styles.ringTrack} />
                  <circle cx="60" cy="60" r="48" className={styles.ringValue} />
                </svg>
                <div className={styles.ringLabel}>
                  <strong>1.840</strong>
                  <span>/ 2.400 kcal</span>
                </div>
              </div>

              <div className={styles.macroList}>
                {macros.map((macro) => (
                  <div key={macro.label} className={styles.macroRow}>
                    <div className={styles.macroMeta}>
                      <span>{macro.label}</span>
                      <strong>
                        {macro.value}
                        <em> / {macro.target}</em>
                      </strong>
                    </div>
                    <div className={styles.progressTrack}>
                      <span
                        className={styles.progressFill}
                        style={{ width: macro.width }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className={styles.recommendation}>
              <span>Recommendation</span>
              <p>
                Great! 260 calories remaining to hit your daily goal. 20g protein is a solid next
                meal target.
              </p>
            </div>
          </motion.article>
        </Reveal>
      </Container>
    </section>
  );
};
