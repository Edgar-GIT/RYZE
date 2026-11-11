import { ArrowRight, Construction } from "lucide-react";

import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { BrandGlobe } from "@/components/brand_globe/brand_globe";
import { joinClassNames } from "@utils/class_names";

import styles from "./under_development_page.module.css";

interface UnderDevelopmentPageProps {
  title: string;
  description: string;
  eyebrow?: string;
}

export const UnderDevelopmentPage = ({
  title,
  description,
  eyebrow = "In progress"
}: UnderDevelopmentPageProps) => (
  <PageWrapper className={ui.page}>
    <section className={joinClassNames(ui.toneNavy, styles.section)} aria-labelledby="under-development-heading">
      <div className={joinClassNames(ui.container, styles.layout)}>
        <div className={styles.copy}>
          <p className={styles.status}>
            <Construction aria-hidden="true" />
            {eyebrow}
          </p>
          <h1 id="under-development-heading" className={ui.heading}>
            {title}
          </h1>
          <p className={ui.lead}>{description}</p>
          <div className={joinClassNames(ui.actions, styles.actions)}>
            <PublicAction to="/" icon={<ArrowRight />}>
              Back Home
            </PublicAction>
            <PublicAction to="/services" variant="ghostOnDark">
              View Services
            </PublicAction>
          </div>
        </div>
        <BrandGlobe className={styles.orb} />
      </div>
    </section>
  </PageWrapper>
);
