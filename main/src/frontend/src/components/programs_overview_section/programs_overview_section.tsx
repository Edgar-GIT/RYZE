import { ArrowRight, Dumbbell, Flame, Gift, Sparkles } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import type { ServiceCatalogState } from "@/hooks/use_service_catalog";
import { useMediaQuery } from "@/hooks/use_media_query";
import { type Copy, useI18n } from "@/i18n/i18n";
import { buildProgramHighlights, type ProgramHighlightCategory } from "@/services/service_catalog";
import { joinClassNames } from "@utils/class_names";

import advert1 from "@resources/img/program_cards/advert1.jpeg";
import advert2 from "@resources/img/program_cards/advert2.jpeg";
import advert3 from "@resources/img/program_cards/advert3.jpeg";

import styles from "./programs_overview_section.module.css";

const images = [advert1, advert2, advert3];

interface ProgramCard {
  key: string;
  title: string;
  description: string;
  ctaLabel: string;
  to: string;
  category: ProgramHighlightCategory;
  Icon: LucideIcon;
}

interface ShowcaseText {
  title: string;
  description: string;
  cta: string;
}

interface SectionText {
  // Only free programs carry a tag. Level names are left to the Services page:
  // the goal cards ("Build muscle", "Lose fat") do not describe the levels.
  freeTag: string;
  label: string;
  headingFind: string;
  headingYour: string;
  headingRest: string;
  lead: string;
  allPlans: string;
  loading: string;
  listLabel: string;
  cta: Record<ProgramHighlightCategory, string>;
  showcase: [ShowcaseText, ShowcaseText, ShowcaseText];
}

const COPY: Copy<SectionText> = {
  en: {
    freeTag: "Free",
    label: "Start training today",
    headingFind: "Find ",
    headingYour: "your",
    headingRest: " next program",
    lead: "Choose your goal and start your first session today.",
    allPlans: "See all plans",
    loading: "Loading the published programs…",
    listLabel: "Programs",
    cta: { generic: "Explore", premium: "View package", premiumLevel2: "View service" },
    showcase: [
      {
        title: "Build muscle",
        description:
          "Hypertrophy programs built around progressive overload. Structured volume, smart periodization and weekly progression that help you grow stronger, block after block.",
        cta: "Explore"
      },
      {
        title: "Lose fat",
        description:
          "Structured training programs for fat loss and conditioning. Calibrated intensity, metabolic circuits and recovery-aware scheduling that keep you cutting without burning out.",
        cta: "Explore"
      },
      {
        title: "Train for free",
        description:
          "A complete, ready-made training plan built on the fundamentals. No questionnaire, no subscription and no catch — start training today, completely free.",
        cta: "Start free"
      }
    ]
  },
  pt: {
    freeTag: "Grátis",
    label: "Começa a treinar hoje",
    headingFind: "Encontra o ",
    headingYour: "teu",
    headingRest: " próximo programa",
    lead: "Escolhe o teu objetivo e faz o teu primeiro treino hoje.",
    allPlans: "Ver todos os planos",
    loading: "A carregar os programas publicados…",
    listLabel: "Programas",
    cta: { generic: "Explorar", premium: "Ver pacote", premiumLevel2: "Ver serviço" },
    showcase: [
      {
        title: "Ganhar massa muscular",
        description:
          "Programas de hipertrofia construídos com sobrecarga progressiva. Volume estruturado, periodização inteligente e progressão semanal para ficares mais forte, bloco após bloco.",
        cta: "Explorar"
      },
      {
        title: "Perder gordura",
        description:
          "Programas de treino estruturados para perda de gordura e condição física. Intensidade calibrada, circuitos metabólicos e um calendário que respeita a recuperação, para emagreceres sem te esgotares.",
        cta: "Explorar"
      },
      {
        title: "Treinar grátis",
        description:
          "Um plano de treino completo e pronto, assente no essencial. Sem questionário, sem subscrição e sem letras pequenas — começa a treinar hoje, totalmente grátis.",
        cta: "Começar grátis"
      }
    ]
  }
};

// The original goal cards (restored from before the section became
// API-driven). Shown only while no program is published, so Home always invites
// people in; published programs replace them with live data.
const SHOWCASE_STRUCTURE: { key: string; to: string; category: ProgramHighlightCategory; Icon: LucideIcon }[] = [
  { key: "build-muscle", to: "/services/premium-level-1", category: "premium", Icon: Dumbbell },
  { key: "lose-fat", to: "/services/premium-level-2", category: "premiumLevel2", Icon: Flame },
  { key: "train-for-free", to: "/services/generic-program", category: "generic", Icon: Gift }
];

// Below 768px the cards are a horizontal, swipeable row that must be
// keyboard-focusable; from 768px they are a static grid.
const SCROLLER_QUERY = "(max-width: 767px)";

interface ProgramsOverviewSectionProps {
  catalog: ServiceCatalogState;
}

export const ProgramsOverviewSection = ({ catalog }: ProgramsOverviewSectionProps) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const scroller = useMediaQuery(SCROLLER_QUERY);
  const highlights =
    catalog.status === "ready" ? buildProgramHighlights(catalog.generic, catalog.premium, catalog.premiumLevel2) : [];
  const cards: ProgramCard[] =
    highlights.length > 0
      ? highlights.map((program) => ({
          key: `${program.category}-${program.id}`,
          title: program.title,
          description: program.description,
          ctaLabel: text.cta[program.category],
          to: program.to,
          category: program.category,
          Icon: program.category === "generic" ? Dumbbell : Sparkles
        }))
      : SHOWCASE_STRUCTURE.map((card, index) => ({
          ...card,
          title: text.showcase[index].title,
          description: text.showcase[index].description,
          ctaLabel: text.showcase[index].cta
        }));

  return (
    <section id="programs" className={joinClassNames(ui.toneNavy, ui.anchorTarget, styles.section)} aria-labelledby="programs-heading">
      <div className={joinClassNames(ui.container, styles.inner)}>
        <div className={styles.header}>
          <div className={styles.intro}>
            <p className={ui.label}>{text.label}</p>
            <h2 id="programs-heading" className={ui.heading}>
              {text.headingFind}
              <span className={ui.hl}>{text.headingYour}</span>
              {text.headingRest}
              <span className={ui.stop}>.</span>
            </h2>
            <p className={ui.lead}>{text.lead}</p>
          </div>
          <PublicAction to="/services" variant="ghostOnDark" icon={<ArrowRight />} className={styles.allPlans}>
            {text.allPlans}
          </PublicAction>
        </div>

        {catalog.status === "loading" ? (
          <div className={joinClassNames(ui.surface, styles.state)}>
            <p className={ui.body}>{text.loading}</p>
          </div>
        ) : (
          <ul className={styles.cards} tabIndex={scroller ? 0 : undefined} aria-label={text.listLabel}>
            {cards.map((program, index) => {
              const Icon = program.Icon;

              return (
                <li key={program.key} className={styles.card}>
                  <img className={styles.image} src={images[index % images.length]} alt="" loading="lazy" decoding="async" />
                  <span className={styles.shade} aria-hidden="true" />
                  <div className={styles.top}>
                    <span className={styles.icon} aria-hidden="true">
                      <Icon strokeWidth={1.9} />
                    </span>
                    {program.category === "generic" ? <span className={styles.tag}>{text.freeTag}</span> : null}
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
