import { SlidersHorizontal } from "lucide-react";

import { AdminPlaceholderPage } from "@/components/admin/admin_placeholder/admin_placeholder";

export const AdminConfigurationPage = () => (
  <AdminPlaceholderPage
    eyebrow="Technical configuration"
    title="Technical configuration"
    description="Platform-wide configuration parameters will be editable from this page once the settings backend is implemented."
    icon={SlidersHorizontal}
  />
);