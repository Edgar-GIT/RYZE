import { beforeEach, describe, expect, it } from "vitest";

import { canGoBackInApp, publicParentPath, startNavigationTracking } from "./navigation_history";

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
    expect(publicParentPath("/services/premium-level-1")).toBe("/services");
    expect(publicParentPath("/exercises/abc")).toBe("/exercises");
    expect(publicParentPath("/coaches/42")).toBe("/services/generic-program");
    expect(publicParentPath("/contact")).toBe("/");
  });
});
