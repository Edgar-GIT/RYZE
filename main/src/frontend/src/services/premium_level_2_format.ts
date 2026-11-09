import type { PremiumLevel2Catalog, PremiumLevel2Question } from "./premium_level_2_api";

export type PremiumLevel2AnswerValue = string | number | boolean | string[];
export type PremiumLevel2Answers = Record<string, PremiumLevel2AnswerValue>;

export const isEmptyPremiumLevel2Answer = (value: unknown): boolean => {
  if (value === undefined || value === null) return true;
  if (typeof value === "string") return value.trim() === "";
  if (Array.isArray(value)) return value.length === 0;
  return false;
};

export const formatPremiumLevel2Answer = (
  question: PremiumLevel2Question,
  value: unknown
): string => {
  if (isEmptyPremiumLevel2Answer(value)) return "Not answered";
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (Array.isArray(value)) {
    return value
      .map((entry) => question.options?.find((option) => option.value === entry)?.label ?? String(entry))
      .join(", ");
  }
  if (question.options) {
    return question.options.find((option) => option.value === value)?.label ?? String(value);
  }
  return String(value);
};

export const questionByKey = (
  catalog: PremiumLevel2Catalog | null,
  key: string
): PremiumLevel2Question | undefined =>
  catalog?.sections.flatMap((section) => section.questions).find((question) => question.key === key);

export const humanizeAnswerKey = (key: string): string =>
  key
    .replaceAll("_", " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
