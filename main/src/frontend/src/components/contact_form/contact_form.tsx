import { useId, useState } from "react";
import { Paperclip, SendHorizontal, Upload } from "lucide-react";

import { Button } from "@/components/button/button";

import styles from "./contact_form.module.css";

export const ContactForm = () => {
  const [selectedFileName, setSelectedFileName] = useState("No file selected");
  const uploadInputId = useId();

  return (
    <form className={styles.form} onSubmit={(event) => event.preventDefault()} noValidate>
      <header className={styles.header}>
        <h2>Write to RYZE</h2>
      </header>

      <div className={styles.fields}>
        <label>
          <span>Name</span>
          <input type="text" name="name" autoComplete="name" placeholder="Your name" />
        </label>
        <label>
          <span>Email</span>
          <input type="email" name="email" autoComplete="email" placeholder="you@email.com" />
        </label>
        <label className={styles.fullWidth}>
          <span>Subject</span>
          <input type="text" name="subject" placeholder="How can we help?" />
        </label>
        <label className={styles.fullWidth}>
          <span>Message</span>
          <textarea name="message" rows={5} placeholder="Tell us a bit more…" />
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
      </div>

      <div className={styles.footer}>
        <Button type="submit" size="large" className={styles.submit} icon={<SendHorizontal />}>
          Send Message
        </Button>
        <p className={styles.note}>We usually reply within 24 hours.</p>
      </div>
    </form>
  );
};
