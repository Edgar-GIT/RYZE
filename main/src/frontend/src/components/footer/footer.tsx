import { Camera, Mail } from "lucide-react";
import { Link } from "react-router-dom";

import { BRAND_ASSETS } from "@/constants/brand_assets";
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
          <img src={BRAND_ASSETS.icon} alt="" aria-hidden="true" />
          <span>RYZE</span>
        </Link>
        <p>Professional fitness guidance made accessible, simple and automated.</p>
      </div>

      <address className={styles.contactColumn}>
        <a href={`mailto:${CONTACT_EMAIL}`}>
          <Mail aria-hidden="true" />
          <span>{CONTACT_EMAIL}</span>
        </a>
        <a href={CONTACT_INSTAGRAM_URL} target="_blank" rel="noreferrer">
          <Camera aria-hidden="true" />
          <span>{CONTACT_INSTAGRAM_HANDLE}</span>
        </a>
      </address>

      <div className={styles.socialColumn} aria-label="Future social links">
        <span className={styles.socialSlot} aria-hidden="true" />
        <span className={styles.socialSlot} aria-hidden="true" />
        <span className={styles.socialSlot} aria-hidden="true" />
      </div>

      <p className={styles.copyright}>
        © {currentYear} RYZE. All rights reserved.
      </p>
    </Container>
  </footer>
);
