import { ArrowRight, Dumbbell, Flame, Gift } from "lucide-react";
import { Link } from "react-router-dom";
import { motion, useReducedMotion } from "framer-motion";

import { Container } from "@/components/container/container";
import { Reveal } from "@/components/reveal/reveal";

import advert1 from "@resources/img/program_cards/advert1.jpeg";
import advert2 from "@resources/img/program_cards/advert2.jpeg";
import advert3 from "@resources/img/program_cards/advert3.jpeg";

import styles from "./programs_overview_section.module.css";

const programs = [
  {
    title: "BUILD MUSCLE",
    description:
      "Hypertrophy programs built around progressive overload. Structured volume, smart periodization and weekly progression that help you grow stronger, block after block.",
    ctaLabel: "EXPLORE",
    to: "/services/premium-level-1",
    accent: "blue",
    image: advert1,
    Icon: Dumbbell
  },
  {
    title: "LOSE FAT",
    description:
      "Structured training programs for fat loss and conditioning. Calibrated intensity, metabolic circuits and recovery-aware scheduling that keep you cutting without burning out.",
    ctaLabel: "EXPLORE",
    to: "/services/premium-level-2",
    accent: "orange",
    image: advert2,
    Icon: Flame
  },
  {
    title: "TRAIN FOR FREE",
    description:
      "A complete, ready-made training plan built on the fundamentals. No questionnaire, no subscription and no catch — start training today, completely free.",
    ctaLabel: "START FREE",
    to: "/services/generic-program",
    accent: "teal",
    image: advert3,
    Icon: Gift
  }
] as const;

export const ProgramsOverviewSection = () => {
  const reduceMotion = useReducedMotion();

  return (
    <section className={styles.section}>
      <Container className={styles.inner}>
        <Reveal className={styles.header}>
          <span className={styles.accentRule} aria-hidden="true" />
          <h2>
            Find <span className={styles.accentWord}>your</span> next program
          </h2>
          <p className={styles.subheading}>Choose your goal and start training.</p>
        </Reveal>

        <div className={styles.cardGrid}>
          {programs.map((program, index) => {
            const Icon = program.Icon;

            return (
              <Reveal key={program.title} delay={index * 0.1}>
                <motion.article
                  className={`${styles.card} ${styles[program.accent]}`}
                  whileHover={reduceMotion ? undefined : { y: -5 }}
                  transition={{ duration: 0.28, ease: "easeOut" }}
                >
                  <img
                    className={styles.cardImage}
                    src={program.image}
                    alt=""
                    loading="lazy"
                  />
                  <div className={styles.cardOverlay} aria-hidden="true" />
                  <div className={styles.cardContent}>
                    <span className={styles.iconBadge}>
                      <Icon aria-hidden="true" strokeWidth={1.8} />
                    </span>
                    <h3>{program.title}</h3>
                    <p className={styles.cardDescription}>{program.description}</p>
                    <Link to={program.to} className={styles.cardLink}>
                      <span>{program.ctaLabel}</span>
                      <ArrowRight aria-hidden="true" strokeWidth={2.2} />
                    </Link>
                  </div>
                </motion.article>
              </Reveal>
            );
          })}
        </div>
      </Container>
    </section>
  );
};