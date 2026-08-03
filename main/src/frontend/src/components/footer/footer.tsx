import { Link } from "react-router-dom";

import { BrandIcon } from "@/components/brand_icon/brand_icon";
import {
  CONTACT_EMAIL,
  CONTACT_INSTAGRAM_HANDLE,
  CONTACT_INSTAGRAM_URL
} from "@/constants/contact";
import { BRAND_ASSETS } from "@/constants/brand_assets";
import { Container } from "@/components/container/container";

import styles from "./footer.module.css";

const currentYear = new Date().getFullYear();

export const Footer = () => (
  <footer className={styles.footer}>
    <Container className={styles.inner}>
      <div className={styles.footerTop}>
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <img className={styles.footerLogo} src={BRAND_ASSETS.slogan} alt="RYZE" />
        </Link>

        <div className={styles.footerContact}>
          <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>
          <a href={CONTACT_INSTAGRAM_URL} target="_blank" rel="noreferrer">
            {CONTACT_INSTAGRAM_HANDLE}
          </a>
        </div>
      </div>

      <div className={styles.footerBottom}>
        <p className={styles.copyright}>
          © {currentYear} RYZE. All rights reserved.
        </p>

        <div className={styles.socialArea}>
          <div className={styles.socialLinks} aria-label="Social links">
            <a
              className={styles.socialButton}
              href={`mailto:${CONTACT_EMAIL}`}
              aria-label="Email RYZE"
            >
              <BrandIcon name="gmail" className={styles.iconSvg} />
            </a>
            <a
              className={styles.socialButton}
              href={CONTACT_INSTAGRAM_URL}
              target="_blank"
              rel="noreferrer"
              aria-label="Open RYZE Instagram"
            >
              <BrandIcon name="instagram" className={styles.iconSvg} />
            </a>
            <button
              className={styles.socialButton}
              type="button"
              aria-label="WhatsApp contact prepared"
              disabled
            >
              <BrandIcon name="whatsapp" className={styles.iconSvg} />
            </button>
          </div>
          <div className={styles.futureLinks} aria-label="Future social links placeholder">
            <span />
            <span />
            <span />
          </div>
        </div>
      </div>
    </Container>
  </footer>
);
