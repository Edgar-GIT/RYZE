import { createServer, type Server } from "node:http";
import { afterEach, describe, expect, it } from "vitest";

import {
  createExerciseMediaPreviewMiddleware,
  isDatasetMediaRequest,
  isLoopbackAddress,
  resolveExerciseMediaPreviewPath
} from "./exercise_media_preview";

const PREVIEW_GIF = "/dev-exercise-media/videos/0001-2gPfomN.gif";
const PREVIEW_IMAGE = "/dev-exercise-media/images/0001-2gPfomN.jpg";

let server: Server | null = null;

afterEach(async () => {
  if (server) {
    await new Promise<void>((resolve) => server?.close(() => resolve()));
    server = null;
  }
});

const startServer = async (previewEnabled = true): Promise<string> => {
  const middleware = createExerciseMediaPreviewMiddleware({ enabled: previewEnabled });
  server = createServer((req, res) => {
    middleware(req, res, () => {
      res.statusCode = 204;
      res.end();
    });
  });
  await new Promise<void>((resolve) => server?.listen(0, "127.0.0.1", () => resolve()));
  const address = server.address();
  if (!address || typeof address === "string") {
    throw new Error("test server did not expose a TCP port");
  }
  return `http://127.0.0.1:${address.port}`;
};

describe("resolveExerciseMediaPreviewPath", () => {
  it("resolves the shipped GIF and image with their content types", () => {
    expect(resolveExerciseMediaPreviewPath(PREVIEW_GIF)).toMatchObject({
      contentType: "image/gif"
    });
    expect(resolveExerciseMediaPreviewPath(PREVIEW_IMAGE)).toMatchObject({
      contentType: "image/jpeg"
    });
  });

  it("ignores query strings and fragments", () => {
    expect(resolveExerciseMediaPreviewPath(`${PREVIEW_GIF}?v=1#frame`)).not.toBeNull();
  });

  it("rejects traversal, unknown directories and non-media files", () => {
    expect(resolveExerciseMediaPreviewPath("/dev-exercise-media/images/../data/exercises.json")).toBeNull();
    expect(
      resolveExerciseMediaPreviewPath("/dev-exercise-media/images/%2e%2e%2fdata%2fexercises.json")
    ).toBeNull();
    expect(resolveExerciseMediaPreviewPath("/dev-exercise-media/data/exercises.json")).toBeNull();
    expect(resolveExerciseMediaPreviewPath("/dev-exercise-media/videos/0001-2gPfomN.json")).toBeNull();
    expect(resolveExerciseMediaPreviewPath("/dev-exercise-media/videos/../videos/0001-2gPfomN.gif")).toBeNull();
  });

  it("rejects paths outside the preview prefix", () => {
    expect(resolveExerciseMediaPreviewPath("/images/0001-2gPfomN.jpg")).toBeNull();
    expect(resolveExerciseMediaPreviewPath(undefined)).toBeNull();
  });
});

describe("isLoopbackAddress", () => {
  it("accepts IPv4 and IPv6 loopback addresses", () => {
    expect(isLoopbackAddress("127.0.0.1")).toBe(true);
    expect(isLoopbackAddress("127.10.20.30")).toBe(true);
    expect(isLoopbackAddress("::1")).toBe(true);
    expect(isLoopbackAddress("::ffff:127.0.0.1")).toBe(true);
  });

  it("rejects LAN, public and missing addresses", () => {
    expect(isLoopbackAddress("192.168.1.20")).toBe(false);
    expect(isLoopbackAddress("8.8.8.8")).toBe(false);
    expect(isLoopbackAddress("::ffff:192.168.1.20")).toBe(false);
    expect(isLoopbackAddress(undefined)).toBe(false);
  });
});

describe("isDatasetMediaRequest", () => {
  it("recognises project-relative and /@fs/ dataset media URLs", () => {
    const projectRelative = "/main/src/resources/exercises/exercises-dataset-main/videos/0001-2gPfomN.gif";
    const filesystemPath = `${process.cwd()}${projectRelative}`;

    expect(isDatasetMediaRequest(projectRelative)).toBe(true);
    expect(isDatasetMediaRequest(`/@fs${filesystemPath}`)).toBe(true);
    expect(isDatasetMediaRequest(`/@fs${filesystemPath}?v=1`)).toBe(true);
  });

  it("does not block normal application assets or the explicit preview path", () => {
    expect(isDatasetMediaRequest("/main/src/frontend/src/main.tsx")).toBe(false);
    expect(isDatasetMediaRequest(PREVIEW_GIF)).toBe(false);
    expect(isDatasetMediaRequest("/api/v1/health")).toBe(false);
  });
});

describe("exercise media preview middleware", () => {
  it("serves a local GIF with private, non-cacheable headers", async () => {
    const baseURL = await startServer();

    const response = await fetch(`${baseURL}${PREVIEW_GIF}`);

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBe("image/gif");
    expect(response.headers.get("cache-control")).toBe("no-store");
    expect(response.headers.get("x-content-type-options")).toBe("nosniff");
    expect(response.headers.get("cross-origin-resource-policy")).toBe("same-origin");
    expect((await response.arrayBuffer()).byteLength).toBeGreaterThan(0);
  });

  it("supports HEAD without a body", async () => {
    const baseURL = await startServer();

    const response = await fetch(`${baseURL}${PREVIEW_IMAGE}`, { method: "HEAD" });

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBe("image/jpeg");
    expect(await response.text()).toBe("");
  });

  it("answers 404 for unknown or disallowed preview files", async () => {
    const baseURL = await startServer();

    expect((await fetch(`${baseURL}/dev-exercise-media/data/exercises.json`)).status).toBe(404);
    expect((await fetch(`${baseURL}/dev-exercise-media/images/missing.jpg`)).status).toBe(404);
  });

  it("keeps both the preview and the direct dataset URLs closed when disabled", async () => {
    const baseURL = await startServer(false);
    const projectRelative = "/main/src/resources/exercises/exercises-dataset-main/videos/0001-2gPfomN.gif";

    expect((await fetch(`${baseURL}${PREVIEW_GIF}`)).status).toBe(404);
    expect((await fetch(`${baseURL}${projectRelative}`)).status).toBe(404);
  });

  it("blocks direct project and /@fs/ access to the dataset media", async () => {
    const baseURL = await startServer();
    const projectRelative = "/main/src/resources/exercises/exercises-dataset-main/videos/0001-2gPfomN.gif";
    const filesystemPath = `${process.cwd()}${projectRelative}`;

    expect((await fetch(`${baseURL}${projectRelative}`)).status).toBe(404);
    expect((await fetch(`${baseURL}/@fs${filesystemPath}`)).status).toBe(404);
  });

  it("passes unrelated paths to the next middleware", async () => {
    const baseURL = await startServer();

    expect((await fetch(`${baseURL}/api/v1/health`)).status).toBe(204);
  });
});
