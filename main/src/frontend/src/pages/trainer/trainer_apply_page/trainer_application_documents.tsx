import { Download, FileText, Loader2, Trash2, Upload } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Button } from "@/components/button/button";
import {
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
import { useI18n } from "@/i18n/i18n";

const KIND_OPTIONS: ApplicationDocumentKindValue[] = [
  ApplicationDocumentKind.CERTIFICATION,
  ApplicationDocumentKind.IDENTIFICATION,
  ApplicationDocumentKind.INSURANCE,
  ApplicationDocumentKind.OTHER
];

// Server document codes are mapped to plain wording; raw tokens are never
// displayed. A state conflict means the application left PENDING while this
// panel was open.
const documentErrorMessage = (reason: unknown, fallback: string, t: ReturnType<typeof useI18n>["t"]): string => {
  if (reason instanceof ApiError) {
    switch (reason.code) {
      case "DOCUMENT_TYPE_NOT_ALLOWED":
        return t("account.trainerApply.errType");
      case "DOCUMENT_TOO_LARGE":
        return t("account.trainerApply.errSize");
      case "APPLICATION_STATE_CONFLICT":
        return t("account.trainerApply.errState");
      case "APPLICATION_NOT_FOUND":
        return t("account.trainerApply.errMissing");
      default:
        return fallback;
    }
  }
  return fallback;
};

const KIND_KEY = {
  certification: "account.trainerApply.kindCertification",
  identification: "account.trainerApply.kindIdentification",
  insurance: "account.trainerApply.kindInsurance",
  other: "account.trainerApply.kindOther"
} as const satisfies Record<ApplicationDocumentKindValue, string>;

// TrainerApplicationDocuments is the applicant-side evidence panel. Documents
// are optional and editable only while the application is PENDING; every
// action is delegated to the backend, which re-checks ownership and state.
export const TrainerApplicationDocuments = () => {
  const { t, formatDate } = useI18n();
  const kindLabel = (kind: string) =>
    kind in KIND_KEY ? t(KIND_KEY[kind as ApplicationDocumentKindValue]) : kind;
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
      setListError(documentErrorMessage(reason, t("account.trainerApply.errLoad"), t));
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
      setActionError(documentErrorMessage(reason, t("account.trainerApply.errUpload"), t));
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
      setActionError(documentErrorMessage(reason, t("account.trainerApply.errDownload"), t));
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
      setActionError(documentErrorMessage(reason, t("account.trainerApply.errRemove"), t));
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className={styles.panel}>
      <div className={styles.heading}>
        <h2 className={styles.title}>{t("account.trainerApply.docsTitle")}</h2>
        <p className={styles.description}>
          {t("account.trainerApply.docsText")}
        </p>
      </div>

      <div className={styles.uploadRow}>
        <label className={styles.kindField}>
          <span className={styles.fieldLabel}>{t("account.trainerApply.docType")}</span>
          <select
            value={kind}
            onChange={(event) => setKind(event.target.value as ApplicationDocumentKindValue)}
            className={styles.select}
            aria-label={t("account.trainerApply.docType")}
          >
            {KIND_OPTIONS.map((option) => (
              <option key={option} value={option}>
                {kindLabel(option)}
              </option>
            ))}
          </select>
        </label>

        <label className={styles.fileField}>
          <span className={styles.fieldLabel}>{t("account.trainerApply.file")}</span>
          <input
            ref={fileInputRef}
            type="file"
            accept=".pdf,.jpg,.jpeg,.png,application/pdf,image/jpeg,image/png"
            className={styles.fileInput}
            aria-label={t("account.trainerApply.chooseFile")}
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
          {uploading ? t("account.trainerApply.uploading") : t("account.trainerApply.upload")}
        </Button>
      </div>

      {actionError ? (
        <p className={styles.error} role="alert">
          {actionError}
        </p>
      ) : null}

      {loading ? (
        <p className={styles.stateText}>{t("account.trainerApply.loadingDocs")}</p>
      ) : listError ? (
        <div className={styles.stateBlock}>
          <p className={styles.stateText}>{listError}</p>
          <Button variant="ghost" size="small" onClick={() => void load()}>
            {t("account.trainerApply.retry")}
          </Button>
        </div>
      ) : documents.length === 0 ? (
        <p className={styles.stateText}>{t("account.trainerApply.noDocs")}</p>
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
                  {kindLabel(document.kind)} · {formatBytes(document.byte_size)} ·{" "}
                  {formatDate(document.created_at, { day: "2-digit", month: "short", year: "numeric" })}
                </span>
              </span>
              <span className={styles.itemActions}>
                <button
                  type="button"
                  className={styles.iconButton}
                  aria-label={t("account.trainerApply.downloadNamed", { name: document.original_filename })}
                  title={t("account.trainerApply.download")}
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
                      {deletingId === document.id ? t("account.trainerApply.removing") : t("account.trainerApply.confirmRemove")}
                    </button>
                    <button
                      type="button"
                      className={styles.iconButton}
                      onClick={() => setConfirmDeleteId(null)}
                    >
                      {t("account.trainerApply.cancel")}
                    </button>
                  </>
                ) : (
                  <button
                    type="button"
                    className={styles.iconButton}
                    aria-label={t("account.trainerApply.removeNamed", { name: document.original_filename })}
                    title={t("account.trainerApply.remove")}
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
