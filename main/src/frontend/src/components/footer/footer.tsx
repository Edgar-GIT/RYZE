import { Link } from "react-router-dom";

import { Container } from "@/components/container/container";
import { BRAND_ASSETS } from "@/constants/brand_assets";
import { CONTACT_EMAIL, CONTACT_INSTAGRAM_URL, CONTACT_WHATSAPP_URL } from "@/constants/contact";

import styles from "./footer.module.css";

const currentYear = new Date().getFullYear();

const footerColumns = [
  {
    title: "Platform",
    links: [
      { label: "Training engine", to: "/services" },
      { label: "Nutrition", to: "/services/premium-level-1" },
      { label: "Analytics", to: "/profile" },
      { label: "Coaches", to: "/about-us" },
      { label: "Mobile app", to: "/profile" }
    ]
  },
  {
    title: "Company",
    links: [
      { label: "About", to: "/about-us" },
      { label: "Careers", to: "/about-us" },
      { label: "Press", to: "/about-us" },
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

export const Footer = () => (
  <footer className={styles.footer}>
    <Container className={styles.inner}>
      <div className={styles.main}>
        <div className={styles.brandArea}>
          <Link className={styles.brand} to="/" aria-label="RYZE home">
            <img className={styles.footerLogo} src={BRAND_ASSETS.slogan} alt="RYZE" />
          </Link>
          <p>
            Premium fitness technology. Human coaching, adaptive programming and
            nutrition intelligence in one system.
          </p>
        </div>

        <nav className={styles.columns} aria-label="Footer navigation">
          {footerColumns.map((column) => (
            <div className={styles.column} key={column.title}>
              <h2>{column.title}</h2>
              <ul>
                {column.links.map((link) => (
                  <li key={link.label}>
                    {"href" in link ? (
                      <a
                        href={link.href}
                        target={link.href.startsWith("http") ? "_blank" : undefined}
                        rel={link.href.startsWith("http") ? "noreferrer" : undefined}
                      >
                        {link.label}
                      </a>
                    ) : (
                      <Link to={link.to}>{link.label}</Link>
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
          © {currentYear} RYZE - All rights reserved
        </p>

        <div className={styles.utilities} aria-label="Legal links">
          {utilityLinks.map((link) => (
            <Link key={link} to="/about-us">
              {link}
            </Link>
          ))}
        </div>
      </div>
    </Container>
  </footer>
);
