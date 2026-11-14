import { Route, Switch } from "react-router-dom";

import { MemberShell } from "@/components/app_shell/member_shell";
import { TrainerSessionProvider } from "@/components/trainer/trainer_session_context";
import { AccountPurchaseHistoryPage } from "@/pages/account_purchase_history_page/account_purchase_history_page";
import { AccountSettingsPage } from "@/pages/account_settings_page/account_settings_page";
import { MemberDashboardPage } from "@/pages/member_dashboard_page/member_dashboard_page";
import { MyProgramsPage } from "@/pages/my_programs_page/my_programs_page";
import { ProgramAccessPage } from "@/pages/program_access_page/program_access_page";
import { ProgressPage } from "@/pages/progress_page/progress_page";
import { SupportPage } from "@/pages/support_page/support_page";
import { TrainerApplyPage } from "@/pages/trainer/trainer_apply_page/trainer_apply_page";
import { WorkoutHistoryPage } from "@/pages/workout_history_page/workout_history_page";

// The member area: one shell (navigation, account) around all of these pages,
// mounted once while the member moves between them. Loaded on demand (see
// app_routes.tsx), so public visitors never download it.
const MemberRoutes = () => (
  <MemberShell>
    <Switch>
      <Route exact path="/profile" component={MemberDashboardPage} />
      <Route exact path="/account/settings" component={AccountSettingsPage} />
      <Route exact path="/account/purchases" component={AccountPurchaseHistoryPage} />
      <Route exact path="/support" component={SupportPage} />
      <Route exact path="/services/my-programs" component={MyProgramsPage} />
      <Route exact path="/services/my-programs/:programId" component={ProgramAccessPage} />
      <Route exact path="/services/workout-history" component={WorkoutHistoryPage} />
      <Route exact path="/services/progress" component={ProgressPage} />
      <Route
        exact
        path="/trainer/apply"
        render={() => (
          <TrainerSessionProvider>
            <TrainerApplyPage />
          </TrainerSessionProvider>
        )}
      />
    </Switch>
  </MemberShell>
);

export default MemberRoutes;
