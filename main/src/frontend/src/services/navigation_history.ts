import type { useHistory } from "react-router-dom";

type AppHistory = ReturnType<typeof useHistory>;
type AppLocation = AppHistory["location"];

// Knows whether "back" stays inside RYZE (owner request, 2026-11-15: a way to
// go back one page everywhere, not only the logo to the home page). It follows
// this tab's history entries; a page opened directly (a link from outside, a
// new tab) has nothing behind it, so the back buttons go to its parent page
// instead of leaving the site.

const KEY = "ryze.navigationStack";

// One history entry of this tab: the router's key and the page's path (the
// path tells the back button where the person came from).
interface Entry {
  key: string;
  path: string;
}

let stack: Entry[] = [];
let index = 0;

const entryOf = (location: AppLocation): Entry => ({ key: location.key ?? "root", path: location.pathname ?? "" });

const save = () => {
  try {
    sessionStorage.setItem(KEY, JSON.stringify({ stack, index }));
  } catch {
    // Storage blocked: the history still works for this page view.
  }
};

const isEntry = (item: unknown): item is Entry =>
  typeof item === "object" && item !== null && typeof (item as Entry).key === "string" && typeof (item as Entry).path === "string";

export const startNavigationTracking = (history: AppHistory): (() => void) => {
  try {
    const saved = JSON.parse(sessionStorage.getItem(KEY) ?? "null") as { stack?: unknown } | null;
    stack = saved && Array.isArray(saved.stack) ? saved.stack.filter(isEntry) : [];
  } catch {
    stack = [];
  }
  const current = entryOf(history.location);
  const found = stack.findIndex((entry) => entry.key === current.key);
  if (found >= 0) {
    index = found;
  } else {
    stack = [current];
    index = 0;
  }
  save();

  return history.listen((location, action) => {
    const entry = entryOf(location);
    if (action === "PUSH") {
      stack = [...stack.slice(0, index + 1), entry];
      index = stack.length - 1;
    } else if (action === "REPLACE") {
      stack[index] = entry;
    } else {
      const position = stack.findIndex((item) => item.key === entry.key);
      if (position >= 0) {
        index = position;
      } else {
        stack = [entry];
        index = 0;
      }
    }
    save();
  });
};

export const canGoBackInApp = () => index > 0;

// back is the whole button text: Portuguese contracts the preposition with
// the article ("Voltar à Área pessoal", "Voltar aos Planos de treino"), so it
// cannot be composed from the page name.
export interface ParentPage {
  path: string;
  back: { en: string; pt: string };
}

type Phrase = { en: string; pt: string };

// The back-button text for each page it can lead to, first match wins.
const PAGE_BACK: Array<[RegExp, Phrase]> = [
  [/^\/profile\/?$/, { en: "Back to Personal area", pt: "Voltar à Área pessoal" }],
  [/^\/services\/my-programs\/?$/, { en: "Back to My programs", pt: "Voltar aos Meus programas" }],
  [/^\/services\/my-programs\/[^/]+/, { en: "Back to the plan", pt: "Voltar ao plano" }],
  [/^\/services\/workout-history/, { en: "Back to History", pt: "Voltar ao Histórico" }],
  [/^\/services\/progress/, { en: "Back to Progress", pt: "Voltar ao Progresso" }],
  [/^\/services\/recovery/, { en: "Back to Recovery", pt: "Voltar à Recuperação" }],
  [/^\/services\/workout-sessions\//, { en: "Back to the workout", pt: "Voltar ao treino" }],
  [/^\/services\/premium-level-2\/academy/, { en: "Back to the Academy", pt: "Voltar à Academia" }],
  [/^\/achievements/, { en: "Back to Achievements", pt: "Voltar às Conquistas" }],
  [/^\/support/, { en: "Back to Support", pt: "Voltar ao Suporte" }],
  [/^\/account\/settings/, { en: "Back to Account", pt: "Voltar à Conta" }],
  [/^\/account\/purchases\/?$/, { en: "Back to Purchases", pt: "Voltar às Compras" }],
  [/^\/account\/purchases\//, { en: "Back to the receipt", pt: "Voltar ao comprovativo" }],
  [/^\/trainer\/apply/, { en: "Back to the application", pt: "Voltar à candidatura" }],
  [/^\/trainer/, { en: "Back to the trainer area", pt: "Voltar à área de treinador" }],
  [/^\/admin/, { en: "Back to administration", pt: "Voltar à administração" }],
  [/^\/services\/generic-program\/?$/, { en: "Back to Training plans", pt: "Voltar aos Planos de treino" }],
  [/^\/services\/generic-program\/[^/]+/, { en: "Back to the plan", pt: "Voltar ao plano" }],
  [/^\/services\/premium-level-1/, { en: "Back to Premium Level 1", pt: "Voltar ao Premium Nível 1" }],
  [/^\/services\/premium-level-2/, { en: "Back to Premium Level 2", pt: "Voltar ao Premium Nível 2" }],
  [/^\/services\/?$/, { en: "Back to Services", pt: "Voltar aos Serviços" }],
  [/^\/services\//, { en: "Back to the service", pt: "Voltar ao serviço" }],
  [/^\/coaches\//, { en: "Back to the coach", pt: "Voltar ao treinador" }],
  [/^\/u\//, { en: "Back to the profile", pt: "Voltar ao perfil" }],
  [/^\/exercises\/?$/, { en: "Back to Exercises", pt: "Voltar aos Exercícios" }],
  [/^\/exercises\//, { en: "Back to the exercise", pt: "Voltar ao exercício" }],
  [/^\/contact/, { en: "Back to Contact", pt: "Voltar ao Contacto" }],
  [/^\/our-vision/, { en: "Back to Our Vision", pt: "Voltar à Nossa visão" }],
  [/^\/feedback/, { en: "Back to Feedback", pt: "Voltar ao Feedback" }],
  [/^\/feed/, { en: "Back to the Feed", pt: "Voltar ao Feed" }],
  [/^\/?$/, { en: "Back to Home", pt: "Voltar ao Início" }]
];

export const pageBack = (path: string): Phrase =>
  PAGE_BACK.find(([pattern]) => pattern.test(path))?.[1] ?? { en: "Back", pt: "Voltar" };

/** The personal-area pages (rendered in the member shell). */
export const MEMBER_PATH =
  /^\/(profile|achievements|support|account\/(settings|purchases(\/[^/]+\/receipt)?)|services\/(my-programs(\/[^/]+)?|premium-level-2\/academy|workout-history|progress|recovery)|trainer\/apply)\/?$/;

type Area = "member" | "trainer" | "admin" | "public";

export const areaOf = (pathname: string): Area => {
  if (MEMBER_PATH.test(pathname) || pathname.startsWith("/services/workout-sessions/")) return "member";
  if (pathname.startsWith("/trainer")) return "trainer";
  if (pathname.startsWith("/admin")) return "admin";
  return "public";
};

// Sign-in and similar steps are never where "back" leads.
const NOT_A_BACK_TARGET = /^\/(login|register|forgot-password|reset-password|verify-email|app|admin\/login)\/?$/;

// The page "above" a personal-area page.
export const memberParent = (pathname: string): ParentPage => {
  let path = "/profile";
  if (/^\/services\/my-programs\/[^/]+/.test(pathname)) path = "/services/my-programs";
  else if (/^\/account\/purchases\/[^/]+\/receipt/.test(pathname)) path = "/account/purchases";
  return { path, back: pageBack(path) };
};

// The page "above" a public page, with its name.
export const publicParent = (pathname: string): ParentPage => {
  const path = publicParentPath(pathname);
  return { path, back: pageBack(path) };
};

const parentPathOf = (pathname: string, area: Area) => (area === "member" ? memberParent(pathname).path : publicParentPath(pathname));

const ancestorsOf = (pathname: string, area: Area): string[] => {
  const chain: string[] = [];
  let current = pathname;
  for (let step = 0; step < 8; step += 1) {
    const parent = parentPathOf(current, area);
    if (parent === current || chain.includes(parent)) break;
    chain.push(parent);
    current = parent;
  }
  return chain;
};

export interface BackTarget extends ParentPage {
  /** true: the previous page of this tab, reached with history back. */
  viaHistory: boolean;
}

// resolveBack decides where the back button goes (owner requests of
// 2026-11-20 and 2026-11-21):
// - came from another area (the personal area to the training plans, a plan
//   to the exercise library): back to exactly that page;
// - the previous page is above this one (plans list → plan): back to it;
// - moved between pages of the same area (Progress → History): up to the
//   parent page, so returning never takes one click per page visited;
// - nothing in this tab before (link from outside): the parent page.
export const resolveBack = (pathname: string): BackTarget => {
  const area = areaOf(pathname);
  const previous = index > 0 ? stack[index - 1]?.path : "";
  if (previous && previous !== pathname && !NOT_A_BACK_TARGET.test(previous)) {
    if (areaOf(previous) !== area || ancestorsOf(pathname, area).includes(previous)) {
      return { path: previous, back: pageBack(previous), viaHistory: true };
    }
  }
  const path = parentPathOf(pathname, area);
  return { path, back: pageBack(path), viaHistory: false };
};

// The page "above" a public page, used when there is nothing to go back to.
export const publicParentPath = (pathname: string): string => {
  // Getting a free plan returns to that plan's overview.
  const getPlan = /^(\/services\/generic-program\/[^/]+)\/get\/?$/.exec(pathname);
  if (getPlan) return getPlan[1];
  if (pathname.startsWith("/services/generic-program/")) return "/services/generic-program";
  if (pathname.startsWith("/coaches/")) return "/services/generic-program";
  if (pathname.startsWith("/exercises/")) return "/exercises";
  if (pathname.startsWith("/services/")) return "/services";
  return "/";
};
