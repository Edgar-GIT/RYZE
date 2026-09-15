import { Code2 } from "lucide-react";

import { AdminPlaceholderPage } from "@/components/admin/admin_placeholder/admin_placeholder";

export const AdminDevelopmentPage = () => (
  <AdminPlaceholderPage
    eyebrow="Development"
    title="Development"
    description="Development utilities, build status and service management controls will appear here once the development operations backend is available."
    icon={Code2}
  />
);