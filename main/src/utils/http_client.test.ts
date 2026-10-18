import { describe, expect, it } from "vitest";

import { encodePathSegment } from "./http_client";

describe("encodePathSegment", () => {
  it("keeps plain identifiers unchanged", () => {
    expect(encodePathSegment("3fa85f64-5717-4562-b3fc-2c963f66afa6")).toBe(
      "3fa85f64-5717-4562-b3fc-2c963f66afa6"
    );
  });

  it("encodes reserved path and query characters", () => {
    expect(encodePathSegment("a/b c?d#e")).toBe("a%2Fb%20c%3Fd%23e");
  });

  it("encodes non-ASCII characters without throwing", () => {
    expect(encodePathSegment("café")).toBe("caf%C3%A9");
  });
});
