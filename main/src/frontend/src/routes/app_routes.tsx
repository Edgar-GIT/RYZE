import { AnimatePresence } from "framer-motion";
import { Redirect, Route, Switch, useLocation } from "react-router-dom";

import { AdminLayout } from "@/components/admin/admin_layout/admin_layout";
import Admin2Layout from "@/components/admin2/admin2_layout/admin2_layout";
import { Admin2SessionProvider } from "@/components/admin2/admin2_session_context";
import { ContactPage } from "@/pages/contact_page/contact_page";
import { FeedbackPage } from "@/pages/feedback_page/feedback_page";
import { GenericPlanMarketplacePage } from "@/pages/generic_plan_marketplace_page/generic_plan_marketplace_page";
import { GenericProgramDetailPage } from "@/pages/generic_program_detail_page/generic_program_detail_page";
import { HomePage } from "@/pages/home_page/home_page";
import { LoginPage } from "@/pages/login_page/login_page";
import { MyProgramsPage } from "@/pages/my_programs_page/my_programs_page";
import { OurVisionPage } from "@/pages/our_vision_page/our_vision_page";
import { ProfilePage } from "@/pages/profile_page/profile_page";
import { ProgramAccessPage } from "@/pages/program_access_page/program_access_page";
import { ServicesPage } from "@/pages/services_page/services_page";
import { UnderDevelopmentPage } from "@/pages/under_development_page/under_development_page";
import { AdminAuditPage } from "@/pages/admin/admin_audit_page/admin_audit_page";
import { AdminConfigurationPage } from "@/pages/admin/admin_configuration_page/admin_configuration_page";
import { AdminDashboardPage } from "@/pages/admin/admin_dashboard_page/admin_dashboard_page";
import { AdminDevelopmentPage } from "@/pages/admin/admin_development_page/admin_development_page";
import { AdminSystemPage } from "@/pages/admin/admin_system_page/admin_system_page";
import { AdminTestModePage } from "@/pages/admin/admin_test_mode_page/admin_test_mode_page";
import { AdminTrainerApplicationsPage } from "@/pages/admin/admin_trainer_applications_page/admin_trainer_applications_page";
import { AdminTrainersPage } from "@/pages/admin/admin_trainers_page/admin_trainers_page";
import { AdminUsersPage } from "@/pages/admin/admin_users_page/admin_users_page";
import AdminLoginPage from "@/pages/admin_login_page/admin_login_page";
import Admin2DashboardPage from "@/pages/admin2/admin2_dashboard_page/admin2_dashboard_page";
import Admin2PlansPage from "@/pages/admin2/admin2_plans_page/admin2_plans_page";
import Admin2PlanCreatePage from "@/pages/admin2/admin2_plan_create_page/admin2_plan_create_page";
import Admin2PlanDetailPage from "@/pages/admin2/admin2_plan_detail_page/admin2_plan_detail_page";
import Admin2MarketplacePage from "@/pages/admin2/admin2_marketplace_page/admin2_marketplace_page";
import Admin2TrainersPage from "@/pages/admin2/admin2_trainers_page/admin2_trainers_page";
import Admin2ClientsPage from "@/pages/admin2/admin2_clients_page/admin2_clients_page";
import Admin2SalesPage from "@/pages/admin2/admin2_sales_page/admin2_sales_page";
import Admin2AnalyticsPage from "@/pages/admin2/admin2_analytics_page/admin2_analytics_page";
import Admin2FeedbackPage from "@/pages/admin2/admin2_feedback_page/admin2_feedback_page";
import Admin2SettingsPage from "@/pages/admin2/admin2_settings_page/admin2_settings_page";

export const AppRoutes = () => {
  const location = useLocation();

  return (
    <AnimatePresence mode="wait">
      <Switch location={location} key={location.pathname}>
        <Route exact path="/" component={HomePage} />
        <Route exact path="/services" component={ServicesPage} />
        <Route exact path="/services/generic-program" component={GenericPlanMarketplacePage} />
        <Route exact path="/services/generic-program/:programId" component={GenericProgramDetailPage} />
        <Route exact path="/services/my-programs" component={MyProgramsPage} />
        <Route exact path="/services/my-programs/:programId" component={ProgramAccessPage} />
        <Route
          path="/services/:serviceSlug"
          render={() => (
            <UnderDevelopmentPage
              eyebrow="Service details"
              title="This program page is under development."
              description="The visual route is ready. Product details, pricing and backend-backed purchase flows will be connected in a later implementation."
            />
          )}
        />
        <Route exact path="/contact" component={ContactPage} />
        <Route exact path="/our-vision" component={OurVisionPage} />
        <Route exact path="/about-us" render={() => <Redirect to="/our-vision" />} />
        <Route exact path="/feedback" component={FeedbackPage} />
        <Route
          path="/feed"
          render={() => (
            <UnderDevelopmentPage
              eyebrow="Feed"
              title="Feed is under development."
              description="The RYZE activity feed will be introduced after the frontend foundation is ready."
            />
          )}
        />
        <Route exact path="/login" component={LoginPage} />
        <Route exact path="/admin/login" component={AdminLoginPage} />
        <Route
          path="/register"
          render={() => <LoginPage initialMode="register" />}
        />
        <Route exact path="/profile" component={ProfilePage} />
        <Route
          path="/admin2"
          render={() => (
            <Admin2SessionProvider>
              <Admin2Layout>
                <Switch>
                  <Redirect from="/admin2" exact to="/admin2/dashboard" />
                  <Route exact path="/admin2/dashboard" component={Admin2DashboardPage} />
                  <Route exact path="/admin2/plans" component={Admin2PlansPage} />
                  <Route exact path="/admin2/plans/create" component={Admin2PlanCreatePage} />
                  <Route exact path="/admin2/plans/:programId" component={Admin2PlanDetailPage} />
                  <Route exact path="/admin2/marketplace" component={Admin2MarketplacePage} />
                  <Route exact path="/admin2/trainers" component={Admin2TrainersPage} />
                  <Route exact path="/admin2/clients" component={Admin2ClientsPage} />
                  <Route exact path="/admin2/sales" component={Admin2SalesPage} />
                  <Route exact path="/admin2/analytics" component={Admin2AnalyticsPage} />
                  <Route exact path="/admin2/feedback" component={Admin2FeedbackPage} />
                  <Route exact path="/admin2/settings" component={Admin2SettingsPage} />
                  <Redirect to="/admin2/dashboard" />
                </Switch>
              </Admin2Layout>
            </Admin2SessionProvider>
          )}
        />
        <Route
          path="/admin"
          render={() => (
            <AdminLayout>
              <Switch>
                <Redirect from="/admin" exact to="/admin/dashboard" />
                <Route exact path="/admin/dashboard" component={AdminDashboardPage} />
                <Route exact path="/admin/users" component={AdminUsersPage} />
                <Route exact path="/admin/trainers" component={AdminTrainersPage} />
                <Route exact path="/admin/trainer-applications" component={AdminTrainerApplicationsPage} />
                <Route exact path="/admin/system" component={AdminSystemPage} />
                <Route exact path="/admin/configuration" component={AdminConfigurationPage} />
                <Route exact path="/admin/test-mode" component={AdminTestModePage} />
                <Route exact path="/admin/development" component={AdminDevelopmentPage} />
                <Route exact path="/admin/audit" component={AdminAuditPage} />
                <Redirect to="/admin/dashboard" />
              </Switch>
            </AdminLayout>
          )}
        />
        <Route
          render={() => (
            <UnderDevelopmentPage
              title="This page is under development."
              description="The route is handled by RYZE and will receive its final screen when the corresponding feature is implemented."
            />
          )}
        />
      </Switch>
    </AnimatePresence>
  );
};
