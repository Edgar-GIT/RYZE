import { ArrowUpRight } from "lucide-react";

import { BrandIcon } from "@/components/brand_icon/brand_icon";
import { ContactForm } from "@/components/contact_form/contact_form";
import ui from "@/components/public_ui/public_ui.module.css";
import {
  CONTACT_EMAIL,
  CONTACT_INSTAGRAM_HANDLE,
  CONTACT_INSTAGRAM_URL,
  CONTACT_WHATSAPP_LABEL,
  CONTACT_WHATSAPP_URL
} from "@/constants/contact";
import { joinClassNames } from "@utils/class_names";

import styles from "./contact_section.module.css";

const channels = [
  { icon: "gmail", name: "Email", detail: CONTACT_EMAIL, href: `mailto:${CONTACT_EMAIL}` },
  { icon: "instagram", name: "Instagram", detail: CONTACT_INSTAGRAM_HANDLE, href: CONTACT_INSTAGRAM_URL },
  { icon: "whatsapp", name: "WhatsApp", detail: CONTACT_WHATSAPP_LABEL, href: CONTACT_WHATSAPP_URL }
] as const;

export const ContactSection = () => (
  <section className={joinClassNames(ui.toneNavy, styles.section)} id="contact" aria-labelledby="contact-heading">
    <div className={joinClassNames(ui.container, styles.layout)}>
      <div className={styles.copy}>
        <p className={ui.label}>Contact</p>
        <h1 id="contact-heading" className={ui.display}>
          Talk to <span className={ui.hl}>RYZE.</span>
        </h1>
        <p className={ui.lead}>
          We&apos;re always available to answer questions, discuss partnerships or help you start
          your fitness journey with a clear program.
        </p>

        <ul className={styles.channels} aria-label="Official channels">
          {channels.map((channel) => {
            const isExternal = channel.href.startsWith("http");

            return (
              <li key={channel.name}>
                <a
                  className={joinClassNames(styles.row, ui.focusRing)}
                  href={channel.href}
                  target={isExternal ? "_blank" : undefined}
                  rel={isExternal ? "noreferrer" : undefined}
                >
                  <span className={styles.rowIcon}>
                    <BrandIcon name={channel.icon} />
                  </span>
                  <span className={styles.rowCopy}>
                    <strong>{channel.name}</strong>
                    <small>{channel.detail}</small>
                  </span>
                  <ArrowUpRight className={styles.rowArrow} aria-hidden="true" />
                  {isExternal ? <span className={ui.srOnly}> (opens in a new tab)</span> : null}
                </a>
              </li>
            );
          })}
        </ul>
      </div>

      <div className={styles.formColumn}>
        <ContactForm />
      </div>
    </div>
  </section>
);
