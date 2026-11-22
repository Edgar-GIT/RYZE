import { Camera, ImagePlus, Lock, RefreshCw, Trash2 } from "lucide-react";
import { useRef, useState, type ChangeEvent } from "react";

import { useI18n } from "@/i18n/i18n";
import { removeWorkoutPhoto, uploadWorkoutPhoto, workoutPhotoUrl } from "@/services/workout_sessions_api";
import { ApiError } from "@utils/http_client";

import styles from "./workout_photo.module.css";

const COPY = {
  en: {
    title: "Your workout photo",
    add: "Add a photo",
    addHint: "A photo of you at this workout, to see your evolution. JPEG or PNG.",
    private: "Only you can see it.",
    change: "Change photo",
    remove: "Remove",
    uploading: "Uploading…",
    alt: "Your photo of this workout",
    tooLarge: "This photo is too large. Try a smaller one.",
    invalid: "Use a JPEG or PNG photo.",
    failed: "We could not save the photo. Try again."
  },
  pt: {
    title: "A tua foto do treino",
    add: "Adicionar foto",
    addHint: "Uma foto tua neste treino, para veres a tua evolução. JPEG ou PNG.",
    private: "Só tu a vês.",
    change: "Trocar foto",
    remove: "Remover",
    uploading: "A enviar…",
    alt: "A tua foto deste treino",
    tooLarge: "Esta foto é demasiado grande. Experimenta uma mais pequena.",
    invalid: "Usa uma foto JPEG ou PNG.",
    failed: "Não foi possível guardar a foto. Tenta de novo."
  }
} as const;

interface WorkoutPhotoProps {
  sessionId: string;
  hasPhoto: boolean;
  // Whether the member can add, change or remove it.
  editable: boolean;
}

// WorkoutPhoto lets the member add a photo of themselves to a workout before
// saving it, and shows it when the workout is opened from the history
// (owner request, 2026-11-22, as in Lyfta).
export const WorkoutPhoto = ({ sessionId, hasPhoto, editable }: WorkoutPhotoProps) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const [present, setPresent] = useState(hasPhoto);
  const [version, setVersion] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  if (!present && !editable) return null;

  const choose = () => inputRef.current?.click();

  const upload = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    setBusy(true);
    setError("");
    try {
      await uploadWorkoutPhoto(sessionId, file);
      setPresent(true);
      setVersion(Date.now());
    } catch (failure) {
      const code = failure instanceof ApiError ? failure.code : "";
      setError(code === "IMAGE_TOO_LARGE" || code === "REQUEST_ENTITY_TOO_LARGE" ? text.tooLarge : code === "INVALID_IMAGE" ? text.invalid : text.failed);
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    setBusy(true);
    setError("");
    try {
      await removeWorkoutPhoto(sessionId);
      setPresent(false);
    } catch {
      setError(text.failed);
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className={styles.photo} aria-labelledby={`workout-photo-${sessionId}`}>
      <h2 id={`workout-photo-${sessionId}`} className={styles.title}>
        <Camera aria-hidden="true" /> {text.title}
      </h2>
      <input ref={inputRef} className={styles.input} type="file" accept="image/jpeg,image/png" onChange={(event) => void upload(event)} tabIndex={-1} aria-hidden="true" />
      {present ? (
        <div className={styles.frame}>
          <img src={workoutPhotoUrl(sessionId, version)} alt={text.alt} decoding="async" />
        </div>
      ) : (
        <button type="button" className={styles.drop} onClick={choose} disabled={busy}>
          <ImagePlus aria-hidden="true" />
          <strong>{busy ? text.uploading : text.add}</strong>
          <span>{text.addHint}</span>
        </button>
      )}
      <div className={styles.foot}>
        <span className={styles.private}>
          <Lock aria-hidden="true" /> {text.private}
        </span>
        {present && editable ? (
          <span className={styles.actions}>
            <button type="button" className={styles.action} onClick={choose} disabled={busy}>
              <RefreshCw aria-hidden="true" /> {busy ? text.uploading : text.change}
            </button>
            <button type="button" className={styles.action} onClick={() => void remove()} disabled={busy}>
              <Trash2 aria-hidden="true" /> {text.remove}
            </button>
          </span>
        ) : null}
      </div>
      {error ? (
        <p className={styles.error} role="alert">
          {error}
        </p>
      ) : null}
    </section>
  );
};
