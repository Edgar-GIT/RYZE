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
      port: 5173,
      // Phone testing on the local network (RYZE_LAN=true, with
      // VITE_API_BASE_URL=/api/v1): the API and uploads go through this server
      // to the local backend, so the phone only talks to one origin. The
      // backend sees the usual local frontend origin.
      proxy:
        env.RYZE_LAN === "true"
          ? Object.fromEntries(
              ["/api", "/uploads"].map((path) => [
                path,
                {
                  target: env.RYZE_LAN_BACKEND ?? "http://127.0.0.1:8080",
                  changeOrigin: true,
                  headers: { origin: "http://localhost:5173" }
                }
              ])
            )
          : undefined
    },
    preview: {
      host: "0.0.0.0",
      port: 4173
    }
  };
});
