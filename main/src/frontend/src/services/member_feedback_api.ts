import { apiGet, apiPost, apiUpload, apiUrl, encodePathSegment } from "@utils/http_client";

// Approved member feedback as shown on the public Feedback page.
export interface PublishedFeedback {
  id: string;
  display_name: string;
  rating: number;
  message: string;
  /** The member attached a picture (v2.130.7). */
  has_image?: boolean;
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
  has_image?: boolean;
  created_at: string;
}

export interface FeedbackEligibility {
  // True once the member holds (or held) any plan, including a free one.
  eligible: boolean;
  // The member's feedback, pending or published: one per member (v2.130.7).
  // A rejected one is not returned, so the member can try again.
  pending: OwnFeedback | null;
}

export const FEEDBACK_MESSAGE_MIN = 10;
export const FEEDBACK_MESSAGE_MAX = 1000;

export const fetchPublishedFeedback = (page = 1, limit = 6) =>
  apiGet<PublishedFeedbackPage>(`/feedback?page=${page}&limit=${limit}`);

export const fetchFeedbackEligibility = () => apiGet<FeedbackEligibility>("/me/feedback/eligibility");

// submitFeedback sends the rating, message and an optional JPEG or PNG image.
export const submitFeedback = (rating: number, message: string, image?: File | null) => {
  if (!image) return apiPost<OwnFeedback>("/me/feedback", { rating, message });
  const body = new FormData();
  body.append("rating", String(rating));
  body.append("message", message);
  body.append("image", image);
  return apiUpload<OwnFeedback>("/me/feedback", body);
};

// The picture of approved feedback.
export const feedbackImageUrl = (id: string) => apiUrl(`/feedback/${encodePathSegment(id)}/image`);
