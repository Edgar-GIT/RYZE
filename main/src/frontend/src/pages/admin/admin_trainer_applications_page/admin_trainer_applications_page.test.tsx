import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/services/admin_api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/admin_api")>();
  return {
    ...actual,
    fetchAdminTrainerApplications: vi.fn(),
    approveAdminTrainerApplication: vi.fn(),
    rejectAdminTrainerApplication: vi.fn(),
    fetchAdminApplicationDocuments: vi.fn(),
    fetchAdminApplicationDecisions: vi.fn(),
    downloadAdminApplicationDocument: vi.fn()
  };
});

import {
  approveAdminTrainerApplication,
  fetchAdminApplicationDecisions,
  fetchAdminApplicationDocuments,
  fetchAdminTrainerApplications,
  rejectAdminTrainerApplication
} from "@/services/admin_api";

import { AdminTrainerApplicationsPage } from "./admin_trainer_applications_page";

const mockedFetchApplications = vi.mocked(fetchAdminTrainerApplications);
const mockedApprove = vi.mocked(approveAdminTrainerApplication);
const mockedReject = vi.mocked(rejectAdminTrainerApplication);
const mockedFetchDocuments = vi.mocked(fetchAdminApplicationDocuments);
const mockedFetchDecisions = vi.mocked(fetchAdminApplicationDecisions);

const application = {
  id: "app-1",
  status: "PENDING",
  user_id: "u1",
  user: {
    id: "u1",
    email: "jane@example.com",
    first_name: "Jane",
    last_name: "Doe",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z"
  },
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z"
};

const documentFixture = {
  id: "doc-1",
  kind: "certification",
  original_filename: "certification.pdf",
  content_type: "application/pdf",
  byte_size: 2048,
  created_at: "2026-01-01T00:00:00Z"
};

const decisionFixture = {
  id: "dec-1",
  application_id: "app-1",
  admin_id: "admin-2",
  from_status: "PENDING",
  to_status: "APPROVED",
  reason: "Credentials verified",
  created_at: "2026-01-02T00:00:00Z"
};

beforeEach(() => {
  mockedFetchApplications.mockReset().mockResolvedValue({
    applications: [application],
    pagination: { page: 1, limit: 20, total: 1, total_pages: 1 }
  });
  mockedApprove.mockReset().mockResolvedValue(application);
  mockedReject.mockReset().mockResolvedValue(application);
  mockedFetchDocuments.mockReset().mockResolvedValue({ documents: [documentFixture] });
  mockedFetchDecisions.mockReset().mockResolvedValue({ decisions: [decisionFixture] });
});

afterEach(cleanup);

describe("AdminTrainerApplicationsPage", () => {
  it("lists applications with the applicant name and email", async () => {
    render(<AdminTrainerApplicationsPage />);

    expect(await screen.findByText("Jane Doe")).toBeTruthy();
    expect(screen.getByText("jane@example.com")).toBeTruthy();
  });

  it("loads documents and decision history when a row is expanded", async () => {
    render(<AdminTrainerApplicationsPage />);
    await screen.findByText("Jane Doe");

    fireEvent.click(screen.getByRole("button", { name: /Review/ }));

    expect(await screen.findByText("certification.pdf")).toBeTruthy();
    expect(mockedFetchDocuments).toHaveBeenCalledWith("app-1");
    expect(mockedFetchDecisions).toHaveBeenCalledWith("app-1");
    expect(screen.getByText("Credentials verified")).toBeTruthy();
  });

  it("approves without a note through the quick action", async () => {
    render(<AdminTrainerApplicationsPage />);
    await screen.findByText("Jane Doe");

    fireEvent.click(screen.getByRole("button", { name: "Approve Jane Doe" }));

    await waitFor(() => {
      expect(mockedApprove).toHaveBeenCalledWith("app-1");
    });
  });

  it("sends the decision note with a reasoned approval", async () => {
    render(<AdminTrainerApplicationsPage />);
    await screen.findByText("Jane Doe");
    fireEvent.click(screen.getByRole("button", { name: /Review/ }));
    await screen.findByText("certification.pdf");

    fireEvent.change(screen.getByLabelText(/Decision note/), { target: { value: "Looks good" } });
    fireEvent.click(screen.getByRole("button", { name: "Approve with note" }));

    await waitFor(() => {
      expect(mockedApprove).toHaveBeenCalledWith("app-1", "Looks good");
    });
    expect(mockedReject).not.toHaveBeenCalled();
  });
});
