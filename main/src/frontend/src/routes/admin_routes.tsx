import { Redirect, Route, Switch } from "react-router-dom";

import { AdminLayout } from "@/components/admin/admin_layout/admin_layout";
import { AdminAuditPage } from "@/pages/admin/admin_audit_page/admin_audit_page";
import { AdminConfigurationPage } from "@/pages/admin/admin_configuration_page/admin_configuration_page";
import { AdminDashboardPage } from "@/pages/admin/admin_dashboard_page/admin_dashboard_page";
import { AdminSystemPage } from "@/pages/admin/admin_system_page/admin_system_page";
import { AdminTestModePage } from "@/pages/admin/admin_test_mode_page/admin_test_mode_page";
import { AdminTrainerApplicationsPage } from "@/pages/admin/admin_trainer_applications_page/admin_trainer_applications_page";
import { AdminTrainersPage } from "@/pages/admin/admin_trainers_page/admin_trainers_page";
import { AdminUsersPage } from "@/pages/admin/admin_users_page/admin_users_page";

// The Technical Administrator console (ADMIN_1): its shell and pages.
// Loaded on demand (see app_routes.tsx), so public visitors never download it.
const AdminRoutes = () => (
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
      <Route exact path="/admin/audit" component={AdminAuditPage} />
      <Redirect to="/admin/dashboard" />
    </Switch>
  </AdminLayout>
);

export default AdminRoutes;
