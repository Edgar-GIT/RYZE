import { Languages } from "lucide-react";
import type { KeyboardEvent } from "react";

import { type Locale, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./language_switcher.module.css";

interface LanguageSwitcherProps {
  className?: string;
  // compact: the navigation bars' version (no icon).
  variant?: "default" | "compact";
}

const OPTIONS: readonly Locale[] = ["pt", "en"];

// LanguageSwitcher reads "PT / EN" like the other navigation labels (owner
// request, 2026-11-15): the language in use is white with a small light under
// it, the other is dimmed, and the light slides across when it changes. A
// click anywhere switches; arrow keys pick a side. The choice is stored as
// manual and saved to the account when signed in.
export const LanguageSwitcher = ({ className, variant = "default" }: LanguageSwitcherProps) => {
  const { locale, setLocale, t } = useI18n();
  const other: Locale = locale === "pt" ? "en" : "pt";
  const name = (value: Locale) => t(value === "pt" ? "language.pt" : "language.en");

  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    const target = event.key === "ArrowLeft" ? "pt" : event.key === "ArrowRight" ? "en" : null;
    if (target) {
      event.preventDefault();
      if (target !== locale) setLocale(target);
    }
  };

  return (
    <button
      type="button"
      className={joinClassNames(styles.toggle, variant === "compact" && styles.compact, className)}
      data-locale={locale}
      aria-label={t("language.toggle", { current: name(locale), other: name(other) })}
      title={t("language.toggle", { current: name(locale), other: name(other) })}
      onClick={() => setLocale(other)}
      onKeyDown={handleKeyDown}
    >
      {variant === "default" ? <Languages className={styles.icon} aria-hidden="true" /> : null}
      <span className={styles.track} aria-hidden="true">
        {OPTIONS.map((option, index) => (
          <span key={option} className={styles.slot}>
            {index > 0 ? <span className={styles.separator}>/</span> : null}
            <span
              className={joinClassNames(styles.label, option === locale && styles.labelActive)}
              lang={option === "pt" ? "pt-PT" : "en"}
            >
              {option.toUpperCase()}
            </span>
          </span>
        ))}
        <span className={styles.light} />
      </span>
    </button>
  );
};
