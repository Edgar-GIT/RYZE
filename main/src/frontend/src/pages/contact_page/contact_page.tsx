import { ContactSection } from "@/components/contact_section/contact_section";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";

import styles from "./contact_page.module.css";

export const ContactPage = () => (
  <PageWrapper className={styles.page}>
    <ContactSection compact />
  </PageWrapper>
);
