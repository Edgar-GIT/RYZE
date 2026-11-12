import { KeyRound } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";

import { useI18n } from "@/i18n/i18n";
import { ApiError, apiDelete, apiGet, apiUrl } from "@utils/http_client";

import styles from "./sign_in_methods.module.css";

interface Identity {
  provider: string;
  email: string;
  linked_at: string;
  last_used_at: string | null;
}

interface SignInMethodsResponse {
  password_set: boolean;
  google_available: boolean;
  identities: Identity[];
}

// Results the API reports on the profile after connecting Google (?google=).
const LINK_RESULTS = ["linked", "in_use", "already_linked", "cancelled", "failed", "unverified", "verify_email"] as const;
type LinkResult = (typeof LINK_RESULTS)[number];

const isLinkResult = (value: string | null): value is LinkResult => LINK_RESULTS.includes(value as LinkResult);

const GoogleMark = () => (
  <svg viewBox="0 0 48 48" aria-hidden="true">
    <path
      fill="#EA4335"
      d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"
    />
    <path
      fill="#4285F4"
      d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"
    />
    <path
      fill="#FBBC05"
      d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"
    />
    <path
      fill="#34A853"
      d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"
    />
  </svg>
);

// SignInMethods lists how the owner signs in (password, Google) and lets them
// connect or disconnect Google. Google is never disconnected while it is the
// only way in; the API enforces the same rule.
export const SignInMethods = () => {
  const { t } = useI18n();
  const history = useHistory();
  const location = useLocation();
  const [methods, setMethods] = useState<SignInMethodsResponse | null>(null);
  const [loadFailed, setLoadFailed] = useState(false);
  const [reloadCount, setReloadCount] = useState(0);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<{ tone: "ok" | "error"; text: string } | null>(() => {
    const value = new URLSearchParams(location.search).get("google");
    if (!isLinkResult(value)) {
      return null;
    }
    return { tone: value === "linked" ? "ok" : "error", text: t(`auth.methods.result.${value}`) };
  });

  // The connect result is shown once, then removed from the address bar.
  useEffect(() => {
    if (new URLSearchParams(location.search).has("google")) {
      history.replace(location.pathname);
    }
  }, [history, location.pathname, location.search]);

  useEffect(() => {
    let cancelled = false;
    setLoadFailed(false);
    apiGet<SignInMethodsResponse>("/me/identities")
      .then((response) => {
        if (!cancelled) {
          setMethods(response);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setLoadFailed(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [reloadCount]);

  if (loadFailed) {
    return (
      <section className={styles.methods} aria-labelledby="sign-in-methods-title">
        <h2 id="sign-in-methods-title" className={styles.title}>
          {t("auth.methods.title")}
        </h2>
        <p className={styles.noticeError} role="alert">
          {t("auth.methods.loadError")}
        </p>
        <button type="button" className={styles.action} onClick={() => setReloadCount((count) => count + 1)}>
          {t("common.retry")}
        </button>
      </section>
    );
  }
  if (!methods) {
    return (
      <section className={styles.methods} aria-busy="true" aria-label={t("auth.methods.title")}>
        <h2 className={styles.title}>{t("auth.methods.title")}</h2>
        <p className={styles.loading}>{t("common.loading")}</p>
      </section>
    );
  }

  const google = methods.identities.find((identity) => identity.provider === "google");

  const disconnect = async () => {
    setBusy(true);
    setNotice(null);
    try {
      await apiDelete("/me/identities/google");
      setMethods({ ...methods, identities: methods.identities.filter((identity) => identity.provider !== "google") });
      setNotice({ tone: "ok", text: t("auth.methods.disconnected") });
    } catch (error) {
      const text =
        error instanceof ApiError && error.code === "PASSWORD_REQUIRED"
          ? t("auth.methods.needsPassword")
          : t("common.genericError");
      setNotice({ tone: "error", text });
    } finally {
      setBusy(false);
    }
  };

  const connect = () => {
    setBusy(true);
    window.location.assign(apiUrl("/auth/google/start?mode=link"));
  };

  return (
    <section className={styles.methods} aria-labelledby="sign-in-methods-title">
      <h2 id="sign-in-methods-title" className={styles.title}>
        {t("auth.methods.title")}
      </h2>

      <div className={styles.row}>
        <span className={styles.icon} aria-hidden="true">
          <KeyRound />
        </span>
        <span className={styles.label}>
          <span className={styles.name}>{t("auth.methods.password")}</span>
          <span className={styles.detail}>
            {methods.password_set ? t("auth.methods.passwordSet") : t("auth.methods.passwordMissing")}
          </span>
        </span>
        {methods.password_set ? null : (
          <Link className={styles.action} to="/forgot-password">
            {t("auth.methods.setPassword")}
          </Link>
        )}
      </div>

      {google || methods.google_available ? (
        <div className={styles.row}>
          <span className={styles.icon} aria-hidden="true">
            <GoogleMark />
          </span>
          <span className={styles.label}>
            <span className={styles.name}>{t("auth.methods.google")}</span>
            <span className={styles.detail}>
              {google ? t("auth.methods.connectedAs", { email: google.email }) : t("auth.methods.notConnected")}
            </span>
          </span>
          {google ? (
            <button
              type="button"
              className={styles.action}
              onClick={() => void disconnect()}
              disabled={busy || !methods.password_set}
              aria-describedby={methods.password_set ? undefined : "google-needs-password"}
            >
              {busy ? t("auth.methods.disconnecting") : t("auth.methods.disconnect")}
            </button>
          ) : (
            <button type="button" className={styles.action} onClick={connect} disabled={busy}>
              {t("auth.methods.connect")}
            </button>
          )}
        </div>
      ) : null}

      {!methods.password_set ? (
        <p id="google-needs-password" className={styles.hint}>
          {t("auth.methods.googleOnly")}
        </p>
      ) : null}

      {notice ? (
        <p className={notice.tone === "ok" ? styles.noticeOk : styles.noticeError} role="status">
          {notice.text}
        </p>
      ) : null}
    </section>
  );
};
