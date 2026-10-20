import { Download, FileText, Loader2, Trash2, Upload } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Button } from "@/components/button/button";
import { formatAdminDate } from "@/services/admin_api";
import {
  APPLICATION_DOCUMENT_KIND_LABELS,
  ApplicationDocumentKind,
  deleteTrainerApplicationDocument,
  downloadTrainerApplicationDocument,
  fetchTrainerApplicationDocuments,
  uploadTrainerApplicationDocument,
  type ApplicationDocumentKindValue,
  type TrainerApplicationDocument
} from "@/services/trainer_api";
import { joinClassNames } from "@utils/class_names";
import { formatBytes } from "@utils/format_bytes";
import { ApiError } from "@utils/http_client";

import styles from "./trainer_application_documents.module.css";

const KIND_OPTIONS: ApplicationDocumentKindValue[] = [
  ApplicationDocumentKind.CERTIFICATION,
  ApplicationDocumentKind.IDENTIFICATION,
  ApplicationDocumentKind.INSURANCE,
  ApplicationDocumentKind.OTHER
];

// Server document codes are mapped to plain wording; raw tokens are never
// displayed. A state conflict means the application left PENDING while this
// panel was open.
const documentErrorMessage = (reason: unknown, fallback: string): string => {
  if (reason instanceof ApiError) {
    switch (reason.code) {
      case "DOCUMENT_TYPE_NOT_ALLOWED":
        return "Only PDF, JPEG and PNG files are accepted.";
      case "DOCUMENT_TOO_LARGE":
        return "This file is larger than the allowed document size.";
      case "APPLICATION_STATE_CONFLICT":
        return "The application is no longer pending, so documents can no longer be changed.";
      case "APPLICATION_NOT_FOUND":
        return "No pending application was found for your account.";
      default:
        return reason.message || fallback;
    }
  }
  return fallback;
};

// TrainerApplicationDocuments is the applicant-side evidence panel. Documents
// are optional and editable only while the application is PENDING; every
// action is delegated to the backend, which re-checks ownership and state.
export const TrainerApplicationDocuments = () => {
  const [documents, setDocuments] = useState<TrainerApplicationDocument[]>([]);
  const [loading, setLoading] = useState(true);
  const [listError, setListError] = useState("");
  const [actionError, setActionError] = useState("");

  const [kind, setKind] = useState<ApplicationDocumentKindValue>(ApplicationDocumentKind.CERTIFICATION);
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setListError("");
    try {
      const result = await fetchTrainerApplicationDocuments();
      setDocuments(result.documents);
    } catch (reason) {
      setListError(documentErrorMessage(reason, "Unable to load your documents."));
      setDocuments([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const upload = async () => {
    if (!file) {
      return;
    }
    setUploading(true);
    setActionError("");
    try {
      await uploadTrainerApplicationDocument(kind, file);
      setFile(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
      await load();
    } catch (reason) {
      setActionError(documentErrorMessage(reason, "Unable to upload the document."));
    } finally {
      setUploading(false);
    }
  };

  const download = async (document: TrainerApplicationDocument) => {
    setDownloadingId(document.id);
    setActionError("");
    try {
      await downloadTrainerApplicationDocument(document.id, document.original_filename);
    } catch (reason) {
      setActionError(documentErrorMessage(reason, "Unable to download the document."));
    } finally {
      setDownloadingId(null);
    }
  };

  const remove = async (document: TrainerApplicationDocument) => {
    setDeletingId(document.id);
    setActionError("");
    try {
      await deleteTrainerApplicationDocument(document.id);
      setConfirmDeleteId(null);
      await load();
    } catch (reason) {
      setActionError(documentErrorMessage(reason, "Unable to remove the document."));
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className={styles.panel}>
      <div className={styles.heading}>
        <h2 className={styles.title}>Supporting documents</h2>
        <p className={styles.description}>
          Optional. You can attach certifications, identification or insurance as PDF, JPEG or PNG
          while your application is under review. Documents are private to the review team.
        </p>
      </div>

      <div className={styles.uploadRow}>
        <label className={styles.kindField}>
          <span className={styles.fieldLabel}>Document type</span>
          <select
            value={kind}
            onChange={(event) => setKind(event.target.value as ApplicationDocumentKindValue)}
            className={styles.select}
            aria-label="Document type"
          >
            {KIND_OPTIONS.map((option) => (
              <option key={option} value={option}>
                {APPLICATION_DOCUMENT_KIND_LABELS[option]}
              </option>
            ))}
          </select>
        </label>

        <label className={styles.fileField}>
          <span className={styles.fieldLabel}>File</span>
          <input
            ref={fileInputRef}
            type="file"
            accept=".pdf,.jpg,.jpeg,.png,application/pdf,image/jpeg,image/png"
            className={styles.fileInput}
            aria-label="Choose a document file"
            onChange={(event) => setFile(event.target.files?.[0] ?? null)}
          />
        </label>

        <Button
          variant="secondary"
          size="small"
          icon={uploading ? <Loader2 size={15} className={styles.spinning} /> : <Upload size={15} />}
          iconPosition="left"
          disabled={!file || uploading}
          onClick={() => void upload()}
        >
          {uploading ? "Uploading…" : "Upload document"}
        </Button>
      </div>

      {actionError ? (
        <p className={styles.error} role="alert">
          {actionError}
        </p>
      ) : null}

      {loading ? (
        <p className={styles.stateText}>Loading documents…</p>
      ) : listError ? (
        <div className={styles.stateBlock}>
          <p className={styles.stateText}>{listError}</p>
          <Button variant="ghost" size="small" onClick={() => void load()}>
            Retry
          </Button>
        </div>
      ) : documents.length === 0 ? (
        <p className={styles.stateText}>No supporting documents attached.</p>
      ) : (
        <ul className={styles.list}>
          {documents.map((document) => (
            <li key={document.id} className={styles.item}>
              <span className={styles.itemIcon}>
                <FileText size={16} aria-hidden="true" />
              </span>
              <span className={styles.itemInfo}>
                <span className={styles.itemName}>{document.original_filename}</span>
                <span className={styles.itemMeta}>
                  {APPLICATION_DOCUMENT_KIND_LABELS[document.kind as ApplicationDocumentKindValue] ??
                    document.kind}{" "}
                  · {formatBytes(document.byte_size)} · {formatAdminDate(document.created_at)}
                </span>
              </span>
              <span className={styles.itemActions}>
                <button
                  type="button"
                  className={styles.iconButton}
                  aria-label={`Download ${document.original_filename}`}
                  title="Download"
                  disabled={downloadingId === document.id}
                  onClick={() => void download(document)}
                >
                  <Download size={15} aria-hidden="true" />
                </button>
                {confirmDeleteId === document.id ? (
                  <>
                    <button
                      type="button"
                      className={joinClassNames(styles.iconButton, styles.confirmDelete)}
                      disabled={deletingId === document.id}
                      onClick={() => void remove(document)}
                    >
                      {deletingId === document.id ? "Removing…" : "Confirm remove"}
                    </button>
                    <button
                      type="button"
                      className={styles.iconButton}
                      onClick={() => setConfirmDeleteId(null)}
                    >
                      Cancel
                    </button>
                  </>
                ) : (
                  <button
                    type="button"
                    className={styles.iconButton}
                    aria-label={`Remove ${document.original_filename}`}
                    title="Remove"
                    onClick={() => setConfirmDeleteId(document.id)}
                  >
                    <Trash2 size={15} aria-hidden="true" />
                  </button>
                )}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};
