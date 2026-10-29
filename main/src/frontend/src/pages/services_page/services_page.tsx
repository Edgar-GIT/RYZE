import { Dumbbell, Heart, Trophy, Zap } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { Reveal } from "@/components/reveal/reveal";
import { ServiceCard } from "@/components/service_card/service_card";
import { useServiceCatalog } from "@/hooks/use_service_catalog";
import { buildServiceCategoryCards } from "@/services/service_catalog";
import servicesBackground from "@resources/img/hero/bg_services.png";

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
      ? buildServiceCategoryCards(catalog.generic, catalog.premium)
      : [];

  return (
    <PageWrapper className={styles.page}>
      <section className={styles.catalog}>
        <img
          className={styles.catalogBackground}
          src={servicesBackground}
          alt=""
          aria-hidden="true"
        />

        <Container className={styles.catalogInner}>
          <Reveal className={styles.intro}>
            <h1>
              Choose how you want to <span className={styles.accent}>RYZE</span>
            </h1>
            <p className={styles.lead}>
              Compare the published RYZE offers and choose the route that fits the guidance and
              personalization you need.
            </p>
          </Reveal>

          {catalog.status === "loading" ? (
            <div className={styles.state}>
              <p className={styles.stateText}>Loading the published offers…</p>
            </div>
          ) : catalog.status === "error" ? (
            <div className={styles.state}>
              <p className={styles.stateText}>Unable to load the services right now.</p>
              <Button variant="secondary" size="small" onClick={catalog.retry}>
                Try again
              </Button>
            </div>
          ) : services.length === 0 ? (
            <div className={styles.state}>
              <p className={styles.stateText}>No offers have been published yet.</p>
              <Button variant="secondary" size="small" to="/services/generic-program">
                Browse training plans
              </Button>
            </div>
          ) : (
            <div className={styles.grid}>
              {services.map((service) => (
                <ServiceCard key={service.title} service={service} />
              ))}
            </div>
          )}

          <Reveal className={styles.evolution} delay={0.18}>
            {evolutionItems.map((item, index) => {
              const Icon = item.icon;

              return (
                <article key={item.title} className={styles.evolutionItem}>
                  <span className={styles.evolutionIcon}>
                    <Icon aria-hidden="true" strokeWidth={1.7} />
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
