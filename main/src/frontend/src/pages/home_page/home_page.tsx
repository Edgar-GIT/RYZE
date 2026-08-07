import { ContactPreview } from "@/components/contact_preview/contact_preview";
import { Container } from "@/components/container/container";
import { Hero } from "@/components/hero/hero";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PlatformSection } from "@/components/platform_section/platform_section";
import { Reveal } from "@/components/reveal/reveal";
import { SectionTitle } from "@/components/section_title/section_title";
import { ServiceCard } from "@/components/service_card/service_card";
import { SERVICE_PROGRAMS } from "@/constants/services";
import servicesBackground from "@resources/img/hero/bg_services.png";
import technologyBackground from "@resources/img/hero/tech_bg.png";

import { Activity, BarChart3, Boxes, Brain, ChartNoAxesCombined, Crosshair, Dumbbell, Leaf, Rocket, ShieldCheck, Sparkles, User, UserPlus, Users, Utensils, Zap } from "lucide-react";

import styles from "./home_page.module.css";

const techFeatures = [
  {
    icon: Boxes,
    title: "Connected Modules",
    text: "Everything works together. Nothing works alone."
  },
  {
    icon: Zap,
    title: "Clear Interfaces",
    text: "Designed for people. Built for precision."
  },
  {
    icon: Sparkles,
    title: "Smart Automation",
    text: "Less manual work. More human impact."
  }
] as const;

export const HomePage = () => (
  <PageWrapper>
    <Hero />

    <section className={`${styles.section} ${styles.sectionTech}`}>
      <img
        className={styles.techBackground}
        src={technologyBackground}
        alt=""
        aria-hidden="true"
      />

      <Container className={styles.techLayout}>
        <Reveal className={styles.techIntro}>
          <p className={styles.techEyebrow}>Technology first</p>
          <h2>
            Built like <span className={styles.accentPhrase}>modern</span> software.
          </h2>
          <p className={styles.sectionLead}>
            Connected modules. Clear interfaces.
            <br />
            Automation where it helps. Intelligence where it matters.
          </p>

          <ul className={styles.techFeatures}>
            {techFeatures.map((feature) => {
              const Icon = feature.icon;

              return (
                <li key={feature.title} className={styles.techFeature}>
                  <span className={styles.techFeatureIcon}>
                    <Icon aria-hidden="true" />
                  </span>
                  <div>
                    <strong>{feature.title}</strong>
                    <p>{feature.text}</p>
                  </div>
                </li>
              );
            })}
          </ul>
        </Reveal>
      </Container>
    </section>

    <PlatformSection />

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

    <Reveal y={24}>
      <ContactPreview />
    </Reveal>
  </PageWrapper>
);
