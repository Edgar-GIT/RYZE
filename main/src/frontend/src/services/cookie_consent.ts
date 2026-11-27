/**
 * The visitor's cookie choice (owner request, 2026-11-23: EU ePrivacy and
 * GDPR). RYZE only uses strictly necessary cookies and functional storage the
 * visitor asks for, so nothing optional runs today; the choice is kept so that
 * any optional category added later is only enabled with prior consent
 * (`hasConsent`). The choice expires after 12 months, then the notice shows
 * again.
 */
export type ConsentCategory = "analytics" | "marketing";

export interface CookieConsent {
  version: number;
  analytics: boolean;
  marketing: boolean;
  decidedAt: string;
}

const KEY = "ryze.cookieConsent";
const VERSION = 1;
const MAX_AGE_MS = 365 * 24 * 60 * 60 * 1000;
export const COOKIE_SETTINGS_EVENT = "ryze:cookie-settings";
// Sent after the choice is saved (the account settings show it).
export const COOKIE_CONSENT_CHANGED_EVENT = "ryze:cookie-consent-changed";

export const readConsent = (): CookieConsent | null => {
  try {
    const raw = window.localStorage.getItem(KEY);
    if (!raw) return null;
    const consent = JSON.parse(raw) as CookieConsent;
    if (consent.version !== VERSION || Date.now() - Date.parse(consent.decidedAt) > MAX_AGE_MS) return null;
    return consent;
  } catch {
    return null;
  }
};

export const saveConsent = (choice: Pick<CookieConsent, ConsentCategory>): CookieConsent => {
  const consent: CookieConsent = { version: VERSION, ...choice, decidedAt: new Date().toISOString() };
  try {
    window.localStorage.setItem(KEY, JSON.stringify(consent));
  } catch {
    // Storage blocked: the notice shows again next time; nothing optional runs.
  }
  window.dispatchEvent(new Event(COOKIE_CONSENT_CHANGED_EVENT));
  return consent;
};

export const hasConsent = (category: ConsentCategory): boolean => readConsent()?.[category] === true;

// openCookieSettings reopens the notice (the Cookie Policy's button).
export const openCookieSettings = () => window.dispatchEvent(new Event(COOKIE_SETTINGS_EVENT));
