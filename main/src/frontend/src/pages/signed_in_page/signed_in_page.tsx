import { useEffect } from "react";
import { useHistory } from "react-router-dom";

import { fetchTrainerAccess } from "@/services/trainer_api";
import { ApiError } from "@utils/http_client";

import styles from "./signed_in_page.module.css";

// SignedInPage (/app) decides where an account lands right after signing in:
// approved trainers open their trainer area first (owner request,
// 2026-11-14), every other member opens their home. The decision comes from
// the backend (GET /trainer/access); a visitor without a session goes to the
// login page.
export const SignedInPage = () => {
  const history = useHistory();

  useEffect(() => {
    let cancelled = false;
    fetchTrainerAccess()
      .then((access) => {
        if (!cancelled) {
          history.replace(access.is_trainer ? "/trainer/dashboard" : "/profile");
        }
      })
      .catch((error) => {
        if (cancelled) return;
        history.replace(error instanceof ApiError && error.status === 401 ? "/login" : "/profile");
      });
    return () => {
      cancelled = true;
    };
  }, [history]);

  return (
    <div className={styles.screen} role="status" aria-live="polite">
      <span className={styles.ring} aria-hidden="true" />
      <span className={styles.visuallyHidden}>Loading…</span>
    </div>
  );
};

export default SignedInPage;
