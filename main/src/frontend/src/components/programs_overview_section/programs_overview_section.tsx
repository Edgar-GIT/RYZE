import { ArrowRight, BarChart3, Dumbbell, Flame, Gift, ShieldCheck, Star, Users } from "lucide-react";
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
    description: "Hypertrophy programs built around progressive overload and structured periodization.",
    ctaLabel: "EXPLORE",
    to: "/services/premium-level-1",
    accent: "blue",
    image: advert1,
    Icon: Dumbbell
  },
  {
    title: "LOSE FAT",
    description: "Calibrated intensity and metabolic circuits for fat loss and conditioning.",
    ctaLabel: "EXPLORE",
    to: "/services/premium-level-2",
    accent: "orange",
    image: advert2,
    Icon: Flame
  },
  {
    title: "TRAIN FOR FREE",
    description: "A complete training plan built on the fundamentals — no subscription, no catch.",
    ctaLabel: "START FREE",
    to: "/services/generic-program",
    accent: "teal",
    image: advert3,
    Icon: Gift
  }
] as const;

const stats = [
  { value: "10K+", label: "Active Members", Icon: Users },
  { value: "4.9/5", label: "Average Rating", Icon: Star },
  { value: "150+", label: "Training Programs", Icon: BarChart3 },
  { value: "100%", label: "Secure & Private", Icon: ShieldCheck }
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

        <Reveal delay={0.3}>
          <div className={styles.statsContainer}>
            <div className={styles.statsBar}>
              {stats.map((stat, index) => {
                const Icon = stat.Icon;

                return (
                  <div key={stat.label} className={styles.statItem}>
                    {index > 0 && <span className={styles.statDivider} aria-hidden="true" />}
                    <Icon className={styles.statIcon} aria-hidden="true" strokeWidth={1.6} />
                    <div className={styles.statText}>
                      <strong>{stat.value}</strong>
                      <span>{stat.label}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </Reveal>
      </Container>
    </section>
  );
};
