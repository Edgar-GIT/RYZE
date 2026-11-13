import { Redirect, Route, Switch } from "react-router-dom";

import Admin2Layout from "@/components/admin2/admin2_layout/admin2_layout";
import { Admin2SessionProvider } from "@/components/admin2/admin2_session_context";
import Admin2DashboardPage from "@/pages/admin2/admin2_dashboard_page/admin2_dashboard_page";
import Admin2PlansPage from "@/pages/admin2/admin2_plans_page/admin2_plans_page";
import Admin2PremiumLevel2Page from "@/pages/admin2/admin2_premium_level2_page/admin2_premium_level2_page";
import Admin2PlanCreatePage from "@/pages/admin2/admin2_plan_create_page/admin2_plan_create_page";
import Admin2PlanDetailPage from "@/pages/admin2/admin2_plan_detail_page/admin2_plan_detail_page";
import Admin2MarketplacePage from "@/pages/admin2/admin2_marketplace_page/admin2_marketplace_page";
import Admin2TrainersPage from "@/pages/admin2/admin2_trainers_page/admin2_trainers_page";
import Admin2ClientsPage from "@/pages/admin2/admin2_clients_page/admin2_clients_page";
import Admin2SalesPage from "@/pages/admin2/admin2_sales_page/admin2_sales_page";
import Admin2AnalyticsPage from "@/pages/admin2/admin2_analytics_page/admin2_analytics_page";
import Admin2FeedbackPage from "@/pages/admin2/admin2_feedback_page/admin2_feedback_page";
import Admin2SupportPage from "@/pages/admin2/admin2_support_page/admin2_support_page";
import Admin2ExercisesPage from "@/pages/admin2/admin2_exercises_page/admin2_exercises_page";
import Admin2SettingsPage from "@/pages/admin2/admin2_settings_page/admin2_settings_page";

// The Management Administrator console (ADMIN_2): its shell and pages.
// Loaded on demand (see app_routes.tsx), so public visitors never download it.
const Admin2Routes = () => (
  <Admin2SessionProvider>
    <Admin2Layout>
      <Switch>
        <Redirect from="/admin2" exact to="/admin2/dashboard" />
        <Route exact path="/admin2/dashboard" component={Admin2DashboardPage} />
        <Route exact path="/admin2/plans" component={Admin2PlansPage} />
        <Route exact path="/admin2/premium-level-2" component={Admin2PremiumLevel2Page} />
        <Route exact path="/admin2/plans/create" component={Admin2PlanCreatePage} />
        <Route exact path="/admin2/plans/:programId" component={Admin2PlanDetailPage} />
        <Route exact path="/admin2/exercises" component={Admin2ExercisesPage} />
        <Route exact path="/admin2/marketplace" component={Admin2MarketplacePage} />
        <Route exact path="/admin2/trainers" component={Admin2TrainersPage} />
        <Route exact path="/admin2/clients" component={Admin2ClientsPage} />
        <Route exact path="/admin2/sales" component={Admin2SalesPage} />
        <Route exact path="/admin2/analytics" component={Admin2AnalyticsPage} />
        <Route exact path="/admin2/feedback" component={Admin2FeedbackPage} />
        <Route exact path="/admin2/support" component={Admin2SupportPage} />
        <Route exact path="/admin2/settings" component={Admin2SettingsPage} />
        <Redirect to="/admin2/dashboard" />
      </Switch>
    </Admin2Layout>
  </Admin2SessionProvider>
);

export default Admin2Routes;
