import { AnimatePresence, motion } from "framer-motion";
import { Eye, EyeOff, KeyRound, Lock, ShieldCheck, User } from "lucide-react";
import type { FormEvent } from "react";
import { useState } from "react";
import { Link } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { Button } from "@/components/button/button";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import loginStyles from "@/pages/login_page/login_page.module.css";
import { ApiError, apiPost } from "@utils/http_client";
import loginBackground from "@resources/img/login_create/background.png";

import styles from "./admin_login_page.module.css";

type AdminStep = "credentials" | "access-code" | "authenticated";

const adminErrorFor = (error: unknown, stage: "credentials" | "access-code"): string => {
  if (error instanceof ApiError) {
    if (error.code === "INVALID_CREDENTIALS") {
      return stage === "credentials"
        ? "Invalid administrator credentials."
        : "Authentication failed. Please sign in again.";
    }
    return error.message || "Something went wrong. Please try again.";
  }
  return "Something went wrong. Please try again.";
};

const AdminLoginPage = () => {
  const [step, setStep] = useState<AdminStep>("credentials");
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [showCode, setShowCode] = useState(false);

  const handleCredentials = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }

    const form = new FormData(event.currentTarget);
    const username = String(form.get("admin-username") ?? "");
    const password = String(form.get("admin-password") ?? "");

    if (!username || !password) {
      setErrorMessage("Please fill in all fields.");
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      await apiPost("/admin/auth/login", { username, password });
      setStep("access-code");
    } catch (error) {
      setErrorMessage(adminErrorFor(error, "credentials"));
    } finally {
      setSubmitting(false);
    }
  };

  const handleVerify = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }

    const form = new FormData(event.currentTarget);
    const accessCode = String(form.get("access-code") ?? "");

    if (!accessCode) {
      setErrorMessage("Please enter your access code.");
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      await apiPost("/admin/auth/verify", { access_code: accessCode });
      setStep("authenticated");
    } catch (error) {
      setStep("credentials");
      setErrorMessage(adminErrorFor(error, "access-code"));
    } finally {
      setSubmitting(false);
    }
  };

  const stepLabels: Record<AdminStep, string> = {
    credentials: "Credentials",
    "access-code": "Access code",
    authenticated: "Authenticated",
  };

  const stepLabel = (label: AdminStep) => (
    <span className={step === label ? styles.stepActive : styles.step}>
      {stepLabels[label]}
    </span>
  );

  return (
    <PageWrapper>
      <section
        className={loginStyles.section}
        style={{ backgroundImage: `url(${loginBackground})` }}
      >
        <Link className={loginStyles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <div className={loginStyles.scrim} aria-hidden="true" />

        <div className={loginStyles.card}>
          <AnimatePresence mode="wait">
            {step === "credentials" ? (
              <motion.form
                key="credentials"
                className={loginStyles.form}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0, transition: { duration: 0.3, ease: "easeOut" } }}
                exit={{ opacity: 0, y: -8, transition: { duration: 0.2 } }}
                onSubmit={handleCredentials}
                noValidate
              >
                <div className={loginStyles.header}>
                  <h2>Admin access</h2>
                  <p>Sign in with your administrator credentials.</p>
                </div>

                <div className={styles.steps}>
                  {stepLabel("credentials")} <span className={styles.stepArrow}>→</span>{" "}
                  {stepLabel("access-code")}
                </div>

                <div className={loginStyles.field}>
                  <User className={loginStyles.fieldIcon} aria-hidden="true" />
                  <input
                    type="text"
                    name="admin-username"
                    placeholder="Username"
                    autoComplete="username"
                    aria-label="Username"
                  />
                </div>

                <div className={loginStyles.field}>
                  <Lock className={loginStyles.fieldIcon} aria-hidden="true" />
                  <input
                    className={loginStyles.inputAction}
                    type="password"
                    name="admin-password"
                    placeholder="Password"
                    autoComplete="current-password"
                    aria-label="Password"
                  />
                </div>

                {errorMessage ? (
                  <p className={loginStyles.formError} role="alert">
                    {errorMessage}
                  </p>
                ) : null}

                <Button
                  type="submit"
                  size="large"
                  className={loginStyles.submit}
                  disabled={submitting}
                >
                  Continue
                </Button>
              </motion.form>
            ) : null}

            {step === "access-code" ? (
              <motion.form
                key="access-code"
                className={loginStyles.form}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0, transition: { duration: 0.3, ease: "easeOut" } }}
                exit={{ opacity: 0, y: -8, transition: { duration: 0.2 } }}
                onSubmit={handleVerify}
                noValidate
              >
                <div className={loginStyles.header}>
                  <h2>Access code</h2>
                  <p>Enter your administrator access code to continue.</p>
                </div>

                <div className={styles.steps}>
                  {stepLabel("credentials")} <span className={styles.stepArrow}>→</span>{" "}
                  {stepLabel("access-code")}
                </div>

                <div className={loginStyles.field}>
                  <KeyRound className={loginStyles.fieldIcon} aria-hidden="true" />
                  <input
                    className={loginStyles.inputAction}
                    type={showCode ? "text" : "password"}
                    name="access-code"
                    placeholder="Access code"
                    autoComplete="one-time-code"
                    aria-label="Access code"
                  />
                  <button
                    className={loginStyles.eyeButton}
                    type="button"
                    aria-label={showCode ? "Hide access code" : "Show access code"}
                    onClick={() => setShowCode((currentValue) => !currentValue)}
                  >
                    {showCode ? <EyeOff aria-hidden="true" /> : <Eye aria-hidden="true" />}
                  </button>
                </div>

                {errorMessage ? (
                  <p className={loginStyles.formError} role="alert">
                    {errorMessage}
                  </p>
                ) : null}

                <Button
                  type="submit"
                  size="large"
                  className={loginStyles.submit}
                  disabled={submitting}
                >
                  Verify access code
                </Button>

                <p className={styles.retryHint}>
                  <button type="button" onClick={() => setStep("credentials")}>
                    Go back
                  </button>
                </p>
              </motion.form>
            ) : null}

            {step === "authenticated" ? (
              <motion.div
                key="authenticated"
                className={styles.success}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0, transition: { duration: 0.3, ease: "easeOut" } }}
                exit={{ opacity: 0, y: -8, transition: { duration: 0.2 } }}
              >
                <div className={styles.successIcon} aria-hidden="true">
                  <ShieldCheck />
                </div>
                <div className={loginStyles.header}>
                  <h2>Admin authenticated</h2>
                  <p>
                    Your administrator session is active. The admin console will be added in a
                    future step.
                  </p>
                </div>
                <Button to="/" variant="secondary" size="medium">
                  Back to site
                </Button>
              </motion.div>
            ) : null}
          </AnimatePresence>
        </div>

        <p className={loginStyles.legal}>
          © 2026 RYZE ·{" "}
          <a href="#" onClick={(event) => event.preventDefault()}>
            Privacy Policy
          </a>{" "}
          ·{" "}
          <a href="#" onClick={(event) => event.preventDefault()}>
            Terms of Service
          </a>
        </p>
      </section>
    </PageWrapper>
  );
};

export default AdminLoginPage;
