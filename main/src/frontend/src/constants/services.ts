export type ServiceIcon = "training" | "nutrition" | "personalized";

export interface ServicePlan {
  title: string;
  description: string;
  ctaLabel: string;
  to: string;
  icon: ServiceIcon;
  features: string[];
}

export const SERVICE_PLANS: ServicePlan[] = [
  {
    title: "Generic Plan",
    description:
      "Structured training packs for users who want a clear starting point without personalization.",
    ctaLabel: "Explore Generic",
    to: "/services/generic-plan",
    icon: "training",
    features: ["Pre-built workouts", "Exercise guidance", "Fast start"]
  },
  {
    title: "Premium Level 1",
    description:
      "A guided plan with a personalized training pack and automatically assigned nutrition plan.",
    ctaLabel: "View Premium 1",
    to: "/services/premium-level-1",
    icon: "nutrition",
    features: ["Questionnaire-based", "Nutrition included", "Immediate delivery"]
  },
  {
    title: "Premium Level 2",
    description:
      "Fully personalized training and nutrition with individual preparation and trainer review.",
    ctaLabel: "View Premium 2",
    to: "/services/premium-level-2",
    icon: "personalized",
    features: ["Trainer reviewed", "Deep personalization", "Highest detail"]
  }
];
