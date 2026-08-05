import { AnimatePresence } from "framer-motion";
import { Redirect, Route, Switch, useLocation } from "react-router-dom";

import { ContactPage } from "@/pages/contact_page/contact_page";
import { HomePage } from "@/pages/home_page/home_page";
import { OurVisionPage } from "@/pages/our_vision_page/our_vision_page";
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
              title="This plan page is under development."
              description="The visual route is ready. Product details, pricing and backend-backed purchase flows will be connected in a later implementation."
            />
          )}
        />
        <Route exact path="/contact" component={ContactPage} />
        <Route exact path="/our-vision" component={OurVisionPage} />
        <Route exact path="/about-us" render={() => <Redirect to="/our-vision" />} />
        <Route
          path="/feedback"
          render={() => (
            <UnderDevelopmentPage
              eyebrow="Feedback"
              title="Feedback is under development."
              description="The community area and digital complaints book will be introduced after the frontend foundation is ready."
            />
          )}
        />
        <Route
          path="/login"
          render={() => (
            <UnderDevelopmentPage
              eyebrow="Log in"
              title="Log in is under development."
              description="Authentication will be connected when the backend account flows are implemented."
            />
          )}
        />
        <Route
          path="/register"
          render={() => (
            <UnderDevelopmentPage
              eyebrow="Start free"
              title="Account creation is under development."
              description="Registration and onboarding will be connected when the backend account flows are implemented."
            />
          )}
        />
        <Route
          path="/profile"
          render={() => (
            <UnderDevelopmentPage
              eyebrow="Profile"
              title="Profile is under development."
              description="Authentication and profile management will be connected when the backend account flows are implemented."
            />
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
