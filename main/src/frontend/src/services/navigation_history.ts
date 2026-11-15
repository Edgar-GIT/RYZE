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

// The page "above" a public page, used when there is nothing to go back to.
export const publicParentPath = (pathname: string): string => {
  if (pathname.startsWith("/services/generic-program/")) return "/services/generic-program";
  if (pathname.startsWith("/coaches/")) return "/services/generic-program";
  if (pathname.startsWith("/exercises/")) return "/exercises";
  if (pathname.startsWith("/services/")) return "/services";
  return "/";
};
