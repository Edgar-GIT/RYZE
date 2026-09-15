import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import {
  ExternalLink,
  LogOut,
  Menu,
  ShieldCheck,
  X
} from "lucide-react";
import { Link, Redirect, useLocation } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import {
  adminNavigationGroups,
  type AdminNavGroup,
  type AdminNavItem
} from "@/components/admin/admin_navigation";
import {
  TECHNICAL_ADMINISTRATOR,
  adminRoleName,
  fetchAdminIdentity,
  signOutAdmin,
  type AdminIdentity
} from "@/services/admin_api";
import { joinClassNames } from "@utils/class_names";
import { AdminSessionContext } from "@/components/admin/admin_session_context";

import styles from "./admin_layout.module.css";

type AdminSessionState =
  | { status: "loading" }
  | { status: "ready"; identity: AdminIdentity }
  | { status: "unauthenticated" };

interface AdminSidebarProps {
  identity: AdminIdentity;
  pathname: string;
  onSignOut: () => void;
  onNavigate: () => void;
}

const AdminNavItemLink = ({ item, pathname, onNavigate }: { item: AdminNavItem; pathname: string; onNavigate: () => void }) => {
  const isActive = pathname === item.to;
  const Icon = item.icon;

  return (
    <Link
      to={item.to}
      className={joinClassNames(styles.navItem, isActive && styles.navItemActive)}
      aria-current={isActive ? "page" : undefined}
      onClick={onNavigate}
    >
      <Icon className={styles.navItemIcon} aria-hidden="true" strokeWidth={1.8} />
      <span>{item.label}</span>
    </Link>
  );
};

const AdminNavGroupSection = ({ group, pathname, onNavigate }: { group: AdminNavGroup; pathname: string; onNavigate: () => void }) => (
  <div className={styles.navGroup}>
    <p className={styles.navGroupTitle}>{group.title}</p>
    <nav className={styles.navList} aria-label={`${group.title} navigation`}>
      {group.items.map((item) => (
        <AdminNavItemLink key={item.to} item={item} pathname={pathname} onNavigate={onNavigate} />
      ))}
    </nav>
  </div>
);

const AdminSidebar = ({ identity, pathname, onSignOut, onNavigate }: AdminSidebarProps) => {
  const isTechnicalAdministrator = identity.role === TECHNICAL_ADMINISTRATOR;

  return (
    <div className={styles.sidebarInner}>
      <div className={styles.sidebarHeader}>
        <Link className={styles.brand} to="/" aria-label="RYZE home" onClick={onNavigate}>
          <BrandMark size="navigation" />
          <span className={styles.brandText}>
            RYZE <em className={styles.brandSub}>Admin</em>
          </span>
        </Link>
        <button
          type="button"
          className={styles.rolePill}
          aria-label={`Role: ${adminRoleName(identity.role)}`}
          title={identity.role}
        >
          <ShieldCheck className={styles.roleIcon} aria-hidden="true" strokeWidth={1.8} />
          <span>{adminRoleName(identity.role)}</span>
        </button>
      </div>

      <div className={styles.sidebarNav}>
        {adminNavigationGroups(isTechnicalAdministrator).map((group) => (
          <AdminNavGroupSection key={group.title} group={group} pathname={pathname} onNavigate={onNavigate} />
        ))}
      </div>

      <div className={styles.sidebarFooter}>
        <Link className={styles.footerLink} to="/" onClick={onNavigate}>
          <ExternalLink className={styles.footerIcon} aria-hidden="true" strokeWidth={1.8} />
          <span>Back to the site</span>
        </Link>
        <button type="button" className={styles.signOutButton} onClick={onSignOut}>
          <LogOut className={styles.footerIcon} aria-hidden="true" strokeWidth={1.8} />
          <span>Sign out</span>
        </button>
      </div>
    </div>
  );
};

const AdminLoadingScreen = () => (
  <div className={styles.loadingScreen} role="status" aria-label="Loading admin session">
    <div className={styles.loadingSpinner} aria-hidden="true" />
    <p>Loading admin session…</p>
  </div>
);

export const AdminLayout = ({ children }: { children: ReactNode }) => {
  const { pathname } = useLocation();
  const [session, setSession] = useState<AdminSessionState>({ status: "loading" });
  const [sidebarOpen, setSidebarOpen] = useState(false);

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

  useEffect(() => {
    if (!sidebarOpen) {
      return;
    }
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [sidebarOpen]);

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
    return <AdminLoadingScreen />;
  }

  if (session.status === "unauthenticated") {
    return <Redirect to="/admin/login" />;
  }

  const identity = session.identity;

  return (
    <div className={styles.adminLayout}>
      <button
        type="button"
        className={styles.menuToggle}
        aria-label={sidebarOpen ? "Close admin menu" : "Open admin menu"}
        aria-expanded={sidebarOpen}
        onClick={() => setSidebarOpen((open) => !open)}
      >
        {sidebarOpen ? <X aria-hidden="true" /> : <Menu aria-hidden="true" />}
      </button>

      <aside
        className={joinClassNames(styles.sidebar, sidebarOpen && styles.sidebarOpen)}
        aria-label="Admin navigation"
      >
        <AdminSidebar
          identity={identity}
          pathname={pathname}
          onSignOut={handleSignOut}
          onNavigate={() => setSidebarOpen(false)}
        />
      </aside>

      {sidebarOpen ? (
        <button
          type="button"
          className={styles.scrim}
          aria-label="Close admin menu"
          onClick={() => setSidebarOpen(false)}
        />
      ) : null}

      <main className={styles.content}>
        <AdminSessionContext.Provider value={{ identity }}>{children}</AdminSessionContext.Provider>
      </main>
    </div>
  );
};