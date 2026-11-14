import {
  ArrowRight,
  BarChart3,
  CheckCircle2,
  Dumbbell,
  Headphones,
  Infinity as InfinityIcon,
  Rocket,
  Sparkles,
  Target,
  Zap,
  type LucideIcon
} from "lucide-react";
import { Link } from "react-router-dom";

import { type Copy, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./service_tiers.module.css";

interface TierCopy {
  level: string;
  title: string;
  titleAccent?: string;
  price?: string;
  priceNote?: string;
  lead: string;
  features: string[];
  cta: string;
}

interface TiersText {
  label: string;
  labelAccent: string;
  heading: string;
  headingAccent: string;
  lead: string;
  tiers: [TierCopy, TierCopy, TierCopy];
  benefits: [string, string, string, string, string];
}

const COPY: Copy<TiersText> = {
  en: {
    label: "RYZE",
    labelAccent: "PLANS",
    heading: "Grow at your pace.",
    headingAccent: "Choose your level.",
    lead: "From the essentials to a plan built around you, with technology and expert support.",
    tiers: [
      {
        level: "Level 1",
        title: "Free",
        lead: "Start today, at no cost.",
        features: ["Ready-made training plans", "Exercise library with demos", "Workout log and progress"],
        cta: "Browse free plans"
      },
      {
        level: "Level 2",
        title: "Premium",
        price: "€14.99",
        priceNote: "one-time",
        lead: "Training and nutrition for your goal.",
        features: ["A plan chosen for your goal", "Nutrition plan included", "Ready right after payment"],
        cta: "Learn more"
      },
      {
        level: "Level 3",
        title: "Personalised",
        titleAccent: "made for you",
        price: "€19.49",
        priceNote: "one-time",
        lead: "The most complete plan: everything adapted to you.",
        features: [
          "A full questionnaire about you",
          "Training 100% personalised",
          "Nutrition adapted to your routine",
          "Prepared and reviewed by a RYZE trainer"
        ],
        cta: "Choose Personalised"
      }
    ],
    benefits: ["Plans for every level", "Continuous progress", "One-time payment, no lock-in", "Expert support", "Focused on your goals"]
  },
  pt: {
    label: "PLANOS",
    labelAccent: "RYZE",
    heading: "Evolui ao teu ritmo.",
    headingAccent: "Escolhe o teu nível.",
    lead: "Do essencial a um plano feito à tua medida, com tecnologia e suporte especializado.",
    tiers: [
      {
        level: "Nível 1",
        title: "Grátis",
        lead: "Começa hoje, sem custos.",
        features: ["Planos de treino prontos", "Biblioteca de exercícios com demonstrações", "Registo de treinos e progresso"],
        cta: "Ver planos grátis"
      },
      {
        level: "Nível 2",
        title: "Premium",
        price: "14,99 €",
        priceNote: "pagamento único",
        lead: "Treino e nutrição para o teu objetivo.",
        features: ["Plano escolhido para o teu objetivo", "Plano de nutrição incluído", "Disponível logo após o pagamento"],
        cta: "Saber mais"
      },
      {
        level: "Nível 3",
        title: "Personalizado",
        titleAccent: "feito para ti",
        price: "19,49 €",
        priceNote: "pagamento único",
        lead: "O plano mais completo: tudo adaptado a ti.",
        features: [
          "Questionário completo sobre ti",
          "Treino 100% personalizado",
          "Nutrição adaptada à tua rotina",
          "Preparado e revisto por um treinador RYZE"
        ],
        cta: "Escolher Personalizado"
      }
    ],
    benefits: ["Planos para todos os níveis", "Evolução contínua", "Pagamento único, sem fidelização", "Suporte especializado", "Foco nos teus objetivos"]
  }
};

const TIER_STRUCTURE: { to: string; icon: LucideIcon }[] = [
  { to: "/services/generic-program", icon: Rocket },
  { to: "/services/premium-level-1", icon: Dumbbell },
  { to: "/services/premium-level-2", icon: Sparkles }
];

const BENEFIT_ICONS: LucideIcon[] = [Zap, BarChart3, InfinityIcon, Headphones, Target];

// ServiceTiers presents the three RYZE services as rising glass panels (owner
// design of 2026-11-14): the free level, Premium Level 1 and the personalised
// Premium Level 2, each a step higher, with the prices fixed by category and a
// strip of shared benefits. The whole panel is a link to its service.
export const ServiceTiers = () => {
  const { locale } = useI18n();
  const text = COPY[locale];

  return (
    <section className={styles.tiers} aria-labelledby="tiers-heading">
      <div className={styles.stage}>
        <header className={styles.intro}>
          <p className={styles.label}>
            {text.label} <span>{text.labelAccent}</span>
          </p>
          <h1 id="tiers-heading" className={styles.heading}>
            {text.heading}
            <span>{text.headingAccent}</span>
          </h1>
          <p className={styles.lead}>{text.lead}</p>
        </header>

        <svg className={styles.arrow} viewBox="0 0 600 260" aria-hidden="true" focusable="false">
          <defs>
            <linearGradient id="tiers-arrow" x1="0" x2="1" y1="1" y2="0">
              <stop offset="0%" stopColor="#0481dc" stopOpacity="0" />
              <stop offset="45%" stopColor="#0481dc" stopOpacity="0.7" />
              <stop offset="100%" stopColor="#7cebff" />
            </linearGradient>
          </defs>
          <path className={styles.arrowGlow} d="M8 250 C 220 230, 380 170, 560 34" />
          <path className={styles.arrowLine} d="M8 250 C 220 230, 380 170, 560 34" stroke="url(#tiers-arrow)" />
          <path className={styles.arrowHead} d="M520 30 L566 28 L552 72" />
        </svg>

        {text.tiers.map((tier, index) => {
          const { to, icon: Icon } = TIER_STRUCTURE[index];
          const top = index === 2;
          return (
            <article
              key={tier.level}
              className={joinClassNames(styles.tier, styles[`tier${index + 1}`])}
              aria-labelledby={`tier-${index}`}
            >
              {top ? (
                <svg className={styles.crown} viewBox="0 0 120 80" aria-hidden="true" focusable="false">
                  <path d="M10 70 L18 22 L42 46 L60 8 L78 46 L102 22 L110 70 Z" />
                  <path d="M10 70 L110 70" />
                </svg>
              ) : null}
              <div className={styles.panel}>
                <span className={styles.level}>{tier.level}</span>
                <h2 id={`tier-${index}`} className={styles.title}>
                  <Link to={to} className={styles.titleLink}>
                    {top ? null : <Icon aria-hidden="true" />}
                    <span>
                      {tier.title}
                      {tier.titleAccent ? <em> {tier.titleAccent}</em> : null}
                    </span>
                  </Link>
                </h2>
                {tier.price ? (
                  <p className={styles.price}>
                    {tier.price}
                    <span>{tier.priceNote}</span>
                  </p>
                ) : null}
                <p className={styles.tierLead}>{tier.lead}</p>
                <ul className={styles.features}>
                  {tier.features.map((feature) => (
                    <li key={feature}>
                      <CheckCircle2 aria-hidden="true" />
                      {feature}
                    </li>
                  ))}
                </ul>
                <span className={top ? styles.ctaPrimary : styles.ctaQuiet} aria-hidden="true">
                  {tier.cta}
                  <ArrowRight />
                </span>
              </div>
            </article>
          );
        })}
      </div>

      <ul className={styles.benefits}>
        {text.benefits.map((benefit, index) => {
          const Icon = BENEFIT_ICONS[index];
          return (
            <li key={benefit}>
              <Icon aria-hidden="true" strokeWidth={1.6} />
              <span>{benefit}</span>
            </li>
          );
        })}
      </ul>
    </section>
  );
};
