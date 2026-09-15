import { AnimatePresence } from "framer-motion";
import { useLocation } from "react-router-dom";

import { Layout } from "@/components/layout/layout";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { useInitialLoading } from "@/hooks/use_initial_loading";
import { AppRoutes } from "@/routes/app_routes";
import { ScrollToTop } from "@/routes/scroll_to_top";

const CHROME_FREE_PATHNAMES = new Set(["/login", "/register", "/profile"]);

export const App = () => {
  const isInitialLoading = useInitialLoading();
  const { pathname } = useLocation();

  // Admin routes render inside their own dedicated shell (AdminLayout) and the
  // public chrome (navbar/footer) never appears in the admin area.
  const isAdminRoute = pathname.startsWith("/admin");
  const showChrome = !isAdminRoute && !CHROME_FREE_PATHNAMES.has(pathname);

  return (
    <>
      <AnimatePresence>{isInitialLoading ? <LoadingScreen /> : null}</AnimatePresence>
      <ScrollToTop />
      <Layout showChrome={showChrome}>
        <AppRoutes />
      </Layout>
    </>
  );
};
