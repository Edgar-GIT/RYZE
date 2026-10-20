import { createReadStream, realpathSync, statSync } from "node:fs";
import type { IncomingMessage, ServerResponse } from "node:http";
import { extname, resolve, sep } from "node:path";

import type { Plugin } from "vite";

// The downloaded dataset media is third-party content (© Gym visual). It is
// served only by this Vite development middleware, only to loopback clients,
// and it is never imported by the application, so it cannot enter the
// production bundle or the static build output.
const DATASET_MEDIA_ROOT = resolve(
  process.cwd(),
  "main/src/resources/exercises/exercises-dataset-main"
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

const decodedPathname = (requestURL: string): string | null => {
  const pathname = requestURL.split("?")[0].split("#")[0];
  try {
    return decodeURIComponent(pathname);
  } catch {
    return null;
  }
};

// isDatasetMediaRequest recognises both the plain project-relative URL and the
// /@fs/ URL Vite uses for filesystem access. Without this block the Vite dev
// server would serve the dataset media directly from the project root, which
// would bypass both the loopback restriction and the explicit preview path.
export const isDatasetMediaRequest = (requestURL: string): boolean => {
  const pathname = decodedPathname(requestURL);
  if (!pathname) {
    return false;
  }

  const datasetRoot = resolve(DATASET_MEDIA_ROOT);
  let candidate: string;
  if (pathname.startsWith("/@fs/")) {
    const filesystemPath = pathname.slice("/@fs/".length);
    candidate = resolve(filesystemPath.startsWith("/") ? filesystemPath : `/${filesystemPath}`);
  } else {
    candidate = resolve(process.cwd(), `.${pathname}`);
  }

  if (candidate === datasetRoot || candidate.startsWith(datasetRoot + sep)) {
    return true;
  }

  try {
    const realDatasetRoot = realpathSync(datasetRoot);
    const realCandidate = realpathSync(candidate);
    return realCandidate === realDatasetRoot || realCandidate.startsWith(realDatasetRoot + sep);
  } catch {
    return false;
  }
};

type NextFunction = (error?: unknown) => void;

export interface ExerciseMediaPreviewOptions {
  // The preview endpoint is registered only when the local media flag and the
  // dedicated base path are both configured. The direct-file block stays active
  // in every development session so Vite can never serve the dataset itself.
  enabled?: boolean;
}

export const createExerciseMediaPreviewMiddleware = (options: ExerciseMediaPreviewOptions = {}) => {
  const previewEnabled = options.enabled ?? true;
  return (req: IncomingMessage, res: ServerResponse, next: NextFunction): void => {
    const requestURL = req.url ?? "";
    const pathname = requestURL.split("?")[0].split("#")[0];
    if (!pathname.startsWith(`${EXERCISE_MEDIA_PREVIEW_PREFIX}/`)) {
      if (isDatasetMediaRequest(requestURL)) {
        res.statusCode = 404;
        res.setHeader("Content-Type", "text/plain; charset=utf-8");
        res.setHeader("Cache-Control", "no-store");
        res.end("Not found.");
        return;
      }
      next();
      return;
    }

    if (!previewEnabled) {
      res.statusCode = 404;
      res.setHeader("Content-Type", "text/plain; charset=utf-8");
      res.setHeader("Cache-Control", "no-store");
      res.end("Not found.");
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
export const exerciseMediaDevPreview = (options: ExerciseMediaPreviewOptions = {}): Plugin => ({
  name: "ryze-exercise-media-dev-preview",
  apply: "serve",
  configureServer(server) {
    const previewEnabled = options.enabled ?? true;
    server.middlewares.use(createExerciseMediaPreviewMiddleware({ enabled: previewEnabled }));
    if (previewEnabled) {
      server.config.logger.info(
        `[ryze] local exercise-media preview at ${EXERCISE_MEDIA_PREVIEW_PREFIX} (localhost only, never bundled)`
      );
    } else {
      server.config.logger.info(
        "[ryze] exercise-media preview disabled; set EXERCISE_MEDIA_ENABLED=true and EXERCISE_MEDIA_BASE_PATH=/dev-exercise-media in the local .env"
      );
    }
  }
});
