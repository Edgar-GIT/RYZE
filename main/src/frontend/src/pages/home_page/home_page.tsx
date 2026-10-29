import { ContactPreview } from "@/components/contact_preview/contact_preview";
import { Container } from "@/components/container/container";
import { Hero } from "@/components/hero/hero";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PlatformSection } from "@/components/platform_section/platform_section";
import { ProgramsOverviewSection } from "@/components/programs_overview_section/programs_overview_section";
import { Reveal } from "@/components/reveal/reveal";
import { SectionTitle } from "@/components/section_title/section_title";
import { ServiceCard } from "@/components/service_card/service_card";
import { useServiceCatalog } from "@/hooks/use_service_catalog";
import { buildServiceCategoryCards } from "@/services/service_catalog";

import styles from "./home_page.module.css";

export const HomePage = () => {
  const catalog = useServiceCatalog();
  const services =
    catalog.status === "ready"
      ? buildServiceCategoryCards(catalog.generic, catalog.premium)
      : [];

  return (
    <PageWrapper>
      <Hero />

      <ProgramsOverviewSection catalog={catalog} />

      <PlatformSection />

      {services.length > 0 ? (
        <section className={styles.servicesPreview}>
          <Container className={styles.servicesContainer}>
            <Reveal>
              <SectionTitle
                eyebrow="Catalog"
                title="Available RYZE program categories."
                description="Explore the published training plans and Premium Level 1 packages."
              />
            </Reveal>
            <div className={styles.serviceGrid}>
              {services.map((service, index) => (
                <Reveal key={service.title} delay={index * 0.07}>
                  <ServiceCard service={service} />
                </Reveal>
              ))}
            </div>
          </Container>
        </section>
      ) : null}

      <ContactPreview />
    </PageWrapper>
  );
};
