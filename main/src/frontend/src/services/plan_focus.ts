import { apiGet, apiPut } from "@utils/http_client";

import { fetchFreeActivations } from "@/services/free_programs_api";
import { fetchEntitlements, type ProgramAccessDetail, type ProgramAccessWorkout } from "@/services/purchases_api";
import type { WorkoutSession } from "@/services/workout_sessions_api";

/**
 * The plan in focus (owner request, 2026-11-22). With several active plans the
 * home and "Começar treino" use the plan of the member's latest action:
 * adding a plan, training one or choosing one with "Trocar plano". A workout
 * in progress always wins. The choice is kept on the account (server checks
 * that the member can open the plan).
 */

export interface LibraryPlan {
  id: string;
  name: string;
  frequency: number | null;
  /** When the member got the plan (purchase or free activation). */
  addedAt: string;
  premium: boolean;
}

export interface PlanFocusChoice {
  program_id: string;
  chosen_at: string;
}

export const fetchPlanFocus = (): Promise<PlanFocusChoice | null> => apiGet<PlanFocusChoice | null>("/me/plan-focus");

export const savePlanFocus = (programId: string): Promise<PlanFocusChoice> =>
  apiPut<PlanFocusChoice>("/me/plan-focus", { program_id: programId });

// loadLibraryPlans lists the member's plans: purchases first-class, then free
// plans (a program owned both ways is listed once).
export const loadLibraryPlans = async (): Promise<LibraryPlan[]> => {
  const [entitlements, free] = await Promise.all([fetchEntitlements().catch(() => []), fetchFreeActivations().catch(() => [])]);
  const plans = new Map<string, LibraryPlan>();
  for (const entry of entitlements) {
    plans.set(entry.program_id, {
      id: entry.program_id,
      name: entry.program.name,
      frequency: entry.program.frequency_per_week ?? null,
      addedAt: entry.created_at,
      premium: entry.program.type !== "free"
    });
  }
  for (const entry of free) {
    if (!plans.has(entry.program.id)) {
      plans.set(entry.program.id, {
        id: entry.program.id,
        name: entry.program.name,
        frequency: entry.program.frequency_per_week,
        addedAt: entry.activated_at,
        premium: false
      });
    }
  }
  return [...plans.values()];
};

const time = (value: string | null | undefined): number => {
  const parsed = value ? Date.parse(value) : NaN;
  return Number.isFinite(parsed) ? parsed : 0;
};

// lastActionAt is when the member last added, trained or chose the plan.
export const lastActionAt = (plan: LibraryPlan, sessions: WorkoutSession[], choice: PlanFocusChoice | null): number => {
  let latest = time(plan.addedAt);
  for (const session of sessions) {
    if (session.program_id === plan.id) latest = Math.max(latest, time(session.completed_at ?? session.started_at));
  }
  if (choice?.program_id === plan.id) latest = Math.max(latest, time(choice.chosen_at));
  return latest;
};

// pickFocus returns the plan in focus: the workout in progress, else the plan
// of the latest action (ties: Premium first, then the newest plan).
export const pickFocus = (
  plans: LibraryPlan[],
  sessions: WorkoutSession[],
  active: WorkoutSession | null,
  choice: PlanFocusChoice | null
): string | null => {
  if (active) return active.program_id;
  let best: { id: string; at: number; premium: boolean } | null = null;
  for (const plan of plans) {
    const at = lastActionAt(plan, sessions, choice);
    if (!best || at > best.at || (at === best.at && plan.premium && !best.premium)) {
      best = { id: plan.id, at, premium: plan.premium };
    }
  }
  return best?.id ?? null;
};

// sortByLastAction orders plans from the most recent action.
export const sortByLastAction = (plans: LibraryPlan[], sessions: WorkoutSession[], choice: PlanFocusChoice | null): LibraryPlan[] =>
  [...plans].sort((a, b) => lastActionAt(b, sessions, choice) - lastActionAt(a, sessions, choice));

export interface NextWorkout {
  week: number;
  day: number;
  workout: ProgramAccessWorkout;
}

// nextWorkoutOf is the first day (in week order) not completed yet; a
// completed day never repeats.
export const nextWorkoutOf = (plan: ProgramAccessDetail, history: WorkoutSession[]): NextWorkout | null => {
  const done = new Set(
    history.filter((session) => session.status === "completed" && session.program_id === plan.id).map((session) => session.program_workout_id)
  );
  for (const week of [...plan.weeks].sort((a, b) => a.week_number - b.week_number)) {
    for (const workout of [...week.workouts].sort((a, b) => a.position - b.position)) {
      if (workout.exercises.length > 0 && !done.has(workout.id)) {
        return { week: week.week_number, day: workout.position, workout };
      }
    }
  }
  return null;
};

// hasTrained reports whether the member completed any workout of the plan.
export const hasTrained = (planId: string, sessions: WorkoutSession[]): boolean =>
  sessions.some((session) => session.program_id === planId && session.status === "completed");
