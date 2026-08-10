import { AnimatePresence } from "framer-motion";
import { Redirect, Route, Switch, useLocation } from "react-router-dom";

import { ContactPage } from "@/pages/contact_page/contact_page";
import { FeedbackPage } from "@/pages/feedback_page/feedback_page";
import { HomePage } from "@/pages/home_page/home_page";
import { LoginPage } from "@/pages/login_page/login_page";
import { OurVisionPage } from "@/pages/our_vision_page/our_vision_page";
import { ProfilePage } from "@/pages/profile_page/profile_page";
import { ServicesPage } from "@/pages/services_page/services_page";
import { UnderDevelopmentPage } from "@/pages/under_development_page/under_development_page";

export const AppRoutes = () => {
  const location = useLocation();

  return (
    <AnimatePresence mode="wait">
      <Switch location={location} key={location.pathname}>
        <Route exact path="/" component={HomePage} />
        <Route exact path="/services" component={ServicesPage} />
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
        <Route
          path="/register"
          render={() => <LoginPage initialMode="register" />}
        />
        <Route exact path="/profile" component={ProfilePage} />
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
