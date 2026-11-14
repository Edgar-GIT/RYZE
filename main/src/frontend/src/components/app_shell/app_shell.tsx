import { ArrowLeft, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Link, useLocation } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { LanguageSwitcher } from "@/components/language_switcher/language_switcher";
import { NotificationBell } from "@/components/notification_bell/notification_bell";
import { joinClassNames } from "@utils/class_names";

import styles from "./app_shell.module.css";

export interface ShellNavItem {
  to: string;
  label: string;
  icon: LucideIcon;
  // Whether the item is the current section; defaults to a prefix match.
  match?: (pathname: string) => boolean;
}

export interface ShellUser {
  name: string;
  email?: string;
}

interface AppShellProps {
  area: "member" | "trainer";
  navLabel: string;
  items: ShellNavItem[];
  secondaryLabel?: string;
  secondaryItems?: ShellNavItem[];
  // Phone bottom tabs (at most five).
  tabs: ShellNavItem[];
  user: ShellUser | null;
  accountTo: string;
  accountLabel: string;
  // The highlighted card at the bottom of the sidebar (e.g. start a workout).
  action?: { to: string; title: string; text: string; icon: LucideIcon };
  // Link to the other area (member <-> trainer), when the account has both.
  switchTo?: { to: string; label: string; icon: LucideIcon };
  backToSite: string;
  children: ReactNode;
}

const initialsOf = (name: string) =>
  name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("") || "R";

const isCurrent = (item: ShellNavItem, pathname: string) =>
  item.match ? item.match(pathname) : pathname === item.to || pathname.startsWith(`${item.to}/`);

// AppShell is the frame of the signed-in areas (member and trainer): a
// sidebar on computers (an icon rail on tablets) with the sections, the
// account and a highlighted action; a top bar with the language, the
// notifications and the account; and app-style tabs at the bottom on phones.
// Colours follow the login page: logo navy, navy panels, cyan light and the
// light-blue wave panel for the one thing to do next.
export const AppShell = ({
  area,
  navLabel,
  items,
  secondaryLabel,
  secondaryItems,
  tabs,
  user,
  accountTo,
  accountLabel,
  action,
  switchTo,
  backToSite,
  children
}: AppShellProps) => {
  const { pathname } = useLocation();
  const ActionIcon = action?.icon;
  const SwitchIcon = switchTo?.icon;

  const renderItem = (item: ShellNavItem) => {
    const current = isCurrent(item, pathname);
    const Icon = item.icon;
    return (
      <li key={item.to}>
        <Link
          to={item.to}
          className={joinClassNames(styles.navItem, current && styles.navItemActive)}
          aria-current={current ? "page" : undefined}
          title={item.label}
        >
          <Icon aria-hidden="true" strokeWidth={current ? 2.2 : 1.8} />
          <span>{item.label}</span>
        </Link>
      </li>
    );
  };

  return (
    <div className={styles.shell} data-area={area}>
      <aside className={styles.sidebar}>
        <Link className={styles.brand} to="/" aria-label={backToSite}>
          <BrandMark size="navigation" />
          <span className={styles.brandName}>RYZE</span>
          {area === "trainer" ? <span className={styles.brandTag}>Coach</span> : null}
        </Link>

        <nav className={styles.nav} aria-label={navLabel}>
          <ul className={styles.navList}>{items.map(renderItem)}</ul>
          {secondaryItems && secondaryItems.length > 0 ? (
            <>
              {secondaryLabel ? <p className={styles.navGroup}>{secondaryLabel}</p> : null}
              <ul className={styles.navList}>{secondaryItems.map(renderItem)}</ul>
            </>
          ) : null}
        </nav>

        <div className={styles.sidebarFoot}>
          {action && ActionIcon ? (
            <Link className={styles.actionCard} to={action.to}>
              <span className={styles.actionIcon} aria-hidden="true">
                <ActionIcon strokeWidth={2.2} />
              </span>
              <span className={styles.actionText}>
                <strong>{action.title}</strong>
                <span>{action.text}</span>
              </span>
            </Link>
          ) : null}
          {switchTo && SwitchIcon ? (
            <Link className={styles.switch} to={switchTo.to}>
              <SwitchIcon aria-hidden="true" strokeWidth={1.9} />
              <span>{switchTo.label}</span>
            </Link>
          ) : null}
          <Link
            className={joinClassNames(styles.userCard, pathname === accountTo && styles.userCardActive)}
            to={accountTo}
            aria-label={accountLabel}
          >
            <span className={styles.avatar} aria-hidden="true">
              {initialsOf(user?.name ?? "")}
            </span>
            <span className={styles.userText}>
              <strong>{user?.name ?? "…"}</strong>
              {user?.email ? <span>{user.email}</span> : null}
            </span>
          </Link>
        </div>
      </aside>

      <div className={styles.body}>
        <header className={styles.topbar}>
          <Link className={styles.topBrand} to="/" aria-label={backToSite}>
            <BrandMark size="navigation" />
            <span className={styles.brandName}>RYZE</span>
          </Link>
          <Link className={styles.backLink} to="/">
            <ArrowLeft aria-hidden="true" />
            {backToSite}
          </Link>
          <div className={styles.topActions}>
            <LanguageSwitcher variant="compact" className={styles.language} />
            <NotificationBell />
            <Link className={styles.topAvatar} to={accountTo} aria-label={accountLabel} title={accountLabel}>
              {initialsOf(user?.name ?? "")}
            </Link>
          </div>
        </header>

        <main className={styles.main} id="main-content">
          {children}
        </main>
      </div>

      <nav className={styles.tabBar} aria-label={navLabel}>
        {tabs.map((item) => {
          const current = isCurrent(item, pathname);
          const Icon = item.icon;
          return (
            <Link
              key={item.to}
              to={item.to}
              className={joinClassNames(styles.tab, current && styles.tabActive)}
              aria-current={current ? "page" : undefined}
            >
              <Icon aria-hidden="true" strokeWidth={current ? 2.2 : 1.8} />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </nav>
    </div>
  );
};
