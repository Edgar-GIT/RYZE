import { useCallback, useEffect, useState } from "react";
import { Link, Redirect, useLocation } from "react-router-dom";
import { AlertCircle, LogOut, Menu, ShieldCheck, X } from "lucide-react";
import ryzeIcon from "@resources/img/logo/ryze_icon.png";
import { signOutAdmin, MANAGEMENT_ADMINISTRATOR, adminRoleName } from "@/services/admin_api";
import { Button } from "@/components/button/button";
import { joinClassNames } from "@utils/class_names";
import { useAdmin2Session } from "../admin2_session_context";
import { admin2NavGroups, isNavActive } from "../admin2_navigation";
import styles from "./admin2_layout.module.css";

export default function Admin2Layout({ children }: { children: React.ReactNode }) {
  const { identity, loaded } = useAdmin2Session();
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [signingOut, setSigningOut] = useState(false);

  useEffect(() => {
    setSidebarOpen(false);
  }, [location.pathname]);

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

  const handleSignOut = useCallback(async () => {
    if (signingOut) {
      return;
    }

    setSigningOut(true);
    try {
      await signOutAdmin();
    } catch {
      // Sign-out is best-effort; the hard redirect leaves the console regardless.
    } finally {
      window.location.assign("/admin/login");
    }
  }, [signingOut]);

  if (!loaded) {
    return (
      <div className={styles.statusScreen} role="status" aria-label="Loading admin session">
        <div className={styles.loadingSpinner} aria-hidden="true" />
        <p className={styles.statusDescription}>Loading admin session…</p>
      </div>
    );
  }

  if (!identity) {
    return <Redirect to="/admin/login" />;
  }

  if (identity.role !== MANAGEMENT_ADMINISTRATOR) {
    return (
      <div className={styles.statusScreen}>
        <AlertCircle size={36} color="#ff5a6a" strokeWidth={1.8} />
        <h1 className={styles.statusTitle}>Access restricted</h1>
        <p className={styles.statusDescription}>
          This console is available only to Management Administrators. Your account
          currently has a different role.
        </p>
        <div className={styles.statusActions}>
          <Button
            variant="secondary"
            size="small"
            onClick={handleSignOut}
            disabled={signingOut}
            icon={<LogOut size={15} />}
          >
            Sign out
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.admin2Layout}>
      <button
        type="button"
        className={styles.mobileMenuToggle}
        aria-label={sidebarOpen ? "Close admin menu" : "Open admin menu"}
        aria-expanded={sidebarOpen}
        onClick={() => setSidebarOpen((open) => !open)}
      >
        {sidebarOpen ? <X size={16} aria-hidden="true" /> : <Menu size={16} aria-hidden="true" />}
      </button>

      <aside
        className={joinClassNames(styles.sidebar, sidebarOpen && styles.sidebarOpen)}
        aria-label="Business administration navigation"
      >
        <div className={styles.sidebarInner}>
          <div className={styles.sidebarHeader}>
            <Link to="/admin2/dashboard" className={styles.brand}>
              <img src={ryzeIcon} alt="" width="28" height="28" />
              <span className={styles.brandText}>
                RYZE
                <i className={styles.brandSub}>Console</i>
              </span>
            </Link>
            <button type="button" className={styles.rolePill} aria-label="Administrator role" title={identity.role}>
              <ShieldCheck size={15} className={styles.roleIcon} aria-hidden="true" strokeWidth={1.8} />
              <span>{adminRoleName(identity.role)}</span>
            </button>
          </div>

          <nav className={styles.sidebarNav}>
            {admin2NavGroups.map((group) => (
              <div key={group.label} className={styles.navGroup}>
                <p className={styles.navGroupTitle}>{group.label}</p>
                <div className={styles.navList}>
                  {group.items.map((item) => {
                    const Icon = item.icon;
                    const active = isNavActive(location.pathname, item.path);

                    return (
                      <Link
                        key={item.path}
                        to={item.path}
                        className={joinClassNames(styles.navItem, active && styles.navItemActive)}
                        aria-current={active ? "page" : undefined}
                        onClick={() => setSidebarOpen(false)}
                      >
                        <Icon size={17} className={styles.navIcon} aria-hidden="true" strokeWidth={1.8} />
                        {item.label}
                      </Link>
                    );
                  })}
                </div>
              </div>
            ))}
          </nav>

          <div className={styles.sidebarFooter}>
            <button
              type="button"
              className={styles.signOutButton}
              onClick={handleSignOut}
              disabled={signingOut}
            >
              <LogOut size={17} aria-hidden="true" strokeWidth={1.8} />
              <span>{signingOut ? "Signing out…" : "Sign out"}</span>
            </button>
          </div>
        </div>
      </aside>

      {sidebarOpen ? (
        <button
          type="button"
          className={styles.mobileBackdrop}
          aria-label="Close admin menu"
          onClick={() => setSidebarOpen(false)}
        />
      ) : null}

      <main className={styles.content}>
        <div className={styles.contentInner}>{children}</div>
      </main>
    </div>
  );
}