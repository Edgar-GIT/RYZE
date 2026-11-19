import { describe, expect, it } from "vitest";

import { isValidTestEmail } from "./test_mode_email";

describe("isValidTestEmail", () => {
  it("accepts a plain deliverable-looking address", () => {
    expect(isValidTestEmail("qa.tester@example.com")).toBe(true);
    expect(isValidTestEmail("  qa+tag@sub.example.com  ")).toBe(true);
  });

  it("rejects empty, decorated and malformed addresses", () => {
    for (const value of ["", "not-an-email", "Name <qa@example.com>", "qa@example.com\r\nBcc: x@example.com"]) {
      expect(isValidTestEmail(value)).toBe(false);
    }
  });
});
