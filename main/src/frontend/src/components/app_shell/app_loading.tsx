import { useI18n } from "@/i18n/i18n";

import styles from "./app_loading.module.css";

// AppLoading is the in-app loading state: placeholder blocks where the
// content will appear (the full-screen splash is only for the site's start).
export const AppLoading = () => {
  const { locale } = useI18n();
  return (
    <div className={styles.loading} role="status" aria-live="polite">
      <span className={styles.srOnly}>{locale === "pt" ? "A carregar…" : "Loading…"}</span>
      <span className={styles.title} aria-hidden="true" />
      <span className={styles.line} aria-hidden="true" />
      <span className={styles.grid} aria-hidden="true">
        <span />
        <span />
        <span />
      </span>
    </div>
  );
};
