import { useEffect, useRef, useState } from "react";
import { SendHorizontal, X } from "lucide-react";

import { BrandIcon } from "@/components/brand_icon/brand_icon";
import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
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
  const nameInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!isEmailPanelOpen) {
      return;
    }

    nameInputRef.current?.focus();

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsEmailPanelOpen(false);
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [isEmailPanelOpen]);

  return (
    <section className={styles.section} id="contact">
      <Container className={styles.inner}>
        <SectionTitle
          eyebrow="Contact"
          title="Talk to RYZE."
          description="Use the official channels while the platform prepares its full account and support flows."
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

      {isEmailPanelOpen ? (
        <div
          className={styles.emailOverlay}
          role="presentation"
          onMouseDown={() => setIsEmailPanelOpen(false)}
        >
          <div
            className={styles.emailPanel}
            role="dialog"
            aria-modal="true"
            aria-labelledby="email-panel-title"
            onMouseDown={(event) => event.stopPropagation()}
          >
            <div className={styles.emailPanelHeader}>
              <div>
                <span className={styles.emailPanelEyebrow}>Direct email</span>
                <h3 id="email-panel-title">Write to RYZE</h3>
              </div>
              <button
                className={styles.closeButton}
                type="button"
                onClick={() => setIsEmailPanelOpen(false)}
                aria-label="Close email panel"
              >
                <X aria-hidden="true" />
              </button>
            </div>

            <form
              className={styles.emailForm}
              onSubmit={(event) => event.preventDefault()}
            >
              <label>
                <span>Name</span>
                <input ref={nameInputRef} type="text" name="name" autoComplete="name" />
              </label>
              <label>
                <span>Email</span>
                <input type="email" name="email" autoComplete="email" />
              </label>
              <label>
                <span>Subject</span>
                <input type="text" name="subject" />
              </label>
              <label className={styles.messageField}>
                <span>Message</span>
                <textarea name="message" rows={7} />
              </label>

              <div className={styles.emailPanelFooter}>
                <small>{CONTACT_EMAIL}</small>
                <Button type="submit" icon={<SendHorizontal />}>
                  Prepare message
                </Button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </section>
  );
};
