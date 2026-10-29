import rccImageOne from "@resources/img/program_cards/rcc1.png";
import rccImageTwo from "@resources/img/program_cards/rcc2.png";

export interface ServiceProgram {
  title: string;
  price: string;
  ctaLabel: string;
  to: string;
  imageSrc: string;
  featured?: boolean;
}

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
  }
} as const;
