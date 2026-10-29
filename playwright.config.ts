import { existsSync } from "node:fs";

import { defineConfig } from "@playwright/test";

const systemChromium = [
  "/usr/bin/chromium",
  "/usr/bin/chromium-browser",
  "/usr/bin/google-chrome"
].find((candidate) => existsSync(candidate));

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  timeout: 30_000,
  reporter: [["list"]],
  use: {
    baseURL: "http://127.0.0.1:5174",
    trace: "retain-on-failure",
    launchOptions: systemChromium ? { executablePath: systemChromium } : {}
  },
  webServer: {
    command: "VITE_API_BASE_URL=/api/v1 npm run dev -- --port 5174 --strictPort",
    url: "http://127.0.0.1:5174",
    reuseExistingServer: false,
    timeout: 120_000
  }
});
