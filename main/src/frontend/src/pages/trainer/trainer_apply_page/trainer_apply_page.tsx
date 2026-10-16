import { useState } from "react";
import { Redirect } from "react-router-dom";
import { ClipboardCheck, Clock3, Send, ShieldCheck } from "lucide-react";

import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useTrainerSession } from "@/components/trainer/trainer_session_context";
import {
  submitTrainerApplication,
  TrainerApplicationStatus
} from "@/services/trainer_api";
import { ApiError } from "@utils/http_client";

import styles from "./trainer_apply_page.module.css";

// The application form intentionally collects nothing beyond the request
// itself: the backend accepts no applicant-supplied fields, and the platform
// has not defined any screening document workflow, so the page must not invent
// one. The flow is a single explicit request, reviewed by an administrator.
export const TrainerApplyPage = () => {
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
          setError(reason.message || "Unable to submit the application.");
        }
      } else {
        setError("Unable to submit the application.");
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (!loaded) {
    return (
      <PageWrapper>
        <AnimatedBackground />
        <LoadingScreen />
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
      <AnimatedBackground />
      <AccountNav />

      <main className={styles.main}>
        {isTrainer ? (
          <section className={styles.card}>
            <span className={styles.icon}>
              <ShieldCheck size={26} aria-hidden="true" />
            </span>
            <h1 className={styles.title}>You are a trainer</h1>
            <p className={styles.text}>
              Your trainer access is active. Open your dashboard to manage clients and your training
              programs.
            </p>
            <div className={styles.actions}>
              <Button to="/trainer/dashboard" variant="primary" size="small">
                Open trainer dashboard
              </Button>
            </div>
          </section>
        ) : null}

        {!isTrainer && isPending ? (
          <section className={styles.card}>
            <span className={styles.icon}>
              <Clock3 size={26} aria-hidden="true" />
            </span>
            <h1 className={styles.title}>Application under review</h1>
            <p className={styles.text}>
              Your request to become a trainer has been submitted and is waiting for an
              administrator to review it. You will be able to open the trainer dashboard once it is
              approved.
            </p>
          </section>
        ) : null}

        {!isTrainer && isRejected ? (
          <section className={styles.card}>
            <span className={styles.icon}>
              <ClipboardCheck size={26} aria-hidden="true" />
            </span>
            <h1 className={styles.title}>Application not approved</h1>
            <p className={styles.text}>
              Your previous request was not approved. You can submit a new request at any time.
            </p>
            {error ? <p className={styles.error}>{error}</p> : null}
            <div className={styles.actions}>
              <Button
                variant="primary"
                size="small"
                icon={<Send size={15} />}
                disabled={submitting}
                onClick={() => void submit()}
              >
                {submitting ? "Submitting…" : "Request trainer access again"}
              </Button>
            </div>
          </section>
        ) : null}

        {!isTrainer && !application ? (
          <section className={styles.card}>
            <span className={styles.icon}>
              <Send size={26} aria-hidden="true" />
            </span>
            <h1 className={styles.title}>Become a RYZE trainer</h1>
            <p className={styles.text}>
              Trainers create their own training programs and manage their clients directly in
              RYZE. If you want to publish programs under your name, request trainer access and an
              administrator will review it.
            </p>
            <ul className={styles.points}>
              <li>Create and publish your own training programs.</li>
              <li>Manage the clients you train in one place.</li>
              <li>Your personal control over your content and client list.</li>
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
                {submitting ? "Submitting…" : "Request trainer access"}
              </Button>
              <Button to="/" variant="secondary" size="small">
                Back to RYZE
              </Button>
            </div>
          </section>
        ) : null}
      </main>
    </PageWrapper>
  );
};
