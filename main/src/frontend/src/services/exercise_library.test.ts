import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@utils/http_client", () => ({
  apiGet: vi.fn()
}));

import { apiGet } from "@utils/http_client";

import {
  countActiveLibraryFilters,
  exerciseBodyPartLabel,
  exerciseMediaView,
  exerciseThumbnailView,
  fetchExerciseDetail,
  fetchExerciseLibrary,
  type ExerciseCatalogEntry
} from "./exercise_library";

const mockedGet = vi.mocked(apiGet);

beforeEach(() => {
  mockedGet.mockReset();
});

const makeEntry = (overrides: Partial<ExerciseCatalogEntry> = {}): ExerciseCatalogEntry => ({
  id: "exercise-1",
  name: "Barbell Squat",
  body_part: "upper legs",
  description: "A compound leg movement.",
  instructions: "Stand tall.",
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

describe("fetchExerciseLibrary", () => {
  it("browses the catalogue with pagination when no query is present", async () => {
    mockedGet.mockResolvedValue({ exercises: [], pagination: { page: 1, limit: 24, total: 0, total_pages: 0 } });

    await fetchExerciseLibrary({}, 1, 24);

    expect(mockedGet).toHaveBeenCalledWith("/exercises?page=1&limit=24");
  });

  it("routes a free-text query to the search endpoint", async () => {
    mockedGet.mockResolvedValue({ exercises: [], pagination: { page: 2, limit: 24, total: 0, total_pages: 0 } });

    await fetchExerciseLibrary({ query: " barbell squat " }, 2, 24);

    expect(mockedGet).toHaveBeenCalledWith("/exercises/search?q=barbell%20squat&page=2&limit=24");
  });

  it("appends every active filter including the body part", async () => {
    mockedGet.mockResolvedValue({ exercises: [], pagination: { page: 1, limit: 24, total: 0, total_pages: 0 } });

    await fetchExerciseLibrary(
      {
        bodyPart: "upper legs",
        muscle: "Quads",
        equipment: "Barbell",
        difficulty: "Intermediate",
        category: "Compound"
      },
      1,
      24
    );

    expect(mockedGet).toHaveBeenCalledWith(
      "/exercises?page=1&limit=24&body_part=upper%20legs&muscle=Quads&equipment=Barbell&difficulty=Intermediate&category=Compound"
    );
  });

  it("keeps the filters on the search endpoint", async () => {
    mockedGet.mockResolvedValue({ exercises: [], pagination: { page: 1, limit: 24, total: 0, total_pages: 0 } });

    await fetchExerciseLibrary({ query: "curl", muscle: "Biceps" }, 1, 24);

    expect(mockedGet).toHaveBeenCalledWith("/exercises/search?q=curl&page=1&limit=24&muscle=Biceps");
  });
});

describe("fetchExerciseDetail", () => {
  it("encodes the exercise id in the path", async () => {
    mockedGet.mockResolvedValue({ exercise: makeEntry(), alternatives: [] });

    await fetchExerciseDetail("id/with slash");

    expect(mockedGet).toHaveBeenCalledWith("/exercises/id%2Fwith%20slash");
  });
});

describe("countActiveLibraryFilters", () => {
  it("counts the body part filter as well", () => {
    expect(countActiveLibraryFilters({ bodyPart: "chest" })).toBe(1);
  });

  it("ignores empty and whitespace-only values", () => {
    expect(countActiveLibraryFilters({ bodyPart: "  ", muscle: "", difficulty: "Beginner" })).toBe(1);
  });
});

describe("exerciseThumbnailView", () => {
  it("uses only the static image, never the animated GIF", () => {
    const view = exerciseThumbnailView(
      makeEntry({
        media_available: true,
        animation_url: "/media/exercises/videos/0001-demo.gif",
        image_url: "/media/exercises/images/0001-still.jpg",
        media_attribution: "© Gym visual — https://gymvisual.com/"
      })
    );

    expect(view).toEqual({
      kind: "image",
      url: "/media/exercises/images/0001-still.jpg",
      attribution: "© Gym visual — https://gymvisual.com/"
    });
  });

  it("shows no card media when only a GIF is available", () => {
    expect(
      exerciseThumbnailView(
        makeEntry({ media_available: true, animation_url: "/media/exercises/videos/0001-demo.gif" })
      )
    ).toEqual({ kind: "none", url: "", attribution: "" });
  });
});

describe("exerciseMediaView", () => {
  it("reports no media while the backend keeps media unavailable", () => {
    expect(exerciseMediaView(makeEntry({ animation_url: "/media/a.gif", media_available: false }))).toEqual({
      kind: "none",
      url: "",
      attribution: ""
    });
  });

  it("prefers the animated demonstration over the still image", () => {
    const view = exerciseMediaView(
      makeEntry({
        media_available: true,
        animation_url: "/media/exercises/videos/0001-demo.gif",
        image_url: "/media/exercises/images/0001-still.jpg",
        media_attribution: "© Gym visual — https://gymvisual.com/"
      })
    );

    expect(view).toEqual({
      kind: "image",
      url: "/media/exercises/videos/0001-demo.gif",
      attribution: "© Gym visual — https://gymvisual.com/"
    });
  });

  it("classifies video files as video media", () => {
    const view = exerciseMediaView(makeEntry({ media_available: true, video_url: "/media/clip.mp4" }));

    expect(view.kind).toBe("video");
    expect(view.url).toBe("/media/clip.mp4");
  });

  it("falls back to the still image when no animation exists", () => {
    const view = exerciseMediaView(makeEntry({ media_available: true, image_url: "/media/still.jpg" }));

    expect(view.kind).toBe("image");
    expect(view.url).toBe("/media/still.jpg");
  });
});

describe("exerciseBodyPartLabel", () => {
  it("maps dataset values to display labels", () => {
    expect(exerciseBodyPartLabel("upper arms")).toBe("Upper Arms");
  });

  it("returns unknown values verbatim and empty values as empty", () => {
    expect(exerciseBodyPartLabel("obliques")).toBe("obliques");
    expect(exerciseBodyPartLabel("  ")).toBe("");
  });
});
