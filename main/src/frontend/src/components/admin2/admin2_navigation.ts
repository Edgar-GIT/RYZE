import type { LucideIcon } from "lucide-react";
import { LayoutDashboard, Package, Users, BarChart3, MessageSquare, Settings } from "lucide-react";

export interface Admin2NavItem {
  label: string;
  path: string;
  icon: LucideIcon;
}

export interface Admin2NavGroup {
  label: string;
  items: Admin2NavItem[];
}

export const ADMIN2_PATHS = {
  ROOT: "/admin2",
  DASHBOARD: "/admin2/dashboard",
  PLANS: "/admin2/plans",
  PLAN_CREATE: "/admin2/plans/create",
  MARKETPLACE: "/admin2/marketplace",
  TRAINERS: "/admin2/trainers",
  CLIENTS: "/admin2/clients",
  SALES: "/admin2/sales",
  ANALYTICS: "/admin2/analytics",
  FEEDBACK: "/admin2/feedback",
  SETTINGS: "/admin2/settings"
} as const;

export const admin2NavGroups: Admin2NavGroup[] = [
  {
    label: "Overview",
    items: [{ label: "Dashboard", path: ADMIN2_PATHS.DASHBOARD, icon: LayoutDashboard }]
  },
  {
    label: "Programs",
    items: [
      { label: "Plans", path: ADMIN2_PATHS.PLANS, icon: Package },
      { label: "Marketplace", path: ADMIN2_PATHS.MARKETPLACE, icon: Package }
    ]
  },
  {
    label: "People",
    items: [
      { label: "Trainers", path: ADMIN2_PATHS.TRAINERS, icon: Users },
      { label: "Clients", path: ADMIN2_PATHS.CLIENTS, icon: Users }
    ]
  },
  {
    label: "Business",
    items: [
      { label: "Sales", path: ADMIN2_PATHS.SALES, icon: BarChart3 },
      { label: "Analytics", path: ADMIN2_PATHS.ANALYTICS, icon: BarChart3 }
    ]
  },
  {
    label: "Engagement",
    items: [{ label: "Feedback", path: ADMIN2_PATHS.FEEDBACK, icon: MessageSquare }]
  },
  {
    label: "Configuration",
    items: [{ label: "Settings", path: ADMIN2_PATHS.SETTINGS, icon: Settings }]
  }
];

export const isNavActive = (pathname: string, path: string): boolean =>
  pathname === path;