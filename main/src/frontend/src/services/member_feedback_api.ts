import { apiGet, apiPost } from "@utils/http_client";

// Approved member feedback as shown on the public Feedback page.
export interface PublishedFeedback {
  id: string;
  display_name: string;
  rating: number;
  message: string;
  published_at: string;
}

export interface PublishedFeedbackPage {
  feedback: PublishedFeedback[];
  pagination: { page: number; limit: number; total: number; total_pages: number };
}

// The member's own submission.
export interface OwnFeedback {
  id: string;
  rating: number;
  message: string;
  status: "pending" | "approved" | "rejected";
  created_at: string;
}

export interface FeedbackEligibility {
  // True once the member holds (or held) any plan, including a free one.
  eligible: boolean;
  // A submission still waiting for review; another cannot be sent until then.
  pending: OwnFeedback | null;
}

export const FEEDBACK_MESSAGE_MIN = 10;
export const FEEDBACK_MESSAGE_MAX = 1000;

export const fetchPublishedFeedback = (page = 1, limit = 6) =>
  apiGet<PublishedFeedbackPage>(`/feedback?page=${page}&limit=${limit}`);

export const fetchFeedbackEligibility = () => apiGet<FeedbackEligibility>("/me/feedback/eligibility");

export const submitFeedback = (rating: number, message: string) =>
  apiPost<OwnFeedback>("/me/feedback", { rating, message });
