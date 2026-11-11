import { CategoryOverview } from "@/components/category_overview/category_overview";
import { ContactPreview } from "@/components/contact_preview/contact_preview";
import { HomeStory } from "@/components/home_story/home_story";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ProgramsOverviewSection } from "@/components/programs_overview_section/programs_overview_section";
import { PurchaseJourney } from "@/components/purchase_journey/purchase_journey";
import ui from "@/components/public_ui/public_ui.module.css";
import { useServiceCatalog } from "@/hooks/use_service_catalog";

export const HomePage = () => {
  const catalog = useServiceCatalog();

  return (
    <PageWrapper className={ui.page}>
      {/* The programs come right after the opening chapter: visitors reach them first. */}
      <HomeStory programs={<ProgramsOverviewSection catalog={catalog} />} />

      <CategoryOverview catalog={catalog} />

      <PurchaseJourney />

      <ContactPreview />
    </PageWrapper>
  );
};
