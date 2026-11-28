import { AnimatePresence } from "framer-motion";
import { useEffect } from "react";
import { useLocation } from "react-router-dom";

import { Layout } from "@/components/layout/layout";
import { LanguagePrompt } from "@/components/language_prompt/language_prompt";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PwaUpdatePrompt } from "@/components/pwa_update_prompt/pwa_update_prompt";
import { CookieConsent } from "@/components/cookie_consent/cookie_consent";
import { useInitialLoading } from "@/hooks/use_initial_loading";
import { markAppReady } from "@/services/app_ready";
import { AppRoutes } from "@/routes/app_routes";
import { NavigationTracker, ScrollToTop } from "@/routes/scroll_to_top";

const CHROME_FREE_PATHNAMES = new Set([
  "/app",
  "/login",
  "/register",
  "/profile",
  "/achievements",
  "/forgot-password",
  "/reset-password",
  "/verify-email"
]);

export const App = () => {
  const isInitialLoading = useInitialLoading();
  const { pathname } = useLocation();

  // Admin routes render inside their own dedicated shell (AdminLayout) and the
  // public chrome (navbar/footer) never appears in the admin area.
  const isAdminRoute = pathname.startsWith("/admin");
  const isAccountRoute =
    pathname.startsWith("/account") ||
    pathname.startsWith("/services/my-programs") ||
    pathname.startsWith("/services/workout-") ||
    pathname === "/services/progress" ||
    pathname === "/services/recovery" ||
    pathname === "/services/next-workout";
  // The trainer area has its own shell (TrainerLayout); the application form
  // at /trainer/apply is part of the member area (MemberShell).
  const isTrainerArea = pathname.startsWith("/trainer");
  const showChrome =
    !isAdminRoute && !CHROME_FREE_PATHNAMES.has(pathname) && !isAccountRoute && !isTrainerArea;

  // Private areas (account, training, trainer, administration) share one app
  // theme; see body.ryze-app in styles/global.css.
  const isPrivateApp =
    isAdminRoute ||
    isAccountRoute ||
    pathname.startsWith("/trainer") ||
    pathname === "/profile" ||
    pathname === "/achievements";
  useEffect(() => {
    document.body.classList.toggle("ryze-app", isPrivateApp);
  }, [isPrivateApp]);

  useEffect(() => {
    if (!isInitialLoading) markAppReady();
  }, [isInitialLoading]);

  return (
    <>
      <AnimatePresence>{isInitialLoading ? <LoadingScreen /> : null}</AnimatePresence>
      <ScrollToTop />
      <NavigationTracker />
      {/* First visit only, on the public site, once the loading screen is gone. */}
      {!isInitialLoading && showChrome ? <LanguagePrompt /> : null}
      <Layout showChrome={showChrome}>
        <AppRoutes />
      </Layout>
      <PwaUpdatePrompt />
      {/* EU cookie notice (v2.130.12), once the loading screen is gone. */}
      {!isInitialLoading ? <CookieConsent /> : null}
    </>
  );
};
