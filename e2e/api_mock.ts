import type { Page, Route } from "@playwright/test";

export interface MockRequest {
  method: string;
  path: string;
  url: URL;
  body: unknown;
}

export interface MockResponse {
  status?: number;
  data?: unknown;
  error?: { code: string; details?: string[] };
}

export type MockHandler = (request: MockRequest) => MockResponse | undefined;

export const installApiMock = async (page: Page, handler: MockHandler): Promise<void> => {
  await page.route("**/api/v1/**", async (route: Route) => {
    if (route.request().method() === "OPTIONS") {
      await route.fulfill({ status: 204, headers: corsHeaders() });
      return;
    }

    const url = new URL(route.request().url());
    const path = url.pathname.replace(/^\/api\/v1/, "");
    let body: unknown = undefined;
    const raw = route.request().postData();
    if (raw) {
      try {
        body = JSON.parse(raw);
      } catch {
        body = raw;
      }
    }

    const response = handler({ method: route.request().method(), path, url, body }) ?? {
      status: 404,
      error: { code: "API_ERROR" }
    };
    const status = response.status ?? 200;
    await route.fulfill({
      status,
      headers: { ...corsHeaders(), "content-type": "application/json" },
      body: JSON.stringify({
        success: status < 400,
        message: status < 400 ? "OK" : "Error",
        data: response.data,
        error: response.error
          ? { code: response.error.code, details: response.error.details ?? [] }
          : undefined
      })
    });
  });
};

const corsHeaders = (): Record<string, string> => ({
  "access-control-allow-origin": "http://127.0.0.1:5174",
  "access-control-allow-credentials": "true",
  "access-control-allow-methods": "GET,POST,PATCH,DELETE,OPTIONS",
  "access-control-allow-headers": "content-type"
});
