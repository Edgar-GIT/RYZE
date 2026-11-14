import { ApiError, apiGet, apiPatch } from "@utils/http_client";

// The member's sex and whether they prefer to see the plans made for it first
// (owner request, 2026-11-14). Signed-in members keep it on their account;
// visitors keep it in this browser only.

export type Sex = "female" | "male";

export interface PlanPreferences {
  sex: Sex | null;
  // null: not answered yet.
  plan_audience_filter: boolean | null;
}

const LOCAL_KEY = "ryze.planPreferences";

export const readLocalPlanPreferences = (): PlanPreferences | null => {
  try {
    const raw = localStorage.getItem(LOCAL_KEY);
    if (!raw) return null;
    const value = JSON.parse(raw) as Partial<PlanPreferences>;
    const sex = value.sex === "female" || value.sex === "male" ? value.sex : null;
    const filter = typeof value.plan_audience_filter === "boolean" ? value.plan_audience_filter : null;
    return { sex, plan_audience_filter: filter };
  } catch {
    return null;
  }
};

export const writeLocalPlanPreferences = (preferences: PlanPreferences) => {
  try {
    localStorage.setItem(LOCAL_KEY, JSON.stringify(preferences));
  } catch {
    // Private mode or blocked storage: the answer lasts for this visit only.
  }
};

// The signed-in member's stored preferences, or null for a visitor.
export const fetchAccountPlanPreferences = async (): Promise<PlanPreferences | null> => {
  try {
    const me = await apiGet<{ sex?: Sex | null; plan_audience_filter?: boolean | null }>("/me");
    return { sex: me.sex ?? null, plan_audience_filter: me.plan_audience_filter ?? null };
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) return null;
    throw error;
  }
};

export const saveAccountPlanPreferences = (input: { sex?: Sex; filter_by_sex?: boolean }) =>
  apiPatch<PlanPreferences>("/me/plan-preferences", input);
