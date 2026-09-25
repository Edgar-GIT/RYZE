import { FlaskConical, LogOut, ShieldAlert } from "lucide-react";
import { useState } from "react";
import { useLocation } from "react-router-dom";

import { Button } from "@/components/button/button";
import { useTestMode } from "@/components/test_mode/test_mode_context";

import styles from "./test_mode_banner.module.css";

const HIDDEN_PATHNAMES = new Set(["/login", "/register", "/admin/login"]);

const personaLabel = (persona: string | null): string =>
  persona === "trainer" ? "Test Trainer" : "Test Client";

/**
 * TestModeBanner is a fixed, persistent control shown whenever the server
 * reports an active Test Mode session. It announces the persona currently in
 * effect and provides the Exit control. It is hidden inside the admin shells
 * and the authentication pages, where impersonating state would only add
 * noise.
 */
export const TestModeBanner = () => {
  const { pathname } = useLocation();
  const { isActive, isLoading, persona, errorMessage, exitTestMode } = useTestMode();
  const [exiting, setExiting] = useState(false);

  if (!isActive) {
    return null;
  }

  const isAdminRoute = pathname.startsWith("/admin");
  if (isAdminRoute || HIDDEN_PATHNAMES.has(pathname)) {
    return null;
  }

  const handleExit = async () => {
    if (exiting) {
      return;
    }
    setExiting(true);
    try {
      await exitTestMode();
    } catch {
      setExiting(false);
    }
  };

  return (
    <aside className={styles.banner} role="status" aria-label="Admin Test Mode is active">
      <div className={styles.inner}>
        <span className={styles.icon} aria-hidden="true">
          <FlaskConical size={16} strokeWidth={1.8} />
        </span>
        <div className={styles.text}>
          <p className={styles.title}>
            TEST MODE
            {isLoading ? "" : ` · Active as ${personaLabel(persona)}`}
          </p>
          <p className={styles.subtitle}>
            Purchases are completed instantly at no cost. No real payments are made.
          </p>
          {errorMessage ? (
            <p className={styles.error}>
              <ShieldAlert size={13} aria-hidden="true" />
              {errorMessage}
            </p>
          ) : null}
        </div>
        <Button
          variant="danger"
          size="small"
          onClick={() => void handleExit()}
          disabled={exiting}
          icon={<LogOut size={14} aria-hidden="true" />}
        >
          {exiting ? "Exiting…" : "Exit Test Mode"}
        </Button>
      </div>
    </aside>
  );
};