import { motion } from "framer-motion";
import { CalendarDays, LogOut, Mail, UserRound } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { BrandMark } from "@/components/brand_mark/brand_mark";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { joinClassNames } from "@utils/class_names";
import { ApiError, apiGet, apiPost } from "@utils/http_client";

import styles from "./profile_page.module.css";

interface ProfileUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  created_at: string;
  updated_at: string;
}

type ProfileState =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; user: ProfileUser };

// No avatar upload system exists yet, so the default person icon is always
// used. The imageUrl prop keeps this component ready for a real image later.
const DEFAULT_AVATAR_IMAGE_URL: string | null = null;

interface ProfileAvatarProps {
  imageUrl: string | null;
  size: "small" | "large";
}

const ProfileAvatar = ({ imageUrl, size }: ProfileAvatarProps) => (
  <span className={joinClassNames(styles.avatar, size === "large" ? styles.avatarLarge : styles.avatarSmall)}>
    {imageUrl ? <img src={imageUrl} alt="" /> : <UserRound aria-hidden="true" strokeWidth={1.6} />}
  </span>
);

export const ProfilePage = () => {
  const history = useHistory();
  const [state, setState] = useState<ProfileState>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);
  const [loggingOut, setLoggingOut] = useState(false);
  const [logoutError, setLogoutError] = useState("");

  useEffect(() => {
    let cancelled = false;

    const loadUser = async () => {
      setState({ status: "loading" });

      try {
        const user = await apiGet<ProfileUser>("/me");
        if (!cancelled) {
          setState({ status: "ready", user });
        }
      } catch (error) {
        if (cancelled) {
          return;
        }

        if (error instanceof ApiError && error.status === 401) {
          history.replace("/login");
          return;
        }

        setState({ status: "error" });
      }
    };

    loadUser();

    return () => {
      cancelled = true;
    };
  }, [history, retryCount]);

  const handleLogout = async () => {
    if (loggingOut) {
      return;
    }

    setLoggingOut(true);
    setLogoutError("");

    try {
      await apiPost("/auth/logout", {});
      history.replace("/login");
    } catch {
      setLoggingOut(false);
      setLogoutError("Unable to log out. Please try again.");
    }
  };

  const fullName = state.status === "ready" ? `${state.user.first_name} ${state.user.last_name}`.trim() : "";
  const memberSince =
    state.status === "ready"
      ? new Date(state.user.created_at).toLocaleDateString(undefined, {
          year: "numeric",
          month: "long"
        })
      : "";

  return (
    <PageWrapper className={styles.page}>
      <AnimatedBackground />

      <header className={styles.navbar}>
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <div className={styles.center} aria-hidden="true" />

        {state.status === "ready" ? (
          <div className={styles.userArea}>
            <span className={styles.userName}>{state.user.first_name}</span>
            <ProfileAvatar imageUrl={DEFAULT_AVATAR_IMAGE_URL} size="small" />
          </div>
        ) : null}
      </header>

      <main className={styles.main}>
        {state.status === "loading" ? <LoadingScreen /> : null}

        {state.status === "error" ? (
          <section className={styles.errorCard}>
            <p>Unable to load your profile.</p>
            <Button variant="secondary" onClick={() => setRetryCount((count) => count + 1)}>
              Try again
            </Button>
          </section>
        ) : null}

        {state.status === "ready" ? (
          <motion.section
            className={styles.card}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.35, ease: "easeOut" }}
          >
            <ProfileAvatar imageUrl={DEFAULT_AVATAR_IMAGE_URL} size="large" />

            <h1>{fullName}</h1>

            <p className={styles.email}>
              <Mail aria-hidden="true" />
              {state.user.email}
            </p>

            <p className={styles.meta}>
              <CalendarDays aria-hidden="true" />
              Member since {memberSince}
            </p>

            <span className={styles.divider} aria-hidden="true" />

            {logoutError ? (
              <p className={styles.logoutError} role="alert">
                {logoutError}
              </p>
            ) : null}

            <Button
              variant="secondary"
              icon={<LogOut aria-hidden="true" />}
              iconPosition="left"
              disabled={loggingOut}
              onClick={handleLogout}
            >
              Log out
            </Button>
          </motion.section>
        ) : null}
      </main>
    </PageWrapper>
  );
};
