import planGenericOne from "@resources/img/plan_cards/plan_generic1.png";
import planGenericTwo from "@resources/img/plan_cards/plan_generic2.png";
import planGenericThree from "@resources/img/plan_cards/plan_generic3.png";

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
    imageSrc: planGenericOne,
    imageAlt: "RYZE Generic Plan card background",
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
    imageSrc: planGenericTwo,
    imageAlt: "RYZE Premium Level 1 card background",
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
    imageSrc: planGenericThree,
    imageAlt: "RYZE Premium Level 2 card background",
    badge: "Coach reviewed",
    features: ["Trainer reviewed", "Every detail accounted for", "Ongoing adjustments"]
  }
];
