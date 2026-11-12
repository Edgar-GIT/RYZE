import { Languages } from "lucide-react";

import { LOCALES, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./language_switcher.module.css";

interface LanguageSwitcherProps {
  className?: string;
}

// LanguageSwitcher is a two-option segmented control (EN / PT). The choice is
// stored as manual and saved to the account when signed in.
export const LanguageSwitcher = ({ className }: LanguageSwitcherProps) => {
  const { locale, setLocale, t } = useI18n();

  return (
    <div className={joinClassNames(styles.switcher, className)} role="group" aria-label={t("language.switchTo")}>
      <Languages className={styles.icon} aria-hidden="true" />
      {LOCALES.map((option) => (
        <button
          key={option}
          type="button"
          className={joinClassNames(styles.option, option === locale && styles.active)}
          aria-pressed={option === locale}
          lang={option === "pt" ? "pt-PT" : "en"}
          title={t(option === "pt" ? "language.pt" : "language.en")}
          onClick={() => setLocale(option)}
        >
          {option.toUpperCase()}
        </button>
      ))}
    </div>
  );
};
