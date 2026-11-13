import { Languages } from "lucide-react";

import { type Locale, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./language_switcher.module.css";

interface LanguageSwitcherProps {
  className?: string;
  // compact: the PT | EN block of the navigation bars (no icon, smaller).
  variant?: "default" | "compact";
}

const OPTIONS: readonly Locale[] = ["pt", "en"];

// LanguageSwitcher is a two-option segmented control (PT | EN); the current
// language is highlighted. A click changes the whole site at once; the
// choice is stored as manual and saved to the account when signed in.
export const LanguageSwitcher = ({ className, variant = "default" }: LanguageSwitcherProps) => {
  const { locale, setLocale, t } = useI18n();

  return (
    <div
      className={joinClassNames(styles.switcher, variant === "compact" && styles.compact, className)}
      role="group"
      aria-label={t("language.switchTo")}
    >
      {variant === "default" ? <Languages className={styles.icon} aria-hidden="true" /> : null}
      {OPTIONS.map((option) => (
        <button
          key={option}
          type="button"
          className={joinClassNames(styles.option, option === locale && styles.active)}
          aria-pressed={option === locale}
          lang={option === "pt" ? "pt-PT" : "en"}
          title={t(option === "pt" ? "language.pt" : "language.en")}
          onClick={() => {
            if (option !== locale) setLocale(option);
          }}
        >
          {option.toUpperCase()}
        </button>
      ))}
    </div>
  );
};
