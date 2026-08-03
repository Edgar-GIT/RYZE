import { ArrowRight, Construction, Home } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";

import styles from "./under_development_page.module.css";

interface UnderDevelopmentPageProps {
  title: string;
  description: string;
  eyebrow?: string;
}

export const UnderDevelopmentPage = ({
  title,
  description,
  eyebrow = "In progress"
}: UnderDevelopmentPageProps) => (
  <PageWrapper>
    <section className={styles.section}>
      <Container className={styles.inner} size="narrow">
        <div className={styles.iconWrap}>
          <Construction aria-hidden="true" />
        </div>
        <div className={styles.copy}>
          <p>{eyebrow}</p>
          <h1>{title}</h1>
          <span>{description}</span>
        </div>
        <div className={styles.actions}>
          <Button to="/" variant="primary" icon={<Home />} iconPosition="left">
            Back Home
          </Button>
          <Button to="/services" variant="secondary" icon={<ArrowRight />}>
            View Services
          </Button>
        </div>
      </Container>
    </section>
  </PageWrapper>
);
