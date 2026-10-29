import { expect, test } from "@playwright/test";

import { installApiMock, type MockRequest, type MockResponse } from "./api_mock";

const genericProgram = {
  id: "generic-1",
  trainer_id: null,
  name: "Free Full Body",
  description: "A complete free starter plan.",
  type: "free",
  status: "published",
  level: "beginner",
  duration_weeks: 6,
  frequency_per_week: 3,
  training_type: "full body",
  price_minor_units: 0,
  currency: "EUR",
  product_type: "generic",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z"
};

const paidGenericProgram = {
  ...genericProgram,
  id: "generic-2",
  name: "Strength Builder",
  description: "A progressive strength plan.",
  type: "premium",
  price_minor_units: 999
};

const premiumPrograms = [
  {
    ...genericProgram,
    id: "premium-1",
    name: "Hypertrophy Foundations",
    description: "Training and nutrition for your first hypertrophy block.",
    type: "premium",
    price_minor_units: 1449,
    product_type: "premium_level_1"
  },
  {
    ...genericProgram,
    id: "premium-2",
    name: "Hypertrophy Advanced",
    description: "Training and nutrition for experienced lifters.",
    type: "premium",
    price_minor_units: 1949,
    product_type: "premium_level_1"
  }
];

const detail = (program: (typeof premiumPrograms)[number]) => ({
  ...program,
  weeks: []
});

const catalogHandler = (request: MockRequest): MockResponse | undefined => {
  const { method, path, url } = request;
  if (method !== "GET") {
    return undefined;
  }
  if (path === "/auth/test-mode") {
    return { data: { active: false } };
  }
  if (path === "/me/entitlements") {
    return { status: 401, error: { code: "UNAUTHORIZED" } };
  }
  if (path === "/payments/methods") {
    return { data: [] };
  }
  if (path === "/programs") {
    const scope = url.searchParams.get("scope");
    const programs = scope === "premium_level_1" ? premiumPrograms : [genericProgram, paidGenericProgram];
    return {
      data: {
        programs,
        pagination: { page: 1, limit: 100, total: programs.length }
      }
    };
  }
  const program = premiumPrograms.find((entry) => path === `/programs/${entry.id}`);
  if (program) {
    return { data: detail(program) };
  }
  return undefined;
};

test.beforeEach(async ({ page }) => {
  await installApiMock(page, catalogHandler);
});

test("Services renders only API-backed product categories and no Premium Level 2", async ({ page }) => {
  await page.goto("/services");

  await expect(page.getByRole("link", { name: /Browse training plans/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /View Premium Level 1/ })).toBeVisible();
  await expect(page.getByText("Free and paid")).toBeVisible();
  await expect(page.getByText("From €14.49")).toBeVisible();
  await expect(page.getByText("Premium Level 2")).toHaveCount(0);
  await expect(page.locator('a[href*="premium-level-2"]')).toHaveCount(0);
});

test("Home highlights published programs instead of hardcoded offers", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("heading", { name: "Hypertrophy Foundations" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Free Full Body" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Strength Builder" })).toBeVisible();
  await expect(page.getByRole("link", { name: "View package" })).toHaveAttribute(
    "href",
    "/services/premium-level-1?program=premium-1"
  );
  await expect(page.getByText("LOSE FAT")).toHaveCount(0);
  await expect(page.getByText("BUILD MUSCLE")).toHaveCount(0);
  await expect(page.locator('a[href*="premium-level-2"]')).toHaveCount(0);
});

test("Premium Level 1 lists every returned package and opens the chosen one", async ({ page }) => {
  await page.goto("/services/premium-level-1");

  await expect(page.getByRole("heading", { name: "Choose your package" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Hypertrophy Foundations" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Hypertrophy Advanced" })).toBeVisible();

  const advanced = page.getByRole("article").filter({ hasText: "Hypertrophy Advanced" });
  await advanced.getByRole("link", { name: "View package" }).click();

  await expect(page).toHaveURL(/\/services\/premium-level-1\?program=premium-2$/);
  await expect(page.getByRole("heading", { name: "Hypertrophy Advanced" })).toBeVisible();
  await expect(page.getByRole("link", { name: "All packages" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Start my questionnaire" })).toBeVisible();
});
