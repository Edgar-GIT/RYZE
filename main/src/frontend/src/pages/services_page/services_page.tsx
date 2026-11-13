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
import { type Copy, useI18n } from "@/i18n/i18n";
import { buildServiceCategoryCards } from "@/services/service_catalog";
import { joinClassNames } from "@utils/class_names";

import styles from "./services_page.module.css";

// What every route includes, in the product's own terms.
const EVOLUTION_ICONS = [CalendarDays, ListChecks, Dumbbell, History] as const;

// Helps a visitor pick a route by what they want, not by level names.
const ROUTE_STRUCTURE = [
  { icon: Dumbbell, to: "/services/generic-program" },
  { icon: UtensilsCrossed, to: "/services/premium-level-1" },
  { icon: UserCheck, to: "/services/premium-level-2" }
] as const;

interface Item {
  title: string;
  text: string;
}

interface ServicesText {
  label: string;
  heading: string;
  lead: string;
  loading: string;
  evolution: Item[];
  whichOne: string;
  whichOneHighlight: string;
  routes: Item[];
  newHere: string;
  startFree: string;
  startFreeHighlight: string;
  startFreeText: string;
  createAccount: string;
  login: string;
}

const COPY: Copy<ServicesText> = {
  en: {
    label: "Services",
    heading: "Choose how you want to ",
    lead: "Compare the published RYZE offers and choose the route that fits the guidance and personalization you need.",
    loading: "Loading the published offers…",
    evolution: [
      { title: "Planned for you", text: "Weeks and training days, laid out in order." },
      { title: "Every detail set", text: "Sets, reps, load, rest and tempo per exercise." },
      { title: "Log as you go", text: "A live session timer with set-by-set logging." },
      { title: "See your progress", text: "Workout history after every session." }
    ],
    whichOne: "Which one is ",
    whichOneHighlight: "for you?",
    routes: [
      { title: "Just training", text: "Ready-made training plans, including free ones." },
      {
        title: "Training and nutrition",
        text: "Premium Level 1: a hypertrophy programme with a nutrition plan, delivered right after purchase."
      },
      {
        title: "Built around you",
        text: "Premium Level 2: a RYZE trainer prepares your programme and nutrition plan after an intake."
      }
    ],
    newHere: "New to RYZE?",
    startFree: "Start free ",
    startFreeHighlight: "today.",
    startFreeText:
      "Create a free account, pick a free training plan and log your first session. Upgrade whenever you're ready.",
    createAccount: "Create free account",
    login: "Log in"
  },
  pt: {
    label: "Serviços",
    heading: "Escolhe como queres fazer ",
    lead: "Compara as ofertas RYZE publicadas e escolhe o caminho com o acompanhamento e a personalização de que precisas.",
    loading: "A carregar as ofertas publicadas…",
    evolution: [
      { title: "Planeado para ti", text: "Semanas e dias de treino, por ordem." },
      { title: "Cada detalhe definido", text: "Séries, repetições, carga, descanso e cadência por exercício." },
      { title: "Regista enquanto treinas", text: "Cronómetro do treino com registo série a série." },
      { title: "Vê a tua evolução", text: "Histórico depois de cada treino." }
    ],
    whichOne: "Qual é ",
    whichOneHighlight: "para ti?",
    routes: [
      { title: "Só treino", text: "Planos de treino prontos, incluindo gratuitos." },
      {
        title: "Treino e nutrição",
        text: "Premium Nível 1: um programa de hipertrofia com plano de nutrição, disponível logo após a compra."
      },
      {
        title: "Feito à tua medida",
        text: "Premium Nível 2: um treinador RYZE prepara o teu programa e plano de nutrição depois de um questionário."
      }
    ],
    newHere: "Novo na RYZE?",
    startFree: "Começa grátis ",
    startFreeHighlight: "hoje.",
    startFreeText:
      "Cria uma conta gratuita, escolhe um plano de treino gratuito e regista o teu primeiro treino. Faz upgrade quando quiseres.",
    createAccount: "Criar conta grátis",
    login: "Entrar"
  }
};

export const ServicesPage = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
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
            <p className={ui.label}>{text.label}</p>
            <h1 className={ui.heading}>
              {text.heading}
              <span className={ui.hl}>RYZE</span>
              <span className={ui.stop}>.</span>
            </h1>
            <p className={joinClassNames(ui.lead, styles.lead)}>{text.lead}</p>
          </Reveal>

          {catalog.status === "loading" ? (
            <div className={joinClassNames(ui.toneNavy, ui.surface, styles.state)}>
              <p className={styles.stateText}>{text.loading}</p>
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
            {text.evolution.map((item, index) => {
              const Icon = EVOLUTION_ICONS[index];

              return (
                <article key={item.title} className={styles.evolutionItem}>
                  <span className={ui.iconTile}>
                    <Icon aria-hidden="true" strokeWidth={1.9} />
                  </span>
                  <div className={styles.evolutionCopy}>
                    <strong>{item.title}</strong>
                    <p>{item.text}</p>
                  </div>
                  {index < text.evolution.length - 1 && (
                    <span className={styles.evolutionDivider} aria-hidden="true" />
                  )}
                </article>
              );
            })}
          </Reveal>

          <Reveal className={joinClassNames(ui.split, styles.guide)} delay={0.1}>
            <div className={joinClassNames(ui.toneNavy, ui.splitPane)}>
              <h2 className={ui.title}>
                {text.whichOne}
                <span className={ui.hl}>{text.whichOneHighlight}</span>
              </h2>
              <ul className={styles.routes}>
                {ROUTE_STRUCTURE.map((route, index) => {
                  const Icon = route.icon;
                  const copy = text.routes[index];
                  return (
                    <li key={route.to}>
                      <Link className={styles.route} to={route.to}>
                        <span className={ui.iconTile} aria-hidden="true">
                          <Icon strokeWidth={1.9} />
                        </span>
                        <span className={styles.routeCopy}>
                          <strong>{copy.title}</strong>
                          <span>{copy.text}</span>
                        </span>
                        <ArrowRight className={styles.routeArrow} aria-hidden="true" />
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>

            <div className={joinClassNames(ui.toneLight, ui.splitPane, ui.splitLight)}>
              <p className={styles.kicker}>{text.newHere}</p>
              <h2 className={joinClassNames(ui.heading, styles.lightTitle)}>
                {text.startFree}
                <span className={ui.hl}>{text.startFreeHighlight}</span>
              </h2>
              <p className={ui.body}>{text.startFreeText}</p>
              <div className={ui.actions}>
                <PublicAction to="/register" variant="ink" icon={<ArrowRight />}>
                  {text.createAccount}
                </PublicAction>
                <PublicAction to="/login" variant="ghostOnLight">
                  {text.login}
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
