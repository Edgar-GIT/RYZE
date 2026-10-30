import { ApiError, apiUpload } from "@utils/http_client";

export interface ContactInput {
  name: string;
  email: string;
  subject: string;
  message: string;
}

export interface ContactAttachment {
  file: File;
}

const CONTACT_UNAVAILABLE = "CONTACT_UNAVAILABLE";
const CONTACT_MAX_ATTACHMENT_BYTES = 1 << 20;

export const CONTACT_ATTACHMENT_MAX_BYTES = CONTACT_MAX_ATTACHMENT_BYTES;

// submitContact sends the public contact form. Success is reported only after
// the backend accepted the message for delivery; provider failures surface as
// an error instead of a fabricated confirmation.
export const submitContact = async (
  input: ContactInput,
  attachment?: File | null
): Promise<void> => {
  if (attachment && attachment.size > CONTACT_MAX_ATTACHMENT_BYTES) {
    throw new ApiError(400, "ATTACHMENT_TOO_LARGE", "The attachment is too large.");
  }

  const form = new FormData();
  form.set("name", input.name);
  form.set("email", input.email);
  form.set("subject", input.subject);
  form.set("message", input.message);
  if (attachment && attachment.size > 0) {
    form.set("attachment", attachment);
  }

  await apiUpload<{ status: string }>("/contact", form);
};

export const contactErrorMessage = (error: unknown): string => {
  if (error instanceof ApiError) {
    if (error.code === CONTACT_UNAVAILABLE) {
      return "The contact form is temporarily unavailable. Please email us directly.";
    }
    if (error.code === "NETWORK_ERROR") {
      return "Unable to reach the server. Please check your connection and try again.";
    }
  }
  return "We could not send your message. Please try again or email us directly.";
};
