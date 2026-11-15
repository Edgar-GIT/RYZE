import { useEffect } from "react";
import { useHistory, useLocation } from "react-router-dom";

import { startNavigationTracking } from "@/services/navigation_history";

export const ScrollToTop = () => {
  const { pathname } = useLocation();

  useEffect(() => {
    window.scrollTo(0, 0);
  }, [pathname]);

  return null;
};

// NavigationTracker records this tab's RYZE history for the back buttons.
export const NavigationTracker = () => {
  const history = useHistory();

  useEffect(() => startNavigationTracking(history), [history]);

  return null;
};
