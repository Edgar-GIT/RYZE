import planGeneric from "@resources/img/plan_cards/card_generic.png";
import planPremiumOne from "@resources/img/plan_cards/card_premium1.png";
import planPremiumTwo from "@resources/img/plan_cards/card_premium2.png";

export interface ServicePlan {
  title: string;
  description: string;
  price: string;
  ctaLabel: string;
  to: string;
  imageSrc: string;
  imageAlt: string;
  badge: string;
  features: string[];
}

export const SERVICE_PLANS: ServicePlan[] = [
  {
    title: "Generic Plan",
    description:
      "Ready-to-use training packs. No questionnaire, no waiting — start training today.",
    price: "FREE",
    ctaLabel: "Start now for FREE",
    to: "/services/generic-plan",
    imageSrc: planGeneric,
    imageAlt: "RYZE basic badge",
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
    imageSrc: planPremiumOne,
    imageAlt: "RYZE premium level 1 badge",
    badge: "Train & Nutrition",
    features: ["Plan built around your goal", "Nutrition matched to your training", "Immediate delivery"]
  },
  {
    title: "Premium Level 2",
    description:
      "Your training and nutrition plan prepared and reviewed by a real coach. Exclusive to your needs.",
    price: "19,49€",
    ctaLabel: "Start now with a Coach",
    to: "/services/premium-level-2",
    imageSrc: planPremiumTwo,
    imageAlt: "RYZE premium level 2 badge",
    badge: "Coach reviewed full plan",
    features: ["Trainer reviewed", "Every detail accounted for", "Ongoing adjustments"]
  }
];
