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

  return (
    <>
      <AnimatePresence>{isInitialLoading ? <LoadingScreen /> : null}</AnimatePresence>
      <ScrollToTop />
      <Layout showChrome={!CHROME_FREE_PATHNAMES.has(pathname)}>
        <AppRoutes />
      </Layout>
    </>
  );
};
