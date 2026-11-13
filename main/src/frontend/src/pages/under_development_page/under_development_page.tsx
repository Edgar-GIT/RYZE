import { ArrowRight, Construction, SearchX } from "lucide-react";

import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { BrandGlobe } from "@/components/brand_globe/brand_globe";
import { type Catalog, useGettext } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./under_development_page.module.css";

// Portuguese for this page and the texts the routes pass to it.
const UNDER_DEVELOPMENT_PT: Catalog = {
  "In progress": "Em desenvolvimento",
  "Back Home": "Voltar ao início",
  "View Services": "Ver serviços",
  "Page not found": "Página não encontrada",
  "We could not find this service.": "Não encontrámos este serviço.",
  "Check the address, or open Services to see every RYZE plan and package.":
    "Verifica o endereço ou abre Serviços para veres todos os planos e pacotes RYZE.",
  "We could not find this page.": "Não encontrámos esta página.",
  "The address may be mistyped or the page may have moved. Go back home or open Services.":
    "O endereço pode estar mal escrito ou a página pode ter mudado. Volta ao início ou abre Serviços.",
  Feed: "Feed",
  "Feed is under development.": "O Feed está em desenvolvimento.",
  "The RYZE activity feed will be introduced after the frontend foundation is ready.":
    "O feed de atividade da RYZE vai chegar depois de a base do frontend estar pronta."
};

interface UnderDevelopmentPageProps {
  title: string;
  description: string;
  eyebrow?: string;
  // "notFound" marks an address RYZE does not have (a missing page), as
  // opposed to a feature that is planned but not built yet.
  kind?: "inProgress" | "notFound";
}

export const UnderDevelopmentPage = ({
  title,
  description,
  eyebrow = "In progress",
  kind = "inProgress"
}: UnderDevelopmentPageProps) => {
  const tr = useGettext(UNDER_DEVELOPMENT_PT);
  return (
  <PageWrapper className={ui.page}>
    <section className={joinClassNames(ui.toneNavy, styles.section)} aria-labelledby="under-development-heading">
      <div className={joinClassNames(ui.container, styles.layout)}>
        <div className={styles.copy}>
          <p className={styles.status}>
            {kind === "notFound" ? <SearchX aria-hidden="true" /> : <Construction aria-hidden="true" />}
            {tr(eyebrow)}
          </p>
          <h1 id="under-development-heading" className={ui.heading}>
            {tr(title)}
          </h1>
          <p className={ui.lead}>{tr(description)}</p>
          <div className={joinClassNames(ui.actions, styles.actions)}>
            <PublicAction to="/" icon={<ArrowRight />}>
              {tr("Back Home")}
            </PublicAction>
            <PublicAction to="/services" variant="ghostOnDark">
              {tr("View Services")}
            </PublicAction>
          </div>
        </div>
        <BrandGlobe className={styles.orb} />
      </div>
    </section>
  </PageWrapper>
  );
};
