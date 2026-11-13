import { Globe2, X } from "lucide-react";
import { useEffect, useId } from "react";

import { useI18n } from "@/i18n/i18n";

import styles from "./language_prompt.module.css";

// LanguagePrompt appears once, on the first visit, in the language detected
// from the device (Portugal: Portuguese; elsewhere: English). The visitor
// keeps it or switches with one tap; either answer is remembered and the
// prompt never shows again. It does not block the page.
export const LanguagePrompt = () => {
  const { locale, isAutoDetected, setLocale, confirmLocale, t } = useI18n();
  const titleId = useId();
  const textId = useId();

  useEffect(() => {
    if (!isAutoDetected) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") confirmLocale();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [isAutoDetected, confirmLocale]);

  if (!isAutoDetected) {
    return null;
  }

  const other = locale === "pt" ? "en" : "pt";

  return (
    <section className={styles.prompt} role="dialog" aria-modal="false" aria-labelledby={titleId} aria-describedby={textId}>
      <Globe2 className={styles.icon} aria-hidden="true" />
      <div className={styles.body}>
        <h2 id={titleId} className={styles.title}>
          {t("language.prompt.title")}
        </h2>
        <p id={textId} className={styles.text}>
          {t("language.prompt.text")}
        </p>
        <div className={styles.actions}>
          <button type="button" className={styles.keep} onClick={confirmLocale}>
            {t("language.prompt.keep")}
          </button>
          <button type="button" className={styles.switch} lang={other === "pt" ? "pt-PT" : "en"} onClick={() => setLocale(other)}>
            {t("language.prompt.switch")}
          </button>
        </div>
      </div>
      <button type="button" className={styles.close} aria-label={t("language.prompt.close")} onClick={confirmLocale}>
        <X aria-hidden="true" />
      </button>
    </section>
  );
};
