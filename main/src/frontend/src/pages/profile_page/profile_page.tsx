import { AnimatePresence, motion } from "framer-motion";
import { BadgeCheck, BarChart3, CalendarDays, Dumbbell, Eye, EyeOff, KeyRound, LayoutGrid, LifeBuoy, Lock, LogOut, Mail, ReceiptText, ShoppingBag, Trash2, UserRound } from "lucide-react";
import type { FormEvent } from "react";
import { useEffect, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LanguageSwitcher } from "@/components/language_switcher/language_switcher";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useI18n } from "@/i18n/i18n";
import { fetchTrainerAccess, type TrainerAccessStatus } from "@/services/trainer_api";
import { joinClassNames } from "@utils/class_names";
import { ApiError, apiGet, apiPost } from "@utils/http_client";

import styles from "./profile_page.module.css";
import { SignInMethods } from "./sign_in_methods";

interface ProfileUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  created_at: string;
  updated_at: string;
  email_verified: boolean;
  preferred_language: string | null;
  // False for an account created through Google until a password is set.
  password_set: boolean;
}

// EmailVerificationStatus shows whether the address is confirmed and lets the
// owner ask for a new confirmation link.
const EmailVerificationStatus = ({ verified }: { verified: boolean }) => {
  const { t } = useI18n();
  const [state, setState] = useState<"idle" | "sending" | "sent" | "error">("idle");

  if (verified) {
    return (
      <p className={styles.verified}>
        <BadgeCheck aria-hidden="true" />
        {t("auth.verification.verified")}
      </p>
    );
  }

  const resend = async () => {
    setState("sending");
    try {
      await apiPost("/me/email/verification", {});
      setState("sent");
    } catch (error) {
      setState(error instanceof ApiError && error.code === "EMAIL_ALREADY_VERIFIED" ? "sent" : "error");
    }
  };

  return (
    <div className={styles.unverified} aria-live="polite">
      <span>{t("auth.verification.unverified")}</span>
      {state === "sent" ? (
        <span className={styles.unverifiedNote}>{t("auth.verification.resent")}</span>
      ) : (
        <button type="button" className={styles.unverifiedAction} onClick={() => void resend()} disabled={state === "sending"}>
          {state === "sending" ? t("common.sending") : t("auth.verification.resend")}
        </button>
      )}
      {state === "error" ? <span className={styles.unverifiedNote}>{t("common.genericError")}</span> : null}
    </div>
  );
};

type ProfileState =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; user: ProfileUser };

// No avatar upload system exists yet, so the default person icon is always
// used. The imageUrl prop keeps this component ready for a real image later.
const DEFAULT_AVATAR_IMAGE_URL: string | null = null;

interface ProfileAvatarProps {
  imageUrl: string | null;
}

const ProfileAvatar = ({ imageUrl }: ProfileAvatarProps) => (
  <span className={styles.avatarLarge}>
    {imageUrl ? <img src={imageUrl} alt="" /> : <UserRound aria-hidden="true" strokeWidth={1.6} />}
  </span>
);

// accountErrorFor maps API errors to clean, user-facing messages. Backend or
// internal details are never surfaced to the user.
const accountErrorFor = (error: unknown, t: ReturnType<typeof useI18n>["t"]): string => {
  if (error instanceof ApiError) {
    if (error.code === "INVALID_CREDENTIALS") {
      return t("account.profile.wrongPassword");
    }
    if (error.code === "VALIDATION_ERROR") {
      return t("account.profile.checkDetails");
    }
  }
  return t("common.genericError");
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
  const { t } = useI18n();
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
        aria-label={isVisible ? t("account.profile.hidePassword") : t("account.profile.showPassword")}
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
  const { t } = useI18n();
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
      setErrorMessage(t("account.profile.fillAll"));
      return;
    }

    if (newPassword !== confirmPassword) {
      setErrorMessage(t("account.profile.mismatch"));
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
      setErrorMessage(accountErrorFor(error, t));
      setSubmitting(false);
    }
  };

  return (
    <div className={styles.panel}>
      <h2>{t("account.profile.changePassword")}</h2>
      <p className={styles.panelHint}>{t("account.profile.changePasswordHint")}</p>
      <form className={styles.panelForm} onSubmit={handleSubmit} noValidate>
        <PasswordField
          name="current_password"
          placeholder={t("account.profile.currentPassword")}
          autoComplete="current-password"
        />
        <PasswordField name="new_password" placeholder={t("account.profile.newPassword")} autoComplete="new-password" />
        <PasswordField
          name="confirm_password"
          placeholder={t("account.profile.confirmPassword")}
          autoComplete="new-password"
        />
        {errorMessage ? (
          <p className={styles.formError} role="alert">
            {errorMessage}
          </p>
        ) : null}
        <div className={styles.panelActions}>
          <Button type="button" variant="secondary" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <Button type="submit" disabled={submitting}>
            {t("account.profile.confirm")}
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
const DeleteAccountPanel = ({ onCancel, onSuccess, passwordSet }: AccountPanelProps & { passwordSet: boolean }) => {
  const { t } = useI18n();
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
      setErrorMessage(t("account.profile.fillAll"));
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      await apiPost("/auth/delete-account", { password });
      onSuccess();
    } catch (error) {
      setErrorMessage(accountErrorFor(error, t));
      setSubmitting(false);
    }
  };

  // Deleting asks for the password; an account created through Google sets
  // one first.
  if (!passwordSet) {
    return (
      <div className={joinClassNames(styles.panel, styles.dangerPanel)}>
        <h2>{t("account.profile.deleteAccount")}</h2>
        <p className={styles.panelHint}>{t("auth.methods.googleOnly")}</p>
        <div className={styles.panelActions}>
          <Button type="button" variant="secondary" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <Link className={styles.panelLink} to="/forgot-password">
            {t("auth.methods.setPassword")}
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className={joinClassNames(styles.panel, styles.dangerPanel)}>
      <h2>{t("account.profile.deleteAccount")}</h2>
      <p className={styles.panelHint}>{t("account.profile.deleteHint")}</p>
      <form className={styles.panelForm} onSubmit={handleSubmit} noValidate>
        <PasswordField name="password" placeholder={t("account.profile.currentPassword")} autoComplete="current-password" />
        {errorMessage ? (
          <p className={styles.formError} role="alert">
            {errorMessage}
          </p>
        ) : null}
        <div className={styles.panelActions}>
          <Button type="button" variant="secondary" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <Button type="submit" variant="danger" disabled={submitting}>
            {t("account.profile.deleteAccount")}
          </Button>
        </div>
      </form>
    </div>
  );
};

type AccountPanel = "change-password" | "delete-account" | null;

export const ProfilePage = () => {
  const { t, formatDate } = useI18n();
  const history = useHistory();
  const [state, setState] = useState<ProfileState>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);
  const [loggingOut, setLoggingOut] = useState(false);
  const [logoutError, setLogoutError] = useState("");
  const [activePanel, setActivePanel] = useState<AccountPanel>(null);
  const [trainerAccess, setTrainerAccess] = useState<TrainerAccessStatus | null>(null);

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

  // The trainer entry resolves from the backend access endpoint. A failure
  // simply hides the entry: it is an additional hub link, never the reason the
  // page fails to render.
  useEffect(() => {
    let cancelled = false;
    fetchTrainerAccess()
      .then((access) => {
        if (!cancelled) {
          setTrainerAccess(access);
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

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
      setLogoutError(t("account.profile.logoutError"));
    }
  };

  const fullName = state.status === "ready" ? `${state.user.first_name} ${state.user.last_name}`.trim() : "";
  const memberSince = state.status === "ready" ? formatDate(state.user.created_at, { year: "numeric", month: "long" }) : "";

  return (
    <PageWrapper className={styles.page}>
      <AnimatedBackground />

      <AccountNav userName={state.status === "ready" ? state.user.first_name : undefined} />

      <main className={styles.main}>
        {state.status === "loading" ? <LoadingScreen /> : null}

        {state.status === "error" ? (
          <section className={styles.errorCard}>
            <p>{t("account.profile.loadError")}</p>
            <Button variant="secondary" onClick={() => setRetryCount((count) => count + 1)}>
              {t("common.retry")}
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
            <ProfileAvatar imageUrl={DEFAULT_AVATAR_IMAGE_URL} />

            <h1>{fullName}</h1>

            <p className={styles.email}>
              <Mail aria-hidden="true" />
              {state.user.email}
            </p>

            <EmailVerificationStatus verified={state.user.email_verified} />

            <p className={styles.meta}>
              <CalendarDays aria-hidden="true" />
              {t("account.profile.memberSince", { date: memberSince })}
            </p>

            <div className={styles.language}>
              <span>{t("language.label")}</span>
              <LanguageSwitcher />
            </div>

            <SignInMethods />

            <span className={styles.divider} aria-hidden="true" />

            <nav className={styles.hub} aria-label={t("account.profile.hubLabel")}>
              <Link className={styles.hubLink} to="/services/my-programs">
                <LayoutGrid className={styles.hubIcon} aria-hidden="true" />
                <span className={styles.hubLabel}>
                  <span className={styles.hubTitle}>{t("account.profile.myPrograms")}</span>
                  <span className={styles.hubHint}>{t("account.profile.myProgramsHint")}</span>
                </span>
              </Link>
              <Link className={styles.hubLink} to="/services/progress">
                <BarChart3 className={styles.hubIcon} aria-hidden="true" />
                <span className={styles.hubLabel}>
                  <span className={styles.hubTitle}>{t("account.profile.progress")}</span>
                  <span className={styles.hubHint}>{t("account.profile.progressHint")}</span>
                </span>
              </Link>
              <Link className={styles.hubLink} to="/account/purchases">
                <ReceiptText className={styles.hubIcon} aria-hidden="true" />
                <span className={styles.hubLabel}>
                  <span className={styles.hubTitle}>{t("account.profile.purchases")}</span>
                  <span className={styles.hubHint}>{t("account.profile.purchasesHint")}</span>
                </span>
              </Link>
              <Link className={styles.hubLink} to="/support">
                <LifeBuoy className={styles.hubIcon} aria-hidden="true" />
                <span className={styles.hubLabel}>
                  <span className={styles.hubTitle}>{t("support.title")}</span>
                  <span className={styles.hubHint}>{t("support.hubHint")}</span>
                </span>
              </Link>
              <Link className={styles.hubLink} to="/services/generic-program">
                <ShoppingBag className={styles.hubIcon} aria-hidden="true" />
                <span className={styles.hubLabel}>
                  <span className={styles.hubTitle}>{t("account.profile.marketplace")}</span>
                  <span className={styles.hubHint}>{t("account.profile.marketplaceHint")}</span>
                </span>
              </Link>
              {trainerAccess?.is_trainer ? (
                <Link className={styles.hubLink} to="/trainer/dashboard">
                  <Dumbbell className={styles.hubIcon} aria-hidden="true" />
                  <span className={styles.hubLabel}>
                    <span className={styles.hubTitle}>{t("account.profile.trainerDashboard")}</span>
                    <span className={styles.hubHint}>{t("account.profile.trainerDashboardHint")}</span>
                  </span>
                </Link>
              ) : null}
              {trainerAccess && !trainerAccess.is_trainer ? (
                <Link className={styles.hubLink} to="/trainer/apply">
                  <Dumbbell className={styles.hubIcon} aria-hidden="true" />
                  <span className={styles.hubLabel}>
                    <span className={styles.hubTitle}>{t("account.profile.becomeTrainer")}</span>
                    <span className={styles.hubHint}>{t("account.profile.becomeTrainerHint")}</span>
                  </span>
                </Link>
              ) : null}
            </nav>

            {logoutError ? (
              <p className={styles.logoutError} role="alert">
                {logoutError}
              </p>
            ) : null}

            <div className={styles.actions}>
              {state.user.password_set ? (
                <Button
                  variant="secondary"
                  icon={<KeyRound aria-hidden="true" />}
                  iconPosition="left"
                  onClick={() => togglePanel("change-password")}
                >
                  {t("account.profile.changePassword")}
                </Button>
              ) : null}
              <Button
                variant="danger"
                icon={<Trash2 aria-hidden="true" />}
                iconPosition="left"
                onClick={() => togglePanel("delete-account")}
              >
                {t("account.profile.deleteAccount")}
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
                  <DeleteAccountPanel
                    onCancel={closePanel}
                    onSuccess={handleAccountActionSuccess}
                    passwordSet={state.user.password_set}
                  />
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
              {t("account.profile.logOut")}
            </Button>
          </motion.section>
        ) : null}
      </main>
    </PageWrapper>
  );
};
