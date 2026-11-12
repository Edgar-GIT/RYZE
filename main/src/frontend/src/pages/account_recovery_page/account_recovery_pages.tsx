import { ArrowRight, CheckCircle2, KeyRound, MailCheck } from "lucide-react";
import type { FormEvent, ReactNode } from "react";
import { useEffect, useId, useRef, useState } from "react";
import { Link } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { LanguageSwitcher } from "@/components/language_switcher/language_switcher";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PublicAction, Waves } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";
import { ApiError, apiPost } from "@utils/http_client";

import styles from "./account_recovery_pages.module.css";

const MIN_PASSWORD_LENGTH = 12;
const MAX_PASSWORD_LENGTH = 128;

// The token travels in the URL fragment (#token=...), which browsers never
// send to a server, so it cannot appear in access logs or Referer headers.
const tokenFromHash = (): string => {
  const params = new URLSearchParams(window.location.hash.replace(/^#/, ""));
  return params.get("token") ?? "";
};

// Removes the token from the address bar once read, so it is not kept in the
// browser history or shared by accident.
const clearHash = () => {
  window.history.replaceState(null, "", window.location.pathname + window.location.search);
};

const RecoveryShell = ({ icon, title, lead, children }: { icon: ReactNode; title: string; lead?: string; children: ReactNode }) => (
  <PageWrapper className={joinClassNames(ui.page, styles.page)}>
    <section className={joinClassNames(ui.toneNavy, styles.section)}>
      <header className={styles.top}>
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>
        <LanguageSwitcher />
      </header>
      <div className={styles.card}>
        <div className={joinClassNames(ui.toneLight, styles.cardHead)}>
          <span className={styles.cardIcon} aria-hidden="true">
            {icon}
          </span>
          <Waves />
        </div>
        <div className={styles.cardBody}>
          <h1 className={styles.title}>{title}</h1>
          {lead ? <p className={ui.body}>{lead}</p> : null}
          {children}
        </div>
      </div>
    </section>
  </PageWrapper>
);

export const ForgotPasswordPage = () => {
  const { t, locale } = useI18n();
  const emailId = useId();
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [sentTo, setSentTo] = useState("");

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const address = email.trim();
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(address)) {
      setError(t("auth.forgot.invalidEmail"));
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      await apiPost("/auth/password/forgot", { email: address, language: locale });
      setSentTo(address);
    } catch (caught) {
      if (caught instanceof ApiError && caught.status === 429) {
        setError(t("auth.forgot.rateLimited"));
      } else if (caught instanceof ApiError && caught.status === 400) {
        setError(t("auth.forgot.invalidEmail"));
      } else {
        setError(t("common.genericError"));
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (sentTo) {
    return (
      <RecoveryShell icon={<MailCheck />} title={t("auth.forgot.sentTitle")}>
        <p className={ui.body} role="status">
          {t("auth.forgot.sent", { email: sentTo })}
        </p>
        <PublicAction to="/login" variant="ghostOnDark">
          {t("common.backToLogin")}
        </PublicAction>
      </RecoveryShell>
    );
  }

  return (
    <RecoveryShell icon={<KeyRound />} title={t("auth.forgot.title")} lead={t("auth.forgot.lead")}>
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <label className={styles.field} htmlFor={emailId}>
          <span>{t("auth.forgot.email")}</span>
          <input
            id={emailId}
            type="email"
            autoComplete="email"
            value={email}
            placeholder={t("auth.forgot.emailPlaceholder")}
            onChange={(event) => setEmail(event.target.value)}
            required
          />
        </label>
        {error ? (
          <p className={styles.error} role="alert">
            {error}
          </p>
        ) : null}
        <PublicAction type="submit" disabled={submitting} icon={<ArrowRight />}>
          {submitting ? t("common.sending") : t("auth.forgot.submit")}
        </PublicAction>
        <Link className={styles.quietLink} to="/login">
          {t("common.backToLogin")}
        </Link>
      </form>
    </RecoveryShell>
  );
};

export const ResetPasswordPage = () => {
  const { t } = useI18n();
  const passwordId = useId();
  const confirmId = useId();
  const token = useRef(tokenFromHash());
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [state, setState] = useState<"form" | "done" | "invalid">(token.current ? "form" : "invalid");

  useEffect(() => {
    if (token.current) {
      clearHash();
    }
  }, []);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (password.length < MIN_PASSWORD_LENGTH || password.length > MAX_PASSWORD_LENGTH) {
      setError(t("auth.reset.tooShort"));
      return;
    }
    if (password !== confirm) {
      setError(t("auth.reset.mismatch"));
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      await apiPost("/auth/password/reset", { token: token.current, password });
      setState("done");
    } catch (caught) {
      if (caught instanceof ApiError && caught.code === "INVALID_OR_EXPIRED_LINK") {
        setState("invalid");
      } else if (caught instanceof ApiError && caught.status === 400) {
        setError(t("auth.reset.tooShort"));
      } else {
        setError(t("common.genericError"));
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (state === "done") {
    return (
      <RecoveryShell icon={<CheckCircle2 />} title={t("auth.reset.title")}>
        <p className={ui.body} role="status">
          {t("auth.reset.done")}
        </p>
        <PublicAction to="/login" icon={<ArrowRight />}>
          {t("common.backToLogin")}
        </PublicAction>
      </RecoveryShell>
    );
  }

  if (state === "invalid") {
    return (
      <RecoveryShell icon={<KeyRound />} title={t("auth.reset.title")}>
        <p className={styles.error} role="alert">
          {token.current ? t("auth.reset.invalidLink") : t("auth.reset.missingToken")}
        </p>
        <PublicAction to="/forgot-password" icon={<ArrowRight />}>
          {t("auth.reset.requestNew")}
        </PublicAction>
      </RecoveryShell>
    );
  }

  return (
    <RecoveryShell icon={<KeyRound />} title={t("auth.reset.title")} lead={t("auth.reset.lead")}>
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <label className={styles.field} htmlFor={passwordId}>
          <span>{t("auth.reset.password")}</span>
          <input
            id={passwordId}
            type="password"
            autoComplete="new-password"
            value={password}
            minLength={MIN_PASSWORD_LENGTH}
            onChange={(event) => setPassword(event.target.value)}
            required
          />
        </label>
        <label className={styles.field} htmlFor={confirmId}>
          <span>{t("auth.reset.confirm")}</span>
          <input
            id={confirmId}
            type="password"
            autoComplete="new-password"
            value={confirm}
            onChange={(event) => setConfirm(event.target.value)}
            required
          />
        </label>
        {error ? (
          <p className={styles.error} role="alert">
            {error}
          </p>
        ) : null}
        <PublicAction type="submit" disabled={submitting} icon={<ArrowRight />}>
          {submitting ? t("common.saving") : t("auth.reset.submit")}
        </PublicAction>
      </form>
    </RecoveryShell>
  );
};

export const VerifyEmailPage = () => {
  const { t } = useI18n();
  const token = useRef(tokenFromHash());
  const [state, setState] = useState<"verifying" | "done" | "invalid" | "error">(token.current ? "verifying" : "invalid");
  const started = useRef(false);

  useEffect(() => {
    if (!token.current || started.current) {
      return;
    }
    started.current = true;
    clearHash();
    apiPost("/auth/email/verify", { token: token.current })
      .then(() => setState("done"))
      .catch((caught) => {
        setState(caught instanceof ApiError && caught.code === "INVALID_OR_EXPIRED_LINK" ? "invalid" : "error");
      });
  }, []);

  return (
    <RecoveryShell icon={state === "done" ? <CheckCircle2 /> : <MailCheck />} title={t("auth.verify.title")}>
      {state === "verifying" ? (
        <p className={ui.body} role="status">
          {t("auth.verify.verifying")}
        </p>
      ) : null}
      {state === "done" ? (
        <p className={ui.body} role="status">
          {t("auth.verify.done")}
        </p>
      ) : null}
      {state === "invalid" ? (
        <p className={styles.error} role="alert">
          {t("auth.verify.invalidLink")}
        </p>
      ) : null}
      {state === "error" ? (
        <p className={styles.error} role="alert">
          {t("common.genericError")}
        </p>
      ) : null}
      {state !== "verifying" ? (
        <PublicAction to="/profile" icon={<ArrowRight />}>
          {t("auth.verify.goToProfile")}
        </PublicAction>
      ) : null}
    </RecoveryShell>
  );
};
