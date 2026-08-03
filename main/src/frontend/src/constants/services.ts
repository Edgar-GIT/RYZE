import testImageOne from "@resources/img/testing/test1.jpg";
import testImageTwo from "@resources/img/testing/test2.jpg";
import testImageThree from "@resources/img/testing/teste3.jpg";

export interface ServicePlan {
  title: string;
  description: string;
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
      "Structured training packs for users who want a clear starting point without personalization.",
    ctaLabel: "FREE",
    to: "/services/generic-plan",
    imageSrc: testImageOne,
    imageAlt: "Dumbbells prepared for a structured workout",
    badge: "Base plan",
    features: ["Pre-built workouts", "Exercise guidance", "Fast start"]
  },
  {
    title: "Premium Level 1",
    description:
      "A guided plan with a personalized training pack and automatically assigned nutrition plan.",
    ctaLabel: "14,49€",
    to: "/services/premium-level-1",
    imageSrc: testImageTwo,
    imageAlt: "Athlete training under modern gym lighting",
    badge: "Train + Nutrition",
    features: ["Questionnaire-based", "Nutrition included", "Immediate delivery"]
  },
  {
    title: "Premium Level 2",
    description:
      "Fully personalized training and nutrition with individual preparation and trainer review.",
    ctaLabel: "19,99€",
    to: "/services/premium-level-2",
    imageSrc: testImageThree,
    imageAlt: "Athlete holding dumbbells in a gym",
    badge: "Coach reviewed full plan",
    features: ["Trainer reviewed", "Deep personalization", "Highest detail"]
  }
];
