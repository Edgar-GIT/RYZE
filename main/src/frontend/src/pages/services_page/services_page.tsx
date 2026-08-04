import { ArrowRight } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { Reveal } from "@/components/reveal/reveal";
import { SectionTitle } from "@/components/section_title/section_title";
import { ServiceCard } from "@/components/service_card/service_card";
import { SERVICE_PLANS } from "@/constants/services";

import styles from "./services_page.module.css";

export const ServicesPage = () => (
  <PageWrapper>
    <section className={styles.hero}>
      <Container className={styles.heroInner}>
        <Reveal className={styles.copy}>
          <p>RYZE services</p>
          <h1>Three plan levels. One clean starting point.</h1>
          <span>
            Compare the initial RYZE offer and select the route that fits the amount of guidance and
            personalization you need.
          </span>
        </Reveal>
        <Reveal delay={0.1}>
          <Button to="/contact" variant="secondary" icon={<ArrowRight />}>
            Ask a Question
          </Button>
        </Reveal>
      </Container>
    </section>

    <section className={styles.plans}>
      <Container className={styles.planInner}>
        <Reveal>
          <SectionTitle
            eyebrow="Catalog"
            title="Available RYZE plan categories."
            description="The product pages are prepared as routes and will receive backend-backed details in a later implementation."
          />
        </Reveal>
        <div className={styles.grid}>
          {SERVICE_PLANS.map((service, index) => (
            <Reveal key={service.title} delay={index * 0.08}>
              <ServiceCard service={service} />
            </Reveal>
          ))}
        </div>
      </Container>
    </section>
  </PageWrapper>
);
