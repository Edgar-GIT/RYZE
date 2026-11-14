import { Globe2, Settings } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Redirect, useLocation } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import { AppShell, type UserMenuItem } from "@/components/app_shell/app_shell";
import { adminRoleName, MANAGEMENT_ADMINISTRATOR, signOutAdmin } from "@/services/admin_api";
import { fetchAdminQueues, type OverviewQueues } from "@/services/admin_overview_api";

import { useAdmin2Session } from "../admin2_session_context";
import { admin2NavGroups, admin2Tabs, ADMIN2_PATHS, type Admin2Badges } from "../admin2_navigation";

const MENU_ITEMS: UserMenuItem[] = [
  { to: ADMIN2_PATHS.SETTINGS, label: "Commerce settings", icon: Settings },
  { to: "/", label: "RYZE website", icon: Globe2 }
];

const badgesOf = (queues: OverviewQueues | null): Admin2Badges =>
  queues
    ? {
        premiumLevel2: queues.premium_level_2_awaiting_trainer,
        applications: queues.applications_pending,
        support: queues.support_open,
        feedback: queues.feedback_pending
      }
    : {};

// Admin2Layout is the Management Administrator console (ADMIN_2): the
// catalogue, the business and the service queues, in the shared app shell of
// the signed-in areas. Navigation badges show the open work of each queue and
// refresh when the administrator moves between pages.
export default function Admin2Layout({ children }: { children: React.ReactNode }) {
  const { identity, loaded } = useAdmin2Session();
  const { pathname } = useLocation();
  const [queues, setQueues] = useState<OverviewQueues | null>(null);

  const isManagement = identity?.role === MANAGEMENT_ADMINISTRATOR;

  useEffect(() => {
    if (!isManagement) return;
    let cancelled = false;
    fetchAdminQueues()
      .then((result) => {
        if (!cancelled) setQueues(result);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [isManagement, pathname]);

  const handleSignOut = useCallback(async () => {
    try {
      await signOutAdmin();
    } catch {
      // Sign-out is best-effort; the hard redirect leaves the console regardless.
    }
    window.location.assign("/admin/login");
  }, []);

  if (!loaded) {
    return <AppLoading />;
  }

  if (!identity) {
    return <Redirect to="/admin/login" />;
  }

  if (!isManagement) {
    // The Technical Administrator has its own console.
    return <Redirect to="/admin/dashboard" />;
  }

  const badges = badgesOf(queues);

  return (
    <AppShell
      area="admin"
      brandTag="Business"
      brandTo={ADMIN2_PATHS.DASHBOARD}
      navLabel="Business administration"
      groups={admin2NavGroups(badges)}
      tabs={admin2Tabs(badges)}
      moreLabel="More"
      user={{ name: adminRoleName(identity.role), email: identity.id, role: "ADMIN_2" }}
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
      {children}
    </AppShell>
  );
}
