import { expect, test } from "@playwright/test";

import { installApiMock, type MockRequest, type MockResponse } from "./api_mock";

const contactHandler = (response: MockResponse) => (request: MockRequest): MockResponse | undefined => {
  if (request.method === "GET" && request.path === "/auth/test-mode") {
    return { data: { active: false } };
  }
  if (request.method === "GET" && request.path === "/me/entitlements") {
    return { status: 401, error: { code: "UNAUTHORIZED" } };
  }
  if (request.method === "POST" && request.path === "/contact") {
    return response;
  }
  return undefined;
};

const fillContactForm = async (page: import("@playwright/test").Page) => {
  await page.getByRole("textbox", { name: "Name", exact: true }).fill("Ada Lovelace");
  await page.getByRole("textbox", { name: "Email", exact: true }).fill("ada@example.com");
  await page.getByRole("textbox", { name: "Subject", exact: true }).fill("Training question");
  await page.getByRole("textbox", { name: "Message", exact: true }).fill("How should I start?");
};

test("Contact reports success only after the API accepts the message", async ({ page }) => {
  await installApiMock(page, contactHandler({ data: { status: "sent" } }));
  await page.goto("/contact");

  await fillContactForm(page);
  await page.getByRole("button", { name: "Send Message" }).click();

  await expect(page.getByRole("status")).toHaveText(
    "Message sent. We usually reply within 24 hours."
  );
});

test("Contact surfaces an unavailable delivery instead of a false success", async ({ page }) => {
  await installApiMock(
    page,
    contactHandler({
      status: 503,
      error: { code: "CONTACT_UNAVAILABLE" }
    })
  );
  await page.goto("/contact");

  await fillContactForm(page);
  await page.getByRole("button", { name: "Send Message" }).click();

  await expect(page.getByRole("status")).toHaveText(
    "The contact form is temporarily unavailable. Please email us directly."
  );
});
