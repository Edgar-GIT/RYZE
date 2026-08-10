import { AnimatePresence, motion, useReducedMotion, type Variants } from "framer-motion";
import { Eye, EyeOff, Lock, Mail, User } from "lucide-react";
import type { FormEvent, ReactNode } from "react";
import { useEffect, useRef, useState } from "react";
import { Link, useHistory } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ApiError, apiPost } from "@utils/http_client";
import loginBackground from "@resources/img/login_create/background.png";

import styles from "./login_page.module.css";

type AuthMode = "login" | "register";

interface LoginPageProps {
  initialMode?: AuthMode;
}

const LOADING_DURATION_MS = 1000;

const containerVariants: Variants = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.05 } }
};

const itemVariants: Variants = {
  hidden: { opacity: 0, y: 8 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.3, ease: "easeOut" } }
};

const FadeItem = ({ className, children }: { className?: string; children: ReactNode }) => (
  <motion.div className={className} variants={itemVariants}>
    {children}
  </motion.div>
);

const GoogleButton = () => (
  <button className={styles.googleButton} type="button" aria-label="Continue with Google">
    <svg viewBox="0 0 48 48" aria-hidden="true">
      <path
        fill="#EA4335"
        d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"
      />
      <path
        fill="#4285F4"
        d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"
      />
      <path
        fill="#FBBC05"
        d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"
      />
      <path
        fill="#34A853"
        d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"
      />
    </svg>
  </button>
);

const PasswordField = ({
  id,
  placeholder,
  autoComplete
}: {
  id: string;
  placeholder: string;
  autoComplete: string;
}) => {
  const [isVisible, setIsVisible] = useState(false);

  return (
    <div className={styles.field}>
      <Lock className={styles.fieldIcon} aria-hidden="true" />
      <input
        id={id}
        className={styles.inputAction}
        type={isVisible ? "text" : "password"}
        name={id}
        placeholder={placeholder}
        autoComplete={autoComplete}
        aria-label={placeholder}
      />
      <button
        className={styles.eyeButton}
        type="button"
        aria-label={isVisible ? "Hide password" : "Show password"}
        onClick={() => setIsVisible((currentValue) => !currentValue)}
      >
        {isVisible ? <EyeOff aria-hidden="true" /> : <Eye aria-hidden="true" />}
      </button>
    </div>
  );
};

interface AuthFormProps {
  onSwitch: () => void;
}

const errorMessageFor = (error: unknown): string => {
  if (error instanceof ApiError) {
    if (error.code === "INVALID_CREDENTIALS") {
      return "Invalid email or password.";
    }
    if (error.code === "EMAIL_ALREADY_REGISTERED") {
      return "This email is already registered.";
    }
    return error.message || "Something went wrong. Please try again.";
  }
  return "Something went wrong. Please try again.";
};

const LoginForm = ({ onSwitch }: AuthFormProps) => {
  const history = useHistory();
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }

    const form = new FormData(event.currentTarget);
    const email = String(form.get("email") ?? "");
    const password = String(form.get("login-password") ?? "");

    if (!email || !password) {
      setErrorMessage("Please fill in all fields.");
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      await apiPost("/auth/login", { email, password });
      history.push("/profile");
    } catch (error) {
      setErrorMessage(errorMessageFor(error));
      setSubmitting(false);
    }
  };

  return (
    <motion.form
      className={styles.form}
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      onSubmit={handleSubmit}
      noValidate
    >
      <FadeItem className={styles.header}>
        <h2>Welcome back</h2>
        <p>Log in to continue your journey.</p>
      </FadeItem>

      <FadeItem className={styles.fields}>
        <div className={styles.field}>
          <Mail className={styles.fieldIcon} aria-hidden="true" />
          <input
            type="email"
            name="email"
            placeholder="Email or username"
            autoComplete="email"
            aria-label="Email or username"
          />
        </div>
      </FadeItem>
      <FadeItem>
        <PasswordField id="login-password" placeholder="Password" autoComplete="current-password" />
      </FadeItem>

      <FadeItem className={styles.options}>
        <label className={styles.check}>
          <input type="checkbox" name="remember" />
          <span>Remember me</span>
        </label>
        <a className={styles.link} href="#" onClick={(event) => event.preventDefault()}>
          Forgot your password?
        </a>
      </FadeItem>

      {errorMessage ? (
        <FadeItem>
          <p className={styles.formError} role="alert">
            {errorMessage}
          </p>
        </FadeItem>
      ) : null}

      <FadeItem>
        <Button type="submit" size="large" className={styles.submit} disabled={submitting}>
          Log in
        </Button>
      </FadeItem>

      <FadeItem className={styles.divider}>
        <span>or continue with</span>
      </FadeItem>

      <FadeItem className={styles.googleRow}>
        <GoogleButton />
      </FadeItem>

      <FadeItem className={styles.formFooter}>
        <p className={styles.switch}>
          Don&apos;t have an account?{" "}
          <button type="button" onClick={onSwitch}>
            Create one here
          </button>
        </p>
      </FadeItem>
    </motion.form>
  );
};

const RegisterForm = ({ onSwitch }: AuthFormProps) => {
  const history = useHistory();
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }

    const form = new FormData(event.currentTarget);
    const firstName = String(form.get("first_name") ?? "");
    const lastName = String(form.get("last_name") ?? "");
    const email = String(form.get("email") ?? "");
    const password = String(form.get("register-password") ?? "");
    const confirmPassword = String(form.get("register-confirm") ?? "");

    if (!firstName || !lastName || !email || !password || !confirmPassword) {
      setErrorMessage("Please fill in all fields.");
      return;
    }

    if (password !== confirmPassword) {
      setErrorMessage("Passwords do not match.");
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      await apiPost("/auth/register", {
        first_name: firstName,
        last_name: lastName,
        email,
        password
      });
      history.push("/profile");
    } catch (error) {
      setErrorMessage(errorMessageFor(error));
      setSubmitting(false);
    }
  };

  return (
    <motion.form
      className={styles.form}
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      onSubmit={handleSubmit}
      noValidate
    >
      <FadeItem className={styles.header}>
        <h2>Create account</h2>
        <p>Join RYZE and start your evolution.</p>
      </FadeItem>

      <FadeItem className={styles.fields}>
        <div className={styles.field}>
          <User className={styles.fieldIcon} aria-hidden="true" />
          <input
            type="text"
            name="first_name"
            placeholder="First name"
            autoComplete="given-name"
            aria-label="First name"
          />
        </div>
        <div className={styles.field}>
          <User className={styles.fieldIcon} aria-hidden="true" />
          <input
            type="text"
            name="last_name"
            placeholder="Last name"
            autoComplete="family-name"
            aria-label="Last name"
          />
        </div>
        <div className={styles.field}>
          <Mail className={styles.fieldIcon} aria-hidden="true" />
          <input type="email" name="email" placeholder="Email" autoComplete="email" aria-label="Email" />
        </div>
      </FadeItem>
      <FadeItem className={styles.fields}>
        <PasswordField id="register-password" placeholder="Password" autoComplete="new-password" />
        <PasswordField
          id="register-confirm"
          placeholder="Confirm password"
          autoComplete="new-password"
        />
      </FadeItem>

      <FadeItem>
        <span className={styles.checkTerms}>
          <input
            id="register-terms"
            type="checkbox"
            name="terms"
            aria-label="I agree to the Terms of Service and the Privacy Policy"
          />
          <span>
            I agree to the{" "}
            <a href="#" onClick={(event) => event.preventDefault()}>
              Terms of Service
            </a>{" "}
            and the{" "}
            <a href="#" onClick={(event) => event.preventDefault()}>
              Privacy Policy
            </a>
          </span>
        </span>
      </FadeItem>

      {errorMessage ? (
        <FadeItem>
          <p className={styles.formError} role="alert">
            {errorMessage}
          </p>
        </FadeItem>
      ) : null}

      <FadeItem>
        <Button type="submit" size="large" className={styles.submit} disabled={submitting}>
          Create account
        </Button>
      </FadeItem>

      <FadeItem className={styles.divider}>
        <span>or continue with</span>
      </FadeItem>

      <FadeItem className={styles.googleRow}>
        <GoogleButton />
      </FadeItem>

      <FadeItem className={styles.formFooter}>
        <p className={styles.switch}>
          Already have an account?{" "}
          <button type="button" onClick={onSwitch}>
            Log in
          </button>
        </p>
      </FadeItem>
    </motion.form>
  );
};

export const LoginPage = ({ initialMode = "login" }: LoginPageProps) => {
  const [mode, setMode] = useState<AuthMode>(initialMode);
  const [loading, setLoading] = useState(false);
  const timerRef = useRef<number | null>(null);
  const reduceMotion = useReducedMotion();

  useEffect(
    () => () => {
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current);
      }
    },
    []
  );

  const switchMode = (nextMode: AuthMode) => {
    if (loading) {
      return;
    }

    if (reduceMotion) {
      setMode(nextMode);
      return;
    }

    setLoading(true);
    timerRef.current = window.setTimeout(() => {
      setMode(nextMode);
      setLoading(false);
      timerRef.current = null;
    }, LOADING_DURATION_MS);
  };

  return (
    <PageWrapper>
      <section
        className={styles.section}
        style={{ backgroundImage: `url(${loginBackground})` }}
      >
        <Link className={styles.brand} to="/" aria-label="RYZE home">
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <div className={styles.scrim} aria-hidden="true" />

        <div className={styles.card}>
          {mode === "login" ? (
            <LoginForm onSwitch={() => switchMode("register")} />
          ) : (
            <RegisterForm onSwitch={() => switchMode("login")} />
          )}
        </div>

        <p className={styles.legal}>
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

      <AnimatePresence>{loading ? <LoadingScreen /> : null}</AnimatePresence>
    </PageWrapper>
  );
};
