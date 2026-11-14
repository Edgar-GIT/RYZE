import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { Globe2, ScrollText, ServerCog } from "lucide-react";
import { Redirect, useLocation } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { AppShell, type UserMenuItem } from "@/components/app_shell/app_shell";
import { adminNavigationGroups, ADMIN_TABS } from "@/components/admin/admin_navigation";
import { AdminSessionContext } from "@/components/admin/admin_session_context";
import {
  MANAGEMENT_ADMINISTRATOR,
  adminRoleName,
  fetchAdminIdentity,
  signOutAdmin,
  type AdminIdentity
} from "@/services/admin_api";

type AdminSessionState =
  | { status: "loading" }
  | { status: "ready"; identity: AdminIdentity }
  | { status: "unauthenticated" };

const MENU_ITEMS: UserMenuItem[] = [
  { to: "/admin/system", label: "System status", icon: ServerCog },
  { to: "/admin/audit", label: "Audit log", icon: ScrollText },
  { to: "/", label: "RYZE website", icon: Globe2 }
];

// AdminLayout is the Technical Administrator console (ADMIN_1). It resolves
// the administrator session from the server first; the Management
// Administrator has its own console (/admin2) and is sent there. The frame is
// the shared app shell of the signed-in areas, in the login page colours.
export const AdminLayout = ({ children }: { children: ReactNode }) => {
  const { pathname } = useLocation();
  const [session, setSession] = useState<AdminSessionState>({ status: "loading" });

  useEffect(() => {
    let cancelled = false;
    fetchAdminIdentity()
      .then((identity) => {
        if (!cancelled) {
          setSession({ status: "ready", identity });
        }
      })
      .catch(() => {
        if (!cancelled) {
          setSession({ status: "unauthenticated" });
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const handleSignOut = async () => {
    try {
      await signOutAdmin();
    } catch {
      // The admin cookie may already be expired; the hard redirect below still
      // leaves the admin console regardless.
    }
    window.location.assign("/admin/login");
  };

  if (session.status === "loading") {
    return <AppLoading />;
  }

  if (session.status === "unauthenticated") {
    return <Redirect to="/admin/login" />;
  }

  const identity = session.identity;
  if (identity.role === MANAGEMENT_ADMINISTRATOR) {
    // The applications review is shared; everything else of the management
    // role lives in its own console.
    return <Redirect to={pathname === "/admin/trainer-applications" ? "/admin2/trainer-applications" : "/admin2/dashboard"} />;
  }

  return (
    <AppShell
      area="admin"
      brandTag="Tech"
      brandTo="/admin/dashboard"
      navLabel="Technical administration"
      groups={adminNavigationGroups()}
      tabs={ADMIN_TABS}
      moreLabel="More"
      user={{ name: adminRoleName(identity.role), email: identity.id, role: "ADMIN_1" }}
      menu={{
        label: "Administrator menu",
        items: MENU_ITEMS,
        signOutLabel: "Sign out",
        signingOutLabel: "Signing out…",
        onSignOut: handleSignOut
      }}
      backToSite="Back to the site"
      compactTopbar
    >
      <AdminSessionContext.Provider value={{ identity }}>{children}</AdminSessionContext.Provider>
    </AppShell>
  );
};
