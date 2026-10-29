import { ArrowRight, Dumbbell, Sparkles } from "lucide-react";
import { Link } from "react-router-dom";
import { motion, useReducedMotion } from "framer-motion";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { Reveal } from "@/components/reveal/reveal";
import type { ServiceCatalogState } from "@/hooks/use_service_catalog";
import { buildProgramHighlights } from "@/services/service_catalog";

import advert1 from "@resources/img/program_cards/advert1.jpeg";
import advert2 from "@resources/img/program_cards/advert2.jpeg";
import advert3 from "@resources/img/program_cards/advert3.jpeg";

import styles from "./programs_overview_section.module.css";

const images = [advert1, advert2, advert3];
const accents = ["blue", "orange", "teal"] as const;

interface ProgramsOverviewSectionProps {
  catalog: ServiceCatalogState;
}

export const ProgramsOverviewSection = ({ catalog }: ProgramsOverviewSectionProps) => {
  const reduceMotion = useReducedMotion();
  const highlights =
    catalog.status === "ready" ? buildProgramHighlights(catalog.generic, catalog.premium) : [];

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

        {catalog.status === "loading" ? (
          <div className={styles.state}>
            <p className={styles.stateText}>Loading the published programs…</p>
          </div>
        ) : catalog.status === "error" ? (
          <div className={styles.state}>
            <p className={styles.stateText}>Unable to load the program catalogue right now.</p>
            <Button variant="secondary" size="small" onClick={catalog.retry}>
              Try again
            </Button>
          </div>
        ) : highlights.length === 0 ? (
          <div className={styles.state}>
            <p className={styles.stateText}>No programs have been published yet.</p>
            <Button variant="secondary" size="small" to="/services">
              Browse services
            </Button>
          </div>
        ) : (
          <div className={styles.cardGrid}>
            {highlights.map((program, index) => {
              const Icon = program.category === "premium" ? Sparkles : Dumbbell;
              const accent = accents[index % accents.length];

              return (
                <Reveal key={`${program.category}-${program.id}`} delay={index * 0.1}>
                  <motion.article
                    className={`${styles.card} ${styles[accent]}`}
                    whileHover={reduceMotion ? undefined : { y: -5 }}
                    transition={{ duration: 0.28, ease: "easeOut" }}
                  >
                    <img
                      className={styles.cardImage}
                      src={images[index % images.length]}
                      alt=""
                      loading="lazy"
                    />
                    <div className={styles.cardOverlay} aria-hidden="true" />
                    <div className={styles.cardContent}>
                      <span className={styles.iconBadge}>
                        <Icon aria-hidden="true" strokeWidth={1.8} />
                      </span>
                      <h3>{program.title}</h3>
                      {program.description ? (
                        <p className={styles.cardDescription}>{program.description}</p>
                      ) : null}
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
        )}
      </Container>
    </section>
  );
};
