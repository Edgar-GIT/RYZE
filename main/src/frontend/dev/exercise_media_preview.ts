import { createReadStream, realpathSync, statSync } from "node:fs";
import type { IncomingMessage, ServerResponse } from "node:http";
import { extname, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

import type { Plugin } from "vite";

// The downloaded dataset media is third-party content (© Gym visual). It is
// served only by this Vite development middleware, only to loopback clients,
// and it is never imported by the application, so it cannot enter the
// production bundle or the static build output.
const DATASET_MEDIA_ROOT = fileURLToPath(
  new URL("../../resources/exercises/exercises-dataset-main/", import.meta.url)
);

export const EXERCISE_MEDIA_PREVIEW_PREFIX = "/dev-exercise-media";

const ALLOWED_DIRECTORIES = new Set(["images", "videos"]);

const CONTENT_TYPES = new Map([
  [".gif", "image/gif"],
  [".jpg", "image/jpeg"],
  [".jpeg", "image/jpeg"],
  [".png", "image/png"]
]);

interface MediaPreviewFile {
  filePath: string;
  contentType: string;
  size: number;
}

// resolveExerciseMediaPreviewPath maps a request path to a file inside the
// dataset media directories. Only the two known directories and the image
// extensions shipped by the dataset are accepted; anything else, including
// traversal attempts and symlinks escaping the root, resolves to null.
export const resolveExerciseMediaPreviewPath = (requestURL: string | undefined): MediaPreviewFile | null => {
  if (!requestURL) {
    return null;
  }

  const pathname = requestURL.split("?")[0].split("#")[0];
  if (!pathname.startsWith(`${EXERCISE_MEDIA_PREVIEW_PREFIX}/`)) {
    return null;
  }

  let decoded: string;
  try {
    decoded = decodeURIComponent(pathname.slice(EXERCISE_MEDIA_PREVIEW_PREFIX.length + 1));
  } catch {
    return null;
  }

  if (!decoded || decoded.includes("\0") || decoded.includes("\\")) {
    return null;
  }

  const segments = decoded.split("/");
  if (segments.length !== 2) {
    return null;
  }

  const [directory, filename] = segments;
  if (!ALLOWED_DIRECTORIES.has(directory) || !filename || filename.startsWith(".")) {
    return null;
  }

  const contentType = CONTENT_TYPES.get(extname(filename).toLowerCase());
  if (!contentType) {
    return null;
  }

  try {
    const mediaRoot = realpathSync(resolve(DATASET_MEDIA_ROOT, directory));
    const filePath = realpathSync(resolve(mediaRoot, filename));
    if (!filePath.startsWith(mediaRoot + sep)) {
      return null;
    }
    const stats = statSync(filePath);
    if (!stats.isFile()) {
      return null;
    }
    return { filePath, contentType, size: stats.size };
  } catch {
    return null;
  }
};

// isLoopbackAddress keeps the preview off every non-local interface even when
// Vite is started with --host 0.0.0.0.
export const isLoopbackAddress = (address: string | undefined | null): boolean => {
  if (!address) {
    return false;
  }
  const normalized = address.startsWith("::ffff:") ? address.slice(7) : address;
  return normalized === "::1" || normalized.startsWith("127.");
};

type NextFunction = (error?: unknown) => void;

export const createExerciseMediaPreviewMiddleware = () => {
  return (req: IncomingMessage, res: ServerResponse, next: NextFunction): void => {
    const requestURL = req.url ?? "";
    const pathname = requestURL.split("?")[0].split("#")[0];
    if (!pathname.startsWith(`${EXERCISE_MEDIA_PREVIEW_PREFIX}/`)) {
      next();
      return;
    }

    if (!isLoopbackAddress(req.socket.remoteAddress)) {
      res.statusCode = 403;
      res.setHeader("Content-Type", "text/plain; charset=utf-8");
      res.setHeader("Cache-Control", "no-store");
      res.end("Exercise media preview is available only from localhost.");
      return;
    }

    if (req.method !== "GET" && req.method !== "HEAD") {
      res.statusCode = 405;
      res.setHeader("Allow", "GET, HEAD");
      res.end();
      return;
    }

    const file = resolveExerciseMediaPreviewPath(requestURL);
    if (!file) {
      res.statusCode = 404;
      res.setHeader("Content-Type", "text/plain; charset=utf-8");
      res.end("Not found.");
      return;
    }

    res.statusCode = 200;
    res.setHeader("Content-Type", file.contentType);
    res.setHeader("Content-Length", file.size);
    res.setHeader("Cache-Control", "no-store");
    res.setHeader("X-Content-Type-Options", "nosniff");
    res.setHeader("X-Robots-Tag", "noindex, nofollow");
    res.setHeader("Cross-Origin-Resource-Policy", "same-origin");

    if (req.method === "HEAD") {
      res.end();
      return;
    }

    const stream = createReadStream(file.filePath);
    stream.on("error", () => {
      if (!res.headersSent) {
        res.statusCode = 500;
        res.end("Could not read the preview file.");
      } else {
        res.destroy();
      }
    });
    stream.pipe(res);
  };
};

// exerciseMediaDevPreview is intentionally a serve-only plugin: it has no
// build-time counterpart and no preview-server hook, so `vite build` never
// copies or references the dataset media.
export const exerciseMediaDevPreview = (): Plugin => ({
  name: "ryze-exercise-media-dev-preview",
  apply: "serve",
  configureServer(server) {
    server.middlewares.use(createExerciseMediaPreviewMiddleware());
    server.config.logger.info(
      `[ryze] local exercise-media preview at ${EXERCISE_MEDIA_PREVIEW_PREFIX} (localhost only, never bundled)`
    );
  }
});
