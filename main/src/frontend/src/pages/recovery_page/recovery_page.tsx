import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import styles from "@/components/training/training_page.module.css";
import { useI18n } from "@/i18n/i18n";
import { MuscleRecoveryPanel } from "@/pages/progress_page/muscle_recovery_panel";

const COPY = {
  en: { title: "Recovery", lead: "Which muscles are ready to train today, from what you trained this week." },
  pt: { title: "Recuperação", lead: "Que músculos estão prontos a treinar hoje, a partir do que treinaste esta semana." }
} as const;

// RecoveryPage gives muscle recovery its own place in the member menu (owner
// request, 2026-11-21: "Recovery" in the left bar, out of Progress).
export const RecoveryPage = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  return (
    <PageWrapper className={styles.page}>
      <div className={styles.main}>
        <header className={styles.header}>
          <h1 className={styles.title}>{text.title}</h1>
          <p className={styles.lead}>{text.lead}</p>
        </header>
        <MuscleRecoveryPanel />
      </div>
    </PageWrapper>
  );
};
