import { useEffect, useState } from "react";
import { ClipboardList, Dumbbell, LayoutDashboard, LifeBuoy, UserRound, Users } from "lucide-react";
import { Link, NavLink, Redirect, useLocation } from "react-router-dom";

import nav from "@/components/account_nav/account_nav.module.css";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { BrandMark } from "@/components/brand_mark/brand_mark";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { NotificationBell } from "@/components/notification_bell/notification_bell";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useTrainerSession } from "@/components/trainer/trainer_session_context";
import { useI18n } from "@/i18n/i18n";
import { fetchTrainerProfile } from "@/services/trainer_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./trainer_layout.module.css";

interface TrainerLayoutProps {
  children: React.ReactNode;
}

// TrainerLayout is the shared shell for the authenticated trainer area. It
// resolves the trainer session before rendering anything: unauthenticated
// visitors are sent to the login page and authenticated non-trainers are sent
// to the application page, so a trainer-only surface is never rendered for
// someone the backend has not confirmed as a trainer.
//
// It uses the member area's navigation (account_nav styles): the sections in
// the header on wider screens and an app-style tab bar at the bottom on
// phones, so clients and trainers find their way around the same way.
export const TrainerLayout = ({ children }: TrainerLayoutProps) => {
  const { access, unauthenticated, loaded } = useTrainerSession();
  const { t } = useI18n();
  const { pathname } = useLocation();
  const [userName, setUserName] = useState("");

  const isTrainer = access?.is_trainer ?? false;

  useEffect(() => {
    if (!isTrainer) {
      return;
    }
    let cancelled = false;
    fetchTrainerProfile()
      .then((profile) => {
        if (!cancelled) {
          const name = [profile.user.first_name, profile.user.last_name].filter(Boolean).join(" ");
          setUserName(name || profile.user.email);
        }
      })
      .catch(() => {
        // The header can render without the name; the surface itself already
        // comes from the backend.
      });
    return () => {
      cancelled = true;
    };
  }, [isTrainer]);

  // Reserve room for the bottom tab bar while the trainer shell is shown.
  useEffect(() => {
    if (!isTrainer) return;
    document.body.classList.add("has-tabbar");
    return () => document.body.classList.remove("has-tabbar");
  }, [isTrainer]);

  if (!loaded) {
    return (
      <PageWrapper>
        <AnimatedBackground />
        <LoadingScreen />
      </PageWrapper>
    );
  }

  if (unauthenticated) {
    return <Redirect to="/login" />;
  }

  if (!isTrainer) {
    return <Redirect to="/trainer/apply" />;
  }

  const sections = [
    { to: "/trainer/dashboard", label: t("trainerNav.dashboard"), icon: LayoutDashboard },
    { to: "/trainer/clients", label: t("trainerNav.clients"), icon: Users },
    { to: "/trainer/premium-level-2", label: t("trainerNav.premium"), icon: ClipboardList },
    { to: "/trainer/programs", label: t("trainerNav.programs"), icon: Dumbbell },
    { to: "/trainer/profile", label: t("trainerNav.profile"), icon: UserRound }
  ];

  return (
    <PageWrapper className={styles.page}>
      <AnimatedBackground />

      <header className={nav.navbar}>
        <Link className={nav.brand} to="/" aria-label={t("trainerNav.home")}>
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <nav className={nav.centerNav} aria-label={t("trainerNav.label")}>
          {sections.map((section) => (
            <NavLink key={section.to} className={nav.navLink} activeClassName={nav.navLinkActive} to={section.to}>
              {section.label}
            </NavLink>
          ))}
          <NavLink className={nav.navLink} activeClassName={nav.navLinkActive} to="/support">
            {t("trainerNav.support")}
          </NavLink>
        </nav>

        <div className={nav.userArea}>
          {userName ? <span className={nav.userName}>{userName}</span> : null}
          <Link className={styles.iconLink} to="/support" aria-label={t("trainerNav.support")} title={t("trainerNav.support")}>
            <LifeBuoy aria-hidden="true" strokeWidth={1.8} />
          </Link>
          <NotificationBell />
        </div>
      </header>

      <main className={styles.main}>{children}</main>

      <nav className={nav.tabBar} aria-label={t("trainerNav.label")}>
        {sections.map(({ to, label, icon: Icon }) => {
          const active = pathname.startsWith(to);
          return (
            <Link key={to} to={to} className={joinClassNames(nav.tab, active && nav.tabActive)} aria-current={active ? "page" : undefined}>
              <Icon aria-hidden="true" strokeWidth={active ? 2.2 : 1.8} />
              <span>{label}</span>
            </Link>
          );
        })}
      </nav>
    </PageWrapper>
  );
};
