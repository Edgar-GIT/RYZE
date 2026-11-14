import {
  BarChart3,
  ClipboardList,
  Dumbbell,
  FileCheck2,
  LayoutDashboard,
  LifeBuoy,
  MessageSquare,
  Package,
  Settings,
  UserRoundCheck,
  Users
} from "lucide-react";

import type { ShellNavGroup, ShellNavItem } from "@/components/app_shell/app_shell";

export const ADMIN2_PATHS = {
  ROOT: "/admin2",
  DASHBOARD: "/admin2/dashboard",
  PLANS: "/admin2/plans",
  PLAN_CREATE: "/admin2/plans/create",
  PREMIUM_LEVEL_2: "/admin2/premium-level-2",
  EXERCISES: "/admin2/exercises",
  TRAINERS: "/admin2/trainers",
  APPLICATIONS: "/admin2/trainer-applications",
  CLIENTS: "/admin2/clients",
  SALES: "/admin2/sales",
  FEEDBACK: "/admin2/feedback",
  SUPPORT: "/admin2/support",
  SETTINGS: "/admin2/settings"
} as const;

export interface Admin2Badges {
  premiumLevel2?: number;
  applications?: number;
  support?: number;
  feedback?: number;
}

const item = (to: string, label: string, icon: ShellNavItem["icon"], badge?: number): ShellNavItem => ({
  to,
  label,
  icon,
  badge
});

// The Management Administrator console (ADMIN_2): the catalogue, the business
// and the service queues. Badges show the open work of each queue.
export const admin2NavGroups = (badges: Admin2Badges = {}): ShellNavGroup[] => [
  { items: [item(ADMIN2_PATHS.DASHBOARD, "Overview", LayoutDashboard)] },
  {
    label: "Catalogue",
    items: [
      item(ADMIN2_PATHS.PLANS, "Plans", Package),
      item(ADMIN2_PATHS.EXERCISES, "Exercises", Dumbbell)
    ]
  },
  {
    label: "Business",
    items: [
      item(ADMIN2_PATHS.SALES, "Sales", BarChart3),
      item(ADMIN2_PATHS.TRAINERS, "Trainers & commissions", UserRoundCheck),
      item(ADMIN2_PATHS.CLIENTS, "Members", Users)
    ]
  },
  {
    label: "Service",
    items: [
      item(ADMIN2_PATHS.PREMIUM_LEVEL_2, "Premium L2 requests", ClipboardList, badges.premiumLevel2),
      item(ADMIN2_PATHS.APPLICATIONS, "Trainer applications", FileCheck2, badges.applications),
      item(ADMIN2_PATHS.SUPPORT, "Support", LifeBuoy, badges.support),
      item(ADMIN2_PATHS.FEEDBACK, "Feedback", MessageSquare, badges.feedback)
    ]
  },
  { label: "Configuration", items: [item(ADMIN2_PATHS.SETTINGS, "Settings", Settings)] }
];

export const admin2Tabs = (badges: Admin2Badges = {}): ShellNavItem[] => [
  item(ADMIN2_PATHS.DASHBOARD, "Overview", LayoutDashboard),
  item(ADMIN2_PATHS.PLANS, "Plans", Package),
  item(ADMIN2_PATHS.SALES, "Sales", BarChart3),
  item(ADMIN2_PATHS.SUPPORT, "Support", LifeBuoy, badges.support)
];

export const isNavActive = (pathname: string, path: string): boolean =>
  pathname === path || pathname.startsWith(`${path}/`);
