export interface FilterGroup {
  id: string;
  title: string;
  options: string[];
}

export interface MarketplaceRow {
  id: string;
  title: string;
  description?: string;
  programs: string[];
}

export const MARKETPLACE_FILTER_GROUPS: FilterGroup[] = [
  {
    id: "training-type",
    title: "Training Type",
    options: ["Hypertrophy", "Strength", "HYROX", "CrossFit", "Fat Loss", "At Home"]
  },
  {
    id: "frequency",
    title: "Training Frequency",
    options: [
      "1 day / week",
      "2 days / week",
      "3 days / week",
      "4 days / week",
      "5 days / week",
      "6 days / week",
      "7 days / week"
    ]
  },
  {
    id: "level",
    title: "Level",
    options: ["Beginner", "Intermediate", "Advanced"]
  },
  {
    id: "duration",
    title: "Plan Duration",
    options: ["1–2 weeks", "3–4 weeks", "5–8 weeks", "9–12 weeks"]
  }
];

export const MARKETPLACE_ROWS: MarketplaceRow[] = [
  {
    id: "most-popular",
    title: "Most Popular",
    description: "The programs most people are running right now.",
    programs: [
      "Upper Body Strength",
      "4-Day Muscle Builder",
      "Full Body Foundation",
      "Beginner Strength",
      "Strength & Size",
      "Home Full Body"
    ]
  },
  {
    id: "trending",
    title: "Trending This Week",
    description: "Programs gaining the most momentum right now.",
    programs: [
      "Push Hypertrophy",
      "Pull Hypertrophy",
      "Chest & Triceps",
      "Back & Biceps",
      "Leg Day Builder",
      "Core & Conditioning"
    ]
  },
  {
    id: "new-noteworthy",
    title: "New & Noteworthy",
    description: "Recently published programs worth checking out.",
    programs: [
      "5-Day Hypertrophy",
      "Dumbbell Fundamentals",
      "Kettlebell Strength",
      "Functional Strength",
      "Bodyweight Basics",
      "Mobility & Core"
    ]
  },
  {
    id: "strength",
    title: "Build Strength",
    programs: [
      "Lower Body Power",
      "Powerlifting Prep",
      "Functional Strength",
      "Strength & Size",
      "Beginner Strength",
      "Upper Body Strength"
    ]
  },
  {
    id: "hypertrophy",
    title: "Build Muscle",
    programs: [
      "Push Hypertrophy",
      "Pull Hypertrophy",
      "4-Day Muscle Builder",
      "5-Day Hypertrophy",
      "Chest & Triceps",
      "Leg Day Builder"
    ]
  },
  {
    id: "fat-loss",
    title: "Fat Loss",
    programs: [
      "Fat Loss Conditioning",
      "Metabolic Conditioning",
      "At-Home HIIT",
      "Sprint Progression",
      "Bodyweight Basics",
      "3-Day Full Body"
    ]
  },
  {
    id: "at-home",
    title: "Train at Home",
    programs: [
      "Home Full Body",
      "Bodyweight Basics",
      "At-Home HIIT",
      "Dumbbell Fundamentals",
      "Mobility & Core",
      "Core & Conditioning"
    ]
  },
  {
    id: "performance",
    title: "Performance & Conditioning",
    programs: [
      "HYROX Performance",
      "Speed & Agility",
      "Endurance Builder",
      "Sprint Progression",
      "Functional Strength",
      "Lower Body Power"
    ]
  }
];