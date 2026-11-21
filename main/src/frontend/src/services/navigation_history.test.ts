import { beforeEach, describe, expect, it } from "vitest";

import { canGoBackInApp, publicParentPath, resolveBack, startNavigationTracking } from "./navigation_history";

type Listener = (location: { key?: string }, action: "PUSH" | "POP" | "REPLACE") => void;

// A minimal stand-in for the router history: only what the tracker reads.
const fakeHistory = (initialKey?: string) => {
  let listener: Listener = () => undefined;
  const history = {
    location: { key: initialKey },
    listen: (next: Listener) => {
      listener = next;
      return () => undefined;
    }
  };
  return {
    history: history as unknown as Parameters<typeof startNavigationTracking>[0],
    emit: (key: string | undefined, action: "PUSH" | "POP" | "REPLACE") => listener({ key }, action)
  };
};

describe("navigation history", () => {
  beforeEach(() => sessionStorage.clear());

  it("has nothing to go back to on a page opened directly", () => {
    const { history } = fakeHistory();
    startNavigationTracking(history);
    expect(canGoBackInApp()).toBe(false);
  });

  it("follows pushes, back and forward", () => {
    const { history, emit } = fakeHistory();
    startNavigationTracking(history);
    emit("a", "PUSH");
    emit("b", "PUSH");
    expect(canGoBackInApp()).toBe(true);
    emit("a", "POP");
    emit(undefined, "POP");
    expect(canGoBackInApp()).toBe(false);
    emit("a", "POP");
    expect(canGoBackInApp()).toBe(true);
  });

  it("keeps the position after a reload of the same tab", () => {
    const first = fakeHistory();
    startNavigationTracking(first.history);
    first.emit("a", "PUSH");
    first.emit("b", "PUSH");
    const reloaded = fakeHistory("b");
    startNavigationTracking(reloaded.history);
    expect(canGoBackInApp()).toBe(true);
  });

  it("knows the parent of public pages", () => {
    expect(publicParentPath("/services/generic-program/123")).toBe("/services/generic-program");
    expect(publicParentPath("/services/generic-program/123/get")).toBe("/services/generic-program/123");
    expect(publicParentPath("/services/premium-level-1")).toBe("/services");
    expect(publicParentPath("/exercises/abc")).toBe("/exercises");
    expect(publicParentPath("/coaches/42")).toBe("/services/generic-program");
    expect(publicParentPath("/contact")).toBe("/");
  });
});

// The router history with paths, for the back-button decisions.
const pathHistory = (initialPath: string) => {
  let listener: (location: { key?: string; pathname?: string }, action: "PUSH" | "POP" | "REPLACE") => void = () => undefined;
  let counter = 0;
  const history = {
    location: { key: "start", pathname: initialPath },
    listen: (next: typeof listener) => {
      listener = next;
      return () => undefined;
    }
  };
  return {
    history: history as unknown as Parameters<typeof startNavigationTracking>[0],
    push: (pathname: string) => {
      counter += 1;
      listener({ key: `k${counter}`, pathname }, "PUSH");
    }
  };
};

describe("back target", () => {
  beforeEach(() => sessionStorage.clear());

  it("returns to the personal area page the person came from", () => {
    const { history, push } = pathHistory("/profile");
    startNavigationTracking(history);
    push("/services/generic-program");
    expect(resolveBack("/services/generic-program")).toMatchObject({ path: "/profile", viaHistory: true });
    expect(resolveBack("/services/generic-program").back.pt).toBe("Voltar à Área pessoal");
  });

  it("goes up instead of through every sibling page of an area", () => {
    const { history, push } = pathHistory("/profile");
    startNavigationTracking(history);
    push("/services/progress");
    push("/services/workout-history");
    push("/achievements");
    expect(resolveBack("/achievements")).toMatchObject({ path: "/profile", viaHistory: false });
  });

  it("steps back to the page above when that is where the person was", () => {
    const { history, push } = pathHistory("/services");
    startNavigationTracking(history);
    push("/services/generic-program");
    push("/services/generic-program/p1");
    expect(resolveBack("/services/generic-program/p1")).toMatchObject({ path: "/services/generic-program", viaHistory: true });
  });

  it("uses the parent page when the tab has nothing before or came from the login", () => {
    const { history, push } = pathHistory("/login");
    startNavigationTracking(history);
    expect(resolveBack("/login")).toMatchObject({ viaHistory: false });
    push("/services/generic-program/p1");
    expect(resolveBack("/services/generic-program/p1")).toMatchObject({ path: "/services/generic-program", viaHistory: false });
  });
});
