import logoCardOne from "@resources/img/plan_cards/logo_card1.png";
import logoCardTwo from "@resources/img/plan_cards/logo_card2.png";
import logoCardThree from "@resources/img/plan_cards/logo_card3.png";

export interface ServicePlan {
  title: string;
  description: string;
  price: string;
  ctaLabel: string;
  to: string;
  logoSrc: string;
  logoAlt: string;
  badge: string;
  features: string[];
  featured?: boolean;
}

export const SERVICE_PLANS: ServicePlan[] = [
  {
    title: "Generic Plan",
    description:
      "Ready-to-use training packs. No questionnaire, no waiting — start training today.",
    price: "FREE",
    ctaLabel: "Start now for FREE",
    to: "/services/generic-plan",
    logoSrc: logoCardOne,
    logoAlt: "RYZE Generic Plan logo",
    badge: "Base plan",
    features: ["Ready-made workouts", "Exercise guidance", "Instant Access"]
  },
  {
    title: "Premium Level 1",
    description:
      "A guided plan with a personalized training pack and automatically assigned nutrition plan.",
    price: "14,49€",
    ctaLabel: "Get my plan",
    to: "/services/premium-level-1",
    logoSrc: logoCardTwo,
    logoAlt: "RYZE Premium Level 1 logo",
    badge: "Train & Nutrition",
    features: ["Plan built around your goal", "Nutrition matched to your training", "Immediate delivery"],
    featured: true
  },
  {
    title: "Premium Level 2",
    description:
      "Your training and nutrition plan prepared and reviewed by a real coach. Exclusive to your needs.",
    price: "19,49€",
    ctaLabel: "Start now with a Coach",
    to: "/services/premium-level-2",
    logoSrc: logoCardThree,
    logoAlt: "RYZE Premium Level 2 logo",
    badge: "Coach reviewed",
    features: ["Trainer reviewed", "Every detail accounted for", "Ongoing adjustments"]
  }
];
