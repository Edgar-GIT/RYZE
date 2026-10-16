import { useEffect, useState } from "react";
import { NavLink, Redirect } from "react-router-dom";
import { UserRound } from "lucide-react";

import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { BrandMark } from "@/components/brand_mark/brand_mark";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useTrainerSession } from "@/components/trainer/trainer_session_context";
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
export const TrainerLayout = ({ children }: TrainerLayoutProps) => {
  const { access, unauthenticated, loaded } = useTrainerSession();
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

  return (
    <PageWrapper className={styles.page}>
      <AnimatedBackground />

      <header className={styles.navbar}>
        <NavLink className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </NavLink>

        <nav className={styles.centerNav} aria-label="Trainer">
          <NavLink
            className={styles.navLink}
            activeClassName={joinClassNames(styles.navLink, styles.navLinkActive)}
            to="/trainer/dashboard"
          >
            Dashboard
          </NavLink>
          <NavLink
            className={styles.navLink}
            activeClassName={joinClassNames(styles.navLink, styles.navLinkActive)}
            to="/trainer/clients"
          >
            Clients
          </NavLink>
          <NavLink
            className={styles.navLink}
            activeClassName={joinClassNames(styles.navLink, styles.navLinkActive)}
            to="/trainer/programs"
          >
            Programs
          </NavLink>
          <NavLink
            className={styles.navLink}
            activeClassName={joinClassNames(styles.navLink, styles.navLinkActive)}
            to="/trainer/profile"
          >
            Profile
          </NavLink>
        </nav>

        {userName ? (
          <div className={styles.userArea}>
            <span className={styles.userName}>{userName}</span>
            <span className={styles.avatar} aria-hidden="true">
              <UserRound strokeWidth={1.6} />
            </span>
          </div>
        ) : (
          <div className={styles.spacer} aria-hidden="true" />
        )}
      </header>

      <main className={styles.main}>{children}</main>
    </PageWrapper>
  );
};
