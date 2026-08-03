import { ArrowRight } from "lucide-react";

import { BrandIcon } from "@/components/brand_icon/brand_icon";
import { Button } from "@/components/button/button";
import { SectionWrapper } from "@/components/section_wrapper/section_wrapper";
import {
  CONTACT_EMAIL,
  CONTACT_INSTAGRAM_HANDLE,
  CONTACT_INSTAGRAM_URL
} from "@/constants/contact";

import styles from "./contact_preview.module.css";

const mailtoHref = `mailto:${CONTACT_EMAIL}`;

export const ContactPreview = () => (
  <SectionWrapper
    id="contact"
    tone="deep"
    className={styles.section}
    containerClassName={styles.inner}
  >
    <div className={styles.copy}>
      <p className={styles.eyebrow}>Get in touch</p>
      <h2>Questions, partnerships, or ready to start?</h2>
      <p className={styles.description}>
        Reach the team through the official channels — or open the full contact page to write
        directly to RYZE.
      </p>
    </div>

    <div className={styles.actions}>
      <a className={styles.chip} href={mailtoHref} aria-label={`Email ${CONTACT_EMAIL}`}>
        <BrandIcon name="gmail" />
        <span>Email</span>
      </a>
      <a
        className={styles.chip}
        href={CONTACT_INSTAGRAM_URL}
        target="_blank"
        rel="noreferrer"
        aria-label={`Instagram ${CONTACT_INSTAGRAM_HANDLE}`}
      >
        <BrandIcon name="instagram" />
        <span>Instagram</span>
      </a>
      <span className={styles.chipDisabled} aria-disabled="true" title="Coming soon">
        <BrandIcon name="whatsapp" />
        <span>WhatsApp</span>
      </span>
    </div>

    <Button className={styles.cta} to="/contact" icon={<ArrowRight />}>
      Contact us
    </Button>
  </SectionWrapper>
);
