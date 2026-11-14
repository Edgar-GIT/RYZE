import {
  CalendarDays,
  ClipboardList,
  Dumbbell,
  History,
  ListChecks,
  type LucideIcon,
  Route,
  Sparkles,
  Target,
  UserCheck,
  UtensilsCrossed,
  Zap
} from "lucide-react";
import { useCallback, useEffect, useRef, useState, type CSSProperties } from "react";

import { ServiceCard } from "@/components/service_card/service_card";
import type { ServiceProgram } from "@/constants/services";
import { type Copy, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./service_showroom.module.css";
import type { ShowroomScene } from "./showroom_scene";

// Each route's goal (the placard) and what it includes (the bubbles), in the
// same words the Services page uses. Keyed by the card's destination.
type RouteKey = "generic" | "premium1" | "premium2";

const ROUTE_BY_PATH: Record<string, RouteKey> = {
  "/services/generic-program": "generic",
  "/services/premium-level-1": "premium1",
  "/services/premium-level-2": "premium2"
};

const FACT_ICONS: Record<RouteKey, readonly LucideIcon[]> = {
  generic: [Dumbbell, CalendarDays, ListChecks, History],
  premium1: [UtensilsCrossed, Zap, Target, Dumbbell],
  premium2: [ClipboardList, UserCheck, Sparkles, Route]
};

interface RouteCopy {
  goal: string;
  goalText: string;
  facts: readonly string[];
}

const COPY: Copy<{ label: string; routes: Record<RouteKey, RouteCopy> }> = {
  en: {
    label: "What's included",
    routes: {
      generic: {
        goal: "Just training",
        goalText: "Ready-made training plans, including free ones",
        facts: ["Ready-made training plans", "Weeks and days laid out in order", "Set-by-set logging", "History and progress"]
      },
      premium1: {
        goal: "Training and nutrition",
        goalText: "A programme with a nutrition plan, right after purchase",
        facts: ["Training plus a nutrition plan", "Available right after purchase", "Daily calorie and macro targets", "Hypertrophy programme"]
      },
      premium2: {
        goal: "Built around you",
        goalText: "A RYZE trainer prepares your programme and nutrition",
        facts: ["A complete intake questionnaire", "A RYZE trainer prepares your plan", "Training and nutrition built around you", "Follow your request until delivery"]
      }
    }
  },
  pt: {
    label: "O que inclui",
    routes: {
      generic: {
        goal: "Só treino",
        goalText: "Planos de treino prontos, incluindo gratuitos",
        facts: ["Planos de treino prontos", "Semanas e dias por ordem", "Registo série a série", "Histórico e progresso"]
      },
      premium1: {
        goal: "Treino e nutrição",
        goalText: "Programa com plano de nutrição, logo após a compra",
        facts: ["Treino com plano de nutrição", "Disponível logo após a compra", "Metas diárias de calorias e macros", "Programa de hipertrofia"]
      },
      premium2: {
        goal: "Feito à tua medida",
        goalText: "Um treinador RYZE prepara o teu programa e nutrição",
        facts: ["Questionário completo", "Um treinador RYZE prepara o teu plano", "Treino e nutrição à tua medida", "Acompanha o pedido até à entrega"]
      }
    }
  }
};

type Position = "left" | "middle" | "right";

const positionsFor = (count: number): Position[] => {
  if (count === 1) return ["middle"];
  if (count === 2) return ["left", "right"];
  return Array.from({ length: count }, (_, index) => (index === 0 ? "left" : index === count - 1 ? "right" : "middle"));
};

// The showroom (3D pedestals, focus on hover) needs a precise pointer and a
// wide screen; touch and small screens get the stacked cards.
const SHOWROOM_QUERY = "(min-width: 1024px) and (hover: hover) and (pointer: fine)";

let webglSupport: boolean | null = null;
const supportsWebGL = (): boolean => {
  if (webglSupport !== null) return webglSupport;
  try {
    const probe = document.createElement("canvas");
    const context = probe.getContext("webgl2");
    webglSupport = context !== null;
    context?.getExtension("WEBGL_lose_context")?.loseContext();
  } catch {
    webglSupport = false;
  }
  return webglSupport;
};

// ServiceShowroom presents the service cards as products on display. On wide
// screens each card stands, flat and crisp, on a lit 3D pedestal with a
// placard naming its goal; hovering or focusing a card lifts it, lights its
// pedestal, steps the others back and floats out what it includes. The cards
// are the unchanged ServiceCard component.
export const ServiceShowroom = ({ services }: { services: ServiceProgram[] }) => {
  const { locale } = useI18n();
  const copy = COPY[locale];
  const positions = positionsFor(services.length);
  const [active, setActive] = useState<number | null>(null);
  const [interactive, setInteractive] = useState(
    () => typeof window !== "undefined" && window.matchMedia(SHOWROOM_QUERY).matches
  );
  const [stage3d, setStage3d] = useState(false);
  const leaveTimer = useRef<number | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const hostRef = useRef<HTMLDivElement>(null);
  const cardRefs = useRef<Array<HTMLDivElement | null>>([]);
  const slotRefs = useRef<Array<HTMLLIElement | null>>([]);
  const sceneRef = useRef<ShowroomScene | null>(null);
  const featuredRef = useRef<boolean[]>([]);
  featuredRef.current = services.map((service) => Boolean(service.featured));
  // A stable identity for the cards on display (the parent may rebuild the
  // array on every render).
  const servicesKey = services.map((service) => `${service.to}:${service.featured ? 1 : 0}`).join("|");

  useEffect(() => {
    const query = window.matchMedia(SHOWROOM_QUERY);
    const update = () => setInteractive(query.matches);
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);

  useEffect(() => {
    setStage3d(interactive && supportsWebGL());
  }, [interactive]);

  useEffect(
    () => () => {
      if (leaveTimer.current !== null) window.clearTimeout(leaveTimer.current);
    },
    []
  );

  // The 3D stage: built once per mount on wide screens, placed from the cards'
  // positions, redrawn only when something changes.
  useEffect(() => {
    const host = hostRef.current;
    const root = rootRef.current;
    if (!stage3d || !host || !root) return;
    const canvas = document.createElement("canvas");
    canvas.className = styles.canvas;
    canvas.setAttribute("aria-hidden", "true");
    host.appendChild(canvas);
    let disposed = false;
    let scene: ShowroomScene | null = null;

    // Layout positions only: the slot box plus the card's offset inside it,
    // so the arrival animation (a transform) never moves the pedestals.
    const measure = () => {
      const current = sceneRef.current;
      if (!current) return;
      const box = host.getBoundingClientRect();
      const featured = featuredRef.current;
      const slots = featured
        .map((isFeatured, index) => {
          const slot = slotRefs.current[index];
          const card = cardRefs.current[index];
          if (!slot || !card) return null;
          const rect = slot.getBoundingClientRect();
          return {
            left: rect.left - box.left + card.offsetLeft,
            top: rect.top - box.top + card.offsetTop,
            width: card.offsetWidth,
            height: card.offsetHeight,
            featured: isFeatured
          };
        })
        .filter((slot): slot is NonNullable<typeof slot> => slot !== null);
      const plinths = current.layout(box.width, box.height, slots);
      // Each goal placard sits just under its pedestal.
      plinths.forEach((plinth, index) => {
        slotRefs.current[index]?.style.setProperty("--plinth", `${Math.round(plinth)}px`);
      });
    };

    const handleContextLost = (event: Event) => {
      event.preventDefault();
      setStage3d(false);
    };
    canvas.addEventListener("webglcontextlost", handleContextLost);
    const resizeObserver = new ResizeObserver(() => measure());
    resizeObserver.observe(root);

    import("./showroom_scene")
      .then(({ createShowroomScene }) => {
        if (disposed) return;
        scene = createShowroomScene(canvas);
        sceneRef.current = scene;
        measure();
        scene.arrive();
      })
      .catch(() => {
        if (!disposed) setStage3d(false);
      });

    return () => {
      disposed = true;
      resizeObserver.disconnect();
      canvas.removeEventListener("webglcontextlost", handleContextLost);
      scene?.dispose();
      if (sceneRef.current === scene) sceneRef.current = null;
      canvas.remove();
    };
  }, [stage3d, servicesKey]);

  useEffect(() => {
    sceneRef.current?.setActive(active);
  }, [active]);

  const activate = useCallback(
    (index: number) => {
      if (!interactive) return;
      if (leaveTimer.current !== null) {
        window.clearTimeout(leaveTimer.current);
        leaveTimer.current = null;
      }
      setActive(index);
    },
    [interactive]
  );

  // A short grace period stops a flicker when the pointer crosses a gap.
  const release = useCallback(() => {
    if (leaveTimer.current !== null) window.clearTimeout(leaveTimer.current);
    leaveTimer.current = window.setTimeout(() => {
      setActive(null);
      leaveTimer.current = null;
    }, 140);
  }, []);

  return (
    <div
      ref={rootRef}
      className={joinClassNames(
        styles.showroom,
        interactive && styles.interactive,
        stage3d && styles.with3d,
        styles[`count${Math.min(services.length, 3)}`]
      )}
      data-active={active !== null ? "true" : undefined}
    >
      {stage3d ? <div ref={hostRef} className={styles.canvasHost} aria-hidden="true" /> : null}

      <ul className={styles.stage}>
        {services.map((service, index) => {
          const position = positions[index];
          const route = ROUTE_BY_PATH[service.to];
          const routeCopy = route ? copy.routes[route] : null;
          const icons = route ? FACT_ICONS[route] : [];
          const isActive = active === index;
          return (
            <li
              key={service.title}
              ref={(element) => {
                slotRefs.current[index] = element;
              }}
              className={joinClassNames(
                styles.slot,
                styles[`slot_${position}`],
                isActive && styles.slotActive,
                active !== null && !isActive && styles.slotBehind
              )}
              style={{ "--i": index } as CSSProperties}
              onMouseEnter={() => activate(index)}
              onMouseLeave={release}
              onFocus={() => activate(index)}
              onBlur={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget as Node | null)) release();
              }}
            >
              <div className={styles.enter}>
                <div
                  className={styles.card}
                  ref={(element) => {
                    cardRefs.current[index] = element;
                  }}
                >
                  <ServiceCard service={service} />
                  {routeCopy ? (
                    <ul className={styles.bubbles} aria-label={`${copy.label}: ${routeCopy.goal}`}>
                      {routeCopy.facts.map((fact, factIndex) => {
                        const Icon = icons[factIndex] ?? Sparkles;
                        return (
                          <li key={fact} className={styles.bubble} style={{ "--b": factIndex } as CSSProperties}>
                            <span className={styles.bubbleIcon} aria-hidden="true">
                              <Icon strokeWidth={2} />
                            </span>
                            {fact}
                          </li>
                        );
                      })}
                    </ul>
                  ) : null}
                </div>
                {routeCopy ? (
                  <p className={styles.placard}>
                    <strong>{routeCopy.goal}</strong>
                    <span>{routeCopy.goalText}</span>
                  </p>
                ) : null}
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
};
