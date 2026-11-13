import { Link } from "react-router-dom";

import { Container } from "@/components/container/container";
import { BRAND_ASSETS } from "@/constants/brand_assets";
import { CONTACT_EMAIL, CONTACT_INSTAGRAM_URL, CONTACT_WHATSAPP_URL } from "@/constants/contact";
import { useI18n } from "@/i18n/i18n";

import styles from "./footer.module.css";

const currentYear = new Date().getFullYear();

const footerColumns = [
  {
    title: "Platform",
    links: [
      { label: "Training engine", to: "/services" },
      { label: "Nutrition", to: "/services/premium-level-1" },
      { label: "Analytics", to: "/profile" },
      { label: "Coaches", to: "/our-vision" },
      { label: "Mobile app", to: "/profile" }
    ]
  },
  {
    title: "Company",
    links: [
      { label: "Our Vision", to: "/our-vision" },
      { label: "Careers", to: "/our-vision" },
      { label: "Press", to: "/our-vision" },
      { label: "Partners", to: "/contact" },
      { label: "Contact", href: `mailto:${CONTACT_EMAIL}` }
    ]
  },
  {
    title: "Resources",
    links: [
      { label: "Method", to: "/services" },
      { label: "Science", to: "/services" },
      { label: "Instagram", href: CONTACT_INSTAGRAM_URL },
      { label: "WhatsApp", href: CONTACT_WHATSAPP_URL },
      { label: "Help centre", to: "/contact" },
      { label: "Changelog", to: "/feedback" }
    ]
  }
] as const;

const utilityLinks = ["Privacy", "Terms", "Cookies"] as const;

const DESCRIPTION =
  "Premium fitness technology. Human coaching, adaptive programming and nutrition intelligence in one system.";

// The footer is a protected part: the English copy above stays exactly as the
// owner set it. Portuguese visitors see the same footer with these labels.
const PT_LABELS: Record<string, string> = {
  Platform: "Plataforma",
  "Training engine": "Motor de treino",
  Nutrition: "Nutrição",
  Analytics: "Análise",
  Coaches: "Treinadores",
  "Mobile app": "App móvel",
  Company: "Empresa",
  "Our Vision": "A nossa visão",
  Careers: "Carreiras",
  Press: "Imprensa",
  Partners: "Parceiros",
  Contact: "Contacto",
  Resources: "Recursos",
  Method: "Método",
  Science: "Ciência",
  "Help centre": "Centro de ajuda",
  Changelog: "Novidades",
  Privacy: "Privacidade",
  Terms: "Termos",
  Cookies: "Cookies",
  [DESCRIPTION]:
    "Tecnologia de fitness premium. Acompanhamento humano, programação adaptativa e inteligência nutricional num só sistema.",
  "RYZE home": "Início RYZE",
  "Footer navigation": "Navegação do rodapé",
  "Legal links": "Links legais",
  "All rights reserved": "Todos os direitos reservados"
};

export const Footer = () => {
  const { locale } = useI18n();
  const l = (text: string) => (locale === "pt" ? (PT_LABELS[text] ?? text) : text);
  return (
  <footer className={styles.footer}>
    <Container className={styles.inner}>
      <div className={styles.main}>
        <div className={styles.brandArea}>
          <Link className={styles.brand} to="/" aria-label={l("RYZE home")}>
            <img className={styles.footerLogo} src={BRAND_ASSETS.slogan} alt="RYZE" />
          </Link>
          <p>{l(DESCRIPTION)}</p>
        </div>

        <nav className={styles.columns} aria-label={l("Footer navigation")}>
          {footerColumns.map((column) => (
            <div className={styles.column} key={column.title}>
              <h2>{l(column.title)}</h2>
              <ul>
                {column.links.map((link) => (
                  <li key={link.label}>
                    {"href" in link ? (
                      <a
                        href={link.href}
                        target={link.href.startsWith("http") ? "_blank" : undefined}
                        rel={link.href.startsWith("http") ? "noreferrer" : undefined}
                      >
                        {l(link.label)}
                      </a>
                    ) : (
                      <Link to={link.to}>{l(link.label)}</Link>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </nav>
      </div>

      <div className={styles.footerBottom}>
        <p className={styles.copyright}>
          © {currentYear} RYZE - {l("All rights reserved")}
        </p>

        <div className={styles.utilities} aria-label={l("Legal links")}>
          {utilityLinks.map((link) => (
            <Link key={link} to="/our-vision">
              {l(link)}
            </Link>
          ))}
        </div>
      </div>
    </Container>
  </footer>
);
};
