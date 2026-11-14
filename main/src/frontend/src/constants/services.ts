export interface ServiceProgram {
  title: string;
  price: string;
  ctaLabel: string;
  to: string;
  featured?: boolean;
  // A label above the card that replaces "Featured" (Premium Level 2: the most
  // personalized route, in contrast with the most chosen Premium Level 1; the
  // generic training plans: free, owner request 2026-11-14).
  tag?: { label: string; tone: "personalized" | "free" };
}

const PERSONALIZED_TAG = { label: "Personalized", tone: "personalized" } as const;
const FREE_TAG = { label: "Free", tone: "free" } as const;

export const SERVICE_CATEGORY_PRESENTATION = {
  generic: {
    title: "Training plans",
    ctaLabel: "Browse training plans",
    to: "/services/generic-program",
    tag: FREE_TAG
  },
  premium: {
    title: "Premium Level 1",
    ctaLabel: "View Premium Level 1",
    to: "/services/premium-level-1",
    featured: true
  },
  premiumLevel2: {
    title: "Premium Level 2",
    ctaLabel: "View Premium Level 2",
    to: "/services/premium-level-2",
    featured: true,
    tag: PERSONALIZED_TAG
  }
} as const;

