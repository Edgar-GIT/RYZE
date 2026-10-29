import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@utils/http_client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@utils/http_client")>();
  return {
    ...actual,
    apiGet: vi.fn(),
    apiPost: vi.fn(),
    apiPatch: vi.fn(),
    apiDelete: vi.fn(),
    apiUpload: vi.fn(),
    downloadFile: vi.fn()
  };
});

import { apiDelete, apiGet, apiPatch, apiPost, apiUpload, downloadFile } from "@utils/http_client";

import {
  ApplicationDocumentKind,
  addTrainerClient,
  addWorkoutExercise,
  createProgramWeek,
  deleteTrainerApplicationDocument,
  downloadTrainerApplicationDocument,
  fetchProgramWeeks,
  fetchTrainerAccess,
  fetchTrainerApplicationDocuments,
  fetchTrainerClients,
  fetchTrainerPrograms,
  fetchWorkoutExercises,
  lookupTrainerClient,
  reorderProgramWeeks,
  reorderWorkoutExercises,
  submitTrainerApplication,
  updateWorkoutExercisePrescription,
  uploadTrainerApplicationDocument
} from "./trainer_api";

const mockedGet = vi.mocked(apiGet);
const mockedPost = vi.mocked(apiPost);
const mockedPatch = vi.mocked(apiPatch);
const mockedDelete = vi.mocked(apiDelete);
const mockedUpload = vi.mocked(apiUpload);
const mockedDownload = vi.mocked(downloadFile);

beforeEach(() => {
  mockedGet.mockReset();
  mockedPost.mockReset();
  mockedPatch.mockReset();
  mockedDelete.mockReset();
  mockedUpload.mockReset();
  mockedDownload.mockReset();
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

  it("replaces one workout exercise prescription with the complete set list", async () => {
    mockedPatch.mockResolvedValue({ id: "we1" });
    const input = {
      instructions: "Drive through midfoot.",
      notes: "",
      sets: [
        {
          set_type: "working" as const,
          reps: 8,
          weight_kg: 60,
          rir: 2,
          rpe: null,
          rest_seconds: 90,
          tempo: ""
        }
      ]
    };

    await updateWorkoutExercisePrescription("p1", "w1", "wo1", "we1", input);

    expect(mockedPatch).toHaveBeenCalledWith(
      "/trainer/programs/p1/weeks/w1/workouts/wo1/exercises/we1",
      input
    );
  });
});

describe("application documents", () => {
  it("lists the documents attached to the pending application", async () => {
    const documents = [
      {
        id: "doc-1",
        kind: "certification",
        original_filename: "cert.pdf",
        content_type: "application/pdf",
        byte_size: 2048,
        created_at: "2026-01-01T00:00:00Z"
      }
    ];
    mockedGet.mockResolvedValue({ documents });

    const result = await fetchTrainerApplicationDocuments();

    expect(mockedGet).toHaveBeenCalledWith("/trainer/application/documents");
    expect(result).toEqual({ documents });
  });

  it("uploads the chosen file as multipart form data with its kind", async () => {
    mockedUpload.mockResolvedValue({
      id: "doc-1",
      kind: "identification",
      original_filename: "id.png",
      content_type: "image/png",
      byte_size: 3,
      created_at: "2026-01-01T00:00:00Z"
    });
    const file = new File(["abc"], "id.png", { type: "image/png" });

    await uploadTrainerApplicationDocument(ApplicationDocumentKind.IDENTIFICATION, file);

    expect(mockedUpload).toHaveBeenCalledTimes(1);
    const [path, form] = mockedUpload.mock.calls[0];
    expect(path).toBe("/trainer/application/documents");
    expect(form).toBeInstanceOf(FormData);
    expect(form.get("kind")).toBe("identification");
    expect(form.get("file")).toBe(file);
  });

  it("url-encodes the document id when deleting", async () => {
    mockedDelete.mockResolvedValue(undefined as never);

    await deleteTrainerApplicationDocument("doc/1");

    expect(mockedDelete).toHaveBeenCalledWith("/trainer/application/documents/doc%2F1");
  });

  it("downloads the document through the download endpoint", async () => {
    mockedDownload.mockResolvedValue(undefined);

    await downloadTrainerApplicationDocument("doc-1", "certification.pdf");

    expect(mockedDownload).toHaveBeenCalledWith(
      "/trainer/application/documents/doc-1/download",
      "certification.pdf"
    );
  });
});
