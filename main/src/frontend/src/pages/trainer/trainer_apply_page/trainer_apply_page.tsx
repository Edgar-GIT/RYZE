import { useState } from "react";
import { Redirect } from "react-router-dom";
import { ClipboardCheck, Clock3, Send, ShieldCheck } from "lucide-react";

import { Button } from "@/components/button/button";
import { AppLoading } from "@/components/app_shell/app_loading";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useTrainerSession } from "@/components/trainer/trainer_session_context";
import {
  submitTrainerApplication,
  TrainerApplicationStatus
} from "@/services/trainer_api";
import { ApiError } from "@utils/http_client";

import { TrainerApplicationDocuments } from "./trainer_application_documents";

import styles from "./trainer_apply_page.module.css";
import { useI18n } from "@/i18n/i18n";

// The application form intentionally collects nothing beyond the request
// itself: the backend accepts no applicant-supplied fields, and the platform
// has not defined any screening document workflow, so the page must not invent
// one. The flow is a single explicit request, reviewed by an administrator.
export const TrainerApplyPage = () => {
  const { t } = useI18n();
  const { access, unauthenticated, loaded, refresh } = useTrainerSession();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const submit = async () => {
    setSubmitting(true);
    setError("");
    try {
      await submitTrainerApplication();
      refresh();
    } catch (reason) {
      if (reason instanceof ApiError) {
        // Both codes mean an active application or trainer profile already
        // exists for this user; refreshing reveals the current state.
        if (reason.code === "ALREADY_TRAINER" || reason.code === "APPLICATION_ALREADY_EXISTS") {
          refresh();
        } else if (reason.status === 401) {
          refresh();
        } else {
          setError(t("account.trainerApply.submitError"));
        }
      } else {
        setError(t("account.trainerApply.submitError"));
      }
    } finally {
      setSubmitting(false);
    }
  };

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
            {error ? <p className={styles.error}>{error}</p> : null}
            <div className={styles.actions}>
              <Button
                variant="primary"
                size="small"
                icon={<Send size={15} />}
                disabled={submitting}
                onClick={() => void submit()}
              >
                {submitting ? t("account.trainerApply.submitting") : t("account.trainerApply.requestAgain")}
              </Button>
            </div>
          </section>
        ) : null}

        {!isTrainer && !application ? (
          <section className={styles.card}>
            <span className={styles.icon}>
              <Send size={26} aria-hidden="true" />
            </span>
            <h1 className={styles.title}>{t("account.trainerApply.title")}</h1>
            <p className={styles.text}>{t("account.trainerApply.text")}</p>
            <ul className={styles.points}>
              <li>{t("account.trainerApply.point1")}</li>
              <li>{t("account.trainerApply.point2")}</li>
              <li>{t("account.trainerApply.point3")}</li>
            </ul>
            {error ? <p className={styles.error}>{error}</p> : null}
            <div className={styles.actions}>
              <Button
                variant="primary"
                size="small"
                icon={<Send size={15} />}
                disabled={submitting}
                onClick={() => void submit()}
              >
                {submitting ? t("account.trainerApply.submitting") : t("account.trainerApply.request")}
              </Button>
              <Button to="/" variant="secondary" size="small">
                {t("account.trainerApply.back")}
              </Button>
            </div>
          </section>
        ) : null}
      </div>
    </PageWrapper>
  );
};
