import { useEffect, useId, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Paperclip, SendHorizontal, Upload, X } from "lucide-react";

import { Button } from "@/components/button/button";
import { CONTACT_EMAIL } from "@/constants/contact";

import styles from "./email_contact_panel.module.css";

interface EmailContactPanelProps {
  onClose: () => void;
}

export const EmailContactPanel = ({ onClose }: EmailContactPanelProps) => {
  const [selectedFileName, setSelectedFileName] = useState("No file selected");
  const nameInputRef = useRef<HTMLInputElement>(null);
  const uploadInputId = useId();

  useEffect(() => {
    nameInputRef.current?.focus();
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };

    window.addEventListener("keydown", closeOnEscape);

    return () => {
      window.removeEventListener("keydown", closeOnEscape);
      document.body.style.overflow = previousOverflow;
    };
  }, [onClose]);

  return createPortal(
    <div className={styles.overlay} role="presentation" onMouseDown={onClose}>
      <div
        className={styles.panel}
        role="dialog"
        aria-modal="true"
        aria-labelledby="email-panel-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className={styles.header}>
          <div>
            <span className={styles.eyebrow}>Direct email</span>
            <h3 id="email-panel-title">Write to RYZE</h3>
          </div>
          <button
            className={styles.closeButton}
            type="button"
            onClick={onClose}
            aria-label="Close email panel"
          >
            <X aria-hidden="true" />
          </button>
        </div>

        <form className={styles.form} onSubmit={(event) => event.preventDefault()}>
          <label>
            <span>Name</span>
            <input ref={nameInputRef} type="text" name="name" autoComplete="name" />
          </label>
          <label>
            <span>Email</span>
            <input type="email" name="email" autoComplete="email" />
          </label>
          <label className={styles.fullWidth}>
            <span>Subject</span>
            <input type="text" name="subject" />
          </label>
          <label className={styles.fullWidth}>
            <span>Message</span>
            <textarea name="message" rows={7} />
          </label>

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

          <div className={styles.footer}>
            <small>{CONTACT_EMAIL}</small>
            <Button type="submit" icon={<SendHorizontal />}>
              Prepare message
            </Button>
          </div>
        </form>
      </div>
    </div>,
    document.body
  );
};
