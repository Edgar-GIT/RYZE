import type { useHistory } from "react-router-dom";

type AppHistory = ReturnType<typeof useHistory>;
type AppLocation = AppHistory["location"];

// Knows whether "back" stays inside RYZE (owner request, 2026-11-15: a way to
// go back one page everywhere, not only the logo to the home page). It follows
// this tab's history entries; a page opened directly (a link from outside, a
// new tab) has nothing behind it, so the back buttons go to its parent page
// instead of leaving the site.

const KEY = "ryze.navigationStack";
let stack: string[] = [];
let index = 0;

const keyOf = (location: AppLocation) => location.key ?? "root";

const save = () => {
  try {
    sessionStorage.setItem(KEY, JSON.stringify({ stack, index }));
  } catch {
    // Storage blocked: the history still works for this page view.
  }
};

export const startNavigationTracking = (history: AppHistory): (() => void) => {
  try {
    const saved = JSON.parse(sessionStorage.getItem(KEY) ?? "null") as { stack?: unknown } | null;
    if (saved && Array.isArray(saved.stack)) stack = saved.stack.filter((item): item is string => typeof item === "string");
  } catch {
    stack = [];
  }
  const current = keyOf(history.location);
  const found = stack.indexOf(current);
  if (found >= 0) {
    index = found;
  } else {
    stack = [current];
    index = 0;
  }
  save();

  return history.listen((location, action) => {
    const key = keyOf(location);
    if (action === "PUSH") {
      stack = [...stack.slice(0, index + 1), key];
      index = stack.length - 1;
    } else if (action === "REPLACE") {
      stack[index] = key;
    } else {
      const position = stack.indexOf(key);
      if (position >= 0) {
        index = position;
      } else {
        stack = [key];
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

// The page "above" a personal-area page, with its name: the back button goes
// straight there (one click from any depth).
export const memberParent = (pathname: string): ParentPage => {
  if (/^\/services\/my-programs\/[^/]+/.test(pathname)) {
    return { path: "/services/my-programs", back: { en: "Back to My programs", pt: "Voltar aos Meus programas" } };
  }
  if (/^\/account\/purchases\/[^/]+\/receipt/.test(pathname)) {
    return { path: "/account/purchases", back: { en: "Back to Purchases", pt: "Voltar às Compras" } };
  }
  return { path: "/profile", back: { en: "Back to Personal area", pt: "Voltar à Área pessoal" } };
};

const PUBLIC_PARENT_BACK: Record<string, { en: string; pt: string }> = {
  "/services/generic-program": { en: "Back to Training plans", pt: "Voltar aos Planos de treino" },
  "/exercises": { en: "Back to Exercises", pt: "Voltar aos Exercícios" },
  "/services": { en: "Back to Services", pt: "Voltar aos Serviços" },
  "/": { en: "Back to Home", pt: "Voltar ao Início" }
};

// The page "above" a public page, with its name.
export const publicParent = (pathname: string): ParentPage => {
  const path = publicParentPath(pathname);
  if (/^\/services\/generic-program\/[^/]+$/.test(path)) {
    return { path, back: { en: "Back to the plan", pt: "Voltar ao plano" } };
  }
  return { path, back: PUBLIC_PARENT_BACK[path] ?? PUBLIC_PARENT_BACK["/"] };
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
