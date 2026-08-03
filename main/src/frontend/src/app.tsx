import { AnimatePresence } from "framer-motion";

import { Layout } from "@/components/layout/layout";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { useInitialLoading } from "@/hooks/use_initial_loading";
import { AppRoutes } from "@/routes/app_routes";
import { ScrollToTop } from "@/routes/scroll_to_top";

export const App = () => {
  const isInitialLoading = useInitialLoading();

  return (
    <>
      <AnimatePresence>{isInitialLoading ? <LoadingScreen /> : null}</AnimatePresence>
      <ScrollToTop />
      <Layout>
        <AppRoutes />
      </Layout>
    </>
  );
};
