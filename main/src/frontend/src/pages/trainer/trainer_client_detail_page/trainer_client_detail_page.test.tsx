import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/services/trainer_api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/trainer_api")>();
  return {
    ...actual,
    fetchTrainerClient: vi.fn(),
    fetchTrainerClientOverview: vi.fn(),
    fetchTrainerClientPrograms: vi.fn(),
    fetchTrainerPrograms: vi.fn(),
    fetchTrainerClientWorkoutSessions: vi.fn(),
    fetchTrainerClientWorkoutSession: vi.fn(),
    fetchTrainerClientExerciseHistory: vi.fn(),
    assignTrainerClientProgram: vi.fn(),
    removeTrainerClientProgram: vi.fn()
  };
});

import {
  assignTrainerClientProgram,
  fetchTrainerClient,
  fetchTrainerClientExerciseHistory,
  fetchTrainerClientOverview,
  fetchTrainerClientPrograms,
  fetchTrainerClientWorkoutSession,
  fetchTrainerClientWorkoutSessions,
  fetchTrainerPrograms,
  removeTrainerClientProgram,
  type TrainerClientOverview
} from "@/services/trainer_api";

import { TrainerClientDetailPage } from "./trainer_client_detail_page";

const mockedClient = vi.mocked(fetchTrainerClient);
const mockedOverview = vi.mocked(fetchTrainerClientOverview);
const mockedAssignments = vi.mocked(fetchTrainerClientPrograms);
const mockedPrograms = vi.mocked(fetchTrainerPrograms);
const mockedSessions = vi.mocked(fetchTrainerClientWorkoutSessions);
const mockedSession = vi.mocked(fetchTrainerClientWorkoutSession);
const mockedExerciseHistory = vi.mocked(fetchTrainerClientExerciseHistory);
const mockedAssign = vi.mocked(assignTrainerClientProgram);
const mockedRemove = vi.mocked(removeTrainerClientProgram);

const client = {
  id: "relation-1",
  trainer_id: "trainer-1",
  user_id: "u1",
  user: {
    id: "u1",
    email: "client@example.com",
    first_name: "Casey",
    last_name: "Client",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z"
  },
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-02T00:00:00Z"
};

const overview: TrainerClientOverview = {
  access: {
    assignments: [],
    purchases: [],
    free_activations: []
  },
  statistics: {
    has_active_assignment: false,
    has_active_program: false,
    current_program_name: "",
    total_executions: 2,
    unique_workouts_completed: 2,
    total_workouts_in_program: 4,
    completion_percentage: 50,
    last_workout_date: "2026-11-05T10:00:00Z",
    completed_sessions: 3,
    abandoned_sessions: 1,
    completed_sets: 21,
    skipped_sets: 2,
    total_active_seconds: 5400
  },
  active_session: null,
  recent_sessions: [],
  legacy: { count: 2, last_completed_at: "2026-11-04T10:00:00Z" },
  exercises: []
};

const summary = {
  id: "session-1",
  program_id: "p1",
  program_name: "Strength Builder",
  week_number: 1,
  workout_position: 2,
  status: "completed" as const,
  started_at: "2026-11-05T10:00:00Z",
  completed_at: "2026-11-05T11:00:00Z",
  abandoned_at: null,
  elapsed_seconds: 3600,
  total_sets: 3,
  completed_sets: 2,
  skipped_sets: 1
};

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/trainer/clients/u1"]}>
      <Route path="/trainer/clients/:userID">
        <TrainerClientDetailPage />
      </Route>
    </MemoryRouter>
  );

beforeEach(() => {
  mockedClient.mockReset().mockResolvedValue(client);
  mockedOverview.mockReset().mockResolvedValue(overview);
  mockedAssignments.mockReset().mockResolvedValue([]);
  mockedPrograms.mockReset().mockResolvedValue({
    programs: [
      {
        id: "p1",
        trainer_id: "trainer-1",
        name: "Strength Builder",
        description: "",
        type: "premium",
        status: "draft",
        price_minor_units: 4900,
        currency: "EUR",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z"
      }
    ],
    pagination: { page: 1, limit: 100, total: 1, total_pages: 1 }
  });
  mockedSessions.mockReset().mockResolvedValue({
    sessions: [summary],
    pagination: { page: 1, limit: 10, total: 1, total_pages: 1 }
  });
  mockedSession.mockReset().mockResolvedValue({
    id: "session-1",
    program_id: "p1",
    program_workout_id: "w1",
    program_name: "Strength Builder",
    week_number: 1,
    workout_position: 2,
    status: "completed",
    started_at: "2026-11-05T10:00:00Z",
    paused_at: null,
    completed_at: "2026-11-05T11:00:00Z",
    abandoned_at: null,
    elapsed_seconds: 3600,
    total_sets: 3,
    completed_sets: 2,
    skipped_sets: 1,
    exercises: [
      {
        id: "se1",
        exercise_id: "e1",
        position: 1,
        name: "Back Squat",
        instructions: "",
        notes: "",
        sets: [
          {
            id: "set-1",
            set_number: 1,
            set_type: "working",
            prescribed_reps: 10,
            prescribed_weight_kg: 80,
            prescribed_rir: 2,
            prescribed_rpe: null,
            prescribed_rest_seconds: 90,
            prescribed_tempo: "",
            actual_reps: 8,
            actual_weight_kg: 75,
            actual_rir: 3,
            actual_rpe: null,
            status: "completed",
            completed_at: "2026-11-05T10:15:00Z",
            rest_ends_at: null
          }
        ]
      }
    ]
  });
  mockedExerciseHistory.mockReset().mockResolvedValue({
    exercise_id: "e1",
    name: "Back Squat",
    sessions: [
      {
        session_id: "session-1",
        program_id: "p1",
        program_name: "Strength Builder",
        week_number: 1,
        workout_position: 2,
        completed_at: "2026-11-05T11:00:00Z",
        exercise_id: "e1",
        exercise_name: "Back Squat",
        instructions: "",
        notes: "",
        sets: [
          {
            id: "set-1",
            set_number: 1,
            set_type: "working",
            prescribed_reps: 10,
            prescribed_weight_kg: 80,
            prescribed_rir: 2,
            prescribed_rpe: null,
            prescribed_rest_seconds: 90,
            prescribed_tempo: "",
            actual_reps: 8,
            actual_weight_kg: 75,
            actual_rir: 3,
            actual_rpe: null,
            status: "completed",
            completed_at: "2026-11-05T10:15:00Z"
          }
        ]
      }
    ]
  });
  mockedAssign.mockReset().mockResolvedValue({} as never);
  mockedRemove.mockReset().mockResolvedValue({} as never);
});

afterEach(cleanup);

describe("TrainerClientDetailPage", () => {
  it("renders the client identity, statistics, access and legacy note", async () => {
    renderPage();

    expect(await screen.findByRole("heading", { name: "Casey Client" })).toBeTruthy();
    expect(screen.getByText(/client@example.com/)).toBeTruthy();
    expect(screen.getByText("50%")).toBeTruthy();
    expect(screen.getByText("3")).toBeTruthy();
    expect(screen.getByText(/2 older completion-only entries/)).toBeTruthy();
  });

  it("expands a session and shows prescribed and actual set values", async () => {
    mockedOverview.mockResolvedValue({ ...overview, recent_sessions: [summary] });
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: /Strength Builder/ }));

    expect(await screen.findByText("Back Squat")).toBeTruthy();
    expect(screen.getByText("10 reps × 80 kg")).toBeTruthy();
    expect(screen.getByText("8 reps × 75 kg")).toBeTruthy();
    expect(mockedSession).toHaveBeenCalledWith("u1", "session-1");
  });

  it("loads paginated history and exercise performance", async () => {
    mockedOverview.mockResolvedValue({
      ...overview,
      recent_sessions: [summary],
      exercises: [{ exercise_id: "e1", name: "Back Squat", session_count: 1, last_performed_at: "2026-11-05T11:00:00Z" }]
    });
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "View all sessions" }));
    await waitFor(() => expect(mockedSessions).toHaveBeenCalledWith("u1", 1, 10));

    fireEvent.change(screen.getByLabelText("Exercise"), { target: { value: "e1" } });
    await waitFor(() => expect(mockedExerciseHistory).toHaveBeenCalledWith("u1", "e1", 20));
    expect(await screen.findByText("8 reps × 75 kg")).toBeTruthy();
  });

  it("keeps recent sessions available when full history fails", async () => {
    mockedOverview.mockResolvedValue({ ...overview, recent_sessions: [summary] });
    mockedSessions.mockRejectedValueOnce(new Error("history failed"));
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "View all sessions" }));

    expect(await screen.findByText("Unable to load the training history right now.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "View all sessions" })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Strength Builder/ })).toBeTruthy();
  });

  it("assigns an available programme and refreshes the overview", async () => {
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "Assign programme" }));

    await waitFor(() => expect(mockedAssign).toHaveBeenCalledWith("u1", "p1"));
    expect(await screen.findByText("Programme assigned to this client.")).toBeTruthy();
    expect(mockedOverview).toHaveBeenCalledTimes(2);
  });

  it("confirms removal of the current trainer assignment", async () => {
    mockedOverview.mockResolvedValue({
      ...overview,
      access: {
        ...overview.access,
        assignments: [
          {
            assignment_id: "a1",
            program_id: "p1",
            program_name: "Strength Builder",
            program_status: "published",
            assigned_at: "2026-11-01T10:00:00Z",
            owned_by_current_trainer: true,
            access_status: "active"
          }
        ]
      }
    });
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "Remove" }));
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));

    await waitFor(() => expect(mockedRemove).toHaveBeenCalledWith("u1", "a1"));
    expect(await screen.findByText(/training history is unchanged/)).toBeTruthy();
  });
});
