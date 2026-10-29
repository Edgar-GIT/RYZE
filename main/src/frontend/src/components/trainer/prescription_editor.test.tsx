import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { TrainerWorkoutExercise } from "@/services/trainer_api";

import { PrescriptionEditor } from "./prescription_editor";

afterEach(cleanup);

const makeEntry = (overrides: Partial<TrainerWorkoutExercise> = {}): TrainerWorkoutExercise => ({
  id: "we1",
  program_workout_id: "wo1",
  position: 1,
  instructions: "Drive through midfoot.",
  notes: "Use the safety bars.",
  exercise: {
    id: "e1",
    name: "Barbell Squat",
    description: "",
    target_muscles: "Quads",
    equipment: "Barbell",
    difficulty: "Intermediate",
    video_url: "",
    image_url: ""
  },
  sets: [
    {
      set_number: 1,
      set_type: "working",
      reps: 8,
      weight_kg: 60,
      rir: 2,
      rpe: null,
      rest_seconds: 90,
      tempo: "3-1-1"
    }
  ],
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  ...overrides
});

describe("PrescriptionEditor", () => {
  it("loads the existing prescription and saves the complete replacement", () => {
    const onSave = vi.fn();
    render(<PrescriptionEditor entry={makeEntry()} busy={false} onSave={onSave} onCancel={vi.fn()} />);

    expect((screen.getByLabelText("Set 1 reps") as HTMLInputElement).value).toBe("8");
    expect((screen.getByLabelText("Set 1 weight in kilograms") as HTMLInputElement).value).toBe("60");
    expect((screen.getByLabelText("Set 1 type") as HTMLSelectElement).value).toBe("working");

    fireEvent.click(screen.getByRole("button", { name: "Add set" }));
    fireEvent.change(screen.getByLabelText("Set 2 type"), { target: { value: "warmup" } });
    fireEvent.change(screen.getByLabelText("Set 2 reps"), { target: { value: "10" } });
    fireEvent.change(screen.getByLabelText("Set 2 weight in kilograms"), { target: { value: "40" } });
    fireEvent.change(screen.getByLabelText("Set 2 RIR"), { target: { value: "4" } });
    fireEvent.change(screen.getByLabelText("Set 2 rest in seconds"), { target: { value: "60" } });

    fireEvent.click(screen.getByRole("button", { name: "Save prescription" }));

    expect(onSave).toHaveBeenCalledWith({
      instructions: "Drive through midfoot.",
      notes: "Use the safety bars.",
      sets: [
        {
          set_type: "working",
          reps: 8,
          weight_kg: 60,
          rir: 2,
          rpe: null,
          rest_seconds: 90,
          tempo: "3-1-1"
        },
        {
          set_type: "warmup",
          reps: 10,
          weight_kg: 40,
          rir: 4,
          rpe: null,
          rest_seconds: 60,
          tempo: ""
        }
      ]
    });
  });

  it("rejects out-of-range values before calling the API", () => {
    const onSave = vi.fn();
    render(<PrescriptionEditor entry={makeEntry()} busy={false} onSave={onSave} onCancel={vi.fn()} />);

    fireEvent.change(screen.getByLabelText("Set 1 RPE"), { target: { value: "11" } });
    fireEvent.click(screen.getByRole("button", { name: "Save prescription" }));

    expect(screen.getByRole("alert").textContent).toBe("Set 1 RPE must be between 1 and 10.");
    expect(onSave).not.toHaveBeenCalled();
  });

  it("removes a set and can start from an empty prescription", () => {
    const onSave = vi.fn();
    render(<PrescriptionEditor entry={makeEntry()} busy={false} onSave={onSave} onCancel={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: "Remove set 1" }));

    expect(screen.getByText("No sets yet. Add the first set to build the prescription.")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Save prescription" }));

    expect(onSave).toHaveBeenCalledWith({
      instructions: "Drive through midfoot.",
      notes: "Use the safety bars.",
      sets: []
    });
  });
});
