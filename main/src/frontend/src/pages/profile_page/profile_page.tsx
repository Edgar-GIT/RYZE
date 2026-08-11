import { AnimatePresence, motion } from "framer-motion";
import { CalendarDays, Eye, EyeOff, KeyRound, Lock, LogOut, Mail, Trash2, UserRound } from "lucide-react";
import type { FormEvent } from "react";
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

// accountErrorFor maps API errors to clean, user-facing messages. Backend or
// internal details are never surfaced to the user.
const accountErrorFor = (error: unknown): string => {
  if (error instanceof ApiError) {
    if (error.code === "INVALID_CREDENTIALS") {
      return "Your current password is incorrect.";
    }
    if (error.code === "VALIDATION_ERROR") {
      return "Please check your details and try again.";
    }
  }
  return "Something went wrong. Please try again.";
};

// PasswordField mirrors the login form's password input (visibility toggle),
// staying consistent with the existing RYZE design language.
const PasswordField = ({
  name,
  placeholder,
  autoComplete
}: {
  name: string;
  placeholder: string;
  autoComplete: string;
}) => {
  const [isVisible, setIsVisible] = useState(false);

  return (
    <div className={styles.field}>
      <Lock className={styles.fieldIcon} aria-hidden="true" />
      <input
        className={styles.inputAction}
        type={isVisible ? "text" : "password"}
        name={name}
        placeholder={placeholder}
        autoComplete={autoComplete}
        aria-label={placeholder}
      />
      <button
        className={styles.eyeButton}
        type="button"
        aria-label={isVisible ? "Hide password" : "Show password"}
        onClick={() => setIsVisible((currentValue) => !currentValue)}
      >
        {isVisible ? <EyeOff aria-hidden="true" /> : <Eye aria-hidden="true" />}
      </button>
    </div>
  );
};

interface AccountPanelProps {
  onCancel: () => void;
  onSuccess: () => void;
}

// ChangePasswordPanel lets the user replace their password. On success the
// backend invalidates the session and clears the cookie, so the user is
// redirected to the login screen and must authenticate with the new password.
const ChangePasswordPanel = ({ onCancel, onSuccess }: AccountPanelProps) => {
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }

    const form = new FormData(event.currentTarget);
    const currentPassword = String(form.get("current_password") ?? "");
    const newPassword = String(form.get("new_password") ?? "");
    const confirmPassword = String(form.get("confirm_password") ?? "");

    if (!currentPassword || !newPassword || !confirmPassword) {
      setErrorMessage("Please fill in all fields.");
      return;
    }

    if (newPassword !== confirmPassword) {
      setErrorMessage("Passwords do not match.");
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      await apiPost("/auth/change-password", {
        current_password: currentPassword,
        new_password: newPassword
      });
      onSuccess();
    } catch (error) {
      setErrorMessage(accountErrorFor(error));
      setSubmitting(false);
    }
  };

  return (
    <div className={styles.panel}>
      <h2>Change password</h2>
      <p className={styles.panelHint}>
        Your current session will end after the change. Log in again with your new password.
      </p>
      <form className={styles.panelForm} onSubmit={handleSubmit} noValidate>
        <PasswordField name="current_password" placeholder="Current password" autoComplete="current-password" />
        <PasswordField name="new_password" placeholder="New password" autoComplete="new-password" />
        <PasswordField name="confirm_password" placeholder="Confirm new password" autoComplete="new-password" />
        {errorMessage ? (
          <p className={styles.formError} role="alert">
            {errorMessage}
          </p>
        ) : null}
        <div className={styles.panelActions}>
          <Button type="button" variant="secondary" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit" disabled={submitting}>
            Confirm
          </Button>
        </div>
      </form>
    </div>
  );
};

// DeleteAccountPanel asks for explicit confirmation and the current password
// before calling the existing delete-account endpoint. The account is never
// deleted from the first click; the backend performs the soft delete and
// clears the session.
const DeleteAccountPanel = ({ onCancel, onSuccess }: AccountPanelProps) => {
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }

    const form = new FormData(event.currentTarget);
    const password = String(form.get("password") ?? "");

    if (!password) {
      setErrorMessage("Please fill in all fields.");
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      await apiPost("/auth/delete-account", { password });
      onSuccess();
    } catch (error) {
      setErrorMessage(accountErrorFor(error));
      setSubmitting(false);
    }
  };

  return (
    <div className={joinClassNames(styles.panel, styles.dangerPanel)}>
      <h2>Delete account</h2>
      <p className={styles.panelHint}>
        Your account will be closed and your current session will end. Enter your password to confirm.
      </p>
      <form className={styles.panelForm} onSubmit={handleSubmit} noValidate>
        <PasswordField name="password" placeholder="Current password" autoComplete="current-password" />
        {errorMessage ? (
          <p className={styles.formError} role="alert">
            {errorMessage}
          </p>
        ) : null}
        <div className={styles.panelActions}>
          <Button type="button" variant="secondary" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit" variant="danger" disabled={submitting}>
            Delete account
          </Button>
        </div>
      </form>
    </div>
  );
};

type AccountPanel = "change-password" | "delete-account" | null;

export const ProfilePage = () => {
  const history = useHistory();
  const [state, setState] = useState<ProfileState>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);
  const [loggingOut, setLoggingOut] = useState(false);
  const [logoutError, setLogoutError] = useState("");
  const [activePanel, setActivePanel] = useState<AccountPanel>(null);

  const togglePanel = (panel: Exclude<AccountPanel, null>) => {
    setActivePanel((current) => (current === panel ? null : panel));
  };

  const closePanel = () => {
    setActivePanel(null);
  };

  // After a successful password change or account deletion the backend
  // invalidates the session and clears the cookie, so the user must go back to
  // the login screen and authenticate again.
  const handleAccountActionSuccess = () => {
    history.replace("/login");
  };

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

            <div className={styles.actions}>
              <Button
                variant="secondary"
                icon={<KeyRound aria-hidden="true" />}
                iconPosition="left"
                onClick={() => togglePanel("change-password")}
              >
                Change password
              </Button>
              <Button
                variant="danger"
                icon={<Trash2 aria-hidden="true" />}
                iconPosition="left"
                onClick={() => togglePanel("delete-account")}
              >
                Delete account
              </Button>
            </div>

            <AnimatePresence initial={false}>
              {activePanel === "change-password" ? (
                <motion.div
                  className={styles.panelWrap}
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ duration: 0.25, ease: "easeOut" }}
                >
                  <ChangePasswordPanel onCancel={closePanel} onSuccess={handleAccountActionSuccess} />
                </motion.div>
              ) : null}
              {activePanel === "delete-account" ? (
                <motion.div
                  className={styles.panelWrap}
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ duration: 0.25, ease: "easeOut" }}
                >
                  <DeleteAccountPanel onCancel={closePanel} onSuccess={handleAccountActionSuccess} />
                </motion.div>
              ) : null}
            </AnimatePresence>

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
