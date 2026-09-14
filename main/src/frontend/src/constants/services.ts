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
}

export const SERVICE_PROGRAMS: ServiceProgram[] = [
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
    imageSrc: rccImageThree
  }
];