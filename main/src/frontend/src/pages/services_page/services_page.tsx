import { Dumbbell, Heart, Trophy, Zap } from "lucide-react";

import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { Reveal } from "@/components/reveal/reveal";
import { ServiceCard } from "@/components/service_card/service_card";
import ui from "@/components/public_ui/public_ui.module.css";
import { SERVICE_SHOWCASE } from "@/constants/services";
import { useServiceCatalog } from "@/hooks/use_service_catalog";
import { buildServiceCategoryCards } from "@/services/service_catalog";
import { joinClassNames } from "@utils/class_names";

import styles from "./services_page.module.css";

const evolutionItems = [
  {
    icon: Dumbbell,
    title: "Train Smarter",
    text: "AI-powered programs that adapt to you."
  },
  {
    icon: Heart,
    title: "Reach your Goals",
    text: "Track every milestone along the way."
  },
  {
    icon: Zap,
    title: "Progress Faster",
    text: "Smart analytics to keep you ahead."
  },
  {
    icon: Trophy,
    title: "Get Stronger",
    text: "Build lasting strength and endurance."
  }
] as const;

export const ServicesPage = () => {
  const catalog = useServiceCatalog();
  const services =
    catalog.status === "ready"
      ? buildServiceCategoryCards(catalog.generic, catalog.premium, catalog.premiumLevel2)
      : [];

  return (
    <PageWrapper className={joinClassNames(ui.page, styles.page)}>
      {/* The section sets no inherited font or colour, so the service cards keep
          their own styling; only the intro, states and benefits use the public tones. */}
      <section className={styles.catalog}>
        <Container className={styles.catalogInner}>
          <Reveal className={joinClassNames(ui.toneNavy, styles.intro)}>
            <p className={ui.label}>Services</p>
            <h1 className={ui.heading}>
              Choose how you want to <span className={ui.hl}>RYZE</span>
              <span className={ui.stop}>.</span>
            </h1>
            <p className={joinClassNames(ui.lead, styles.lead)}>
              Compare the published RYZE offers and choose the route that fits the guidance and
              personalization you need.
            </p>
          </Reveal>

          {catalog.status === "loading" ? (
            <div className={joinClassNames(ui.toneNavy, ui.surface, styles.state)}>
              <p className={styles.stateText}>Loading the published offers…</p>
            </div>
          ) : (
            // Live offers when published; otherwise the original showcase cards.
            <div className={styles.grid}>
              {(services.length > 0 ? services : SERVICE_SHOWCASE).map((service) => (
                <ServiceCard key={service.title} service={service} />
              ))}
            </div>
          )}

          <Reveal className={joinClassNames(ui.toneNavy, ui.surface, styles.evolution)} delay={0.18}>
            {evolutionItems.map((item, index) => {
              const Icon = item.icon;

              return (
                <article key={item.title} className={styles.evolutionItem}>
                  <span className={ui.iconTile}>
                    <Icon aria-hidden="true" strokeWidth={1.9} />
                  </span>
                  <div className={styles.evolutionCopy}>
                    <strong>{item.title}</strong>
                    <p>{item.text}</p>
                  </div>
                  {index < evolutionItems.length - 1 && (
                    <span className={styles.evolutionDivider} aria-hidden="true" />
                  )}
                </article>
              );
            })}
          </Reveal>
        </Container>
      </section>
    </PageWrapper>
  );
};
