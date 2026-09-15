import type { LucideIcon } from "lucide-react";
import {
  Code2,
  FileCheck2,
  LayoutDashboard,
  ScrollText,
  ServerCog,
  SlidersHorizontal,
  Users,
  UserRoundCheck
} from "lucide-react";

export interface AdminNavItem {
  to: string;
  label: string;
  icon: LucideIcon;
}

export interface AdminNavGroup {
  title: string;
  items: AdminNavItem[];
}

// MANAGEMENT_NAV is the navigation shared by every admin role.
const MANAGEMENT_NAV: AdminNavGroup[] = [
  {
    title: "Overview",
    items: [{ to: "/admin/dashboard", label: "Dashboard", icon: LayoutDashboard }]
  },
  {
    title: "Management",
    items: [
      { to: "/admin/users", label: "Users", icon: Users },
      { to: "/admin/trainers", label: "Trainers", icon: UserRoundCheck },
      { to: "/admin/trainer-applications", label: "Trainer applications", icon: FileCheck2 }
    ]
  }
];

// TECHNICAL_NAV is available only to the Technical Administrator (ADMIN_1).
// Business sections (plans, finance, marketing) intentionally stay out of this
// navigation because the Technical Administrator does not hold those
// permissions.
const TECHNICAL_NAV: AdminNavGroup[] = [
  {
    title: "Technical",
    items: [
      { to: "/admin/system", label: "System", icon: ServerCog },
      { to: "/admin/configuration", label: "Technical configuration", icon: SlidersHorizontal },
      { to: "/admin/development", label: "Development", icon: Code2 },
      { to: "/admin/audit", label: "Audit log", icon: ScrollText }
    ]
  }
];

export const adminNavigationGroups = (isTechnicalAdministrator: boolean): AdminNavGroup[] => [
  ...MANAGEMENT_NAV,
  ...(isTechnicalAdministrator ? TECHNICAL_NAV : [])
];