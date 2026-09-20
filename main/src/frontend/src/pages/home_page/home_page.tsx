import { ContactPreview } from "@/components/contact_preview/contact_preview";
import { Container } from "@/components/container/container";
import { Hero } from "@/components/hero/hero";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PlatformSection } from "@/components/platform_section/platform_section";
import { ProgramsOverviewSection } from "@/components/programs_overview_section/programs_overview_section";
import { Reveal } from "@/components/reveal/reveal";
import { SectionTitle } from "@/components/section_title/section_title";
import { ServiceCard } from "@/components/service_card/service_card";
import { SERVICE_PROGRAMS } from "@/constants/services";

import styles from "./home_page.module.css";

export const HomePage = () => (
  <PageWrapper>
    <Hero />

    <ProgramsOverviewSection />

    <PlatformSection />

    <section className={styles.servicesPreview}>
      <Container className={styles.servicesContainer}>
        <Reveal>
          <SectionTitle
            eyebrow="Catalog"
            title="Available RYZE program categories."
            description="Start free, add automatic nutrition, or move into a coach-reviewed Elite program."
            align="center"
          />
        </Reveal>
        <div className={styles.serviceGrid}>
          {SERVICE_PROGRAMS.map((service, index) => (
            <Reveal key={service.title} delay={index * 0.07}>
              <ServiceCard service={service} />
            </Reveal>
          ))}
        </div>
      </Container>
    </section>

    <ContactPreview />
  </PageWrapper>
);
