import { FlaskConical, LogOut, ShieldAlert, UserRound, Dumbbell } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/button/button";
import { useAdminSession } from "@/components/admin/admin_session_context";
import { TECHNICAL_ADMINISTRATOR } from "@/services/admin_api";
import {
  enterTestMode,
  exitTestMode,
  fetchTestModeStatus,
  TestModePersonas,
  type TestModePersona
} from "@/services/test_mode_api";
import { ApiError } from "@utils/http_client";
import { joinClassNames } from "@utils/class_names";

import styles from "./admin_test_mode_page.module.css";

const DEFAULT_RETURN_PATH = "/admin/test-mode";

interface PersonaOption {
  persona: TestModePersona;
  label: string;
  description: string;
  icon: typeof UserRound;
}

const PERSONA_OPTIONS: PersonaOption[] = [
  {
    persona: TestModePersonas.CLIENT,
    label: "Test Client",
    description:
      "Browse the marketplace and buy Generic Programs. Every purchase completes instantly at no cost.",
    icon: UserRound
  },
  {
    persona: TestModePersonas.TRAINER,
    label: "Test Trainer",
    description:
      "A persona with a linked trainer profile, ready to exercise the trainer customer experience.",
    icon: Dumbbell
  }
];

type PageState =
  | { status: "loading" }
  | { status: "ready" }
  | { status: "error"; message: string };

export const AdminTestModePage = () => {
  const { identity } = useAdminSession();
  const [active, setActive] = useState(false);
  const [activePersona, setActivePersona] = useState<TestModePersona | null>(null);
  const [pageState, setPageState] = useState<PageState>({ status: "loading" });
  const [returnPath, setReturnPath] = useState(DEFAULT_RETURN_PATH);
  const [submitting, setSubmitting] = useState<TestModePersona | null>(null);
  const [actionError, setActionError] = useState("");

  const load = useCallback(async () => {
    setPageState({ status: "loading" });
    try {
      const status = await fetchTestModeStatus();
      setActive(status.active);
      setActivePersona(status.persona && status.persona === "trainer" ? "trainer" : status.persona === "client" ? "client" : null);
      setPageState({ status: "ready" });
    } catch {
      setPageState({
        status: "error",
        message: "Unable to reach the Test Mode endpoint. Make sure the backend is running."
      });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const isTechnicalAdministrator = identity.role === TECHNICAL_ADMINISTRATOR;

  const handleEnter = async (persona: TestModePersona) => {
    if (submitting) {
      return;
    }
    setSubmitting(persona);
    setActionError("");
    try {
      // The backend swaps the session cookies; the hard redirect guarantees the
      // client continues with exactly the persona the server minted.
      await enterTestMode(persona, returnPath.trim() || DEFAULT_RETURN_PATH);
    } catch (error) {
      if (error instanceof ApiError && error.code === "TEST_MODE_DISABLED") {
        setActionError(
          "Test Mode is disabled on the server. Set TEST_MODE_ENABLED=true in the environment and restart the backend."
        );
      } else {
        setActionError(
          error instanceof Error ? error.message : "Unable to enter Test Mode. Please try again."
        );
      }
      setSubmitting(null);
    }
  };

  const handleExit = async () => {
    if (submitting) {
      return;
    }
    setSubmitting(TestModePersonas.CLIENT);
    setActionError("");
    try {
      await exitTestMode();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Unable to exit Test Mode. Please try again.");
      setSubmitting(null);
    }
  };

  if (!isTechnicalAdministrator) {
    return (
      <div className={styles.page}>
        <header className={styles.header}>
          <p className={styles.eyebrow}>Test Mode</p>
          <h1 className={styles.title}>Admin Test Mode</h1>
          <p className={styles.description}>
            Only the Technical Administrator can enter Test Mode.
          </p>
        </header>
        <div className={styles.card} role="status">
          <ShieldAlert className={styles.cardIcon} aria-hidden="true" />
          <h2 className={styles.cardTitle}>Restricted</h2>
          <p className={styles.cardDescription}>
            The backend enforces this restriction on every Test Mode action.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <p className={styles.eyebrow}>Test Mode</p>
        <h1 className={styles.title}>Admin Test Mode</h1>
        <p className={styles.description}>
          Impersonate a predefined persona to exercise the customer commerce
          flows without real payments. Inside Test Mode every purchasable
          Generic Program renders as Free and purchases complete server-side
          with no payment provider.
        </p>
      </header>

      {pageState.status === "loading" ? (
        <div className={styles.card} role="status">
          <p className={styles.statusText}>Checking Test Mode status…</p>
        </div>
      ) : pageState.status === "error" ? (
        <div className={styles.card} role="alert">
          <p className={styles.statusText}>{pageState.message}</p>
          <Button variant="secondary" size="small" onClick={() => void load()}>
            Retry
          </Button>
        </div>
      ) : active ? (
        <div className={styles.card} role="status">
          <div className={styles.activeRow}>
            <span className={styles.activeIcon} aria-hidden="true">
              <FlaskConical size={18} strokeWidth={1.8} />
            </span>
            <div className={styles.activeText}>
              <h2 className={styles.cardTitle}>
                Test Mode is active as {activePersona === "trainer" ? "Test Trainer" : "Test Client"}
              </h2>
              <p className={styles.cardDescription}>
                Exiting restores the original administrator session and returns
                to {returnPath || "/"}.
              </p>
            </div>
          </div>
          {actionError ? (
            <p className={styles.actionError}>{actionError}</p>
          ) : null}
          <div className={styles.actions}>
            <Button
              variant="danger"
              size="small"
              onClick={() => void handleExit()}
              disabled={submitting !== null}
              icon={<LogOut size={14} aria-hidden="true" />}
            >
              {submitting ? "Exiting…" : "Exit Test Mode"}
            </Button>
          </div>
        </div>
      ) : (
        <>
          <div className={styles.returnPath}>
            <label className={styles.returnPathLabel} htmlFor="test-mode-return-path">
              Return path after exiting
            </label>
            <input
              id="test-mode-return-path"
              className={styles.returnPathInput}
              type="text"
              value={returnPath}
              onChange={(event) => setReturnPath(event.target.value)}
              placeholder="/admin/test-mode"
            />
          </div>

          {actionError ? <p className={styles.actionError}>{actionError}</p> : null}

          <div className={styles.personas}>
            {PERSONA_OPTIONS.map((option) => {
              const Icon = option.icon;
              const isSubmitting = submitting === option.persona;
              return (
                <article key={option.persona} className={styles.personaCard}>
                  <div className={joinClassNames(styles.personaHeader, submitting !== null && styles.personaDisabled)}>
                    <span className={styles.personaIcon} aria-hidden="true">
                      <Icon size={18} strokeWidth={1.8} />
                    </span>
                    <h2 className={styles.personaTitle}>{option.label}</h2>
                  </div>
                  <p className={styles.personaDescription}>{option.description}</p>
                  <div className={styles.personaActions}>
                    <Button
                      variant="primary"
                      size="small"
                      onClick={() => void handleEnter(option.persona)}
                      disabled={submitting !== null}
                      icon={<FlaskConical size={14} aria-hidden="true" />}
                    >
                      {isSubmitting ? "Entering…" : `Enter as ${option.label}`}
                    </Button>
                  </div>
                </article>
              );
            })}
          </div>
        </>
      )}
    </div>
  );
};