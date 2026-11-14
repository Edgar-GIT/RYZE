import { describe, expect, it } from "vitest";

import { formatDuration, parseDuration } from "./duration";

describe("formatDuration", () => {
  it("keeps seconds under a minute", () => {
    expect(formatDuration(45)).toBe("45 s");
    expect(formatDuration(0)).toBe("0 s");
  });

  it("switches to minutes and seconds from one minute", () => {
    expect(formatDuration(61)).toBe("1 min 1 s");
    expect(formatDuration(90)).toBe("1 min 30 s");
    expect(formatDuration(120)).toBe("2 min");
    expect(formatDuration(3725)).toBe("1 h 2 min 5 s");
  });

  it("returns an empty string without a value", () => {
    expect(formatDuration(null)).toBe("");
    expect(formatDuration(undefined)).toBe("");
  });
});

describe("parseDuration", () => {
  it("reads plain seconds and clock notation", () => {
    expect(parseDuration("90")).toBe(90);
    expect(parseDuration("1:30")).toBe(90);
    expect(parseDuration("0:45")).toBe(45);
  });

  it("reads units", () => {
    expect(parseDuration("1 min 1 s")).toBe(61);
    expect(parseDuration("2 min")).toBe(120);
    expect(parseDuration("45s")).toBe(45);
    expect(parseDuration("1m30")).toBe(90);
  });

  it("rejects empty or unreadable text", () => {
    expect(parseDuration("")).toBeNull();
    expect(parseDuration("soon")).toBeNull();
  });
});
