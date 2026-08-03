import { Link } from "react-router-dom";

import { BrandIcon } from "@/components/brand_icon/brand_icon";
import { BrandMark } from "@/components/brand_mark/brand_mark";
import {
  CONTACT_EMAIL,
  CONTACT_INSTAGRAM_HANDLE,
  CONTACT_INSTAGRAM_URL
} from "@/constants/contact";
import { Container } from "@/components/container/container";

import styles from "./footer.module.css";

const currentYear = new Date().getFullYear();

export const Footer = () => (
  <footer className={styles.footer}>
    <Container className={styles.inner}>
      <div className={styles.brandColumn}>
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="footer" />
          <span>RYZE</span>
        </Link>
        <p>
          Professional fitness guidance made accessible, simple and automated
          through a clean digital experience.
        </p>
      </div>

      <address className={styles.contactColumn}>
        <a className={styles.contactLink} href={`mailto:${CONTACT_EMAIL}`}>
          <span className={styles.contactIcon}>
            <BrandIcon name="gmail" className={styles.iconSvg} />
          </span>
          <span>
            <strong>Email</strong>
            <small>{CONTACT_EMAIL}</small>
          </span>
        </a>
        <a className={styles.contactLink} href={CONTACT_INSTAGRAM_URL} target="_blank" rel="noreferrer">
          <span className={styles.contactIcon}>
            <BrandIcon name="instagram" className={styles.iconSvg} />
          </span>
          <span>
            <strong>Instagram</strong>
            <small>{CONTACT_INSTAGRAM_HANDLE}</small>
          </span>
        </a>
        <span className={styles.contactPending}>
          <span className={styles.contactIcon}>
            <BrandIcon name="whatsapp" className={styles.iconSvg} />
          </span>
          <span>
            <strong>WhatsApp</strong>
            <small>Prepared for activation</small>
          </span>
        </span>
      </address>

      <div className={styles.futureSocials} aria-label="Future social links">
        <span>Future social links</span>
        <div>
          <span aria-hidden="true" />
          <span aria-hidden="true" />
          <span aria-hidden="true" />
        </div>
      </div>

      <p className={styles.copyright}>
        © {currentYear} RYZE. All rights reserved.
      </p>
    </Container>
  </footer>
);
