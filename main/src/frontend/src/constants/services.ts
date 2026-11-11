import rccImageOne from "@resources/img/program_cards/rcc1.png";
import rccImageTwo from "@resources/img/program_cards/rcc2.png";
import rccImageThree from "@resources/img/program_cards/rcc3.png";

export interface ServiceProgram {
  title: string;
  price: string;
  ctaLabel: string;
  to: string;
  imageSrc: string;
  featured?: boolean;
  // A label above the card that replaces "Featured" (Premium Level 2: the most
  // personalized route, in contrast with the most chosen Premium Level 1).
  tag?: { label: string; tone: "personalized" };
}

const PERSONALIZED_TAG = { label: "Personalized", tone: "personalized" } as const;

export const SERVICE_CATEGORY_PRESENTATION = {
  generic: {
    title: "Training plans",
    ctaLabel: "Browse training plans",
    to: "/services/generic-program",
    imageSrc: rccImageOne
  },
  premium: {
    title: "Premium Level 1",
    ctaLabel: "View Premium Level 1",
    to: "/services/premium-level-1",
    imageSrc: rccImageTwo,
    featured: true
  },
  premiumLevel2: {
    title: "Premium Level 2",
    ctaLabel: "View Premium Level 2",
    to: "/services/premium-level-2",
    imageSrc: rccImageTwo,
    featured: true,
    tag: PERSONALIZED_TAG
  }
} as const;

// The original Services showcase (restored from before the catalogue became
// API-driven). Shown only while no offer is published, so the page always
// presents RYZE's three routes; published offers replace it with live data.
export const SERVICE_SHOWCASE: ServiceProgram[] = [
  {
    title: "Generic Program",
    price: "FREE",
    ctaLabel: "Start now for FREE",
    to: "/services/generic-program",
    imageSrc: rccImageOne
  },
  {
    title: "Premium Level 1",
    price: "14,49€",
    ctaLabel: "Get my program",
    to: "/services/premium-level-1",
    imageSrc: rccImageTwo,
    featured: true
  },
  {
    title: "Premium Level 2",
    price: "19,49€",
    ctaLabel: "Start now with a Coach",
    to: "/services/premium-level-2",
    imageSrc: rccImageThree,
    tag: PERSONALIZED_TAG
  }
];
