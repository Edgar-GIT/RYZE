import { createContext, useContext } from "react";

import type { AdminIdentity } from "@/services/admin_api";

// AdminSession carries the authenticated administrator identity resolved by the
// AdminLayout guard to every admin page. It is always derived from the server
// `/admin/auth/me` response, never from client input.
export interface AdminSession {
  identity: AdminIdentity;
}

export const AdminSessionContext = createContext<AdminSession | null>(null);

export const useAdminSession = (): AdminSession => {
  const session = useContext(AdminSessionContext);
  if (!session) {
    throw new Error("useAdminSession must be used within AdminLayout");
  }
  return session;
};