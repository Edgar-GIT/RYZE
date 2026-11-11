import { ContactSection } from "@/components/contact_section/contact_section";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import ui from "@/components/public_ui/public_ui.module.css";

export const ContactPage = () => (
  <PageWrapper className={ui.page}>
    <ContactSection />
  </PageWrapper>
);
