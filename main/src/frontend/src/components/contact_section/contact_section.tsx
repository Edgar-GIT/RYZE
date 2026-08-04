import { ArrowUpRight } from "lucide-react";

import { BrandIcon } from "@/components/brand_icon/brand_icon";
import { ContactForm } from "@/components/contact_form/contact_form";
import { Container } from "@/components/container/container";
import {
  CONTACT_EMAIL,
  CONTACT_INSTAGRAM_HANDLE,
  CONTACT_INSTAGRAM_URL,
  CONTACT_WHATSAPP_LABEL,
  CONTACT_WHATSAPP_URL
} from "@/constants/contact";

import styles from "./contact_section.module.css";

const mailtoHref = `mailto:${CONTACT_EMAIL}`;

export const ContactSection = () => (
  <section className={styles.section} id="contact" aria-labelledby="contact-heading">
    <Container className={styles.layout}>
      <div className={styles.copy}>
        <p className={styles.eyebrow}>Contact</p>
        <h1 id="contact-heading" className={styles.title}>
          Talk to RYZE.
        </h1>
        <p className={styles.description}>
          We&apos;re always available to answer questions, discuss partnerships or help you start
          your fitness journey with a clear plan.
        </p>

        <div className={styles.channels} role="list">
          <a className={styles.row} href={mailtoHref} role="listitem">
            <span className={styles.rowIcon}>
              <BrandIcon name="gmail" />
            </span>
            <span className={styles.rowCopy}>
              <strong>Email</strong>
              <small>{CONTACT_EMAIL}</small>
            </span>
            <ArrowUpRight className={styles.rowArrow} aria-hidden="true" />
          </a>

          <a
            className={styles.row}
            href={CONTACT_INSTAGRAM_URL}
            target="_blank"
            rel="noreferrer"
            role="listitem"
          >
            <span className={styles.rowIcon}>
              <BrandIcon name="instagram" />
            </span>
            <span className={styles.rowCopy}>
              <strong>Instagram</strong>
              <small>{CONTACT_INSTAGRAM_HANDLE}</small>
            </span>
            <ArrowUpRight className={styles.rowArrow} aria-hidden="true" />
          </a>

          <a
            className={styles.row}
            href={CONTACT_WHATSAPP_URL}
            target="_blank"
            rel="noreferrer"
            role="listitem"
          >
            <span className={styles.rowIcon}>
              <BrandIcon name="whatsapp" />
            </span>
            <span className={styles.rowCopy}>
              <strong>WhatsApp</strong>
              <small>{CONTACT_WHATSAPP_LABEL}</small>
            </span>
            <ArrowUpRight className={styles.rowArrow} aria-hidden="true" />
          </a>
        </div>
      </div>

      <div className={styles.formColumn}>
        <ContactForm />
      </div>
    </Container>
  </section>
);
