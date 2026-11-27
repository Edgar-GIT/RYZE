import { Cookie, X } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";

import { LEGAL_PATHS } from "@/constants/legal";
import { type Copy, useI18n } from "@/i18n/i18n";
import { COOKIE_SETTINGS_EVENT, readConsent, saveConsent } from "@/services/cookie_consent";

import styles from "./cookie_consent.module.css";

const COPY: Copy<{
  title: string;
  text: string;
  policy: string;
  accept: string;
  reject: string;
  customise: string;
  save: string;
  close: string;
  necessary: string;
  necessaryText: string;
  always: string;
  analytics: string;
  analyticsText: string;
  marketing: string;
  marketingText: string;
  notUsed: string;
}> = {
  en: {
    title: "Cookies at RYZE",
    text: "We use cookies and storage that are strictly necessary to sign you in, pay and remember your language. We do not use analytics or advertising cookies; if we ever do, they will only run with your consent.",
    policy: "Cookie Policy",
    accept: "Accept all",
    reject: "Reject optional",
    customise: "Customise",
    save: "Save choice",
    close: "Close",
    necessary: "Strictly necessary",
    necessaryText: "Session, security, payment and the language you choose. Always on.",
    always: "Always on",
    analytics: "Statistics",
    analyticsText: "Anonymous usage measurement.",
    marketing: "Marketing",
    marketingText: "Personalised advertising.",
    notUsed: "Not used today"
  },
  pt: {
    title: "Cookies na RYZE",
    text: "Usamos cookies e armazenamento estritamente necessários para iniciares sessão, pagares e guardarmos o teu idioma. Não usamos cookies de análise nem de publicidade; se algum dia os usarmos, só funcionam com o teu consentimento.",
    policy: "Política de Cookies",
    accept: "Aceitar todos",
    reject: "Rejeitar opcionais",
    customise: "Personalizar",
    save: "Guardar escolha",
    close: "Fechar",
    necessary: "Estritamente necessários",
    necessaryText: "Sessão, segurança, pagamento e o idioma que escolhes. Sempre ativos.",
    always: "Sempre ativos",
    analytics: "Estatística",
    analyticsText: "Medição anónima de utilização.",
    marketing: "Marketing",
    marketingText: "Publicidade personalizada.",
    notUsed: "Não usados de momento"
  }
};

// CookieConsent is the cookie notice (owner request, 2026-11-23): shown until
// the visitor decides, with "Accept" and "Reject" given the same weight, and
// reopened from the Cookie Policy. Hidden in the administration area.
export const CookieConsent = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const { pathname } = useLocation();
  const [open, setOpen] = useState(() => readConsent() === null);
  const [details, setDetails] = useState(false);
  const [analytics, setAnalytics] = useState(false);
  const [marketing, setMarketing] = useState(false);

  useEffect(() => {
    const reopen = () => {
      const current = readConsent();
      setAnalytics(current?.analytics ?? false);
      setMarketing(current?.marketing ?? false);
      setDetails(true);
      setOpen(true);
    };
    window.addEventListener(COOKIE_SETTINGS_EVENT, reopen);
    return () => window.removeEventListener(COOKIE_SETTINGS_EVENT, reopen);
  }, []);

  if (!open || pathname.startsWith("/admin")) return null;

  const decide = (choice: { analytics: boolean; marketing: boolean }) => {
    saveConsent(choice);
    setOpen(false);
    setDetails(false);
  };

  return (
    <section className={styles.notice} role="dialog" aria-modal="false" aria-labelledby="cookie-title">
      <div className={styles.head}>
        <span className={styles.icon} aria-hidden="true">
          <Cookie />
        </span>
        <h2 id="cookie-title" className={styles.title}>
          {text.title}
        </h2>
        {readConsent() ? (
          <button type="button" className={styles.close} onClick={() => setOpen(false)} aria-label={text.close}>
            <X aria-hidden="true" />
          </button>
        ) : null}
      </div>
      <p className={styles.text}>
        {text.text} <Link to={LEGAL_PATHS.cookies}>{text.policy}</Link>
      </p>

      {details ? (
        <ul className={styles.categories}>
          <li>
            <span>
              <strong>{text.necessary}</strong>
              <small>{text.necessaryText}</small>
            </span>
            <em>{text.always}</em>
          </li>
          {(
            [
              ["analytics", text.analytics, text.analyticsText, analytics, setAnalytics],
              ["marketing", text.marketing, text.marketingText, marketing, setMarketing]
            ] as const
          ).map(([key, label, description, value, set]) => (
            <li key={key}>
              <span>
                <strong>{label}</strong>
                <small>
                  {description} {text.notUsed}.
                </small>
              </span>
              <label className={styles.switch}>
                <input type="checkbox" checked={value} onChange={(event) => set(event.target.checked)} aria-label={label} />
                <i aria-hidden="true" />
              </label>
            </li>
          ))}
        </ul>
      ) : null}

      <div className={styles.actions}>
        <button type="button" className={styles.choice} onClick={() => decide({ analytics: false, marketing: false })}>
          {text.reject}
        </button>
        <button type="button" className={styles.choice} onClick={() => decide({ analytics: true, marketing: true })}>
          {text.accept}
        </button>
        {details ? (
          <button type="button" className={styles.link} onClick={() => decide({ analytics, marketing })}>
            {text.save}
          </button>
        ) : (
          <button type="button" className={styles.link} onClick={() => setDetails(true)}>
            {text.customise}
          </button>
        )}
      </div>
    </section>
  );
};
