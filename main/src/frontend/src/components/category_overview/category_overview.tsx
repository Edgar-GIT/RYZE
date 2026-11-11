import { ArrowRight, Check } from "lucide-react";

import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { SERVICE_CATEGORY_PRESENTATION } from "@/constants/services";
import type { ServiceCatalogState } from "@/hooks/use_service_catalog";
import type { MarketplaceProgram } from "@/services/marketplace_api";
import { describeCatalogPrice } from "@/services/service_catalog";
import { joinClassNames } from "@utils/class_names";

import styles from "./category_overview.module.css";

type CategoryKey = keyof typeof SERVICE_CATEGORY_PRESENTATION;

interface CategoryCopy {
  description: string;
  personalization: 1 | 2 | 3;
  included: string[];
  accent: "level1" | "level2" | "level3";
  action: "ghostOnDark" | "primary";
}

// Copy summarises docs/INSTRUCTIONS/project.MD. Personalization follows the
// documented ordering of the three program categories; prices come from the API.
const CATEGORY_COPY: Record<CategoryKey, CategoryCopy> = {
  generic: {
    description: "Ready-made training plans for common goals, free and paid.",
    personalization: 1,
    included: ["Structured weekly training plan", "Exercise demonstrations", "Session logging and history"],
    accent: "level1",
    action: "ghostOnDark"
  },
  premium: {
    description: "A structured training programme paired with a nutrition plan.",
    personalization: 2,
    included: ["Structured training programme", "Nutrition plan with calorie and macro targets", "Delivered right after purchase"],
    accent: "level2",
    action: "primary"
  },
  premiumLevel2: {
    description: "Training and nutrition prepared individually for you by a RYZE trainer.",
    personalization: 3,
    included: ["Programme authored by a RYZE trainer", "Personal nutrition plan", "Status tracking from intake to delivery"],
    accent: "level3",
    action: "primary"
  }
};

const PERSONALIZATION_STEPS = [1, 2, 3] as const;

interface CategoryOverviewProps {
  catalog: ServiceCatalogState;
}

export const CategoryOverview = ({ catalog }: CategoryOverviewProps) => {
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
          <p className={ui.label}>Choose your level</p>
          <h2 id="category-overview-heading" className={ui.heading}>
            Pick how much <span className={ui.hl}>guidance you want.</span>
          </h2>
          <p className={ui.lead}>
            Explore training plans, Premium Level 1 packages and trainer-delivered Premium Level 2
            services.
          </p>
        </div>

        <ol className={styles.levels}>
          {categories.map(([key, programs]) => {
            const presentation = SERVICE_CATEGORY_PRESENTATION[key];
            const copy = CATEGORY_COPY[key];

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
                  <span>Personalization {copy.personalization} of 3</span>
                </div>
                <h3 className={ui.title}>{presentation.title}</h3>
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
                  <p className={styles.price}>{describeCatalogPrice(programs)}</p>
                  <PublicAction to={presentation.to} variant={copy.action} icon={<ArrowRight />} className={styles.cta}>
                    {presentation.ctaLabel}
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
