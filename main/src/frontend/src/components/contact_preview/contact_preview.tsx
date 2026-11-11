import { ArrowRight, ArrowUpRight } from "lucide-react";

import { BrandIcon } from "@/components/brand_icon/brand_icon";
import { PublicAction, Waves } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import {
  CONTACT_EMAIL,
  CONTACT_INSTAGRAM_HANDLE,
  CONTACT_INSTAGRAM_URL,
  CONTACT_WHATSAPP_LABEL,
  CONTACT_WHATSAPP_URL
} from "@/constants/contact";
import { joinClassNames } from "@utils/class_names";

import styles from "./contact_preview.module.css";

const channels = [
  { icon: "gmail", label: "Email", detail: CONTACT_EMAIL, href: `mailto:${CONTACT_EMAIL}` },
  { icon: "instagram", label: "Instagram", detail: CONTACT_INSTAGRAM_HANDLE, href: CONTACT_INSTAGRAM_URL },
  { icon: "whatsapp", label: "WhatsApp", detail: CONTACT_WHATSAPP_LABEL, href: CONTACT_WHATSAPP_URL }
] as const;

export const ContactPreview = () => (
  <section
    id="contact"
    className={joinClassNames(ui.section, ui.toneNavy, styles.section)}
    aria-labelledby="contact-preview-heading"
  >
    <div className={joinClassNames(ui.container, ui.toneLight, ui.lightPanel, styles.layout)}>
      <Waves />
      <div className={styles.copy}>
        <p className={ui.label}>Get in touch</p>
        <h2 id="contact-preview-heading" className={ui.heading}>
          Questions, partnerships, or <span className={ui.hl}>ready to start?</span>
        </h2>
        <p className={ui.lead}>
          Reach the team through the official channels — or open the full contact page to write
          directly to RYZE.
        </p>
        <div className={joinClassNames(ui.actions, styles.actions)}>
          <PublicAction to="/services/generic-program" variant="ink" icon={<ArrowRight />}>
            Join for Free
          </PublicAction>
          <PublicAction to="/contact" variant="ghostOnLight">
            Contact us
          </PublicAction>
        </div>
      </div>

      <ul className={styles.channels} aria-label="Official channels">
        {channels.map((channel) => {
          const isExternal = channel.href.startsWith("http");

          return (
            <li key={channel.label}>
              <a
                className={joinClassNames(styles.channel, ui.focusRing)}
                href={channel.href}
                target={isExternal ? "_blank" : undefined}
                rel={isExternal ? "noreferrer" : undefined}
              >
                <span className={styles.channelIcon}>
                  <BrandIcon name={channel.icon} />
                </span>
                <span className={styles.channelCopy}>
                  <strong>{channel.label}</strong>
                  <small>{channel.detail}</small>
                </span>
                <ArrowUpRight className={styles.channelArrow} aria-hidden="true" />
                {isExternal ? <span className={ui.srOnly}> (opens in a new tab)</span> : null}
              </a>
            </li>
          );
        })}
      </ul>
    </div>
  </section>
);
