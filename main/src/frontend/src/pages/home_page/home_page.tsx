import { CtaBand } from "@/components/cta_band/cta_band";
import { ContactSection } from "@/components/contact_section/contact_section";
import { Container } from "@/components/container/container";
import { Hero } from "@/components/hero/hero";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { SectionTitle } from "@/components/section_title/section_title";
import { ServiceCard } from "@/components/service_card/service_card";
import { SERVICE_PLANS } from "@/constants/services";

import styles from "./home_page.module.css";

export const HomePage = () => (
  <PageWrapper>
    <Hero />

    <section className={styles.servicesPreview}>
      <Container className={styles.servicesContainer}>
        <SectionTitle
          eyebrow="Services"
          title="Choose the level of guidance that matches your start."
          description="RYZE begins with three clear plan levels, from structured generic training to fully personalized training and nutrition."
          align="center"
        />
        <div className={styles.serviceGrid}>
          {SERVICE_PLANS.map((service) => (
            <ServiceCard key={service.title} service={service} />
          ))}
        </div>
      </Container>
    </section>

    <CtaBand
      eyebrow="Start with structure"
      title="Stop guessing what to train next."
      description="Pick a plan level and move through a focused digital experience designed to make the first step simple."
      primaryLabel="View Services"
      primaryTo="/services"
      secondaryLabel="Contact RYZE"
      secondaryTo="/contact"
    />

    <CtaBand
      eyebrow="Built to scale"
      title="A platform ready for trainers and automation."
      description="RYZE is designed to grow into a marketplace where certified trainers can publish, manage and improve plan delivery."
      primaryLabel="About the Platform"
      primaryTo="/about-us"
      secondaryLabel="Community Feedback"
      secondaryTo="/feedback"
    />

    <ContactSection />
  </PageWrapper>
);
