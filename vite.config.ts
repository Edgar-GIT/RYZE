import { fileURLToPath, URL } from "node:url";

import react from "@vitejs/plugin-react";
import { loadEnv } from "vite";
import { defineConfig } from "vitest/config";

import {
  EXERCISE_MEDIA_PREVIEW_PREFIX,
  exerciseMediaDevPreview
} from "./main/src/frontend/dev/exercise_media_preview";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const exerciseMediaPreviewEnabled =
    env.EXERCISE_MEDIA_ENABLED === "true" &&
    (env.EXERCISE_MEDIA_BASE_PATH ?? "").replace(/\/+$/, "") === EXERCISE_MEDIA_PREVIEW_PREFIX;

  return {
    plugins: [react(), exerciseMediaDevPreview({ enabled: exerciseMediaPreviewEnabled })],
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
  };
});
