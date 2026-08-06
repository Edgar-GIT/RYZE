import { ContactPreview } from "@/components/contact_preview/contact_preview";
import { Container } from "@/components/container/container";
import { Hero } from "@/components/hero/hero";
import { NutritionSection } from "@/components/nutrition_section/nutrition_section";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PlatformSection } from "@/components/platform_section/platform_section";
import { ProgressSection } from "@/components/progress_section/progress_section";
import { Reveal } from "@/components/reveal/reveal";
import { SectionTitle } from "@/components/section_title/section_title";
import { ServiceCard } from "@/components/service_card/service_card";
import { TrainingSection } from "@/components/training_section/training_section";
import { SERVICE_PLANS } from "@/constants/services";
import servicesBackground from "@resources/img/hero/bg_services.png";

import styles from "./home_page.module.css";

export const HomePage = () => (
  <PageWrapper>
    <Hero />

    <PlatformSection />

    <TrainingSection />

    <NutritionSection />

    <ProgressSection />

    <section className={styles.servicesPreview}>
      <img
        className={styles.servicesBackground}
        src={servicesBackground}
        alt=""
        aria-hidden="true"
      />
      <Container className={styles.servicesContainer}>
        <Reveal>
          <SectionTitle
            eyebrow="Catalog"
            title="Available RYZE plan categories."
            description="Start free, add automatic nutrition, or move into a coach-reviewed Elite plan."
            align="center"
          />
        </Reveal>
        <div className={styles.serviceGrid}>
          {SERVICE_PLANS.map((service, index) => (
            <Reveal key={service.title} delay={index * 0.07}>
              <ServiceCard service={service} />
            </Reveal>
          ))}
        </div>
      </Container>
    </section>

    <Reveal y={24}>
      <ContactPreview />
    </Reveal>
  </PageWrapper>
);
