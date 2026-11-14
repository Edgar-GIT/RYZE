import { ArrowLeft, LayoutGrid, X, type LucideIcon } from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Link, useLocation } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { LanguageSwitcher } from "@/components/language_switcher/language_switcher";
import { NotificationBell } from "@/components/notification_bell/notification_bell";
import { joinClassNames } from "@utils/class_names";

import { UserMenu, type ShellUser, type UserMenuItem } from "./user_menu";
import styles from "./app_shell.module.css";

export type { ShellUser, UserMenuItem } from "./user_menu";

export interface ShellNavItem {
  to: string;
  label: string;
  icon: LucideIcon;
  // Whether the item is the current section; defaults to a prefix match.
  match?: (pathname: string) => boolean;
  // Optional count shown next to the label (e.g. pending items).
  badge?: number;
}

export interface ShellNavGroup {
  label?: string;
  items: ShellNavItem[];
}

export interface ShellMenu {
  label: string;
  items: UserMenuItem[];
  signOutLabel: string;
  signingOutLabel: string;
  onSignOut: () => Promise<void> | void;
}

interface AppShellProps {
  area: "member" | "trainer" | "admin";
  // Small tag after the RYZE wordmark (e.g. "Coach", "Admin").
  brandTag?: string;
  // Where the brand links to; the public home by default.
  brandTo?: string;
  navLabel: string;
  groups: ShellNavGroup[];
  // Phone bottom tabs (at most four; "More" is added when moreLabel is set).
  tabs: ShellNavItem[];
  moreLabel?: string;
  closeLabel?: string;
  user: ShellUser | null;
  menu: ShellMenu;
  // The highlighted card at the bottom of the sidebar (e.g. start a workout).
  action?: { to: string; title: string; text: string; icon: LucideIcon };
  // Link to the other area (member <-> trainer), when the account has both.
  switchTo?: { to: string; label: string; icon: LucideIcon };
  backToSite: string;
  // Hides the language and notification controls (admin consoles).
  compactTopbar?: boolean;
  children: ReactNode;
}

const isCurrent = (item: ShellNavItem, pathname: string) =>
  item.match ? item.match(pathname) : pathname === item.to || pathname.startsWith(`${item.to}/`);

// AppShell is the frame of every signed-in area (member, trainer and the two
// administration consoles): a sidebar on computers (an icon rail on tablets)
// with the sections and the account; a top bar with the language, the
// notifications and the account menu; and app-style tabs on phones, with a
// "More" sheet for the remaining sections. Colours follow the login page:
// logo navy, navy panels, cyan light and the light-blue wave panel for the one
// thing to do next.
export const AppShell = ({
  area,
  brandTag,
  brandTo = "/",
  navLabel,
  groups,
  tabs,
  moreLabel,
  closeLabel = "Close",
  user,
  menu,
  action,
  switchTo,
  backToSite,
  compactTopbar = false,
  children
}: AppShellProps) => {
  const { pathname } = useLocation();
  const [moreOpen, setMoreOpen] = useState(false);
  const sheetRef = useRef<HTMLDivElement>(null);
  const moreButtonRef = useRef<HTMLButtonElement>(null);
  const ActionIcon = action?.icon;
  const SwitchIcon = switchTo?.icon;

  useEffect(() => {
    setMoreOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!moreOpen) {
      return;
    }
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    sheetRef.current?.querySelector<HTMLElement>("a, button")?.focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMoreOpen(false);
        moreButtonRef.current?.focus();
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [moreOpen]);

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
          {item.badge ? <em className={styles.navBadge}>{item.badge > 99 ? "99+" : item.badge}</em> : null}
        </Link>
      </li>
    );
  };

  const moreCurrent =
    moreLabel !== undefined && !tabs.some((tab) => isCurrent(tab, pathname));

  return (
    <div className={styles.shell} data-area={area}>
      <aside className={styles.sidebar}>
        <Link className={styles.brand} to={brandTo} aria-label={backToSite}>
          <BrandMark size="navigation" />
          <span className={styles.brandName}>RYZE</span>
          {brandTag ? <span className={styles.brandTag}>{brandTag}</span> : null}
        </Link>

        <nav className={styles.nav} aria-label={navLabel}>
          {groups.map((group, index) => (
            <div key={group.label ?? index} className={styles.navSection}>
              {group.label ? <p className={styles.navGroup}>{group.label}</p> : null}
              <ul className={styles.navList}>{group.items.map(renderItem)}</ul>
            </div>
          ))}
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
          <UserMenu
            variant="card"
            user={user}
            items={menu.items}
            label={menu.label}
            signOutLabel={menu.signOutLabel}
            signingOutLabel={menu.signingOutLabel}
            onSignOut={menu.onSignOut}
          />
        </div>
      </aside>

      <div className={styles.body}>
        <header className={styles.topbar}>
          <Link className={styles.topBrand} to={brandTo} aria-label={backToSite}>
            <BrandMark size="navigation" />
            <span className={styles.brandName}>RYZE</span>
            {brandTag ? <span className={styles.topTag}>{brandTag}</span> : null}
          </Link>
          <Link className={styles.backLink} to="/">
            <ArrowLeft aria-hidden="true" />
            {backToSite}
          </Link>
          <div className={styles.topActions}>
            {compactTopbar ? null : <LanguageSwitcher variant="compact" className={styles.language} />}
            {compactTopbar ? null : <NotificationBell />}
            <UserMenu
              variant="avatar"
              user={user}
              items={menu.items}
              label={menu.label}
              signOutLabel={menu.signOutLabel}
              signingOutLabel={menu.signingOutLabel}
              onSignOut={menu.onSignOut}
            />
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
              {item.badge ? <em className={styles.tabBadge}>{item.badge > 9 ? "9+" : item.badge}</em> : null}
            </Link>
          );
        })}
        {moreLabel ? (
          <button
            ref={moreButtonRef}
            type="button"
            className={joinClassNames(styles.tab, moreCurrent && styles.tabActive)}
            aria-haspopup="dialog"
            aria-expanded={moreOpen}
            onClick={() => setMoreOpen(true)}
          >
            <LayoutGrid aria-hidden="true" strokeWidth={moreCurrent ? 2.2 : 1.8} />
            <span>{moreLabel}</span>
          </button>
        ) : null}
      </nav>

      {moreOpen && moreLabel ? (
        <div className={styles.sheetLayer}>
          <button
            type="button"
            className={styles.sheetBackdrop}
            aria-label={closeLabel}
            tabIndex={-1}
            onClick={() => setMoreOpen(false)}
          />
          <div ref={sheetRef} className={styles.sheet} role="dialog" aria-modal="true" aria-label={moreLabel}>
            <div className={styles.sheetHead}>
              <strong>{moreLabel}</strong>
              <button
                type="button"
                className={styles.sheetClose}
                aria-label={closeLabel}
                onClick={() => {
                  setMoreOpen(false);
                  moreButtonRef.current?.focus();
                }}
              >
                <X aria-hidden="true" />
              </button>
            </div>
            {groups.map((group, index) => (
              <div key={group.label ?? index} className={styles.sheetGroup}>
                {group.label ? <p className={styles.sheetLabel}>{group.label}</p> : null}
                <div className={styles.sheetGrid}>
                  {group.items.map((item) => {
                    const Icon = item.icon;
                    const current = isCurrent(item, pathname);
                    return (
                      <Link
                        key={item.to}
                        to={item.to}
                        className={joinClassNames(styles.sheetItem, current && styles.sheetItemActive)}
                        aria-current={current ? "page" : undefined}
                      >
                        <Icon aria-hidden="true" strokeWidth={1.9} />
                        <span>{item.label}</span>
                        {item.badge ? <em className={styles.navBadge}>{item.badge}</em> : null}
                      </Link>
                    );
                  })}
                </div>
              </div>
            ))}
            {switchTo && SwitchIcon ? (
              <Link className={styles.sheetSwitch} to={switchTo.to}>
                <SwitchIcon aria-hidden="true" />
                <span>{switchTo.label}</span>
              </Link>
            ) : null}
          </div>
        </div>
      ) : null}
    </div>
  );
};
