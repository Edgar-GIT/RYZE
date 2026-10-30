import { useId, useState } from "react";
import type { FormEvent } from "react";
import { Paperclip, SendHorizontal, Upload } from "lucide-react";

import { Button } from "@/components/button/button";
import {
  CONTACT_ATTACHMENT_MAX_BYTES,
  contactErrorMessage,
  submitContact
} from "@/services/contact_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./contact_form.module.css";

const ALLOWED_ATTACHMENT_TYPES = new Set([
  "image/png",
  "image/jpeg",
  "image/webp",
  "application/pdf"
]);

type SubmitStatus = "idle" | "submitting" | "success" | "error";

export const ContactForm = () => {
  const [selectedFileName, setSelectedFileName] = useState("No file selected");
  const [status, setStatus] = useState<SubmitStatus>("idle");
  const [feedback, setFeedback] = useState("");
  const uploadInputId = useId();

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (status === "submitting") {
      return;
    }

    const form = event.currentTarget;
    const data = new FormData(form);
    const attachment = data.get("attachment");
    if (attachment instanceof File && attachment.size > 0) {
      if (attachment.size > CONTACT_ATTACHMENT_MAX_BYTES) {
        setStatus("error");
        setFeedback("Attachments must be 1 MB or smaller.");
        return;
      }
      if (!ALLOWED_ATTACHMENT_TYPES.has(attachment.type)) {
        setStatus("error");
        setFeedback("Please attach a PNG, JPEG, WebP image or PDF.");
        return;
      }
    }

    setStatus("submitting");
    setFeedback("");
    try {
      await submitContact(
        {
          name: String(data.get("name") ?? ""),
          email: String(data.get("email") ?? ""),
          subject: String(data.get("subject") ?? ""),
          message: String(data.get("message") ?? "")
        },
        attachment instanceof File ? attachment : null
      );
      form.reset();
      setSelectedFileName("No file selected");
      setStatus("success");
      setFeedback("Message sent. We usually reply within 24 hours.");
    } catch (error) {
      setStatus("error");
      setFeedback(contactErrorMessage(error));
    }
  };

  return (
    <form
      className={styles.form}
      onSubmit={(event) => void handleSubmit(event)}
      aria-busy={status === "submitting"}
    >
      <header className={styles.header}>
        <h2>Write to RYZE</h2>
      </header>

      <div className={styles.fields}>
        <label>
          <span>Name</span>
          <input
            type="text"
            name="name"
            autoComplete="name"
            placeholder="Your name"
            required
            maxLength={100}
          />
        </label>
        <label>
          <span>Email</span>
          <input
            type="email"
            name="email"
            autoComplete="email"
            placeholder="you@email.com"
            required
            maxLength={254}
          />
        </label>
        <label className={styles.fullWidth}>
          <span>Subject</span>
          <input
            type="text"
            name="subject"
            placeholder="How can we help?"
            required
            maxLength={200}
          />
        </label>
        <label className={styles.fullWidth}>
          <span>Message</span>
          <textarea
            name="message"
            rows={5}
            placeholder="Tell us a bit more…"
            required
            maxLength={5000}
          />
        </label>
        <input
          className={styles.honeypot}
          type="text"
          name="website"
          tabIndex={-1}
          autoComplete="off"
          aria-hidden="true"
        />

        <div className={styles.uploadArea}>
          <div className={styles.uploadCopy}>
            <Paperclip aria-hidden="true" />
            <div>
              <span>Attachment</span>
              <small>{selectedFileName}</small>
            </div>
          </div>
          <label className={styles.uploadButton} htmlFor={uploadInputId}>
            <Upload aria-hidden="true" />
            <span>Choose file</span>
          </label>
          <input
            id={uploadInputId}
            type="file"
            name="attachment"
            accept="image/png,image/jpeg,image/webp,application/pdf"
            onChange={(event) => {
              const fileName = event.target.files?.[0]?.name;
              setSelectedFileName(fileName ?? "No file selected");
            }}
          />
        </div>
      </div>

      <div className={styles.footer}>
        {feedback ? (
          <p
            className={joinClassNames(
              styles.feedback,
              status === "success" ? styles.feedbackSuccess : styles.feedbackError
            )}
            role="status"
            aria-live="polite"
          >
            {feedback}
          </p>
        ) : null}
        <Button
          type="submit"
          size="large"
          className={styles.submit}
          icon={<SendHorizontal />}
          disabled={status === "submitting"}
        >
          {status === "submitting" ? "Sending…" : "Send Message"}
        </Button>
        <p className={styles.note}>We usually reply within 24 hours.</p>
      </div>
    </form>
  );
};
