import { fileURLToPath, URL } from "node:url";

import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

import { exerciseMediaDevPreview } from "./main/src/frontend/dev/exercise_media_preview";

export default defineConfig({
  plugins: [react(), exerciseMediaDevPreview()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./main/src/frontend/src", import.meta.url)),
      "@resources": fileURLToPath(new URL("./main/src/resources", import.meta.url)),
      "@utils": fileURLToPath(new URL("./main/src/utils", import.meta.url))
    }
  },
  test: {
    environment: "jsdom",
    include: ["main/src/**/*.test.{ts,tsx}"],
    restoreMocks: true
  },
  server: {
    host: "0.0.0.0",
    port: 5173
  },
  preview: {
    host: "0.0.0.0",
    port: 4173
  }
});
