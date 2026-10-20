import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@utils/http_client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@utils/http_client")>();
  return {
    ...actual,
    apiGet: vi.fn(),
    apiPost: vi.fn(),
    apiPatch: vi.fn(),
    downloadFile: vi.fn()
  };
});

import { apiGet, apiPost, downloadFile } from "@utils/http_client";

import {
  approveAdminTrainerApplication,
  downloadAdminApplicationDocument,
  fetchAdminApplicationDecisions,
  fetchAdminApplicationDocuments,
  rejectAdminTrainerApplication
} from "./admin_api";

const mockedGet = vi.mocked(apiGet);
const mockedPost = vi.mocked(apiPost);
const mockedDownload = vi.mocked(downloadFile);

beforeEach(() => {
  mockedGet.mockReset();
  mockedPost.mockReset();
  mockedDownload.mockReset();
});

describe("application review", () => {
  it("fetches the evidence documents of an application", async () => {
    mockedGet.mockResolvedValue({ documents: [] });

    await fetchAdminApplicationDocuments("app-1");

    expect(mockedGet).toHaveBeenCalledWith("/admin/trainer-applications/app-1/documents");
  });

  it("fetches the decision history of an application", async () => {
    mockedGet.mockResolvedValue({ decisions: [] });

    await fetchAdminApplicationDecisions("app-1");

    expect(mockedGet).toHaveBeenCalledWith("/admin/trainer-applications/app-1/decisions");
  });

  it("sends a trimmed reason with an approval decision", async () => {
    mockedPost.mockResolvedValue({});

    await approveAdminTrainerApplication("app-1", "  verified credentials  ");

    expect(mockedPost).toHaveBeenCalledWith("/admin/trainer-applications/app-1/approve", {
      reason: "verified credentials"
    });
  });

  it("omits an empty reason from a rejection decision", async () => {
    mockedPost.mockResolvedValue({});

    await rejectAdminTrainerApplication("app-1", "   ");

    expect(mockedPost).toHaveBeenCalledWith("/admin/trainer-applications/app-1/reject", {});
  });

  it("downloads one evidence document of an application", async () => {
    mockedDownload.mockResolvedValue(undefined);

    await downloadAdminApplicationDocument("app-1", "doc-1", "cert.pdf");

    expect(mockedDownload).toHaveBeenCalledWith(
      "/admin/trainer-applications/app-1/documents/doc-1/download",
      "cert.pdf"
    );
  });
});
