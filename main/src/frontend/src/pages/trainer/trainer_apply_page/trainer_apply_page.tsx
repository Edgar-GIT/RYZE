import { Redirect } from "react-router-dom";
import { ClipboardCheck, Clock3, ShieldCheck } from "lucide-react";

import { Button } from "@/components/button/button";
import { AppLoading } from "@/components/app_shell/app_loading";
import { useMember } from "@/components/app_shell/member_shell";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useTrainerSession } from "@/components/trainer/trainer_session_context";
import { TrainerApplicationStatus } from "@/services/trainer_api";

import { TrainerApplicationDocuments } from "./trainer_application_documents";
import { TrainerApplicationForm } from "./trainer_application_form";

import styles from "./trainer_apply_page.module.css";
import { useI18n } from "@/i18n/i18n";

// The trainer application (owner request, 2026-11-20): a complete form —
// the applicant's details, background, a free text and the evidence files —
// sent together and screened by the server, then reviewed by a person.
export const TrainerApplyPage = () => {
  const { t } = useI18n();
  const { access, unauthenticated, loaded, refresh } = useTrainerSession();
  const { user } = useMember();
  const defaultName = user ? `${user.first_name} ${user.last_name}`.trim() : "";

  if (!loaded) {
    return (
      <PageWrapper>
        <AppLoading />
      </PageWrapper>
    );
  }

  if (unauthenticated) {
    return <Redirect to="/login" />;
  }

  const isTrainer = access?.is_trainer ?? false;
  const application = access?.application ?? null;
  const isPending = application?.status === TrainerApplicationStatus.PENDING;
  const isRejected = application?.status === TrainerApplicationStatus.REJECTED;

  return (
    <PageWrapper className={styles.page}>

      <div className={styles.main}>
        {isTrainer ? (
          <section className={styles.card}>
            <span className={styles.icon}>
              <ShieldCheck size={26} aria-hidden="true" />
            </span>
            <h1 className={styles.title}>{t("account.trainerApply.trainerTitle")}</h1>
            <p className={styles.text}>{t("account.trainerApply.trainerText")}</p>
            <div className={styles.actions}>
              <Button to="/trainer/dashboard" variant="primary" size="small">
                {t("account.trainerApply.openDashboard")}
              </Button>
            </div>
          </section>
        ) : null}

        {!isTrainer && isPending ? (
          <section className={styles.card}>
            <span className={styles.icon}>
              <Clock3 size={26} aria-hidden="true" />
            </span>
            <h1 className={styles.title}>{t("account.trainerApply.pendingTitle")}</h1>
            <p className={styles.text}>{t("account.trainerApply.pendingText")}</p>
            <TrainerApplicationDocuments />
          </section>
        ) : null}

        {!isTrainer && isRejected ? (
          <section className={styles.card}>
            <span className={styles.icon}>
              <ClipboardCheck size={26} aria-hidden="true" />
            </span>
            <h1 className={styles.title}>{t("account.trainerApply.rejectedTitle")}</h1>
            <p className={styles.text}>{t("account.trainerApply.rejectedText")}</p>
          </section>
        ) : null}

        {!isTrainer && isRejected ? <TrainerApplicationForm defaultName={defaultName} onSubmitted={refresh} /> : null}

        {!isTrainer && !application ? <TrainerApplicationForm defaultName={defaultName} onSubmitted={refresh} /> : null}
      </div>
    </PageWrapper>
  );
};
