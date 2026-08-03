import { useCallback, useState } from "react";

import { BrandIcon } from "@/components/brand_icon/brand_icon";
import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { EmailContactPanel } from "@/components/email_contact_panel/email_contact_panel";
import { SectionTitle } from "@/components/section_title/section_title";
import {
  CONTACT_EMAIL,
  CONTACT_INSTAGRAM_HANDLE,
  CONTACT_INSTAGRAM_URL
} from "@/constants/contact";

import styles from "./contact_section.module.css";

interface ContactSectionProps {
  compact?: boolean;
}

export const ContactSection = ({ compact = false }: ContactSectionProps) => {
  const [isEmailPanelOpen, setIsEmailPanelOpen] = useState(false);
  const closeEmailPanel = useCallback(() => setIsEmailPanelOpen(false), []);

  return (
    <section className={styles.section} id="contact">
      <Container className={styles.inner}>
        <SectionTitle
          eyebrow="Contact"
          title="Talk to RYZE."
          description="Use the official channels or open the direct email panel to draft a message with an attachment ready for future delivery."
          className={compact ? styles.compactTitle : undefined}
        />

        <div className={styles.contactGrid}>
          <button
            className={styles.contactItem}
            type="button"
            onClick={() => setIsEmailPanelOpen(true)}
          >
            <span className={styles.icon}>
              <BrandIcon name="gmail" />
            </span>
            <span>
              <strong>Email</strong>
              <small>{CONTACT_EMAIL}</small>
            </span>
          </button>

          <a
            className={styles.contactItem}
            href={CONTACT_INSTAGRAM_URL}
            target="_blank"
            rel="noreferrer"
          >
            <span className={styles.icon}>
              <BrandIcon name="instagram" />
            </span>
            <span>
              <strong>Instagram</strong>
              <small>{CONTACT_INSTAGRAM_HANDLE}</small>
            </span>
          </a>

          <div className={styles.whatsappItem}>
            <span className={styles.icon}>
              <BrandIcon name="whatsapp" />
            </span>
            <span>
              <strong>WhatsApp</strong>
              <small>Channel prepared for activation.</small>
            </span>
            <Button disabled variant="secondary" icon={<BrandIcon name="whatsapp" />}>
              WhatsApp
            </Button>
          </div>
        </div>
      </Container>

      {isEmailPanelOpen ? <EmailContactPanel onClose={closeEmailPanel} /> : null}
    </section>
  );
};
