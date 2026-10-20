import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
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

let intersectionCallback: IntersectionObserverCallback | null = null;

class MockIntersectionObserver {
  root = null;
  rootMargin = "";
  thresholds = [];

  constructor(callback: IntersectionObserverCallback) {
    intersectionCallback = callback;
  }

  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() {
    return [];
  }
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

beforeEach(() => {
  mockedFetch.mockReset();
  intersectionCallback = null;
  vi.stubGlobal("IntersectionObserver", MockIntersectionObserver);
});

const triggerLoadMore = () => {
  act(() => {
    intersectionCallback?.(
      [{ isIntersecting: true } as IntersectionObserverEntry],
      {} as IntersectionObserver
    );
  });
};

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
    limit: 10,
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
      10
    );
  });

  it("renders available media with attribution and falls back when it fails", async () => {
    mockedFetch.mockResolvedValue(
      makeResult([
        makeEntry("e1", "Barbell Squat", {
          media_available: true,
          image_url: "/dev-exercise-media/images/0001.jpg",
          animation_url: "/dev-exercise-media/videos/0001.gif",
          media_attribution: "© Gym visual — https://gymvisual.com/"
        })
      ])
    );

    renderPage();

    const image = await screen.findByRole("img", { name: "Barbell Squat demonstration" });
    expect(image.getAttribute("src")).toBe("/dev-exercise-media/images/0001.jpg");
    expect(screen.getByText("© Gym visual — https://gymvisual.com/")).toBeTruthy();

    fireEvent.error(image);

    expect(screen.getByText("Media unavailable")).toBeTruthy();
    expect(screen.queryByRole("img", { name: "Barbell Squat demonstration" })).toBeNull();
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
        10
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
        10
      )
    );
  });

  it("loads the next batch when the sentinel enters the viewport", async () => {
    mockedFetch.mockResolvedValueOnce(
      makeResult([makeEntry("e1", "Barbell Squat")], { total: 50, total_pages: 5 })
    );
    mockedFetch.mockResolvedValueOnce(
      makeResult([makeEntry("e2", "Dumbbell Curl")], { page: 2, total: 50, total_pages: 5 })
    );

    renderPage();

    expect(await screen.findByText("Barbell Squat")).toBeTruthy();
    expect(screen.getByText("Scroll to load more exercises.")).toBeTruthy();

    triggerLoadMore();

    expect(await screen.findByText("Dumbbell Curl")).toBeTruthy();
    expect(screen.getByText("Barbell Squat")).toBeTruthy();
    expect(mockedFetch).toHaveBeenLastCalledWith(expect.anything(), 2, 10);
  });

  it("shows the end state when the last batch is loaded", async () => {
    mockedFetch.mockResolvedValue(makeResult([makeEntry("e1", "Barbell Squat")], { total: 1, total_pages: 1 }));

    renderPage();

    expect(await screen.findByText("You have reached the end of the library.")).toBeTruthy();
  });
});
