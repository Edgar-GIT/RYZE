import { ContactPreview } from "@/components/contact_preview/contact_preview";
import { HomeStory } from "@/components/home_story/home_story";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ProgramsOverviewSection } from "@/components/programs_overview_section/programs_overview_section";
import { PurchaseJourney } from "@/components/purchase_journey/purchase_journey";
import ui from "@/components/public_ui/public_ui.module.css";
import { joinClassNames } from "@utils/class_names";

import styles from "./home_page.module.css";

export const HomePage = () => (
  // The hero video's deep blues carry on to the end of the homepage (owner
  // request, 2026-11-28).
  <PageWrapper className={joinClassNames(ui.page, styles.home)}>
    {/* The programs come right after the opening chapter: visitors reach them first. */}
    <HomeStory programs={<ProgramsOverviewSection />} />

    <PurchaseJourney />

    <ContactPreview />
  </PageWrapper>
);
