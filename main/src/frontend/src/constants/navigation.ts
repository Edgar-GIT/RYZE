import type { TranslationKey } from "@/i18n/i18n";

export interface NavigationItem {
  labelKey: TranslationKey;
  // Shorter label for the desktop bar, where every link sits on one line.
  shortKey?: TranslationKey;
  to: string;
}

export const PUBLIC_NAVIGATION_ITEMS: NavigationItem[] = [
  { labelKey: "publicNav.services", to: "/services" },
  { labelKey: "publicNav.exercises", shortKey: "publicNav.exercisesShort", to: "/exercises" },
  { labelKey: "publicNav.contact", to: "/contact" },
  { labelKey: "publicNav.feedback", to: "/feedback" },
  { labelKey: "publicNav.vision", shortKey: "publicNav.visionShort", to: "/our-vision" },
  { labelKey: "publicNav.feed", to: "/feed" }
];
