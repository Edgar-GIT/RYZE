import { describe, expect, it } from "vitest";

import { parseReps, parseWeight } from "./set_row";

// Letters are an error, never a guessed weight (owner report, 2026-11-21).
describe("set inputs", () => {
  it("accepts numbers with a comma or a point", () => {
    expect(parseWeight("62,5")).toEqual({ ok: true, value: 62.5 });
    expect(parseWeight(" 60 ")).toEqual({ ok: true, value: 60 });
    expect(parseWeight("")).toEqual({ ok: true, value: null });
    expect(parseReps("10")).toEqual({ ok: true, value: 10 });
  });

  it("rejects letters, signs and out-of-range values", () => {
    for (const value of ["abc", "6o", "-5", "1e3", "60kg", "1001", "62,555"]) {
      expect(parseWeight(value).ok).toBe(false);
    }
    for (const value of ["dez", "8.5", "-1", "1000"]) {
      expect(parseReps(value).ok).toBe(false);
    }
  });
});
