import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/services/exercise_library", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/exercise_library")>();
  return {
    ...actual,
    fetchExerciseLibrary: vi.fn(),
    fetchExerciseDetail: vi.fn()
  };
});

import {
  fetchExerciseDetail,
  type ExerciseCatalogEntry,
  type ExerciseDetail
} from "@/services/exercise_library";
import { ApiError } from "@utils/http_client";

import { ExerciseDetailPage } from "./exercise_detail_page";

const mockedFetch = vi.mocked(fetchExerciseDetail);

afterEach(cleanup);

beforeEach(() => {
  mockedFetch.mockReset();
  window.scrollTo = vi.fn();
});

const makeEntry = (overrides: Partial<ExerciseCatalogEntry> = {}): ExerciseCatalogEntry => ({
  id: "e1",
  name: "Barbell Squat",
  body_part: "upper legs",
  description: "A compound leg movement.",
  instructions: "",
  instruction_steps: ["Unrack the bar", "Squat to depth"],
  target_muscles: "quadriceps",
  primary_muscle_group: "Quads",
  secondary_muscle_groups: "Glutes",
  equipment: "Barbell",
  difficulty: "Intermediate",
  movement_category: "Compound",
  video_url: null,
  image_url: null,
  animation_url: null,
  three_d_key: null,
  media_attribution: "",
  media_available: false,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  ...overrides
});

const makeDetail = (overrides: Partial<ExerciseDetail> = {}): ExerciseDetail => ({
  exercise: makeEntry(),
  alternatives: [],
  ...overrides
});

const renderDetail = (exerciseId: string) =>
  render(
    <MemoryRouter initialEntries={[`/exercises/${exerciseId}`]}>
      <Route path="/exercises/:exerciseId" component={ExerciseDetailPage} />
    </MemoryRouter>
  );

describe("ExerciseDetailPage", () => {
  it("renders the exercise with its ordered instruction steps", async () => {
    mockedFetch.mockResolvedValue(makeDetail());

    renderDetail("e1");

    expect(await screen.findByRole("heading", { level: 1, name: "Barbell Squat" })).toBeTruthy();
    const steps = screen.getAllByRole("listitem");
    expect(steps.map((step) => step.textContent)).toEqual(["Unrack the bar", "Squat to depth"]);
    expect(screen.queryByRole("button", { name: /Open 3D view/ })).toBeNull();
  });

  it("offers the 3D demonstration only for exercises that ship one", async () => {
    mockedFetch.mockResolvedValue(makeDetail({ exercise: makeEntry({ three_d_key: "squat" }) }));

    renderDetail("e1");

    expect(await screen.findByRole("button", { name: /Open 3D view/ })).toBeTruthy();
  });

  it("renders the media attribution only while media is available", async () => {
    mockedFetch.mockResolvedValue(
      makeDetail({
        exercise: makeEntry({
          media_available: true,
          animation_url: "/media/demo.gif",
          media_attribution: "© Gym visual — https://gymvisual.com/"
        })
      })
    );

    renderDetail("e1");

    expect(await screen.findByText("© Gym visual — https://gymvisual.com/")).toBeTruthy();
  });

  it("shows the unavailable state while the backend keeps media disabled", async () => {
    mockedFetch.mockResolvedValue(makeDetail());

    renderDetail("e1");

    expect(await screen.findByText(/RYZE media placeholder/)).toBeTruthy();
  });

  it("renders the honest not-found state for an unknown exercise", async () => {
    mockedFetch.mockRejectedValue(new ApiError(404, "EXERCISE_NOT_FOUND", "Exercise not found."));

    renderDetail("missing");

    expect(await screen.findByText(/no longer available/)).toBeTruthy();
    expect(screen.queryByRole("heading", { level: 1 })).toBeNull();
  });

  it("renders a retryable error state for a failed request", async () => {
    mockedFetch.mockRejectedValueOnce(new ApiError(500, "INTERNAL_ERROR", "Internal server error."));
    mockedFetch.mockResolvedValueOnce(makeDetail());

    renderDetail("e1");

    expect(await screen.findByText(/Unable to load this exercise/)).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(await screen.findByRole("heading", { level: 1, name: "Barbell Squat" })).toBeTruthy();
  });

  it("navigates to an alternative exercise", async () => {
    mockedFetch.mockResolvedValueOnce(
      makeDetail({
        alternatives: [
          {
            id: "alt-1",
            exercise_id: "e1",
            alternative_exercise_id: "e2",
            alternative_name: "Goblet Squat"
          }
        ]
      })
    );
    mockedFetch.mockResolvedValueOnce(makeDetail({ exercise: makeEntry({ id: "e2", name: "Goblet Squat" }) }));

    renderDetail("e1");

    expect(await screen.findByText("Goblet Squat")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: /View/ }));

    await waitFor(() => expect(mockedFetch).toHaveBeenLastCalledWith("e2"));
    expect(await screen.findByRole("heading", { level: 1, name: "Goblet Squat" })).toBeTruthy();
  });
});
