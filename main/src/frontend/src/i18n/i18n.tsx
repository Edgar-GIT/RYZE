import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";

import { apiPatch } from "@utils/http_client";

import { en } from "./messages/en";
import type { Messages } from "./messages/en";
import { pt } from "./messages/pt";

/*
 * RYZE internationalisation (no external library).
 *
 * - Dictionaries are typed: pt.ts must provide every key en.ts has, so a page
 *   can never be half translated by omission.
 * - First visit: Portuguese when the device is in Portugal (time zone
 *   Europe/Lisbon, Atlantic/Madeira, Atlantic/Azores) or the browser prefers
 *   pt-PT; English otherwise. The detected language is stored, so it does not
 *   change from visit to visit.
 * - A manual choice is stored as manual, always wins over detection, and is
 *   saved to the account (it drives the language of RYZE emails).
 */

export type Locale = "en" | "pt";

export const LOCALES: readonly Locale[] = ["en", "pt"];

const STORAGE_KEY = "ryze.locale";
const PORTUGAL_TIME_ZONES = ["Europe/Lisbon", "Atlantic/Madeira", "Atlantic/Azores"];

const DICTIONARIES: Record<Locale, Messages> = { en, pt };

// Intl locale per interface language (European formats in both).
const INTL_LOCALE: Record<Locale, string> = { en: "en-GB", pt: "pt-PT" };

interface StoredLocale {
  locale: Locale;
  // auto: detected and not yet confirmed by the visitor; manual: chosen or
  // confirmed (the first-visit language prompt stops showing).
  source: "auto" | "manual";
}

type Leaves<T, P extends string = ""> = {
  [K in keyof T & string]: T[K] extends string ? `${P}${K}` : Leaves<T[K], `${P}${K}.`>;
}[keyof T & string];

export type TranslationKey = Leaves<Messages>;

export type TranslationParams = Record<string, string | number>;

const isLocale = (value: unknown): value is Locale => value === "en" || value === "pt";

const readStored = (): StoredLocale | null => {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as Partial<StoredLocale>;
    if (isLocale(parsed.locale) && (parsed.source === "auto" || parsed.source === "manual")) {
      return { locale: parsed.locale, source: parsed.source };
    }
  } catch {
    // Storage may be unavailable (private mode); detection still works.
  }
  return null;
};

const writeStored = (value: StoredLocale) => {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(value));
  } catch {
    // Ignore: the choice still applies for this visit.
  }
};

export const detectLocale = (
  timeZone: string | undefined = Intl.DateTimeFormat().resolvedOptions().timeZone,
  languages: readonly string[] = typeof navigator === "undefined" ? [] : navigator.languages ?? [navigator.language]
): Locale => {
  if (timeZone && PORTUGAL_TIME_ZONES.includes(timeZone)) {
    return "pt";
  }
  if (languages.some((language) => language.toLowerCase() === "pt-pt")) {
    return "pt";
  }
  return "en";
};

const initialState = (): StoredLocale => {
  const stored = readStored();
  if (stored) {
    return stored;
  }
  const detected: StoredLocale = { locale: detectLocale(), source: "auto" };
  writeStored(detected);
  return detected;
};

const lookup = (messages: Messages, key: string): string | undefined => {
  let node: unknown = messages;
  for (const part of key.split(".")) {
    if (node && typeof node === "object" && part in node) {
      node = (node as Record<string, unknown>)[part];
    } else {
      return undefined;
    }
  }
  return typeof node === "string" ? node : undefined;
};

const interpolate = (template: string, params?: TranslationParams): string =>
  params ? template.replace(/\{(\w+)\}/g, (match, name: string) => (name in params ? String(params[name]) : match)) : template;

export const translate = (locale: Locale, key: TranslationKey, params?: TranslationParams): string =>
  interpolate(lookup(DICTIONARIES[locale], key) ?? lookup(en, key) ?? key, params);

interface I18nValue {
  locale: Locale;
  // True until the visitor confirms or changes the detected language.
  isAutoDetected: boolean;
  intlLocale: string;
  setLocale: (locale: Locale) => void;
  // Keeps the detected language and stops the first-visit prompt.
  confirmLocale: () => void;
  t: (key: TranslationKey, params?: TranslationParams) => string;
  formatDate: (value: string | number | Date, options?: Intl.DateTimeFormatOptions) => string;
  formatNumber: (value: number, options?: Intl.NumberFormatOptions) => string;
  formatMoney: (minorUnits: number, currency: string) => string;
}

const I18nContext = createContext<I18nValue | null>(null);

export const I18nProvider = ({ children, initialLocale: fixedLocale }: { children: ReactNode; initialLocale?: Locale }) => {
  // initialLocale pins the language (tests); otherwise it comes from the
  // stored choice or the browser.
  const [state, setState] = useState<StoredLocale>(() =>
    fixedLocale ? { locale: fixedLocale, source: "manual" } : initialState()
  );
  const locale = state.locale;

  useEffect(() => {
    document.documentElement.lang = INTL_LOCALE[locale];
  }, [locale]);

  const setLocale = useCallback((next: Locale) => {
    writeStored({ locale: next, source: "manual" });
    setState({ locale: next, source: "manual" });
    // Saved to the account when signed in; a signed-out visitor keeps the
    // local choice only.
    apiPatch("/me/preferences", { language: next }).catch(() => undefined);
  }, []);

  const confirmLocale = useCallback(() => {
    setState((current) => {
      const confirmed: StoredLocale = { locale: current.locale, source: "manual" };
      writeStored(confirmed);
      return confirmed;
    });
  }, []);

  const value = useMemo<I18nValue>(() => {
    const intlLocale = INTL_LOCALE[locale];
    return {
      locale,
      isAutoDetected: state.source === "auto",
      intlLocale,
      setLocale,
      confirmLocale,
      t: (key, params) => translate(locale, key, params),
      formatDate: (input, options = { day: "numeric", month: "short", year: "numeric" }) =>
        new Date(input).toLocaleDateString(intlLocale, options),
      formatNumber: (input, options) => new Intl.NumberFormat(intlLocale, options).format(input),
      formatMoney: (minorUnits, currency) =>
        new Intl.NumberFormat(intlLocale, { style: "currency", currency }).format(minorUnits / 100)
    };
  }, [locale, state.source, setLocale, confirmLocale]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
};

// Outside the provider (isolated component tests) the interface is English.
const ENGLISH_FALLBACK: I18nValue = {
  locale: "en",
  isAutoDetected: false,
  intlLocale: INTL_LOCALE.en,
  setLocale: () => undefined,
  confirmLocale: () => undefined,
  t: (key, params) => translate("en", key, params),
  formatDate: (input, options = { day: "numeric", month: "short", year: "numeric" }) =>
    new Date(input).toLocaleDateString(INTL_LOCALE.en, options),
  formatNumber: (input, options) => new Intl.NumberFormat(INTL_LOCALE.en, options).format(input),
  formatMoney: (minorUnits, currency) =>
    new Intl.NumberFormat(INTL_LOCALE.en, { style: "currency", currency }).format(minorUnits / 100)
};

export const useI18n = (): I18nValue => useContext(I18nContext) ?? ENGLISH_FALLBACK;

// Copy is page text kept next to the page that shows it: one object per
// language with the same shape (pt must provide every key en has). Values may
// be functions for text with numbers or names.
export interface Copy<T> {
  en: T;
  pt: T;
}

export const useCopy = <T,>(copy: Copy<T>): T => copy[useI18n().locale];

// Catalog is a gettext-style table for pages with a lot of English prose:
// the English text stays in the page as the key and the Portuguese text sits
// in a catalogue next to it. A missing entry falls back to English.
export type Catalog = Readonly<Record<string, string>>;

export type Gettext = (text: string, params?: TranslationParams) => string;

export const gettext = (locale: Locale, catalogs: readonly Catalog[], text: string, params?: TranslationParams): string => {
  let template = text;
  if (locale === "pt") {
    for (const catalog of catalogs) {
      const found = catalog[text];
      if (found !== undefined) {
        template = found;
        break;
      }
    }
  }
  return interpolate(template, params);
};

// useGettext returns tr("English text", {params}) for the current language.
export const useGettext = (...catalogs: Catalog[]): Gettext => {
  const { locale } = useI18n();
  return (text, params) => gettext(locale, catalogs, text, params);
};
