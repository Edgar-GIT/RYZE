import { expect, test } from "@playwright/test";

import { installApiMock, type MockRequest, type MockResponse } from "./api_mock";

const trainer = {
  id: "trainer-1",
  user_id: "user-1",
  user: {
    id: "user-1",
    email: "trainer@example.com",
    first_name: "Alex",
    last_name: "Trainer",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z"
  },
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z"
};

const program = {
  id: "p1",
  trainer_id: "trainer-1",
  name: "Strength Foundations",
  description: "A four-week strength block.",
  type: "premium",
  status: "draft",
  price_minor_units: 4900,
  currency: "EUR",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z"
};

const week = {
  id: "w1",
  program_id: "p1",
  week_number: 1,
  workouts: [
    {
      id: "wo1",
      program_week_id: "w1",
      position: 1,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z"
    }
  ],
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z"
};

const exercise = {
  id: "we1",
  program_workout_id: "wo1",
  position: 1,
  instructions: "",
  notes: "",
  exercise: {
    id: "e1",
    name: "Barbell Squat",
    description: "A compound leg movement.",
    target_muscles: "Quads",
    equipment: "Barbell",
    difficulty: "Intermediate",
    video_url: "",
    image_url: ""
  },
  sets: [],
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z"
};

let patchBody: unknown = null;

const trainerHandler = (request: MockRequest): MockResponse | undefined => {
  const { method, path, body } = request;
  if (method === "GET" && path === "/auth/test-mode") {
    return { data: { active: false } };
  }
  if (method === "GET" && path === "/trainer/access") {
    return { data: { is_trainer: true, application: null } };
  }
  if (method === "GET" && path === "/trainer/profile") {
    return { data: trainer };
  }
  if (method === "GET" && path === "/trainer/programs/p1") {
    return { data: program };
  }
  if (method === "GET" && path === "/trainer/programs/p1/weeks") {
    return { data: { weeks: [week] } };
  }
  if (method === "GET" && path === "/trainer/programs/p1/weeks/w1/workouts/wo1/exercises") {
    return { data: { exercises: [exercise] } };
  }
  if (method === "PATCH" && path === "/trainer/programs/p1/weeks/w1/workouts/wo1/exercises/we1") {
    patchBody = body;
    const input = body as {
      instructions: string;
      notes: string;
      sets: Array<Record<string, unknown>>;
    };
    return {
      data: {
        ...exercise,
        instructions: input.instructions,
        notes: input.notes,
        sets: input.sets.map((set, index) => ({ ...set, set_number: index + 1 })),
        updated_at: "2026-01-02T00:00:00Z"
      }
    };
  }
  return undefined;
};

test.beforeEach(async ({ page }) => {
  patchBody = null;
  await installApiMock(page, trainerHandler);
});

test("a trainer edits instructions and a prescription set in the browser", async ({ page }) => {
  await page.goto("/trainer/programs/p1");

  await expect(page.getByRole("heading", { name: "Strength Foundations" })).toBeVisible();
  await page.getByRole("button", { name: /Workout 1/ }).click();
  await expect(page.getByText("Barbell Squat")).toBeVisible();

  await page.getByRole("button", { name: "Edit prescription for Barbell Squat" }).click();
  await expect(page.getByRole("region", { name: "Prescription for Barbell Squat" })).toBeVisible();

  await page.getByLabel("Instructions").fill("Brace before descending.");
  await page.getByRole("button", { name: "Add set" }).click();
  await page.getByLabel("Set 1 reps").fill("8");
  await page.getByLabel("Set 1 weight in kilograms").fill("60");
  await page.getByLabel("Set 1 RIR").fill("2");
  await page.getByLabel("Set 1 rest in seconds").fill("120");
  await page.getByRole("button", { name: "Save prescription" }).click();

  await expect(page.getByText("Prescription saved.")).toBeVisible();
  await expect(page.getByText("1 set · 8 reps · 60 kg · RIR 2 · 120s rest")).toBeVisible();
  expect(patchBody).toEqual({
    instructions: "Brace before descending.",
    notes: "",
    sets: [
      {
        set_type: "working",
        reps: 8,
        weight_kg: 60,
        rir: 2,
        rpe: null,
        rest_seconds: 120,
        tempo: ""
      }
    ]
  });
});
