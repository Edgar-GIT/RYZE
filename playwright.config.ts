import { existsSync } from "node:fs";

import { defineConfig } from "@playwright/test";

// CI uses the Playwright-managed browser installed by the workflow. Local
// developers may keep using a system Chromium; an explicit override wins.
const systemChromium = process.env.CI
  ? undefined
  : ["/usr/bin/chromium", "/usr/bin/chromium-browser", "/usr/bin/google-chrome"].find(
      (candidate) => existsSync(candidate)
    );
const executablePath =
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH || systemChromium;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  timeout: 30_000,
  reporter: [["list"]],
  use: {
    baseURL: "http://127.0.0.1:5174",
    trace: "retain-on-failure",
    launchOptions: executablePath ? { executablePath } : {}
  },
  webServer: {
    command: "VITE_API_BASE_URL=/api/v1 npm run dev -- --port 5174 --strictPort",
    url: "http://127.0.0.1:5174",
    reuseExistingServer: false,
    timeout: 120_000
  }
});
