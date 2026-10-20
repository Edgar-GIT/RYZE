import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/services/trainer_api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/trainer_api")>();
  return {
    ...actual,
    fetchTrainerApplicationDocuments: vi.fn(),
    uploadTrainerApplicationDocument: vi.fn(),
    deleteTrainerApplicationDocument: vi.fn(),
    downloadTrainerApplicationDocument: vi.fn()
  };
});

import {
  deleteTrainerApplicationDocument,
  downloadTrainerApplicationDocument,
  fetchTrainerApplicationDocuments,
  uploadTrainerApplicationDocument
} from "@/services/trainer_api";
import { ApiError } from "@utils/http_client";

import { TrainerApplicationDocuments } from "./trainer_application_documents";

const mockedFetch = vi.mocked(fetchTrainerApplicationDocuments);
const mockedUpload = vi.mocked(uploadTrainerApplicationDocument);
const mockedDelete = vi.mocked(deleteTrainerApplicationDocument);
const mockedDownload = vi.mocked(downloadTrainerApplicationDocument);

const certification = {
  id: "doc-1",
  kind: "certification",
  original_filename: "certification.pdf",
  content_type: "application/pdf",
  byte_size: 2048,
  created_at: "2026-01-01T00:00:00Z"
};

beforeEach(() => {
  mockedFetch.mockReset().mockResolvedValue({ documents: [certification] });
  mockedUpload.mockReset().mockResolvedValue(certification);
  mockedDelete.mockReset().mockResolvedValue(undefined as never);
  mockedDownload.mockReset().mockResolvedValue(undefined);
});

afterEach(cleanup);

describe("TrainerApplicationDocuments", () => {
  it("lists the fetched documents with their kind and size", async () => {
    render(<TrainerApplicationDocuments />);

    expect(await screen.findByText("certification.pdf")).toBeTruthy();
    expect(screen.getByText(/Certification · 2 KB/)).toBeTruthy();
  });

  it("shows an empty state when nothing is attached", async () => {
    mockedFetch.mockResolvedValue({ documents: [] });

    render(<TrainerApplicationDocuments />);

    expect(await screen.findByText("No supporting documents attached.")).toBeTruthy();
  });

  it("uploads the chosen file with the selected kind and reloads the list", async () => {
    render(<TrainerApplicationDocuments />);
    await screen.findByText("certification.pdf");

    const file = new File(["abc"], "id.png", { type: "image/png" });
    fireEvent.change(screen.getByLabelText("Choose a document file"), { target: { files: [file] } });
    fireEvent.change(screen.getByLabelText("Document type"), { target: { value: "identification" } });
    fireEvent.click(screen.getByRole("button", { name: /Upload document/ }));

    await waitFor(() => {
      expect(mockedUpload).toHaveBeenCalledWith("identification", file);
    });
    await waitFor(() => {
      expect(mockedFetch).toHaveBeenCalledTimes(2);
    });
  });

  it("removes a document only after the confirmation step", async () => {
    render(<TrainerApplicationDocuments />);
    await screen.findByText("certification.pdf");

    fireEvent.click(screen.getByRole("button", { name: "Remove certification.pdf" }));
    fireEvent.click(screen.getByRole("button", { name: "Confirm remove" }));

    await waitFor(() => {
      expect(mockedDelete).toHaveBeenCalledWith("doc-1");
    });
  });

  it("maps a server document error code to plain wording", async () => {
    mockedUpload.mockRejectedValue(new ApiError(415, "DOCUMENT_TYPE_NOT_ALLOWED", "document_type_not_allowed"));
    render(<TrainerApplicationDocuments />);
    await screen.findByText("certification.pdf");

    const file = new File(["abc"], "notes.txt", { type: "text/plain" });
    fireEvent.change(screen.getByLabelText("Choose a document file"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: /Upload document/ }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Only PDF, JPEG and PNG files are accepted.");
    expect(screen.queryByText("document_type_not_allowed")).toBeNull();
  });

  it("downloads a document through the download endpoint", async () => {
    render(<TrainerApplicationDocuments />);
    await screen.findByText("certification.pdf");

    fireEvent.click(screen.getByRole("button", { name: "Download certification.pdf" }));

    await waitFor(() => {
      expect(mockedDownload).toHaveBeenCalledWith("doc-1", "certification.pdf");
    });
  });
});
