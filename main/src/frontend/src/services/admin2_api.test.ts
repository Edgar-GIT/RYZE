import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@utils/http_client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@utils/http_client")>();
  return {
    ...actual,
    apiGet: vi.fn(),
    apiPost: vi.fn(),
    apiPatch: vi.fn(),
    apiDelete: vi.fn()
  };
});

import { apiGet, apiPatch, apiPost } from "@utils/http_client";

import { createAdminExercise, fetchAdminExercises, updateAdminExercise } from "./admin2_api";

const mockedGet = vi.mocked(apiGet);
const mockedPost = vi.mocked(apiPost);
const mockedPatch = vi.mocked(apiPatch);

const exerciseInput = {
  description: "A squat.",
  instructions: "Bend the knees.",
  instruction_steps: ["Stand tall.", "Squat down."],
  target_muscles: "Quads",
  primary_muscle_group: "upper legs",
  secondary_muscle_groups: "glutes",
  equipment: "Barbell",
  difficulty: "Beginner",
  movement_category: "Compound",
  body_part: "upper legs",
  image_url: "exercises/squat.jpg",
  animation_url: "exercises/squat.gif",
  source_media_id: "0043",
  media_attribution: "© Gym visual"
};

beforeEach(() => {
  mockedGet.mockReset();
  mockedPost.mockReset();
  mockedPatch.mockReset();
});

describe("admin exercise catalogue", () => {
  it("builds the filtered catalogue query", async () => {
    mockedGet.mockResolvedValue({ exercises: [], pagination: { page: 2, limit: 25, total: 0, total_pages: 0 } });

    await fetchAdminExercises(2, 25, { query: "  squat ", status: "published", bodyPart: "upper legs" });

    expect(mockedGet).toHaveBeenCalledWith(
      "/admin/exercises?page=2&limit=25&q=squat&status=published&body_part=upper+legs"
    );
  });

  it("omits empty filters from the query", async () => {
    mockedGet.mockResolvedValue({ exercises: [], pagination: { page: 1, limit: 25, total: 0, total_pages: 0 } });

    await fetchAdminExercises(1, 25);

    expect(mockedGet).toHaveBeenCalledWith("/admin/exercises?page=1&limit=25");
  });

  it("creates a draft entry", async () => {
    mockedPost.mockResolvedValue({ exercise: { id: "ex-1" } });

    await createAdminExercise({ ...exerciseInput, name: "Squat" });

    expect(mockedPost).toHaveBeenCalledWith("/admin/exercises", { ...exerciseInput, name: "Squat" });
  });

  it("updates an entry through its encoded id", async () => {
    mockedPatch.mockResolvedValue({ exercise: { id: "ex-1" } });

    await updateAdminExercise("ex/1", { ...exerciseInput, status: "published" });

    expect(mockedPatch).toHaveBeenCalledWith("/admin/exercises/ex%2F1", {
      ...exerciseInput,
      status: "published"
    });
  });
});
