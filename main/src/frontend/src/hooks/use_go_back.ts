import { useCallback } from "react";
import { useHistory } from "react-router-dom";

import { canGoBackInApp } from "@/services/navigation_history";

// useGoBack returns "go back one page": the previous RYZE page when there is
// one in this tab, otherwise the given parent page (never another site).
export const useGoBack = (fallback: string | (() => string)) => {
  const history = useHistory();
  return useCallback(() => {
    if (canGoBackInApp()) {
      history.goBack();
      return;
    }
    history.push(typeof fallback === "function" ? fallback() : fallback);
  }, [history, fallback]);
};
