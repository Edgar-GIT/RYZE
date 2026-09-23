export interface FilterGroup {
  id: string;
  title: string;
  options: string[];
}

export interface MarketplaceRow {
  id: string;
  title: string;
  description?: string;
  // Discovery rows have no real popularity metrics yet. Each source maps to a
  // deterministic, honest fallback ordering of the published catalog:
  //   catalog → the order the API returns (newest published first)
  //   recent  → most recently updated programs
  //   newest  → newest published programs
  source?: "catalog" | "recent" | "newest";
  // Category rows group published programs by their training_type value. A
  // program belongs to the row when its training_type is in this list.
  trainingTypes?: string[];
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
    source: "catalog"
  },
  {
    id: "trending",
    title: "Trending This Week",
    description: "Programs gaining the most momentum right now.",
    source: "recent"
  },
  {
    id: "new-noteworthy",
    title: "New & Noteworthy",
    description: "Recently published programs worth checking out.",
    source: "newest"
  },
  {
    id: "strength",
    title: "Build Strength",
    trainingTypes: ["Strength"]
  },
  {
    id: "hypertrophy",
    title: "Build Muscle",
    trainingTypes: ["Hypertrophy"]
  },
  {
    id: "fat-loss",
    title: "Fat Loss",
    trainingTypes: ["Fat Loss"]
  },
  {
    id: "at-home",
    title: "Train at Home",
    trainingTypes: ["At Home"]
  },
  {
    id: "performance",
    title: "Performance & Conditioning",
    trainingTypes: ["HYROX", "CrossFit"]
  }
];

// Plan Duration filter options, each mapped to its inclusive week range.
export const DURATION_RANGES: Record<string, [number, number]> = {
  "1–2 weeks": [1, 2],
  "3–4 weeks": [3, 4],
  "5–8 weeks": [5, 8],
  "9–12 weeks": [9, 12]
};

// Training Frequency filter options, each mapped to its exact days-per-week
// value (parsed from the leading number in the label).
export const FREQUENCY_DAYS: string[] = [
  "1",
  "2",
  "3",
  "4",
  "5",
  "6",
  "7"
];