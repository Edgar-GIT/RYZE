import { ServerCog } from "lucide-react";

import { AdminPlaceholderPage } from "@/components/admin/admin_placeholder/admin_placeholder";

export const AdminSystemPage = () => (
  <AdminPlaceholderPage
    eyebrow="System"
    title="System & Infrastructure"
    description="Server health, API uptime and database connectivity will be shown here once the monitoring layer is implemented."
    icon={ServerCog}
  />
);