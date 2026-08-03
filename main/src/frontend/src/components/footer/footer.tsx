import { Link } from "react-router-dom";

import { BrandIcon } from "@/components/brand_icon/brand_icon";
import { BrandMark } from "@/components/brand_mark/brand_mark";
import {
  CONTACT_EMAIL,
  CONTACT_INSTAGRAM_URL
} from "@/constants/contact";
import { Container } from "@/components/container/container";

import styles from "./footer.module.css";

const currentYear = new Date().getFullYear();

export const Footer = () => (
  <footer className={styles.footer}>
    <Container className={styles.inner}>
      <div className={styles.footerTop}>
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="footer" />
          <span>RYZE</span>
        </Link>

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
      </div>

      <p className={styles.copyright}>
        © {currentYear} RYZE. All rights reserved.
      </p>
    </Container>
  </footer>
);
