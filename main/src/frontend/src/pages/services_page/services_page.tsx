import { Dumbbell, Heart, Trophy, Zap } from "lucide-react";

import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { Reveal } from "@/components/reveal/reveal";
import { ServiceCard } from "@/components/service_card/service_card";
import { SERVICE_PROGRAMS } from "@/constants/services";
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
        <Reveal className={styles.intro}>
          <p className={styles.eyebrow}>Catalog</p>
          <h1>
            Choose how you want to <span className={styles.accent}>RYZE</span>
          </h1>
          <p className={styles.lead}>
            Compare the initial RYZE offer and choose the route that fits the guidance and
            personalization you need.
          </p>
        </Reveal>

        <div className={styles.grid}>
          {SERVICE_PROGRAMS.map((service, index) => (
            <Reveal key={service.title} delay={0.1 + index * 0.08}>
              <ServiceCard service={service} />
            </Reveal>
          ))}
        </div>

        <Reveal className={styles.evolution} delay={0.18}>
          {evolutionItems.map((item, index) => {
            const Icon = item.icon;

            return (
              <article
                key={item.title}
                className={styles.evolutionItem}
              >
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
