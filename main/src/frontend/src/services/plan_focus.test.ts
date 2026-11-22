import { describe, expect, it } from "vitest";

import { pickFocus, sortByLastAction, type LibraryPlan } from "./plan_focus";
import type { WorkoutSession } from "./workout_sessions_api";

const plan = (id: string, addedAt: string, premium = false): LibraryPlan => ({ id, name: id, frequency: 3, addedAt, premium });
const session = (programId: string, at: string, status = "completed") =>
  ({ id: `${programId}-${at}`, program_id: programId, status, started_at: at, completed_at: status === "completed" ? at : null }) as unknown as WorkoutSession;

// The plan in focus is the one of the latest action: added, trained or chosen
// (owner request, 2026-11-22); a workout in progress always wins.
describe("plan focus", () => {
  const plans = [plan("a", "2026-11-01T10:00:00Z"), plan("b", "2026-11-10T10:00:00Z")];

  it("puts a newly added plan in focus", () => {
    expect(pickFocus(plans, [session("a", "2026-11-05T10:00:00Z")], null, null)).toBe("b");
  });

  it("follows the plan trained most recently", () => {
    expect(pickFocus(plans, [session("a", "2026-11-12T10:00:00Z")], null, null)).toBe("a");
  });

  it("follows a later choice, until another plan is trained", () => {
    const choice = { program_id: "a", chosen_at: "2026-11-13T10:00:00Z" };
    expect(pickFocus(plans, [session("b", "2026-11-12T10:00:00Z")], null, choice)).toBe("a");
    expect(pickFocus(plans, [session("b", "2026-11-14T10:00:00Z")], null, choice)).toBe("b");
  });

  it("lets a workout in progress win and has no focus without plans", () => {
    expect(pickFocus(plans, [], session("a", "2026-11-01T11:00:00Z", "in_progress"), null)).toBe("a");
    expect(pickFocus([], [], null, null)).toBeNull();
  });

  it("orders the other plans from the latest action", () => {
    expect(sortByLastAction(plans, [session("a", "2026-11-12T10:00:00Z")], null).map((entry) => entry.id)).toEqual(["a", "b"]);
  });
});
