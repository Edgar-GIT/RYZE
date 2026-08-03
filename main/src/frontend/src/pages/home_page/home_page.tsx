import { CoachesSection } from "@/components/coaches_section/coaches_section";
import { ContactSection } from "@/components/contact_section/contact_section";
import { FaqSection } from "@/components/faq_section/faq_section";
import { Hero } from "@/components/hero/hero";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { SectionTitle } from "@/components/section_title/section_title";
import { SectionWrapper } from "@/components/section_wrapper/section_wrapper";
import { ServiceCard } from "@/components/service_card/service_card";
import { TestimonialsSection } from "@/components/testimonials_section/testimonials_section";
import { WhyChooseRyze } from "@/components/why_choose_ryze/why_choose_ryze";
import { SERVICE_PLANS } from "@/constants/services";

import styles from "./home_page.module.css";

export const HomePage = () => (
  <PageWrapper>
    <Hero />

    <SectionWrapper
      className={styles.servicesPreview}
      containerClassName={styles.servicesContainer}
    >
      <SectionTitle
        eyebrow="Services"
        title="Three ways to begin with structure."
        description="Start simple, add automatic nutrition guidance or move into a fully personalized plan prepared with trainer oversight."
        align="center"
      />
      <div className={styles.serviceGrid}>
        {SERVICE_PLANS.map((service) => (
          <ServiceCard key={service.title} service={service} />
        ))}
      </div>
    </SectionWrapper>

    <WhyChooseRyze />
    <CoachesSection />
    <TestimonialsSection />
    <FaqSection />
    <ContactSection />
  </PageWrapper>
);
