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
import { type Copy, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./contact_preview.module.css";

const channels = [
  { icon: "gmail", label: "Email", detail: CONTACT_EMAIL, href: `mailto:${CONTACT_EMAIL}` },
  { icon: "instagram", label: "Instagram", detail: CONTACT_INSTAGRAM_HANDLE, href: CONTACT_INSTAGRAM_URL },
  { icon: "whatsapp", label: "WhatsApp", detail: CONTACT_WHATSAPP_LABEL, href: CONTACT_WHATSAPP_URL }
] as const;

const COPY: Copy<{
  label: string;
  heading: string;
  headingHighlight: string;
  lead: string;
  join: string;
  contact: string;
  channels: string;
  newTab: string;
}> = {
  en: {
    label: "Get in touch",
    heading: "Questions, partnerships, or ",
    headingHighlight: "ready to start?",
    lead: "Reach the team through the official channels — or open the full contact page to write directly to RYZE.",
    join: "Join for Free",
    contact: "Contact us",
    channels: "Official channels",
    newTab: " (opens in a new tab)"
  },
  pt: {
    label: "Fala connosco",
    heading: "Dúvidas, parcerias ou ",
    headingHighlight: "pronto para começar?",
    lead: "Fala com a equipa pelos canais oficiais — ou abre a página de contacto para escreveres diretamente à RYZE.",
    join: "Junta-te grátis",
    contact: "Contactar",
    channels: "Canais oficiais",
    newTab: " (abre num novo separador)"
  }
};

export const ContactPreview = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  return (
  <section
    id="contact"
    className={joinClassNames(ui.section, ui.toneNavy, styles.section)}
    aria-labelledby="contact-preview-heading"
  >
    <div className={joinClassNames(ui.container, ui.toneLight, ui.lightPanel, styles.layout)}>
      <Waves />
      <div className={styles.copy}>
        <p className={ui.label}>{text.label}</p>
        <h2 id="contact-preview-heading" className={ui.heading}>
          {text.heading}
          <span className={ui.hl}>{text.headingHighlight}</span>
        </h2>
        <p className={ui.lead}>{text.lead}</p>
        <div className={joinClassNames(ui.actions, styles.actions)}>
          <PublicAction to="/services/generic-program" variant="ink" icon={<ArrowRight />}>
            {text.join}
          </PublicAction>
          <PublicAction to="/contact" variant="ghostOnLight">
            {text.contact}
          </PublicAction>
        </div>
      </div>

      <ul className={styles.channels} aria-label={text.channels}>
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
                {isExternal ? <span className={ui.srOnly}>{text.newTab}</span> : null}
              </a>
            </li>
          );
        })}
      </ul>
    </div>
  </section>
);
};
