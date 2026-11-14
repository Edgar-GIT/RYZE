import { AnimatePresence, motion } from "framer-motion";
import { BadgeCheck, CalendarDays, Dumbbell, Eye, EyeOff, KeyRound, Languages, Lock, LogOut, Mail, ShieldAlert, Trash2, Users } from "lucide-react";
import type { FormEvent } from "react";
import { useState } from "react";
import { Link, useHistory } from "react-router-dom";

import { useMember } from "@/components/app_shell/member_shell";
import { Button } from "@/components/button/button";
import { LanguageSwitcher } from "@/components/language_switcher/language_switcher";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { type Copy, useI18n } from "@/i18n/i18n";
import { SignInMethods } from "@/pages/profile_page/sign_in_methods";
import { joinClassNames } from "@utils/class_names";
import { ApiError, apiPost } from "@utils/http_client";

import styles from "./account_settings_page.module.css";

const COPY: Copy<{
  kicker: string;
  title: string;
  highlight: string;
  lead: string;
  profile: string;
  preferences: string;
  languageText: string;
  access: string;
  security: string;
  securityText: string;
  trainer: string;
  trainerOpenText: string;
  trainerApplyText: string;
  session: string;
  sessionText: string;
  danger: string;
  dangerText: string;
}> = {
  en: {
    kicker: "Account",
    title: "Your ",
    highlight: "account.",
    lead: "Your details, preferences, sign-in and security in one place.",
    profile: "Profile",
    preferences: "Preferences",
    languageText: "The language of the site and of your emails.",
    access: "Sign-in methods",
    security: "Security",
    securityText: "Change the password you sign in with. You will sign in again afterwards.",
    trainer: "Trainer area",
    trainerOpenText: "Manage your clients and programmes.",
    trainerApplyText: "Apply to publish programmes and coach clients on RYZE.",
    session: "Session",
    sessionText: "Sign out of RYZE on this device.",
    danger: "Delete account",
    dangerText: "Permanently close your account. This cannot be undone."
  },
  pt: {
    kicker: "Conta",
    title: "A tua ",
    highlight: "conta.",
    lead: "Os teus dados, preferências, acesso e segurança num só sítio.",
    profile: "Perfil",
    preferences: "Preferências",
    languageText: "A língua do site e dos teus emails.",
    access: "Métodos de entrada",
    security: "Segurança",
    securityText: "Muda a password com que entras. Depois voltas a entrar.",
    trainer: "Área de treinador",
    trainerOpenText: "Gere os teus clientes e programas.",
    trainerApplyText: "Candidata-te para publicares programas e treinares clientes na RYZE.",
    session: "Sessão",
    sessionText: "Sai da RYZE neste dispositivo.",
    danger: "Apagar conta",
    dangerText: "Fecha a tua conta de forma definitiva. Não é possível desfazer."
  }
};

interface ProfileUser {
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

const initialsOf = (first: string, last: string) => `${first[0] ?? ""}${last[0] ?? ""}`.toUpperCase() || "R";

// AccountSettingsPage: the member's details, language, sign-in methods,
// password, trainer entry, sign out and account deletion. Two columns on
// computers, one on phones.
export const AccountSettingsPage = () => {
  const { t, locale, formatDate } = useI18n();
  const text = COPY[locale];
  const history = useHistory();
  const { user, isTrainer } = useMember();
  const [loggingOut, setLoggingOut] = useState(false);
  const [logoutError, setLogoutError] = useState("");
  const [activePanel, setActivePanel] = useState<AccountPanel>(null);

  const togglePanel = (panel: Exclude<AccountPanel, null>) => {
    setActivePanel((current) => (current === panel ? null : panel));
  };

  // After a password change or account deletion the backend ends the session,
  // so the member signs in again.
  const handleAccountActionSuccess = () => {
    history.replace("/login");
  };

  const handleLogout = async () => {
    if (loggingOut) return;
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

  const fullName = user ? `${user.first_name} ${user.last_name}`.trim() : "";

  return (
    <PageWrapper className={styles.page}>
      <header className={styles.header}>
        <p className={styles.kicker}>{text.kicker}</p>
        <h1 className={styles.title}>
          {text.title}
          <span>{text.highlight}</span>
        </h1>
        <p className={styles.lead}>{text.lead}</p>
      </header>

      {user ? (
        <div className={styles.grid}>
          <div className={styles.column}>
            <section className={joinClassNames(styles.card, styles.profile)} aria-labelledby="account-profile">
              <span className={styles.avatar} aria-hidden="true">
                {initialsOf(user.first_name, user.last_name)}
              </span>
              <div className={styles.identity}>
                <h2 id="account-profile" className={styles.name}>
                  {fullName}
                </h2>
                <p className={styles.line}>
                  <Mail aria-hidden="true" />
                  {user.email}
                </p>
                <p className={styles.line}>
                  <CalendarDays aria-hidden="true" />
                  {t("account.profile.memberSince", { date: formatDate(user.created_at, { year: "numeric", month: "long" }) })}
                </p>
                <EmailVerificationStatus verified={user.email_verified} />
              </div>
            </section>

            <section className={styles.card} aria-labelledby="account-preferences">
              <h2 id="account-preferences" className={styles.cardTitle}>
                <Languages aria-hidden="true" /> {text.preferences}
              </h2>
              <div className={styles.row}>
                <div>
                  <strong>{t("language.label")}</strong>
                  <p>{text.languageText}</p>
                </div>
                <LanguageSwitcher variant="compact" />
              </div>
            </section>

            <section className={styles.card} aria-label={text.access}>
              <SignInMethods />
            </section>
          </div>

          <div className={styles.column}>
            <section className={styles.card} aria-labelledby="account-security">
              <h2 id="account-security" className={styles.cardTitle}>
                <KeyRound aria-hidden="true" /> {text.security}
              </h2>
              <p className={styles.text}>{text.securityText}</p>
              {user.password_set ? (
                <div className={styles.actions}>
                  <Button variant="secondary" icon={<KeyRound aria-hidden="true" />} iconPosition="left" onClick={() => togglePanel("change-password")}>
                    {t("account.profile.changePassword")}
                  </Button>
                </div>
              ) : (
                <p className={styles.text}>{t("auth.methods.googleOnly")}</p>
              )}
              <AnimatePresence initial={false}>
                {activePanel === "change-password" ? (
                  <motion.div
                    className={styles.panelWrap}
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    transition={{ duration: 0.25, ease: "easeOut" }}
                  >
                    <ChangePasswordPanel onCancel={() => setActivePanel(null)} onSuccess={handleAccountActionSuccess} />
                  </motion.div>
                ) : null}
              </AnimatePresence>
            </section>

            <Link className={styles.trainerCard} to={isTrainer ? "/trainer/dashboard" : "/trainer/apply"}>
              <span className={styles.trainerIcon} aria-hidden="true">
                {isTrainer ? <Users /> : <Dumbbell />}
              </span>
              <span className={styles.trainerText}>
                <strong>{isTrainer ? text.trainer : t("account.profile.becomeTrainer")}</strong>
                <span>{isTrainer ? text.trainerOpenText : text.trainerApplyText}</span>
              </span>
            </Link>

            <section className={styles.card} aria-labelledby="account-session">
              <h2 id="account-session" className={styles.cardTitle}>
                <LogOut aria-hidden="true" /> {text.session}
              </h2>
              <p className={styles.text}>{text.sessionText}</p>
              {logoutError ? (
                <p className={styles.formError} role="alert">
                  {logoutError}
                </p>
              ) : null}
              <div className={styles.actions}>
                <Button variant="secondary" icon={<LogOut aria-hidden="true" />} iconPosition="left" disabled={loggingOut} onClick={handleLogout}>
                  {t("account.profile.logOut")}
                </Button>
              </div>
            </section>

            <section className={joinClassNames(styles.card, styles.danger)} aria-labelledby="account-danger">
              <h2 id="account-danger" className={styles.cardTitle}>
                <ShieldAlert aria-hidden="true" /> {text.danger}
              </h2>
              <p className={styles.text}>{text.dangerText}</p>
              <div className={styles.actions}>
                <Button variant="danger" icon={<Trash2 aria-hidden="true" />} iconPosition="left" onClick={() => togglePanel("delete-account")}>
                  {t("account.profile.deleteAccount")}
                </Button>
              </div>
              <AnimatePresence initial={false}>
                {activePanel === "delete-account" ? (
                  <motion.div
                    className={styles.panelWrap}
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    transition={{ duration: 0.25, ease: "easeOut" }}
                  >
                    <DeleteAccountPanel
                      onCancel={() => setActivePanel(null)}
                      onSuccess={handleAccountActionSuccess}
                      passwordSet={(user as ProfileUser).password_set}
                    />
                  </motion.div>
                ) : null}
              </AnimatePresence>
            </section>
          </div>
        </div>
      ) : (
        <div className={styles.loading} role="status" aria-busy="true" />
      )}
    </PageWrapper>
  );
};
