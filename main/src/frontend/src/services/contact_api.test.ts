import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@utils/http_client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@utils/http_client")>();
  return {
    ...actual,
    apiUpload: vi.fn()
  };
});

import { ApiError, apiUpload } from "@utils/http_client";

import { contactErrorMessage, submitContact } from "./contact_api";

const mockedUpload = vi.mocked(apiUpload);

beforeEach(() => {
  mockedUpload.mockReset();
});

describe("submitContact", () => {
  it("posts every field and the optional attachment as multipart data", async () => {
    mockedUpload.mockResolvedValue({ status: "sent" });
    const attachment = new File(["pdf"], "brief.pdf", { type: "application/pdf" });

    await submitContact(
      { name: "Ada", email: "ada@example.com", subject: "Hello", message: "A message" },
      attachment
    );

    expect(mockedUpload).toHaveBeenCalledTimes(1);
    const [path, body] = mockedUpload.mock.calls[0];
    expect(path).toBe("/contact");
    expect(body).toBeInstanceOf(FormData);
    const form = body as FormData;
    expect(form.get("name")).toBe("Ada");
    expect(form.get("email")).toBe("ada@example.com");
    expect(form.get("subject")).toBe("Hello");
    expect(form.get("message")).toBe("A message");
    expect(form.get("attachment")).toBe(attachment);
  });

  it("omits an empty attachment", async () => {
    mockedUpload.mockResolvedValue({ status: "sent" });

    await submitContact(
      { name: "Ada", email: "ada@example.com", subject: "Hello", message: "A message" },
      new File([], "empty.pdf", { type: "application/pdf" })
    );

    const form = mockedUpload.mock.calls[0][1] as FormData;
    expect(form.get("attachment")).toBeNull();
  });

  it("rejects an oversized attachment before making a request", async () => {
    const oversized = new File([new Uint8Array((1 << 20) + 1)], "large.pdf", {
      type: "application/pdf"
    });

    await expect(
      submitContact({ name: "Ada", email: "ada@example.com", subject: "Hello", message: "A message" }, oversized)
    ).rejects.toMatchObject({ code: "ATTACHMENT_TOO_LARGE" });
    expect(mockedUpload).not.toHaveBeenCalled();
  });
});

describe("contactErrorMessage", () => {
  it("maps an unavailable backend to an honest fallback", () => {
    expect(contactErrorMessage(new ApiError(503, "CONTACT_UNAVAILABLE", "unavailable"))).toContain(
      "temporarily unavailable"
    );
  });

  it("does not expose a raw server message", () => {
    expect(contactErrorMessage(new ApiError(500, "INTERNAL_ERROR", "database exploded"))).not.toContain(
      "database exploded"
    );
  });
});
