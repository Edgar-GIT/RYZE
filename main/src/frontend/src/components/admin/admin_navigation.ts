import {
  FileCheck2,
  FlaskConical,
  LayoutDashboard,
  ScrollText,
  ServerCog,
  SlidersHorizontal,
  Users,
  UserRoundCheck
} from "lucide-react";

import type { ShellNavGroup, ShellNavItem } from "@/components/app_shell/app_shell";

const DASHBOARD: ShellNavItem = { to: "/admin/dashboard", label: "Dashboard", icon: LayoutDashboard };
const USERS: ShellNavItem = { to: "/admin/users", label: "Users", icon: Users };
const TRAINERS: ShellNavItem = { to: "/admin/trainers", label: "Trainers", icon: UserRoundCheck };
const APPLICATIONS: ShellNavItem = { to: "/admin/trainer-applications", label: "Applications", icon: FileCheck2 };
const SYSTEM: ShellNavItem = { to: "/admin/system", label: "System status", icon: ServerCog };
const CONFIGURATION: ShellNavItem = { to: "/admin/configuration", label: "Configuration", icon: SlidersHorizontal };
const TEST_MODE: ShellNavItem = { to: "/admin/test-mode", label: "Test Mode", icon: FlaskConical };
const AUDIT: ShellNavItem = { to: "/admin/audit", label: "Audit log", icon: ScrollText };

// The Technical Administrator console (ADMIN_1): accounts and the platform.
// Business sections (plans, sales, commissions, support) belong to the
// Management Administrator console and are never listed here.
export const adminNavigationGroups = (): ShellNavGroup[] => [
  { items: [DASHBOARD] },
  { label: "People", items: [USERS, TRAINERS, APPLICATIONS] },
  { label: "Platform", items: [SYSTEM, CONFIGURATION, TEST_MODE, AUDIT] }
];

// Phone tabs; every other section is in the "More" sheet.
export const ADMIN_TABS: ShellNavItem[] = [DASHBOARD, USERS, SYSTEM, AUDIT];
