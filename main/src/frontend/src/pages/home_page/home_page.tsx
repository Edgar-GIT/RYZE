import { ContactPreview } from "@/components/contact_preview/contact_preview";
import { CtaBand } from "@/components/cta_band/cta_band";
import { Hero } from "@/components/hero/hero";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PlatformSection } from "@/components/platform_section/platform_section";
import { NutritionDemo } from "@/components/product_demos/nutrition_demo";
import { ProgressDemo } from "@/components/product_demos/progress_demo";
import { Reveal } from "@/components/reveal/reveal";
import { SectionTitle } from "@/components/section_title/section_title";
import { SectionWrapper } from "@/components/section_wrapper/section_wrapper";
import { ServiceCard } from "@/components/service_card/service_card";
import { ShowcaseSection } from "@/components/showcase_section/showcase_section";
import { TrainingSection } from "@/components/training_section/training_section";
import { SERVICE_PLANS } from "@/constants/services";

import styles from "./home_page.module.css";

export const HomePage = () => (
  <PageWrapper>
    <Hero />

    <PlatformSection />

    <TrainingSection />

    <ShowcaseSection
      eyebrow="Nutrition"
      title="Fuel that moves with the plan."
      description="Macros, timing and guidance that adapt to the training week — not a static PDF."
      tone="card"
    >
      <NutritionDemo />
    </ShowcaseSection>

    <ShowcaseSection
      eyebrow="Progress"
      title="Proof, not motivation posters."
      description="Strength trends, body metrics and PRs in one calm analytics surface."
      reverse
      tone="surface"
    >
      <ProgressDemo />
    </ShowcaseSection>

    <SectionWrapper
      tone="card"
      className={styles.servicesPreview}
      containerClassName={styles.servicesContainer}
    >
      <Reveal>
        <SectionTitle
          eyebrow="Plans"
          title="Choose how far you want to go."
          description="Start free, add automatic nutrition, or move into a fully personalized Elite plan."
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
    </SectionWrapper>

    <Reveal y={24}>
      <CtaBand
        eyebrow="Start"
        title="Begin with structure."
        description="Pick a plan and experience how RYZE turns training into a clear weekly system."
        primaryLabel="Join for Free"
        primaryTo="/services/generic-plan"
        secondaryLabel="View plans"
        secondaryTo="/services"
      />
    </Reveal>

    <Reveal y={24}>
      <ContactPreview />
    </Reveal>
  </PageWrapper>
);
