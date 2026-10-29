import { describe, expect, it } from "vitest";

import type { MarketplaceProgram } from "./marketplace_api";
import {
  buildProgramHighlights,
  buildServiceCategoryCards,
  describeCatalogPrice
} from "./service_catalog";

const program = (overrides: Partial<MarketplaceProgram>): MarketplaceProgram => ({
  id: "p1",
  trainer_id: null,
  name: "Program",
  description: "Description",
  type: "premium",
  status: "published",
  level: null,
  duration_weeks: 8,
  frequency_per_week: 3,
  training_type: null,
  price_minor_units: 1449,
  currency: "EUR",
  product_type: "generic",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  ...overrides
});

describe("describeCatalogPrice", () => {
  it("renders Free when every program is free", () => {
    expect(describeCatalogPrice([program({ type: "free", price_minor_units: 0 })])).toBe("Free");
  });

  it("renders the exact price for one paid program", () => {
    expect(describeCatalogPrice([program({})])).toBe("€14.49");
  });

  it("renders the cheapest price when several paid programs exist", () => {
    expect(
      describeCatalogPrice([
        program({ id: "p1", price_minor_units: 1999 }),
        program({ id: "p2", price_minor_units: 1449 })
      ])
    ).toBe("From €14.49");
  });

  it("renders Free and paid when the catalogue mixes both", () => {
    expect(
      describeCatalogPrice([
        program({ id: "p1", type: "free", price_minor_units: 0 }),
        program({ id: "p2", price_minor_units: 1449 })
      ])
    ).toBe("Free and paid");
  });
});

describe("buildServiceCategoryCards", () => {
  it("only builds cards for families that have published programs", () => {
    const cards = buildServiceCategoryCards([program({ id: "g1", type: "free", price_minor_units: 0 })], []);

    expect(cards).toHaveLength(1);
    expect(cards[0].title).toBe("Training plans");
    expect(cards[0].price).toBe("Free");
    expect(cards[0].to).toBe("/services/generic-program");
  });

  it("marks the Premium Level 1 card as featured", () => {
    const cards = buildServiceCategoryCards([], [program({ id: "p1" })]);

    expect(cards).toHaveLength(1);
    expect(cards[0].title).toBe("Premium Level 1");
    expect(cards[0].featured).toBe(true);
    expect(cards[0].price).toBe("€14.49");
  });
});

describe("buildProgramHighlights", () => {
  it("puts a Premium Level 1 package first and then generic programs", () => {
    const highlights = buildProgramHighlights(
      [program({ id: "g1", name: "Generic one", product_type: "generic" })],
      [program({ id: "p1", name: "Premium one", product_type: "premium_level_1" })]
    );

    expect(highlights.map((entry) => entry.title)).toEqual(["Premium one", "Generic one"]);
    expect(highlights[0].to).toBe("/services/premium-level-1");
    expect(highlights[1].to).toBe("/services/generic-program/g1");
  });

  it("fills the remaining slots with additional Premium packages when needed", () => {
    const highlights = buildProgramHighlights(
      [],
      [program({ id: "p1", name: "First" }), program({ id: "p2", name: "Second" })]
    );

    expect(highlights.map((entry) => entry.title)).toEqual(["First", "Second"]);
  });
});
