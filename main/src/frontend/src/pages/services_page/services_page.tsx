import {
  Award,
  Headphones,
  Lock,
  RefreshCw,
  ShieldCheck,
  Zap
} from "lucide-react";

import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { Reveal } from "@/components/reveal/reveal";
import { ServiceCard } from "@/components/service_card/service_card";
import { SERVICE_PROGRAMS } from "@/constants/services";
import servicesBackground from "@resources/img/hero/bg_services.png";

import styles from "./services_page.module.css";

const catalogHighlights = [
  {
    icon: ShieldCheck,
    title: "Built by experts",
    text: "Professionally designed training and nutrition programs."
  },
  {
    icon: Zap,
    title: "Instant access",
    text: "Start your journey immediately."
  },
  {
    icon: RefreshCw,
    title: "Always evolving",
    text: "Programs are updated continuously."
  }
] as const;

const trustBadges = [
  {
    icon: Lock,
    title: "Secure & private",
    text: "Your data is safe with us."
  },
  {
    icon: Award,
    title: "Satisfaction guarantee",
    text: "Not satisfied? We'll make it right."
  },
  {
    icon: Headphones,
    title: "Support 24/7",
    text: "We're here whenever you need us."
  }
] as const;

export const ServicesPage = () => (
  <PageWrapper className={styles.page}>
    <section className={styles.catalog}>
      <img
        className={styles.catalogBackground}
        src={servicesBackground}
        alt=""
        aria-hidden="true"
      />

      <Container className={styles.catalogInner}>
        <div className={styles.headerRow}>
          <Reveal className={styles.intro}>
            <p className={styles.eyebrow}>Catalog</p>
            <h1>
              Available RYZE program <span className={styles.accent}>categories.</span>
            </h1>
            <p className={styles.lead}>
              Compare the initial RYZE offer and choose the route that fits the guidance and
              personalization you need.
            </p>
          </Reveal>

          <Reveal className={styles.highlights} delay={0.08}>
            {catalogHighlights.map((item) => {
              const Icon = item.icon;

              return (
                <article key={item.title} className={styles.highlight}>
                  <span className={styles.highlightIcon}>
                    <Icon aria-hidden="true" strokeWidth={1.7} />
                  </span>
                  <div>
                    <strong>{item.title}</strong>
                    <p>{item.text}</p>
                  </div>
                </article>
              );
            })}
          </Reveal>
        </div>

        <div className={styles.grid}>
          {SERVICE_PROGRAMS.map((service, index) => (
            <Reveal key={service.title} delay={0.1 + index * 0.08}>
              <ServiceCard service={service} />
            </Reveal>
          ))}
        </div>

        <Reveal className={styles.trust} delay={0.18}>
          {trustBadges.map((item) => {
            const Icon = item.icon;

            return (
              <article key={item.title} className={styles.trustItem}>
                <span className={styles.trustIcon}>
                  <Icon aria-hidden="true" strokeWidth={1.7} />
                </span>
                <div>
                  <strong>{item.title}</strong>
                  <p>{item.text}</p>
                </div>
              </article>
            );
          })}
        </Reveal>
      </Container>
    </section>
  </PageWrapper>
);
