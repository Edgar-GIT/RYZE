import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@utils/http_client", () => ({
  apiGet: vi.fn(),
  apiPost: vi.fn(),
  apiPatch: vi.fn(),
  apiDelete: vi.fn()
}));

import { apiDelete, apiGet, apiPatch, apiPost } from "@utils/http_client";

import {
  addTrainerClient,
  addWorkoutExercise,
  createProgramWeek,
  fetchProgramWeeks,
  fetchTrainerAccess,
  fetchTrainerClients,
  fetchTrainerPrograms,
  fetchWorkoutExercises,
  lookupTrainerClient,
  reorderProgramWeeks,
  reorderWorkoutExercises,
  submitTrainerApplication
} from "./trainer_api";

const mockedGet = vi.mocked(apiGet);
const mockedPost = vi.mocked(apiPost);
const mockedPatch = vi.mocked(apiPatch);
const mockedDelete = vi.mocked(apiDelete);

beforeEach(() => {
  mockedGet.mockReset();
  mockedPost.mockReset();
  mockedPatch.mockReset();
  mockedDelete.mockReset();
});

describe("access and application", () => {
  it("reads the trainer access status from the access endpoint", async () => {
    mockedGet.mockResolvedValue({ is_trainer: false, application: null });

    const result = await fetchTrainerAccess();

    expect(mockedGet).toHaveBeenCalledWith("/trainer/access");
    expect(result).toEqual({ is_trainer: false, application: null });
  });

  it("submits the application without a body", async () => {
    mockedPost.mockResolvedValue({ id: "app-1", status: "PENDING" });

    await submitTrainerApplication();

    expect(mockedPost).toHaveBeenCalledWith("/trainer/apply", {});
  });
});

describe("clients", () => {
  it("requests one page of clients with pagination parameters", async () => {
    mockedGet.mockResolvedValue({ clients: [], pagination: { page: 2, limit: 20, total: 0, total_pages: 0 } });

    await fetchTrainerClients(2, 20);

    expect(mockedGet).toHaveBeenCalledWith("/trainer/clients?page=2&limit=20");
  });

  it("encodes the lookup email", async () => {
    mockedGet.mockResolvedValue({ user_id: "u1", email: "a@b.c", first_name: "", last_name: "" });

    await lookupTrainerClient("a b@c.d");

    expect(mockedGet).toHaveBeenCalledWith("/trainer/clients/lookup?email=a%20b%40c.d");
  });

  it("adds a client by user id", async () => {
    mockedPost.mockResolvedValue({ user_id: "u1" });

    await addTrainerClient("u1");

    expect(mockedPost).toHaveBeenCalledWith("/trainer/clients", { user_id: "u1" });
  });
});

describe("programs and structure", () => {
  it("requests one page of programs", async () => {
    mockedGet.mockResolvedValue({ programs: [], pagination: { page: 1, limit: 20, total: 0, total_pages: 0 } });

    await fetchTrainerPrograms(1, 20);

    expect(mockedGet).toHaveBeenCalledWith("/trainer/programs?page=1&limit=20");
  });

  it("creates a week with an empty body", async () => {
    mockedPost.mockResolvedValue({ id: "w1" });

    await createProgramWeek("p1");

    expect(mockedPost).toHaveBeenCalledWith("/trainer/programs/p1/weeks", {});
  });

  it("unwraps the weeks envelope returned by the API", async () => {
    const weeks = [{ id: "w1", week_number: 1 }];
    mockedGet.mockResolvedValue({ weeks });

    const result = await fetchProgramWeeks("p1");

    expect(mockedGet).toHaveBeenCalledWith("/trainer/programs/p1/weeks");
    expect(result).toEqual(weeks);
  });

  it("sends the full id list when reordering weeks", async () => {
    mockedPatch.mockResolvedValue({});

    await reorderProgramWeeks("p1", ["w2", "w1"]);

    expect(mockedPatch).toHaveBeenCalledWith("/trainer/programs/p1/weeks/order", { ids: ["w2", "w1"] });
  });

  it("adds an exercise to a workout with only the exercise id", async () => {
    mockedPost.mockResolvedValue({ id: "we1" });

    await addWorkoutExercise("p1", "w1", "wo1", "e1");

    expect(mockedPost).toHaveBeenCalledWith(
      "/trainer/programs/p1/weeks/w1/workouts/wo1/exercises",
      { exercise_id: "e1" }
    );
  });

  it("unwraps the exercises envelope returned by the API", async () => {
    const exercises = [{ id: "we1", position: 1 }];
    mockedGet.mockResolvedValue({ exercises });

    const result = await fetchWorkoutExercises("p1", "w1", "wo1");

    expect(mockedGet).toHaveBeenCalledWith("/trainer/programs/p1/weeks/w1/workouts/wo1/exercises");
    expect(result).toEqual(exercises);
  });

  it("sends the full id list when reordering workout exercises", async () => {
    mockedPatch.mockResolvedValue({});

    await reorderWorkoutExercises("p1", "w1", "wo1", ["we2", "we1"]);

    expect(mockedPatch).toHaveBeenCalledWith(
      "/trainer/programs/p1/weeks/w1/workouts/wo1/exercises/order",
      { ids: ["we2", "we1"] }
    );
  });
});
