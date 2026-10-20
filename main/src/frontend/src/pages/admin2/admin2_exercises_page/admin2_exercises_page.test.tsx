import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/services/admin2_api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/admin2_api")>();
  return {
    ...actual,
    fetchAdminExercises: vi.fn(),
    createAdminExercise: vi.fn(),
    updateAdminExercise: vi.fn()
  };
});

import {
  createAdminExercise,
  fetchAdminExercises,
  updateAdminExercise,
  type AdminExercise
} from "@/services/admin2_api";
import { ApiError } from "@utils/http_client";

import Admin2ExercisesPage from "./admin2_exercises_page";

const mockedFetch = vi.mocked(fetchAdminExercises);
const mockedCreate = vi.mocked(createAdminExercise);
const mockedUpdate = vi.mocked(updateAdminExercise);

const exercise: AdminExercise = {
  id: "ex-1",
  name: "Barbell Squat",
  source: "gymvisual",
  source_id: "0043",
  body_part: "upper legs",
  description: "A compound leg movement.",
  instructions: "Squat with a barbell.",
  instruction_steps: ["Stand tall.", "Squat down."],
  target_muscles: "Quadriceps",
  primary_muscle_group: "upper legs",
  secondary_muscle_groups: "glutes",
  equipment: "Barbell",
  difficulty: "Intermediate",
  movement_category: "Compound",
  video_url: "",
  image_url: "exercises/squat.jpg",
  animation_url: "exercises/squat.gif",
  source_media_id: "0043",
  media_attribution: "© Gym visual",
  status: "draft",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z"
};

const page = { page: 1, limit: 25, total: 1, total_pages: 1 };

beforeEach(() => {
  mockedFetch.mockReset().mockResolvedValue({ exercises: [exercise], pagination: page });
  mockedCreate.mockReset().mockResolvedValue({ exercise });
  mockedUpdate.mockReset().mockResolvedValue({ exercise });
});

afterEach(cleanup);

describe("Admin2ExercisesPage", () => {
  it("lists catalogue entries with their publication status", async () => {
    render(<Admin2ExercisesPage />);

    expect(await screen.findByText("Barbell Squat")).toBeTruthy();
    expect(within(screen.getByRole("table")).getByText("Draft")).toBeTruthy();
    expect(mockedFetch).toHaveBeenCalledWith(1, 25, { query: "", status: "", bodyPart: "" });
  });

  it("creates a new catalogue entry from the editor", async () => {
    render(<Admin2ExercisesPage />);
    await screen.findByText("Barbell Squat");

    fireEvent.click(screen.getByRole("button", { name: "New exercise" }));
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Front Squat" } });
    fireEvent.click(screen.getByRole("button", { name: "Create exercise" }));

    await waitFor(() => {
      expect(mockedCreate).toHaveBeenCalledTimes(1);
    });
    expect(mockedCreate.mock.calls[0][0].name).toBe("Front Squat");
    await waitFor(() => {
      expect(mockedFetch).toHaveBeenCalledTimes(2);
    });
  });

  it("edits an entry without renaming it and updates its status", async () => {
    render(<Admin2ExercisesPage />);
    await screen.findByText("Barbell Squat");

    fireEvent.click(screen.getByRole("button", { name: "Edit Barbell Squat" }));

    const nameInput = screen.getByLabelText("Name") as HTMLInputElement;
    expect(nameInput.disabled).toBe(true);

    fireEvent.change(screen.getByLabelText("Status"), { target: { value: "published" } });
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => {
      expect(mockedUpdate).toHaveBeenCalledTimes(1);
    });
    expect(mockedUpdate.mock.calls[0][0]).toBe("ex-1");
    expect(mockedUpdate.mock.calls[0][1].status).toBe("published");
  });

  it("maps a duplicate name error to plain wording", async () => {
    mockedCreate.mockRejectedValue(new ApiError(409, "EXERCISE_NAME_TAKEN", "exercise_name_taken"));
    render(<Admin2ExercisesPage />);
    await screen.findByText("Barbell Squat");

    fireEvent.click(screen.getByRole("button", { name: "New exercise" }));
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Barbell Squat" } });
    fireEvent.click(screen.getByRole("button", { name: "Create exercise" }));

    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toContain("An exercise with this name already exists.");
    expect(screen.queryByText("exercise_name_taken")).toBeNull();
  });
});
