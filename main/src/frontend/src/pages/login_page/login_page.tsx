import { motion, type Variants } from "framer-motion";
import { ArrowRight, Eye, EyeOff, Lock, Mail, User } from "lucide-react";
import type { FormEvent, ReactNode, RefObject } from "react";
import { useEffect, useRef, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";

import { BrandMark } from "@/components/brand_mark/brand_mark";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PublicAction, Waves } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { BRAND_ASSETS } from "@/constants/brand_assets";
import { translate, useGettext, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";
import { ApiError, apiGet, apiPost, apiUrl } from "@utils/http_client";

import { LOGIN_PT } from "./login_page.pt";
import styles from "./login_page.module.css";

type AuthMode = "login" | "register";

interface LoginPageProps {
  initialMode?: AuthMode;
}

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

// Results Google sign-in can report back on this page (?oauth=...).
const GOOGLE_RESULTS = ["cancelled", "failed", "expired", "unverified", "link_required", "unavailable"] as const;
type GoogleResult = (typeof GOOGLE_RESULTS)[number];

const isGoogleResult = (value: string | null): value is GoogleResult => GOOGLE_RESULTS.includes(value as GoogleResult);

// The login page is a protected part: its English copy is unchanged and the
// Portuguese comes from login_page.pt.ts when the visitor chooses PT. The
// chosen locale also travels to the API so a new account gets emails in the
// reader's language.
const loginText = translate;

// GoogleButton starts Google sign-in: the API redirects to Google and back.
// When the provider is not configured it reports that instead of navigating.
const GoogleButton = ({ onMessage }: { onMessage: (message: string) => void }) => {
  const { locale } = useI18n();
  const t = (key: Parameters<typeof translate>[1]) => loginText(locale, key);
  const tr = useGettext(LOGIN_PT);
  const [pending, setPending] = useState(false);

  const handleClick = async () => {
    if (pending) {
      return;
    }
    setPending(true);
    onMessage("");
    try {
      const providers = await apiGet<{ google: boolean }>("/auth/providers");
      if (!providers.google) {
        onMessage(t("auth.google.unavailable"));
        setPending(false);
        return;
      }
      window.location.assign(apiUrl(`/auth/google/start?mode=login&lang=${locale}`));
    } catch {
      onMessage(t("common.genericError"));
      setPending(false);
    }
  };

  return (
    <button
      className={styles.googleButton}
      type="button"
      aria-label={pending ? t("auth.google.redirecting") : tr("Continue with Google")}
      aria-busy={pending}
      disabled={pending}
      onClick={() => void handleClick()}
    >
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
      <span>Google</span>
    </button>
  );
};

const PasswordField = ({
  id,
  placeholder,
  autoComplete
}: {
  id: string;
  placeholder: string;
  autoComplete: string;
}) => {
  const tr = useGettext(LOGIN_PT);
  const [isVisible, setIsVisible] = useState(false);

  return (
    <div className={styles.field}>
      <Lock className={styles.fieldIcon} aria-hidden="true" />
      <input
        id={id}
        className={styles.inputAction}
        type={isVisible ? "text" : "password"}
        name={id}
        placeholder={tr(placeholder)}
        autoComplete={autoComplete}
        aria-label={tr(placeholder)}
      />
      <button
        className={styles.eyeButton}
        type="button"
        aria-label={isVisible ? tr("Hide password") : tr("Show password")}
        onClick={() => setIsVisible((currentValue) => !currentValue)}
      >
        {isVisible ? <EyeOff aria-hidden="true" /> : <Eye aria-hidden="true" />}
      </button>
    </div>
  );
};

interface AuthFormProps {
  onSwitch: () => void;
  headingRef: RefObject<HTMLHeadingElement | null>;
  // A notice to show on arrival (for example a Google sign-in result).
  initialMessage?: string;
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

const LoginForm = ({ onSwitch, headingRef, initialMessage = "" }: AuthFormProps) => {
  const tr = useGettext(LOGIN_PT);
  const history = useHistory();
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState(initialMessage);

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

    // Member accounts sign in with their email; only the configured
    // administrators have usernames, and they sign in at /admin/login.
    if (!email.includes("@")) {
      setErrorMessage("Use the email address of your account. Administrators sign in at /admin/login.");
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
        <h1 ref={headingRef} tabIndex={-1}>
          {tr("Welcome")} <span className={ui.hl}>{tr("back.")}</span>
        </h1>
        <p>{tr("Log in to continue your journey.")}</p>
      </FadeItem>

      <FadeItem className={styles.fields}>
        <div className={styles.field}>
          <Mail className={styles.fieldIcon} aria-hidden="true" />
          <input
            type="email"
            name="email"
            placeholder={tr("Email or username")}
            autoComplete="email"
            aria-label={tr("Email or username")}
          />
        </div>
        <PasswordField id="login-password" placeholder="Password" autoComplete="current-password" />
      </FadeItem>

      <FadeItem className={styles.options}>
        <label className={styles.check}>
          <input type="checkbox" name="remember" />
          <span>{tr("Remember me")}</span>
        </label>
        <Link className={styles.link} to="/forgot-password">
          {tr("Forgot your password?")}
        </Link>
      </FadeItem>

      {errorMessage ? (
        <FadeItem>
          <p className={styles.formError} role="alert">
            {tr(errorMessage)}
          </p>
        </FadeItem>
      ) : null}

      <FadeItem>
        <PublicAction type="submit" className={styles.submit} disabled={submitting} icon={<ArrowRight />}>
          {tr("Log in")}
        </PublicAction>
      </FadeItem>

      <FadeItem className={styles.divider}>
        <span>{tr("or continue with")}</span>
      </FadeItem>

      <FadeItem className={styles.googleRow}>
        <GoogleButton onMessage={setErrorMessage} />
      </FadeItem>

      <FadeItem className={styles.formFooter}>
        <p className={styles.switch}>
          {tr("Don't have an account?")}{" "}
          <button type="button" onClick={onSwitch}>
            {tr("Create one here")}
          </button>
        </p>
      </FadeItem>
    </motion.form>
  );
};

const RegisterForm = ({ onSwitch, headingRef }: AuthFormProps) => {
  // The interface language at sign-up becomes the language of RYZE emails.
  const { locale } = useI18n();
  const tr = useGettext(LOGIN_PT);
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
        password,
        language: locale
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
        <h1 ref={headingRef} tabIndex={-1}>
          {tr("Create")} <span className={ui.hl}>{tr("account.")}</span>
        </h1>
        <p>{tr("Join RYZE and start your evolution.")}</p>
      </FadeItem>

      <FadeItem className={styles.fields}>
        <div className={styles.nameRow}>
          <div className={styles.field}>
            <User className={styles.fieldIcon} aria-hidden="true" />
            <input
              type="text"
              name="first_name"
              placeholder={tr("First name")}
              autoComplete="given-name"
              aria-label={tr("First name")}
            />
          </div>
          <div className={styles.field}>
            <User className={styles.fieldIcon} aria-hidden="true" />
            <input
              type="text"
              name="last_name"
              placeholder={tr("Last name")}
              autoComplete="family-name"
              aria-label={tr("Last name")}
            />
          </div>
        </div>
        <div className={styles.field}>
          <Mail className={styles.fieldIcon} aria-hidden="true" />
          <input type="email" name="email" placeholder={tr("Email")} autoComplete="email" aria-label={tr("Email")} />
        </div>
        <PasswordField id="register-password" placeholder="Password" autoComplete="new-password" />
        <PasswordField id="register-confirm" placeholder="Confirm password" autoComplete="new-password" />
      </FadeItem>

      <FadeItem>
        <span className={styles.checkTerms}>
          <input
            id="register-terms"
            type="checkbox"
            name="terms"
            aria-label={tr("I agree to the Terms of Service and the Privacy Policy")}
          />
          <span>
            {tr("I agree to the")}{" "}
            <a href="#" onClick={(event) => event.preventDefault()}>
              {tr("Terms of Service")}
            </a>{" "}
            {tr("and the")}{" "}
            <a href="#" onClick={(event) => event.preventDefault()}>
              {tr("Privacy Policy")}
            </a>
          </span>
        </span>
      </FadeItem>

      {errorMessage ? (
        <FadeItem>
          <p className={styles.formError} role="alert">
            {tr(errorMessage)}
          </p>
        </FadeItem>
      ) : null}

      <FadeItem>
        <PublicAction type="submit" className={styles.submit} disabled={submitting} icon={<ArrowRight />}>
          {tr("Create account")}
        </PublicAction>
      </FadeItem>

      <FadeItem className={styles.divider}>
        <span>{tr("or continue with")}</span>
      </FadeItem>

      <FadeItem className={styles.googleRow}>
        <GoogleButton onMessage={setErrorMessage} />
      </FadeItem>

      <FadeItem className={styles.formFooter}>
        <p className={styles.switch}>
          {tr("Already have an account?")}{" "}
          <button type="button" onClick={onSwitch}>
            {tr("Log in")}
          </button>
        </p>
      </FadeItem>
    </motion.form>
  );
};

// Copy of the light-blue panel for each mode: it invites to the *other* mode.
const PANEL_COPY: Record<AuthMode, { kicker: string; title: string; text: string; action: string; target: AuthMode }> = {
  login: {
    kicker: "New to RYZE?",
    title: "Start your evolution.",
    text: "Create a free account to follow your programmes, log every session and see your progress.",
    action: "Create account",
    target: "register"
  },
  register: {
    kicker: "Already a member?",
    title: "Welcome back.",
    text: "Log in to continue with your programmes, nutrition plans and workout history.",
    action: "Log in",
    target: "login"
  }
};

// Login and registration share one card: the form takes one half and the
// light-blue wave panel the other. Switching mode slides the panel across and
// the other form takes its place. The inactive form is inert and hidden from
// assistive technology; focus moves to the new form's heading.
export const LoginPage = ({ initialMode = "login" }: LoginPageProps) => {
  const { locale } = useI18n();
  const tr = useGettext(LOGIN_PT);
  const history = useHistory();
  const location = useLocation();
  // A Google sign-in result arrives as ?oauth=...; it is read once and removed
  // from the address bar.
  const [googleResult] = useState(() => {
    const value = new URLSearchParams(location.search).get("oauth");
    return isGoogleResult(value) ? value : null;
  });
  const [mode, setMode] = useState<AuthMode>(googleResult ? "login" : initialMode);

  useEffect(() => {
    if (new URLSearchParams(location.search).has("oauth")) {
      history.replace(location.pathname);
    }
  }, [history, location.pathname, location.search]);

  const switched = useRef(false);
  const loginHeading = useRef<HTMLHeadingElement>(null);
  const registerHeading = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    if (!switched.current) {
      return;
    }
    (mode === "login" ? loginHeading : registerHeading).current?.focus({ preventScroll: true });
  }, [mode]);

  const switchMode = (nextMode: AuthMode) => {
    switched.current = true;
    setMode(nextMode);
  };

  return (
    <PageWrapper className={joinClassNames(ui.page, styles.page)}>
      <section className={joinClassNames(ui.toneNavy, styles.section)}>
        <Link className={styles.brand} to="/" aria-label={tr("RYZE home")}>
          <BrandMark size="navigation" />
          <span>RYZE</span>
        </Link>

        <div className={styles.shell} data-mode={mode}>
          <div
            className={joinClassNames(styles.pane, styles.paneLogin)}
            data-active={mode === "login"}
            inert={mode !== "login"}
            aria-hidden={mode !== "login"}
          >
            <LoginForm
              onSwitch={() => switchMode("register")}
              headingRef={loginHeading}
              initialMessage={googleResult ? loginText(locale, `auth.google.result.${googleResult}`) : ""}
            />
          </div>
          <div
            className={joinClassNames(styles.pane, styles.paneRegister)}
            data-active={mode === "register"}
            inert={mode !== "register"}
            aria-hidden={mode !== "register"}
          >
            <RegisterForm onSwitch={() => switchMode("login")} headingRef={registerHeading} />
          </div>

          <aside className={joinClassNames(ui.toneLight, styles.panel)} aria-label={tr("RYZE account")}>
            <Waves />
            <img className={styles.panelMark} src={BRAND_ASSETS.icon} alt="" width="600" height="658" />
            {(["login", "register"] as const).map((panelMode) => {
              const copy = PANEL_COPY[panelMode];
              const active = panelMode === mode;

              return (
                <div
                  key={panelMode}
                  className={styles.panelCopy}
                  data-panel={panelMode}
                  data-active={active}
                  inert={!active}
                  aria-hidden={!active}
                >
                  <p className={styles.panelKicker}>{tr(copy.kicker)}</p>
                  <p className={styles.panelTitle}>{tr(copy.title)}</p>
                  <p className={styles.panelText}>{tr(copy.text)}</p>
                  <PublicAction variant="ink" icon={<ArrowRight />} onClick={() => switchMode(copy.target)}>
                    {tr(copy.action)}
                  </PublicAction>
                </div>
              );
            })}
          </aside>
        </div>

        <p className={styles.legal}>
          © 2026 RYZE ·{" "}
          <a href="#" onClick={(event) => event.preventDefault()}>
            {tr("Privacy Policy")}
          </a>{" "}
          ·{" "}
          <a href="#" onClick={(event) => event.preventDefault()}>
            {tr("Terms of Service")}
          </a>
        </p>
      </section>
    </PageWrapper>
  );
};
