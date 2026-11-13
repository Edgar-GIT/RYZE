import { ArrowRight, Check } from "lucide-react";

import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { SERVICE_CATEGORY_PRESENTATION } from "@/constants/services";
import type { ServiceCatalogState } from "@/hooks/use_service_catalog";
import { type Copy, useI18n } from "@/i18n/i18n";
import type { MarketplaceProgram } from "@/services/marketplace_api";
import { describeCatalogPrice } from "@/services/service_catalog";
import { joinClassNames } from "@utils/class_names";

import styles from "./category_overview.module.css";

type CategoryKey = keyof typeof SERVICE_CATEGORY_PRESENTATION;

interface CategoryStyle {
  personalization: 1 | 2 | 3;
  accent: "level1" | "level2" | "level3";
  action: "ghostOnDark" | "primary";
}

// Personalization follows the documented ordering of the three program
// categories; prices come from the API.
const CATEGORY_STYLE: Record<CategoryKey, CategoryStyle> = {
  generic: { personalization: 1, accent: "level1", action: "ghostOnDark" },
  premium: { personalization: 2, accent: "level2", action: "primary" },
  premiumLevel2: { personalization: 3, accent: "level3", action: "primary" }
};

interface CategoryText {
  title: string;
  description: string;
  included: string[];
  cta: string;
}

interface OverviewText {
  label: string;
  heading: string;
  headingHighlight: string;
  lead: string;
  personalization: (level: number) => string;
  categories: Record<CategoryKey, CategoryText>;
}

// Copy summarises docs/INSTRUCTIONS/project.MD.
const COPY: Copy<OverviewText> = {
  en: {
    label: "Choose your level",
    heading: "Pick how much ",
    headingHighlight: "guidance you want.",
    lead: "Explore training plans, Premium Level 1 packages and trainer-delivered Premium Level 2 services.",
    personalization: (level) => `Personalization ${level} of 3`,
    categories: {
      generic: {
        title: "Training plans",
        description: "Ready-made training plans for common goals, free and paid.",
        included: ["Structured weekly training plan", "Exercise demonstrations", "Session logging and history"],
        cta: "Browse training plans"
      },
      premium: {
        title: "Premium Level 1",
        description: "A structured training programme paired with a nutrition plan.",
        included: ["Structured training programme", "Nutrition plan with calorie and macro targets", "Delivered right after purchase"],
        cta: "View Premium Level 1"
      },
      premiumLevel2: {
        title: "Premium Level 2",
        description: "Training and nutrition prepared individually for you by a RYZE trainer.",
        included: ["Programme authored by a RYZE trainer", "Personal nutrition plan", "Status tracking from intake to delivery"],
        cta: "View Premium Level 2"
      }
    }
  },
  pt: {
    label: "Escolhe o teu nível",
    heading: "Escolhe quanto ",
    headingHighlight: "acompanhamento queres.",
    lead: "Explora planos de treino, pacotes Premium Nível 1 e serviços Premium Nível 2 entregues por treinadores.",
    personalization: (level) => `Personalização ${level} de 3`,
    categories: {
      generic: {
        title: "Planos de treino",
        description: "Planos de treino prontos para objetivos comuns, gratuitos e pagos.",
        included: ["Plano de treino semanal estruturado", "Demonstrações de exercícios", "Registo de treinos e histórico"],
        cta: "Ver planos de treino"
      },
      premium: {
        title: "Premium Nível 1",
        description: "Um programa de treino estruturado com um plano de nutrição.",
        included: ["Programa de treino estruturado", "Plano de nutrição com metas de calorias e macros", "Disponível logo após a compra"],
        cta: "Ver Premium Nível 1"
      },
      premiumLevel2: {
        title: "Premium Nível 2",
        description: "Treino e nutrição preparados individualmente para ti por um treinador RYZE.",
        included: ["Programa criado por um treinador RYZE", "Plano de nutrição pessoal", "Acompanhamento do estado, do questionário à entrega"],
        cta: "Ver Premium Nível 2"
      }
    }
  }
};

const PERSONALIZATION_STEPS = [1, 2, 3] as const;

interface CategoryOverviewProps {
  catalog: ServiceCatalogState;
}

export const CategoryOverview = ({ catalog }: CategoryOverviewProps) => {
  const { locale } = useI18n();
  const text = COPY[locale];

  if (catalog.status !== "ready") {
    return null;
  }

  const families: Array<[CategoryKey, MarketplaceProgram[]]> = [
    ["generic", catalog.generic],
    ["premium", catalog.premium],
    ["premiumLevel2", catalog.premiumLevel2]
  ];
  const categories = families.filter(([, programs]) => programs.length > 0);

  if (categories.length === 0) {
    return null;
  }

  return (
    <section className={joinClassNames(ui.section, ui.toneNavy)} aria-labelledby="category-overview-heading">
      <div className={joinClassNames(ui.container, styles.layout)}>
        <div className={styles.intro}>
          <p className={ui.label}>{text.label}</p>
          <h2 id="category-overview-heading" className={ui.heading}>
            {text.heading}
            <span className={ui.hl}>{text.headingHighlight}</span>
          </h2>
          <p className={ui.lead}>{text.lead}</p>
        </div>

        <ol className={styles.levels}>
          {categories.map(([key, programs]) => {
            const presentation = SERVICE_CATEGORY_PRESENTATION[key];
            const copy = { ...CATEGORY_STYLE[key], ...text.categories[key] };

            return (
              <li
                key={key}
                className={joinClassNames(ui.surface, styles.level, styles[copy.accent])}
                data-personalization={copy.personalization}
              >
                <div className={styles.meter}>
                  <span className={styles.meterBars} aria-hidden="true">
                    {PERSONALIZATION_STEPS.map((step) => (
                      <i key={step} className={step <= copy.personalization ? styles.barOn : undefined} />
                    ))}
                  </span>
                  <span>{text.personalization(copy.personalization)}</span>
                </div>
                <h3 className={ui.title}>{copy.title}</h3>
                <p className={styles.description}>{copy.description}</p>
                <ul className={styles.included}>
                  {copy.included.map((item) => (
                    <li key={item}>
                      <Check aria-hidden="true" strokeWidth={2.4} />
                      {item}
                    </li>
                  ))}
                </ul>
                <div className={styles.footer}>
                  <p className={styles.price}>{describeCatalogPrice(programs, locale)}</p>
                  <PublicAction to={presentation.to} variant={copy.action} icon={<ArrowRight />} className={styles.cta}>
                    {copy.cta}
                  </PublicAction>
                </div>
              </li>
            );
          })}
        </ol>
      </div>
    </section>
  );
};
