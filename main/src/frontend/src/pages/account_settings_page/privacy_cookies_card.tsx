import { BellRing, Cookie, FileText, HeartPulse, ShieldCheck } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { Button } from "@/components/button/button";
import { LEGAL_PATHS } from "@/constants/legal";
import { useI18n } from "@/i18n/i18n";
import { fetchHealthConsent, withdrawHealthConsent } from "@/services/body_stats_api";
import { fetchTrainingReminders, saveTrainingReminders } from "@/services/training_reminders_api";
import { COOKIE_CONSENT_CHANGED_EVENT, type CookieConsent, openCookieSettings, readConsent } from "@/services/cookie_consent";

import styles from "./account_settings_page.module.css";

const COPY = {
  en: {
    title: "Privacy and notifications",
    reminders: "Training reminders",
    remindersText: "A short email (and a note in the app) when you have not trained for a few days. At most once a week.",
    text: "RYZE only uses the cookies needed to sign you in, pay and remember your language. Review your choice for optional cookies at any time.",
    choice: "Your choice",
    none: "Not chosen yet",
    essential: "Strictly necessary only",
    all: "All accepted",
    custom: "Customised",
    on: "on",
    manage: "Manage cookies",
    privacy: "Privacy Policy",
    cookies: "Cookie Policy",
    terms: "Terms and Conditions",
    health: "Health data",
    healthGiven: (date: string) => `You consented on ${date} to RYZE processing your health data (measurements and questionnaire answers).`,
    healthNone: "You have not given consent for health data. It is asked when you save a measurement or send a questionnaire.",
    withdraw: "Withdraw consent",
    withdrawConfirm:
      "Withdraw your consent? RYZE erases your body measurements and the answers to your Premium Level 2 questionnaires. A personalised service not yet delivered can no longer be prepared.",
    withdrawn: "Consent withdrawn. Your health data was erased.",
    withdrawError: "We could not withdraw the consent. Please try again."
  },
  pt: {
    title: "Privacidade e notificações",
    reminders: "Lembretes de treino",
    remindersText: "Um email curto (e um aviso na app) quando não treinas há uns dias. No máximo uma vez por semana.",
    text: "A RYZE só usa os cookies necessários para iniciares sessão, pagares e guardarmos o teu idioma. Podes rever a tua escolha sobre cookies opcionais a qualquer momento.",
    choice: "A tua escolha",
    none: "Ainda não escolheste",
    essential: "Só os estritamente necessários",
    all: "Todos aceites",
    custom: "Personalizada",
    on: "a",
    manage: "Gerir cookies",
    privacy: "Política de Privacidade",
    cookies: "Política de Cookies",
    terms: "Termos e Condições",
    health: "Dados de saúde",
    healthGiven: (date: string) => `Deste consentimento a ${date} para a RYZE tratar os teus dados de saúde (medições e respostas aos questionários).`,
    healthNone: "Não deste consentimento para dados de saúde. É pedido quando guardas uma medição ou envias um questionário.",
    withdraw: "Retirar consentimento",
    withdrawConfirm:
      "Retirar o consentimento? A RYZE apaga as tuas medições corporais e as respostas aos questionários do Premium Nível 2. Um serviço Personalizado ainda não entregue deixa de poder ser preparado.",
    withdrawn: "Consentimento retirado. Os teus dados de saúde foram apagados.",
    withdrawError: "Não foi possível retirar o consentimento. Tenta de novo."
  }
} as const;

// PrivacyCookiesCard: the visitor's cookie choice in the account settings
// (owner request, 2026-11-23), reopening the cookie notice to change it.
export const PrivacyCookiesCard = () => {
  const { locale, formatDate } = useI18n();
  const text = COPY[locale];
  const [consent, setConsent] = useState<CookieConsent | null>(() => readConsent());
  const [healthAt, setHealthAt] = useState<string | null | undefined>(undefined);
  const [reminders, setReminders] = useState<boolean | null>(null);

  useEffect(() => {
    fetchTrainingReminders()
      .then(setReminders)
      .catch(() => setReminders(null));
  }, []);

  const toggleReminders = async (enabled: boolean) => {
    const previous = reminders;
    setReminders(enabled);
    try {
      setReminders(await saveTrainingReminders(enabled));
    } catch {
      setReminders(previous);
    }
  };
  const [healthMessage, setHealthMessage] = useState("");
  const [withdrawing, setWithdrawing] = useState(false);

  useEffect(() => {
    fetchHealthConsent()
      .then(setHealthAt)
      .catch(() => setHealthAt(null));
  }, []);

  const withdraw = async () => {
    if (!window.confirm(text.withdrawConfirm)) return;
    setWithdrawing(true);
    setHealthMessage("");
    try {
      await withdrawHealthConsent();
      setHealthAt(null);
      setHealthMessage(text.withdrawn);
    } catch {
      setHealthMessage(text.withdrawError);
    } finally {
      setWithdrawing(false);
    }
  };

  useEffect(() => {
    const refresh = () => setConsent(readConsent());
    window.addEventListener(COOKIE_CONSENT_CHANGED_EVENT, refresh);
    return () => window.removeEventListener(COOKIE_CONSENT_CHANGED_EVENT, refresh);
  }, []);

  const label = !consent
    ? text.none
    : consent.analytics && consent.marketing
      ? text.all
      : !consent.analytics && !consent.marketing
        ? text.essential
        : text.custom;

  return (
    <section className={styles.card} aria-labelledby="account-privacy">
      <h2 id="account-privacy" className={styles.cardTitle}>
        <Cookie aria-hidden="true" /> {text.title}
      </h2>
      <p className={styles.text}>{text.text}</p>
      <div className={styles.row}>
        <div>
          <strong>{text.choice}</strong>
          <p>
            {label}
            {consent ? ` · ${text.on} ${formatDate(consent.decidedAt)}` : ""}
          </p>
        </div>
        <Button variant="secondary" icon={<Cookie aria-hidden="true" />} iconPosition="left" onClick={openCookieSettings}>
          {text.manage}
        </Button>
      </div>
      {reminders !== null ? (
        <div className={styles.row}>
          <div>
            <strong>
              <BellRing aria-hidden="true" className={styles.inlineIcon} /> {text.reminders}
            </strong>
            <p>{text.remindersText}</p>
          </div>
          <label className={styles.switch}>
            <input type="checkbox" checked={reminders} onChange={(event) => void toggleReminders(event.target.checked)} aria-label={text.reminders} />
            <i aria-hidden="true" />
          </label>
        </div>
      ) : null}
      <div className={styles.row}>
        <div>
          <strong>
            <HeartPulse aria-hidden="true" className={styles.inlineIcon} /> {text.health}
          </strong>
          <p>
            {healthAt === undefined ? "…" : healthAt ? text.healthGiven(formatDate(healthAt)) : text.healthNone}
          </p>
          {healthMessage ? <p role="status">{healthMessage}</p> : null}
        </div>
        {healthAt ? (
          <Button variant="secondary" disabled={withdrawing} onClick={() => void withdraw()}>
            {text.withdraw}
          </Button>
        ) : null}
      </div>
      <div className={styles.legalLinks}>
        <Link to={LEGAL_PATHS.privacy}>
          <ShieldCheck aria-hidden="true" /> {text.privacy}
        </Link>
        <Link to={LEGAL_PATHS.cookies}>
          <Cookie aria-hidden="true" /> {text.cookies}
        </Link>
        <Link to={LEGAL_PATHS.terms}>
          <FileText aria-hidden="true" /> {text.terms}
        </Link>
      </div>
    </section>
  );
};
