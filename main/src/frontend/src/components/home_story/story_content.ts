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

import { type Copy, useI18n } from "@/i18n/i18n";

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

export type ScreenKey = "training" | "nutrition" | "synced" | "platform";

const SCREEN_SOURCES: Record<ScreenKey, string> = {
  training: trainingPhone,
  nutrition: nutritionPhone,
  synced: syncedPhone,
  platform: programAccessDesktop
};

// Pieces of the phone screens lifted out beside the devices, ordered top to
// bottom like the screen elements they come from (the 3D scene joins each card
// to its element). Decorative: the chapter copy lists the same features.
export const CALLOUTS = {
  train: [calloutWeek, calloutSet, calloutRest],
  fuel: [calloutTarget, calloutProtein, calloutMeal],
  sync: calloutSync
} as const;

// Language-free structure of the story: ids, icons and screens.
const STRUCTURE: { id: StoryChapterId; icons?: LucideIcon[]; screen?: ScreenKey }[] = [
  { id: "identity" },
  { id: "train", icons: [CalendarDays, ListChecks, Timer, History], screen: "training" },
  { id: "fuel", icons: [Target, UtensilsCrossed, PackageCheck, UserCheck], screen: "nutrition" },
  { id: "platform", icons: [Search, Library, LayoutGrid], screen: "platform" },
  { id: "start" }
];

const ENTRY_ICONS = [Dumbbell, Crown, UserCheck] as const;

interface ChapterText {
  navLabel: string;
  label: string;
  headline: HeadlineSegment[];
  lead: string;
  points?: string[];
}

interface StoryText {
  chapters: Record<StoryChapterId, ChapterText>;
  screens: Record<ScreenKey, string>;
  sampleNote: string;
  entryPoints: [string, string, string];
  waysToTrain: string;
  startFree: string;
  explorePlans: string;
  discover: string;
  steps: string;
  scrollCue: string;
}

const en: StoryText = {
  chapters: {
    identity: {
      navLabel: "RYZE",
      label: "Training & nutrition platform",
      headline: [
        { text: "Train " },
        { text: "anywhere.", highlight: true },
        { text: " Progress " },
        { text: "everywhere.", highlight: true }
      ],
      lead: "Free training plans, premium programs and personalized coaching in one platform that goes wherever you train. Follow every session and track your progress from your phone or your computer."
    },
    train: {
      navLabel: "Train",
      label: "Train",
      headline: [{ text: "Every session, " }, { text: "already planned.", highlight: true }],
      lead: "Programmes are organised into weeks and training days. Each exercise arrives with its sets, target reps, load, rest and tempo — and you log every set as you go.",
      points: [
        "Weeks and training days, laid out in order",
        "Sets, reps, load, rest and tempo for every exercise",
        "A live session timer with set-by-set logging",
        "Workout history and progress after every session"
      ]
    },
    fuel: {
      navLabel: "Fuel",
      label: "Fuel",
      headline: [{ text: "Nutrition that " }, { text: "fits the plan.", highlight: true }],
      lead: "Premium packages pair your training with a nutrition plan: a daily calorie target, protein, carbohydrate and fat, and every meal broken down into foods and portions.",
      points: [
        "Daily calorie and macro targets",
        "Meals and snacks, with foods and portions",
        "Premium Level 1: delivered right after purchase",
        "Premium Level 2: prepared for you by a RYZE trainer"
      ]
    },
    platform: {
      navLabel: "Evolve",
      label: "Evolve",
      headline: [{ text: "One platform, " }, { text: "every screen.", highlight: true }],
      lead: "Browse the catalogue, check out and find everything you own in My Programs — then follow it on your phone at the gym or on your laptop at home.",
      points: [
        "Search and filter the training-plan catalogue",
        "Exercise demonstrations in the Exercise Library",
        "My Programs, workout history and progress in one place"
      ]
    },
    start: {
      navLabel: "Start",
      label: "Get started",
      headline: [{ text: "Your next level " }, { text: "starts here.", highlight: true }],
      lead: "Start with a free training plan, step up to a Premium package with nutrition, or get a programme prepared for you by a trainer."
    }
  },
  screens: {
    training:
      "RYZE training screen on a phone: today's upper-strength session in week 4 of a sample programme, with the week's training days, four exercises with sets, reps and load, rest and tempo, and the last result for bench press.",
    nutrition:
      "RYZE nutrition screen on a phone: a sample daily target of 1800 kcal with protein, carbohydrate and fat, and four meals broken down into foods and grams.",
    synced: "RYZE today screen on a phone: day 3, upper strength, two of four sessions done this week and a start session button.",
    platform:
      "RYZE programme page on a laptop: a sample Premium programme with its weeks and training days, today's exercises with sets, reps, load, rest and tempo, the day's nutrition targets and the week's completed sessions."
  },
  sampleNote: "Illustrative RYZE screens with sample data.",
  entryPoints: ["Free programs", "Premium plans", "Personalized coaching"],
  waysToTrain: "Ways to train with RYZE",
  startFree: "Start training free",
  explorePlans: "Explore plans",
  discover: "Discover RYZE",
  steps: "Story steps",
  scrollCue: "Scroll to explore"
};

const pt: StoryText = {
  chapters: {
    identity: {
      navLabel: "RYZE",
      label: "Plataforma de treino e nutrição",
      headline: [
        { text: "Treina " },
        { text: "em qualquer lado.", highlight: true },
        { text: " Evolui " },
        { text: "em todo o lado.", highlight: true }
      ],
      lead: "Planos de treino gratuitos, programas premium e acompanhamento personalizado numa só plataforma, que vai contigo para onde treinares. Segue cada treino e acompanha a tua evolução no telemóvel ou no computador."
    },
    train: {
      navLabel: "Treinar",
      label: "Treinar",
      headline: [{ text: "Cada treino, " }, { text: "já planeado.", highlight: true }],
      lead: "Os programas estão organizados em semanas e dias de treino. Cada exercício chega com séries, repetições-alvo, carga, descanso e cadência — e registas cada série à medida que treinas.",
      points: [
        "Semanas e dias de treino, por ordem",
        "Séries, repetições, carga, descanso e cadência em cada exercício",
        "Cronómetro do treino com registo série a série",
        "Histórico e progresso depois de cada treino"
      ]
    },
    fuel: {
      navLabel: "Nutrição",
      label: "Nutrição",
      headline: [{ text: "Nutrição que " }, { text: "acompanha o plano.", highlight: true }],
      lead: "Os pacotes Premium juntam ao treino um plano de nutrição: meta diária de calorias, proteína, hidratos e gordura, e cada refeição dividida em alimentos e porções.",
      points: [
        "Metas diárias de calorias e macronutrientes",
        "Refeições e snacks, com alimentos e porções",
        "Premium Nível 1: disponível logo após a compra",
        "Premium Nível 2: preparado para ti por um treinador RYZE"
      ]
    },
    platform: {
      navLabel: "Evoluir",
      label: "Evoluir",
      headline: [{ text: "Uma plataforma, " }, { text: "todos os ecrãs.", highlight: true }],
      lead: "Explora o catálogo, compra e encontra tudo o que é teu em Os meus programas — depois segue no telemóvel no ginásio ou no portátil em casa.",
      points: [
        "Pesquisa e filtra o catálogo de planos de treino",
        "Demonstrações de exercícios na Biblioteca de exercícios",
        "Os meus programas, histórico e progresso num só sítio"
      ]
    },
    start: {
      navLabel: "Começar",
      label: "Começar",
      headline: [{ text: "O teu próximo nível " }, { text: "começa aqui.", highlight: true }],
      lead: "Começa com um plano de treino gratuito, sobe para um pacote Premium com nutrição, ou recebe um programa preparado para ti por um treinador."
    }
  },
  screens: {
    training:
      "Ecrã de treino RYZE num telemóvel: o treino de força de tronco de hoje, na semana 4 de um programa de exemplo, com os dias de treino da semana, quatro exercícios com séries, repetições e carga, descanso e cadência, e o último resultado do supino.",
    nutrition:
      "Ecrã de nutrição RYZE num telemóvel: uma meta diária de exemplo de 1800 kcal com proteína, hidratos e gordura, e quatro refeições divididas em alimentos e gramas.",
    synced: "Ecrã de hoje RYZE num telemóvel: dia 3, força de tronco, dois de quatro treinos feitos esta semana e um botão para iniciar o treino.",
    platform:
      "Página de programa RYZE num portátil: um programa Premium de exemplo com as suas semanas e dias de treino, os exercícios de hoje com séries, repetições, carga, descanso e cadência, as metas de nutrição do dia e os treinos concluídos da semana."
  },
  sampleNote: "Ecrãs RYZE ilustrativos, com dados de exemplo.",
  entryPoints: ["Programas gratuitos", "Planos premium", "Acompanhamento personalizado"],
  waysToTrain: "Formas de treinar com a RYZE",
  startFree: "Começar a treinar grátis",
  explorePlans: "Ver planos",
  discover: "Descobre a RYZE",
  steps: "Passos da história",
  scrollCue: "Desliza para explorar"
};

export const STORY_COPY: Copy<StoryText> = { en, pt };

const buildChapters = (text: StoryText): StoryChapter[] =>
  STRUCTURE.map(({ id, icons, screen }) => {
    const chapter = text.chapters[id];
    return {
      id,
      tone: "navy",
      navLabel: chapter.navLabel,
      label: chapter.label,
      headline: chapter.headline,
      lead: chapter.lead,
      points: icons && chapter.points ? chapter.points.map((point, index) => ({ icon: icons[index], text: point })) : undefined,
      screen: screen ? { src: SCREEN_SOURCES[screen], alt: text.screens[screen] } : undefined
    };
  });

const CHAPTERS_BY_LOCALE = { en: buildChapters(en), pt: buildChapters(pt) };

// The language-free order of the chapters (the 3D scroll logic uses the ids).
export const STORY_CHAPTER_IDS: StoryChapterId[] = STRUCTURE.map((chapter) => chapter.id);

// SCREENS keeps the screen images for the 3D scene (no text).
export const SCREENS = SCREEN_SOURCES;

// useStory returns the chapters and labels in the current language.
export const useStory = () => {
  const { locale } = useI18n();
  const text = STORY_COPY[locale];
  const chapters = CHAPTERS_BY_LOCALE[locale];
  const entryPoints = ENTRY_ICONS.map((icon, index) => ({ icon, label: text.entryPoints[index] }));
  return { text, chapters, entryPoints };
};
