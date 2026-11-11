import {
  CalendarDays,
  Crown,
  Dumbbell,
  History,
  LayoutGrid,
  Library,
  ListChecks,
  PackageCheck,
  Search,
  Target,
  Timer,
  UserCheck,
  UtensilsCrossed
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

import nutritionPhone from "@resources/img/showcase/phone_nutrition.webp";
import programAccessDesktop from "@resources/img/showcase/laptop_programme.webp";
import trainingPhone from "@resources/img/showcase/phone_training.webp";
import syncedPhone from "@resources/img/showcase/phone_synced.webp";
import calloutMeal from "@resources/img/showcase/callout_meal.webp";
import calloutProtein from "@resources/img/showcase/callout_protein.webp";
import calloutRest from "@resources/img/showcase/callout_rest.webp";
import calloutSet from "@resources/img/showcase/callout_set.webp";
import calloutSync from "@resources/img/showcase/callout_sync.webp";
import calloutTarget from "@resources/img/showcase/callout_target.webp";
import calloutWeek from "@resources/img/showcase/callout_week.webp";

// Every chapter sits on the shared navy page.
export type StoryTone = "navy";
export type StoryChapterId = "identity" | "train" | "fuel" | "platform" | "start";

export interface HeadlineSegment {
  text: string;
  highlight?: boolean;
}

export interface StoryPoint {
  icon: LucideIcon;
  text: string;
}

export interface StoryScreen {
  src: string;
  alt: string;
}

export interface StoryChapter {
  id: StoryChapterId;
  navLabel: string;
  tone: StoryTone;
  label: string;
  headline: HeadlineSegment[];
  lead: string;
  points?: StoryPoint[];
  screen?: StoryScreen;
}

export const SCREENS = {
  training: {
    src: trainingPhone,
    alt: "RYZE training screen on a phone: today's upper-strength session in week 4 of a sample programme, with the week's training days, four exercises with sets, reps and load, rest and tempo, and the last result for bench press.",
  },
  nutrition: {
    src: nutritionPhone,
    alt: "RYZE nutrition screen on a phone: a sample daily target of 1800 kcal with protein, carbohydrate and fat, and four meals broken down into foods and grams.",
  },
  synced: {
    src: syncedPhone,
    alt: "RYZE today screen on a phone: day 3, upper strength, two of four sessions done this week and a start session button."
  },
  platform: {
    src: programAccessDesktop,
    alt: "RYZE programme page on a laptop: a sample Premium programme with its weeks and training days, today's exercises with sets, reps, load, rest and tempo, the day's nutrition targets and the week's completed sessions.",
  }
} as const satisfies Record<string, StoryScreen>;

// Pieces of the phone screens lifted out beside the devices, one per feature
// point of their chapter. Decorative: the chapter copy lists the same features.
export const CALLOUTS = {
  train: [calloutSet, calloutRest, calloutWeek],
  fuel: [calloutProtein, calloutMeal, calloutTarget],
  sync: calloutSync
} as const;

export const SAMPLE_DATA_NOTE = "Illustrative RYZE screens with sample data.";

// The three entry points named in the original hero (shown as labels).
export const ENTRY_POINTS = [
  { icon: Dumbbell, label: "Free programs" },
  { icon: Crown, label: "Premium plans" },
  { icon: UserCheck, label: "Personalized coaching" }
] as const;

export const STORY_CHAPTERS: StoryChapter[] = [
  {
    id: "identity",
    navLabel: "RYZE",
    tone: "navy",
    label: "Training & nutrition platform",
    headline: [
      { text: "Train " },
      { text: "anywhere.", highlight: true },
      { text: " Progress " },
      { text: "everywhere.", highlight: true }
    ],
    lead: "Free training plans, premium programs and personalized coaching in one platform that goes wherever you train. Follow every session and track your progress from your phone or your computer."
  },
  {
    id: "train",
    navLabel: "Train",
    tone: "navy",
    label: "Train",
    headline: [{ text: "Every session, " }, { text: "already planned.", highlight: true }],
    lead: "Programmes are organised into weeks and training days. Each exercise arrives with its sets, target reps, load, rest and tempo — and you log every set as you go.",
    points: [
      { icon: CalendarDays, text: "Weeks and training days, laid out in order" },
      { icon: ListChecks, text: "Sets, reps, load, rest and tempo for every exercise" },
      { icon: Timer, text: "A live session timer with set-by-set logging" },
      { icon: History, text: "Workout history and progress after every session" }
    ],
    screen: SCREENS.training
  },
  {
    id: "fuel",
    navLabel: "Fuel",
    tone: "navy",
    label: "Fuel",
    headline: [{ text: "Nutrition that " }, { text: "fits the plan.", highlight: true }],
    lead: "Premium packages pair your training with a nutrition plan: a daily calorie target, protein, carbohydrate and fat, and every meal broken down into foods and portions.",
    points: [
      { icon: Target, text: "Daily calorie and macro targets" },
      { icon: UtensilsCrossed, text: "Meals and snacks, with foods and portions" },
      { icon: PackageCheck, text: "Premium Level 1: delivered right after purchase" },
      { icon: UserCheck, text: "Premium Level 2: prepared for you by a RYZE trainer" }
    ],
    screen: SCREENS.nutrition
  },
  {
    id: "platform",
    navLabel: "Evolve",
    tone: "navy",
    label: "Evolve",
    headline: [{ text: "One platform, " }, { text: "every screen.", highlight: true }],
    lead: "Browse the catalogue, check out and find everything you own in My Programs — then follow it on your phone at the gym or on your laptop at home.",
    points: [
      { icon: Search, text: "Search and filter the training-plan catalogue" },
      { icon: Library, text: "Exercise demonstrations in the Exercise Library" },
      { icon: LayoutGrid, text: "My Programs, workout history and progress in one place" }
    ],
    screen: SCREENS.platform
  },
  {
    id: "start",
    navLabel: "Start",
    tone: "navy",
    label: "Get started",
    headline: [{ text: "Your next level " }, { text: "starts here.", highlight: true }],
    lead: "Start with a free training plan, step up to a Premium package with nutrition, or get a programme prepared for you by a trainer."
  }
];
