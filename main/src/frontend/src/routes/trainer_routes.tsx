import TrainerPlanBuilderPage from "@/pages/trainer/trainer_plan_builder_page/trainer_plan_builder_page";
import { Redirect, Route, Switch } from "react-router-dom";

import { TrainerSessionProvider } from "@/components/trainer/trainer_session_context";
import { TrainerLayout } from "@/components/trainer/trainer_layout/trainer_layout";
import { TrainerClientDetailPage } from "@/pages/trainer/trainer_client_detail_page/trainer_client_detail_page";
import { TrainerClientsPage } from "@/pages/trainer/trainer_clients_page/trainer_clients_page";
import { TrainerDashboardPage } from "@/pages/trainer/trainer_dashboard_page/trainer_dashboard_page";
import { TrainerProfilePage } from "@/pages/trainer/trainer_profile_page/trainer_profile_page";
import { TrainerProgramsPage } from "@/pages/trainer/trainer_programs_page/trainer_programs_page";
import { TrainerPremiumLevel2Page } from "@/pages/trainer/trainer_premium_level2_page/trainer_premium_level2_page";
import { TrainerPremiumLevel2DetailPage } from "@/pages/trainer/trainer_premium_level2_detail_page/trainer_premium_level2_detail_page";

// The approved-trainer area: its shell and pages.
// Loaded on demand (see app_routes.tsx), so public visitors never download it.
const TrainerRoutes = () => (
  <TrainerSessionProvider>
    <TrainerLayout>
      <Switch>
        <Redirect from="/trainer" exact to="/trainer/dashboard" />
        <Route exact path="/trainer/dashboard" component={TrainerDashboardPage} />
        <Route exact path="/trainer/clients" component={TrainerClientsPage} />
        <Route exact path="/trainer/clients/:userID" component={TrainerClientDetailPage} />
        <Route exact path="/trainer/premium-level-2" component={TrainerPremiumLevel2Page} />
        <Route exact path="/trainer/premium-level-2/:requestID" component={TrainerPremiumLevel2DetailPage} />
        <Route exact path="/trainer/programs" component={TrainerProgramsPage} />
        <Route exact path="/trainer/programs/new" component={TrainerPlanBuilderPage} />
        <Route exact path="/trainer/programs/:programId" component={TrainerPlanBuilderPage} />
        <Route exact path="/trainer/profile" component={TrainerProfilePage} />
        <Redirect to="/trainer/dashboard" />
      </Switch>
    </TrainerLayout>
  </TrainerSessionProvider>
);

export default TrainerRoutes;
