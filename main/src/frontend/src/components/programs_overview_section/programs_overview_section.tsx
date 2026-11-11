import { ArrowRight, Dumbbell, Flame, Gift, Sparkles, UserCheck } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import type { ServiceCatalogState } from "@/hooks/use_service_catalog";
import { useMediaQuery } from "@/hooks/use_media_query";
import { buildProgramHighlights, type ProgramHighlightCategory } from "@/services/service_catalog";
import { joinClassNames } from "@utils/class_names";

import advert1 from "@resources/img/program_cards/advert1.jpeg";
import advert2 from "@resources/img/program_cards/advert2.jpeg";
import advert3 from "@resources/img/program_cards/advert3.jpeg";

import styles from "./programs_overview_section.module.css";

const images = [advert1, advert2, advert3];

// Same level colours as the Services page: free cyan, Premium Level 1 blue,
// Premium Level 2 (the most personalized) purple.
const TAGS: Record<ProgramHighlightCategory, { label: string; tone: "free" | "premium" | "personalized"; icon: LucideIcon }> = {
  generic: { label: "Free", tone: "free", icon: Gift },
  premium: { label: "Premium Level 1", tone: "premium", icon: Sparkles },
  premiumLevel2: { label: "Premium Level 2", tone: "personalized", icon: UserCheck }
};

interface ProgramCard {
  key: string;
  title: string;
  description: string;
  ctaLabel: string;
  to: string;
  category: ProgramHighlightCategory;
  Icon: LucideIcon;
}

// The original goal cards (restored from before the section became
// API-driven). Shown only while no program is published, so Home always invites
// people in; published programs replace them with live data.
const SHOWCASE: ProgramCard[] = [
  {
    key: "build-muscle",
    title: "Build muscle",
    description:
      "Hypertrophy programs built around progressive overload. Structured volume, smart periodization and weekly progression that help you grow stronger, block after block.",
    ctaLabel: "Explore",
    to: "/services/premium-level-1",
    category: "premium",
    Icon: Dumbbell
  },
  {
    key: "lose-fat",
    title: "Lose fat",
    description:
      "Structured training programs for fat loss and conditioning. Calibrated intensity, metabolic circuits and recovery-aware scheduling that keep you cutting without burning out.",
    ctaLabel: "Explore",
    to: "/services/premium-level-2",
    category: "premiumLevel2",
    Icon: Flame
  },
  {
    key: "train-for-free",
    title: "Train for free",
    description:
      "A complete, ready-made training plan built on the fundamentals. No questionnaire, no subscription and no catch — start training today, completely free.",
    ctaLabel: "Start free",
    to: "/services/generic-program",
    category: "generic",
    Icon: Gift
  }
];

// Below 768px the cards are a horizontal, swipeable row that must be
// keyboard-focusable; from 768px they are a static grid.
const SCROLLER_QUERY = "(max-width: 767px)";

interface ProgramsOverviewSectionProps {
  catalog: ServiceCatalogState;
}

export const ProgramsOverviewSection = ({ catalog }: ProgramsOverviewSectionProps) => {
  const scroller = useMediaQuery(SCROLLER_QUERY);
  const highlights =
    catalog.status === "ready" ? buildProgramHighlights(catalog.generic, catalog.premium, catalog.premiumLevel2) : [];
  const cards: ProgramCard[] =
    highlights.length > 0
      ? highlights.map((program) => ({
          key: `${program.category}-${program.id}`,
          title: program.title,
          description: program.description,
          ctaLabel: program.ctaLabel,
          to: program.to,
          category: program.category,
          Icon: program.category === "generic" ? Dumbbell : Sparkles
        }))
      : SHOWCASE;

  return (
    <section id="programs" className={joinClassNames(ui.toneNavy, ui.anchorTarget, styles.section)} aria-labelledby="programs-heading">
      <div className={joinClassNames(ui.container, styles.inner)}>
        <div className={styles.header}>
          <div className={styles.intro}>
            <p className={ui.label}>Start training today</p>
            <h2 id="programs-heading" className={ui.heading}>
              Find <span className={ui.hl}>your</span> next program<span className={ui.stop}>.</span>
            </h2>
            <p className={ui.lead}>Choose your goal and start your first session today.</p>
          </div>
          <PublicAction to="/services" variant="ghostOnDark" icon={<ArrowRight />} className={styles.allPlans}>
            See all plans
          </PublicAction>
        </div>

        {catalog.status === "loading" ? (
          <div className={joinClassNames(ui.surface, styles.state)}>
            <p className={ui.body}>Loading the published programs…</p>
          </div>
        ) : (
          <ul className={styles.cards} tabIndex={scroller ? 0 : undefined} aria-label="Programs">
            {cards.map((program, index) => {
              const Icon = program.Icon;
              const tag = TAGS[program.category];

              return (
                <li key={program.key} className={joinClassNames(styles.card, styles[tag.tone])}>
                  <img className={styles.image} src={images[index % images.length]} alt="" loading="lazy" decoding="async" />
                  <span className={styles.shade} aria-hidden="true" />
                  <div className={styles.top}>
                    <span className={styles.icon} aria-hidden="true">
                      <Icon strokeWidth={1.9} />
                    </span>
                    <span className={styles.tag}>{tag.label}</span>
                  </div>
                  <div className={styles.body}>
                    <h3 className={styles.title}>{program.title}</h3>
                    {program.description ? <p className={styles.description}>{program.description}</p> : null}
                    <PublicAction to={program.to} icon={<ArrowRight />} className={styles.cta}>
                      {program.ctaLabel}
                    </PublicAction>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
};
