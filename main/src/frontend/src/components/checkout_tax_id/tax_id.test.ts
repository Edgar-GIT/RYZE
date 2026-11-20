import { describe, expect, it } from "vitest";

import { isValidOptionalTaxId, normalizeTaxId } from "./tax_id";

describe("isValidOptionalTaxId", () => {
  it("accepts an empty field and valid NIFs as typed", () => {
    for (const value of ["", "   ", "123456789", " 123 456 789 ", "PT123456789", "pt 501 964 843", "999999990"]) {
      expect(isValidOptionalTaxId(value)).toBe(true);
    }
    expect(normalizeTaxId("pt 501 964 843")).toBe("501964843");
  });

  it("refuses anything else", () => {
    for (const value of ["12345678", "1234567890", "123456780", "012345678", "12345678a", "ES123456789"]) {
      expect(isValidOptionalTaxId(value)).toBe(false);
    }
  });
});
