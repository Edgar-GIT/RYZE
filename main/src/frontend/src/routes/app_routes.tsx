import { AnimatePresence } from "framer-motion";
import { lazy, Suspense } from "react";
import { Redirect, Route, Switch, useLocation } from "react-router-dom";

import { CoachProfilePage } from "@/pages/coach_profile_page/coach_profile_page";
import { ContactPage } from "@/pages/contact_page/contact_page";
import { ExerciseDetailPage } from "@/pages/exercise_detail_page/exercise_detail_page";
import { ExerciseLibraryPage } from "@/pages/exercise_library_page/exercise_library_page";
import { FeedbackPage } from "@/pages/feedback_page/feedback_page";
import { GenericPlanMarketplacePage } from "@/pages/generic_plan_marketplace_page/generic_plan_marketplace_page";
import { GenericProgramDetailPage } from "@/pages/generic_program_detail_page/generic_program_detail_page";
import { HomePage } from "@/pages/home_page/home_page";
import { LoginPage } from "@/pages/login_page/login_page";
import { ForgotPasswordPage, ResetPasswordPage, VerifyEmailPage } from "@/pages/account_recovery_page/account_recovery_pages";
import { OurVisionPage } from "@/pages/our_vision_page/our_vision_page";
import { PremiumLevel1Page } from "@/pages/premium_level1_page/premium_level1_page";
import { PremiumLevel2Page } from "@/pages/premium_level2_page/premium_level2_page";
import { PremiumLevel2RequestPage } from "@/pages/premium_level2_request_page/premium_level2_request_page";
import { WorkoutSessionPage } from "@/pages/workout_session_page/workout_session_page";
import { ServicesPage } from "@/pages/services_page/services_page";
import { UnderDevelopmentPage } from "@/pages/under_development_page/under_development_page";
import AdminLoginPage from "@/pages/admin_login_page/admin_login_page";

// The consoles and the trainer area are separate bundles: a visitor of the
// public site or the member app never downloads them.
const AdminRoutes = lazy(() => import("./admin_routes"));
const Admin2Routes = lazy(() => import("./admin2_routes"));
const TrainerRoutes = lazy(() => import("./trainer_routes"));
const MemberRoutes = lazy(() => import("./member_routes"));

const RouteFallback = () => <div role="status" aria-busy="true" style={{ minHeight: "100svh" }} />;

// The member area: one shell (navigation, account) around all of these pages,
// mounted once while the member moves between them. The workout player stays
// full screen, outside the shell.
const MEMBER_PATHS = [
  "/profile",
  "/account/settings",
  "/account/purchases",
  "/support",
  "/services/my-programs",
  "/services/my-programs/:programId",
  "/services/workout-history",
  "/services/progress",
  "/trainer/apply"
];

const memberPathPattern = /^\/(profile|support|account\/(settings|purchases)|services\/(my-programs(\/[^/]+)?|workout-history|progress)|trainer\/apply)\/?$/;

// The route transition key: one key per area shell, so moving inside the
// member or trainer area never remounts its shell.
const transitionKey = (pathname: string) => {
  if (memberPathPattern.test(pathname)) return "member-area";
  if (pathname.startsWith("/trainer")) return "trainer-area";
  return pathname;
};


export const AppRoutes = () => {
  const location = useLocation();

  return (
    <AnimatePresence mode="wait">
      <Switch location={location} key={transitionKey(location.pathname)}>
        <Route exact path="/" component={HomePage} />
        <Route exact path="/services" component={ServicesPage} />
        <Route exact path="/services/generic-program" component={GenericPlanMarketplacePage} />
        <Route exact path="/coaches/:trainerId" component={CoachProfilePage} />
        {/* Premium Level 1 must be declared before the dynamic /services/:serviceSlug
            route, otherwise the slug route would swallow it. */}
        <Route exact path="/services/premium-level-1" component={PremiumLevel1Page} />
        <Route exact path="/services/premium-level-2" component={PremiumLevel2Page} />
        <Route exact path="/services/premium-level-2/requests/:requestID" component={PremiumLevel2RequestPage} />
        <Route exact path="/services/generic-program/:programId" component={GenericProgramDetailPage} />
        {/* The configured payment return URLs land here (Stripe appends
            session_id, PayPal appends token, both with purchase_id). They must
            be declared before the dynamic /services/:serviceSlug route so the
            return leg of a completed or cancelled checkout is never swallowed. */}
        <Route exact path="/services/generic-program/:programId/purchase/success" component={GenericProgramDetailPage} />
        <Route exact path="/services/generic-program/:programId/purchase/cancelled" component={GenericProgramDetailPage} />
        <Route exact path={MEMBER_PATHS}>
          <Suspense fallback={<RouteFallback />}>
            <MemberRoutes />
          </Suspense>
        </Route>
        <Route exact path="/services/workout-sessions/:sessionId" component={WorkoutSessionPage} />
        <Route exact path="/exercises" component={ExerciseLibraryPage} />
        <Route exact path="/exercises/:exerciseId" component={ExerciseDetailPage} />
        <Route
          path="/services/:serviceSlug"
          render={() => (
            <UnderDevelopmentPage
              kind="notFound"
              eyebrow="Page not found"
              title="We could not find this service."
              description="Check the address, or open Services to see every RYZE plan and package."
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
        <Route exact path="/forgot-password" component={ForgotPasswordPage} />
        <Route exact path="/reset-password" component={ResetPasswordPage} />
        <Route exact path="/verify-email" component={VerifyEmailPage} />
        <Route exact path="/admin/login" component={AdminLoginPage} />
        <Route
          path="/register"
          render={() => <LoginPage initialMode="register" />}
        />
        {/* /trainer/apply belongs to the member area (declared above, before
            the /trainer shell route): it is reachable by any signed-in member,
            while every other /trainer route requires an approved trainer. */}
        <Route path="/trainer">
          <Suspense fallback={<RouteFallback />}>
            <TrainerRoutes />
          </Suspense>
        </Route>
        <Route path="/admin2">
          <Suspense fallback={<RouteFallback />}>
            <Admin2Routes />
          </Suspense>
        </Route>
        <Route path="/admin">
          <Suspense fallback={<RouteFallback />}>
            <AdminRoutes />
          </Suspense>
        </Route>
        <Route
          render={() => (
            <UnderDevelopmentPage
              kind="notFound"
              eyebrow="Page not found"
              title="We could not find this page."
              description="The address may be mistyped or the page may have moved. Go back home or open Services."
            />
          )}
        />
      </Switch>
    </AnimatePresence>
  );
};
