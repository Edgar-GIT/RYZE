import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
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
  fetchExerciseLibrary,
  type ExerciseCatalogEntry,
  type ExerciseLibraryResult
} from "@/services/exercise_library";

import { ExerciseLibraryPage } from "./exercise_library_page";

const mockedFetch = vi.mocked(fetchExerciseLibrary);

afterEach(cleanup);

beforeEach(() => {
  mockedFetch.mockReset();
});

const makeEntry = (id: string, name: string, overrides: Partial<ExerciseCatalogEntry> = {}): ExerciseCatalogEntry => ({
  id,
  name,
  body_part: "upper legs",
  description: "A compound leg movement.",
  instructions: "",
  instruction_steps: ["Step one", "Step two"],
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

const makeResult = (exercises: ExerciseCatalogEntry[], overrides: Partial<ExerciseLibraryResult["pagination"]> = {}): ExerciseLibraryResult => ({
  exercises,
  pagination: {
    page: 1,
    limit: 24,
    total: exercises.length,
    total_pages: 1,
    ...overrides
  }
});

const renderPage = () =>
  render(
    <MemoryRouter>
      <ExerciseLibraryPage />
    </MemoryRouter>
  );

describe("ExerciseLibraryPage", () => {
  it("renders the catalogue cards returned by the backend", async () => {
    mockedFetch.mockResolvedValue(makeResult([makeEntry("e1", "Barbell Squat"), makeEntry("e2", "Dumbbell Curl")]));

    renderPage();

    expect(await screen.findByText("Barbell Squat")).toBeTruthy();
    expect(screen.getByText("Dumbbell Curl")).toBeTruthy();
    expect(screen.getByText(/Showing 1–2 of 2 exercises/)).toBeTruthy();
    expect(mockedFetch).toHaveBeenCalledWith(
      expect.objectContaining({ query: "", bodyPart: "", muscle: "", equipment: "", difficulty: "", category: "" }),
      1,
      24
    );
  });

  it("renders the honest empty state when nothing matches", async () => {
    mockedFetch.mockResolvedValue(makeResult([]));

    renderPage();

    expect(await screen.findByText("No exercises match the current search or filters.")).toBeTruthy();
  });

  it("renders the error state and retries the request", async () => {
    mockedFetch.mockRejectedValueOnce(new Error("Network down"));
    mockedFetch.mockResolvedValueOnce(makeResult([makeEntry("e1", "Barbell Squat")]));

    renderPage();

    expect(await screen.findByText("Network down")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(await screen.findByText("Barbell Squat")).toBeTruthy();
    expect(mockedFetch).toHaveBeenCalledTimes(2);
  });

  it("sends the body part filter and resets to the first page", async () => {
    mockedFetch.mockResolvedValue(makeResult([]));

    renderPage();
    await screen.findByText("No exercises match the current search or filters.");

    fireEvent.change(screen.getByLabelText("Filter by body part"), { target: { value: "upper legs" } });

    await waitFor(() =>
      expect(mockedFetch).toHaveBeenLastCalledWith(
        expect.objectContaining({ bodyPart: "upper legs" }),
        1,
        24
      )
    );
  });

  it("debounces the search input and uses the search request", async () => {
    mockedFetch.mockResolvedValue(makeResult([]));

    renderPage();
    await screen.findByText("No exercises match the current search or filters.");

    fireEvent.change(screen.getByLabelText("Search exercises"), { target: { value: "squat" } });

    await waitFor(() =>
      expect(mockedFetch).toHaveBeenLastCalledWith(
        expect.objectContaining({ query: "squat" }),
        1,
        24
      )
    );
  });

  it("paginates through the catalogue", async () => {
    mockedFetch.mockResolvedValueOnce(makeResult([makeEntry("e1", "Barbell Squat")], { total: 50, total_pages: 3 }));
    mockedFetch.mockResolvedValueOnce(makeResult([makeEntry("e2", "Dumbbell Curl")], { page: 2, total: 50, total_pages: 3 }));

    renderPage();

    expect(await screen.findByText("Barbell Squat")).toBeTruthy();
    expect(screen.getByText("Page 1 of 3")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: /Next/ }));

    expect(await screen.findByText("Dumbbell Curl")).toBeTruthy();
    expect(mockedFetch).toHaveBeenLastCalledWith(expect.anything(), 2, 24);
  });
});
