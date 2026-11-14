import { SERVICE_CATEGORY_PRESENTATION, type ServiceProgram } from "@/constants/services";
import {
  formatMarketplacePrice,
  type MarketplaceProgram
} from "@/services/marketplace_api";

export type ProgramHighlightCategory = "generic" | "premium" | "premiumLevel2";

const isFree = (program: MarketplaceProgram): boolean =>
  program.type === "free" || program.price_minor_units === 0;

export const describeCatalogPrice = (programs: MarketplaceProgram[], locale: "en" | "pt" = "en"): string => {
  if (programs.length === 0) {
    return "";
  }
  const pt = locale === "pt";
  const paid = programs.filter((program) => !isFree(program));
  if (paid.length === 0) {
    return pt ? "Grátis" : "Free";
  }
  if (paid.length < programs.length) {
    return pt ? "Gratuitos e pagos" : "Free and paid";
  }
  const cheapest = paid.reduce((lowest, program) =>
    program.price_minor_units < lowest.price_minor_units ? program : lowest
  );
  const label = formatMarketplacePrice(cheapest.price_minor_units, cheapest.currency, cheapest.type, locale);
  if (paid.length === 1) return label;
  return pt ? `Desde ${label}` : `From ${label}`;
};

export const buildServiceCategoryCards = (
  generic: MarketplaceProgram[],
  premium: MarketplaceProgram[],
  premiumLevel2: MarketplaceProgram[] = []
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
  if (premiumLevel2.length > 0) {
    cards.push({
      ...SERVICE_CATEGORY_PRESENTATION.premiumLevel2,
      price: describeCatalogPrice(premiumLevel2)
    });
  }
  return cards;
};
