import { ArrowRight, CalendarDays, Dumbbell, History, ListChecks, UserCheck, UtensilsCrossed } from "lucide-react";
import { Link } from "react-router-dom";

import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { PublicAction, Waves } from "@/components/public_ui/public_ui";
import { Reveal } from "@/components/reveal/reveal";
import { ServiceCard } from "@/components/service_card/service_card";
import ui from "@/components/public_ui/public_ui.module.css";
import { SERVICE_SHOWCASE } from "@/constants/services";
import { useServiceCatalog } from "@/hooks/use_service_catalog";
import { buildServiceCategoryCards } from "@/services/service_catalog";
import { joinClassNames } from "@utils/class_names";

import styles from "./services_page.module.css";

// What every route includes, in the product's own terms.
const evolutionItems = [
  {
    icon: CalendarDays,
    title: "Planned for you",
    text: "Weeks and training days, laid out in order."
  },
  {
    icon: ListChecks,
    title: "Every detail set",
    text: "Sets, reps, load, rest and tempo per exercise."
  },
  {
    icon: Dumbbell,
    title: "Log as you go",
    text: "A live session timer with set-by-set logging."
  },
  {
    icon: History,
    title: "See your progress",
    text: "Workout history after every session."
  }
] as const;

// Helps a visitor pick a route by what they want, not by level names.
const ROUTES = [
  {
    icon: Dumbbell,
    title: "Just training",
    text: "Ready-made training plans, including free ones.",
    to: "/services/generic-program"
  },
  {
    icon: UtensilsCrossed,
    title: "Training and nutrition",
    text: "Premium Level 1: a hypertrophy programme with a nutrition plan, delivered right after purchase.",
    to: "/services/premium-level-1"
  },
  {
    icon: UserCheck,
    title: "Built around you",
    text: "Premium Level 2: a RYZE trainer prepares your programme and nutrition plan after an intake.",
    to: "/services/premium-level-2"
  }
] as const;

export const ServicesPage = () => {
  const catalog = useServiceCatalog();
  const services =
    catalog.status === "ready"
      ? buildServiceCategoryCards(catalog.generic, catalog.premium, catalog.premiumLevel2)
      : [];

  return (
    <PageWrapper className={joinClassNames(ui.page, styles.page)}>
      {/* The section sets no inherited font or colour, so the service cards keep
          their own styling; only the intro, states and benefits use the public tones. */}
      <section className={styles.catalog}>
        <Container className={styles.catalogInner}>
          <Reveal className={joinClassNames(ui.toneNavy, styles.intro)}>
            <p className={ui.label}>Services</p>
            <h1 className={ui.heading}>
              Choose how you want to <span className={ui.hl}>RYZE</span>
              <span className={ui.stop}>.</span>
            </h1>
            <p className={joinClassNames(ui.lead, styles.lead)}>
              Compare the published RYZE offers and choose the route that fits the guidance and
              personalization you need.
            </p>
          </Reveal>

          {catalog.status === "loading" ? (
            <div className={joinClassNames(ui.toneNavy, ui.surface, styles.state)}>
              <p className={styles.stateText}>Loading the published offers…</p>
            </div>
          ) : (
            // Live offers when published; otherwise the original showcase cards.
            <div className={styles.grid}>
              {(services.length > 0 ? services : SERVICE_SHOWCASE).map((service) => (
                <ServiceCard key={service.title} service={service} />
              ))}
            </div>
          )}

          <Reveal className={joinClassNames(ui.toneNavy, ui.surface, styles.evolution)} delay={0.18}>
            {evolutionItems.map((item, index) => {
              const Icon = item.icon;

              return (
                <article key={item.title} className={styles.evolutionItem}>
                  <span className={ui.iconTile}>
                    <Icon aria-hidden="true" strokeWidth={1.9} />
                  </span>
                  <div className={styles.evolutionCopy}>
                    <strong>{item.title}</strong>
                    <p>{item.text}</p>
                  </div>
                  {index < evolutionItems.length - 1 && (
                    <span className={styles.evolutionDivider} aria-hidden="true" />
                  )}
                </article>
              );
            })}
          </Reveal>

          <Reveal className={joinClassNames(ui.split, styles.guide)} delay={0.1}>
            <div className={joinClassNames(ui.toneNavy, ui.splitPane)}>
              <h2 className={ui.title}>
                Which one is <span className={ui.hl}>for you?</span>
              </h2>
              <ul className={styles.routes}>
                {ROUTES.map((route) => {
                  const Icon = route.icon;
                  return (
                    <li key={route.to}>
                      <Link className={styles.route} to={route.to}>
                        <span className={ui.iconTile} aria-hidden="true">
                          <Icon strokeWidth={1.9} />
                        </span>
                        <span className={styles.routeCopy}>
                          <strong>{route.title}</strong>
                          <span>{route.text}</span>
                        </span>
                        <ArrowRight className={styles.routeArrow} aria-hidden="true" />
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>

            <div className={joinClassNames(ui.toneLight, ui.splitPane, ui.splitLight)}>
              <p className={styles.kicker}>New to RYZE?</p>
              <h2 className={joinClassNames(ui.heading, styles.lightTitle)}>
                Start free <span className={ui.hl}>today.</span>
              </h2>
              <p className={ui.body}>
                Create a free account, pick a free training plan and log your first session. Upgrade whenever you&apos;re
                ready.
              </p>
              <div className={ui.actions}>
                <PublicAction to="/register" variant="ink" icon={<ArrowRight />}>
                  Create free account
                </PublicAction>
                <PublicAction to="/login" variant="ghostOnLight">
                  Log in
                </PublicAction>
              </div>
              <Waves />
            </div>
          </Reveal>
        </Container>
      </section>
    </PageWrapper>
  );
};
