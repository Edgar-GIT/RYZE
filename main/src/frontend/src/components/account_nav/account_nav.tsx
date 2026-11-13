import { BarChart3, BookOpen, Dumbbell, History, UserRound } from "lucide-react";
import { useEffect } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { NotificationBell } from "@/components/notification_bell/notification_bell";
import { useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./account_nav.module.css";

// AccountNav is the navigation of the signed-in member area. On wider screens
// the sections sit in the header; on phones the header keeps only the brand,
// notifications and the account, and the main sections move to an app-style
// tab bar at the bottom (Train, Library, Progress, History, Profile).
interface AccountNavProps {
  userName?: string;
}

const isTrainPath = (pathname: string) =>
  pathname.startsWith("/services/my-programs") || pathname.startsWith("/services/workout-sessions");

export const AccountNav = ({ userName }: AccountNavProps) => {
  const { t } = useI18n();
  const { pathname } = useLocation();

  // Reserve room for the bottom tab bar while this navigation is shown.
  useEffect(() => {
    document.body.classList.add("has-tabbar");
    return () => document.body.classList.remove("has-tabbar");
  }, []);

  const tabs = [
    { to: "/services/my-programs", label: t("training.nav.train"), icon: Dumbbell, active: isTrainPath(pathname) },
    { to: "/exercises", label: t("training.nav.library"), icon: BookOpen, active: pathname.startsWith("/exercises") },
    { to: "/services/progress", label: t("training.nav.progress"), icon: BarChart3, active: pathname === "/services/progress" },
    { to: "/services/workout-history", label: t("training.nav.history"), icon: History, active: pathname === "/services/workout-history" },
    {
      to: "/profile",
      label: t("training.nav.profile"),
      icon: UserRound,
      active: pathname === "/profile" || pathname.startsWith("/account") || pathname === "/support"
    }
  ];

  return (
    <>
      <header className={styles.navbar}>
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <nav className={styles.centerNav} aria-label={t("account.nav.label")}>
          <NavLink className={styles.navLink} activeClassName={styles.navLinkActive} to="/services/my-programs" isActive={() => isTrainPath(pathname)}>
            {t("account.nav.myPrograms")}
          </NavLink>
          <NavLink className={styles.navLink} activeClassName={styles.navLinkActive} to="/services/progress">
            {t("account.nav.progress")}
          </NavLink>
          <NavLink className={styles.navLink} activeClassName={styles.navLinkActive} to="/services/workout-history">
            {t("training.nav.history")}
          </NavLink>
          <NavLink className={styles.navLink} activeClassName={styles.navLinkActive} to="/account/purchases">
            {t("account.nav.purchases")}
          </NavLink>
          <NavLink className={styles.navLink} activeClassName={styles.navLinkActive} to="/support">
            {t("account.nav.support")}
          </NavLink>
          <NavLink className={styles.navLink} activeClassName={styles.navLinkActive} to="/profile">
            {t("account.nav.profile")}
          </NavLink>
        </nav>

        <div className={styles.userArea}>
          {userName ? (
            <>
              <span className={styles.userName}>{userName}</span>
              <span className={styles.avatar} aria-hidden="true">
                <UserRound strokeWidth={1.6} />
              </span>
            </>
          ) : null}
          <NotificationBell />
        </div>
      </header>

      <nav className={styles.tabBar} aria-label={t("training.nav.label")}>
        {tabs.map(({ to, label, icon: Icon, active }) => (
          <Link key={to} to={to} className={joinClassNames(styles.tab, active && styles.tabActive)} aria-current={active ? "page" : undefined}>
            <Icon aria-hidden="true" strokeWidth={active ? 2.2 : 1.8} />
            <span>{label}</span>
          </Link>
        ))}
      </nav>
    </>
  );
};
