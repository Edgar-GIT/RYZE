import { SERVICE_CATEGORY_PRESENTATION, type ServiceProgram } from "@/constants/services";
import {
  formatMarketplacePrice,
  type MarketplaceProgram
} from "@/services/marketplace_api";

export type ProgramHighlightCategory = "generic" | "premium";

export interface ProgramHighlight {
  id: string;
  category: ProgramHighlightCategory;
  title: string;
  description: string;
  ctaLabel: string;
  to: string;
}

const isFree = (program: MarketplaceProgram): boolean =>
  program.type === "free" || program.price_minor_units === 0;

export const describeCatalogPrice = (programs: MarketplaceProgram[]): string => {
  if (programs.length === 0) {
    return "";
  }
  const paid = programs.filter((program) => !isFree(program));
  if (paid.length === 0) {
    return "Free";
  }
  if (paid.length < programs.length) {
    return "Free and paid";
  }
  const cheapest = paid.reduce((lowest, program) =>
    program.price_minor_units < lowest.price_minor_units ? program : lowest
  );
  const label = formatMarketplacePrice(cheapest.price_minor_units, cheapest.currency, cheapest.type);
  return paid.length === 1 ? label : `From ${label}`;
};

export const buildServiceCategoryCards = (
  generic: MarketplaceProgram[],
  premium: MarketplaceProgram[]
): ServiceProgram[] => {
  const cards: ServiceProgram[] = [];
  if (generic.length > 0) {
    cards.push({
      ...SERVICE_CATEGORY_PRESENTATION.generic,
      price: describeCatalogPrice(generic)
    });
  }
  if (premium.length > 0) {
    cards.push({
      ...SERVICE_CATEGORY_PRESENTATION.premium,
      price: describeCatalogPrice(premium)
    });
  }
  return cards;
};

const premiumHighlight = (program: MarketplaceProgram): ProgramHighlight => ({
  id: program.id,
  category: "premium",
  title: program.name,
  description: program.description,
  ctaLabel: "View package",
  to: "/services/premium-level-1"
});

const genericHighlight = (program: MarketplaceProgram): ProgramHighlight => ({
  id: program.id,
  category: "generic",
  title: program.name,
  description: program.description,
  ctaLabel: "Explore",
  to: `/services/generic-program/${program.id}`
});

export const buildProgramHighlights = (
  generic: MarketplaceProgram[],
  premium: MarketplaceProgram[],
  limit = 3
): ProgramHighlight[] => {
  const featured = premium.slice(0, 1).map(premiumHighlight);
  const remaining = limit - featured.length;
  const highlights = [...featured, ...generic.slice(0, Math.max(remaining, 0)).map(genericHighlight)];
  if (highlights.length < limit) {
    highlights.push(
      ...premium.slice(1, 1 + (limit - highlights.length)).map(premiumHighlight)
    );
  }
  return highlights;
};
