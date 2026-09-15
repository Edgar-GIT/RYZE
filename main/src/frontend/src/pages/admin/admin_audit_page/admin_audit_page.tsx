import { ScrollText } from "lucide-react";

import { AdminPlaceholderPage } from "@/components/admin/admin_placeholder/admin_placeholder";

export const AdminAuditPage = () => (
  <AdminPlaceholderPage
    eyebrow="Audit log"
    title="Audit log"
    description="Administrative and security-relevant activity will be recorded here once the audit service is implemented."
    icon={ScrollText}
  />
);