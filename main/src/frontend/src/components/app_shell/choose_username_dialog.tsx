import { AtSign } from "lucide-react";
import { type FormEvent, type KeyboardEvent, useEffect, useId, useRef, useState } from "react";

import { useI18n } from "@/i18n/i18n";
import { chooseUsername, isValidUsername } from "@/services/profile_api";
import { ApiError } from "@utils/http_client";

import styles from "./choose_username_dialog.module.css";

const COPY = {
  en: {
    title: "Choose your @username",
    lead: "It is your reference on RYZE, and the name people see while you have no nickname. You can change it later in Account.",
    label: "@username",
    hint: "3 to 30 lowercase letters, digits, dots or underscores.",
    save: "Save",
    saving: "Saving…",
    invalid: "An @username has 3 to 30 lowercase letters, digits, dots or underscores.",
    taken: "This @username is already in use. Try another one.",
    generic: "Something went wrong. Please try again."
  },
  pt: {
    title: "Escolhe a tua @referência",
    lead: "É a tua referência na RYZE e o nome que os outros veem enquanto não tiveres alcunha. Podes mudá-la depois em Conta.",
    label: "@referência",
    hint: "3 a 30 letras minúsculas, números, pontos ou underscores.",
    save: "Guardar",
    saving: "A guardar…",
    invalid: "A @referência tem 3 a 30 letras minúsculas, números, pontos ou underscores.",
    taken: "Esta @referência já está a ser usada. Experimenta outra.",
    generic: "Algo correu mal. Tenta de novo."
  }
} as const;

// ChooseUsernameDialog asks an account whose @username was generated (Google
// sign-up, an older or administrator-made account) to choose its own (owner
// decision, 2026-11-21). It stays until a valid one is saved.
export const ChooseUsernameDialog = ({ onChosen }: { onChosen: () => void }) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const ids = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [value, setValue] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  // Keep the keyboard inside the dialog (it cannot be dismissed).
  const trapFocus = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== "Tab") return;
    const first = inputRef.current;
    const last = buttonRef.current;
    if (!first || !last) return;
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  };

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!isValidUsername(value)) {
      setError(text.invalid);
      return;
    }
    setBusy(true);
    setError("");
    try {
      await chooseUsername(value);
      onChosen();
    } catch (caught) {
      const code = caught instanceof ApiError ? caught.code : "";
      setError(code === "USERNAME_TAKEN" ? text.taken : code === "INVALID_USERNAME" ? text.invalid : text.generic);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className={styles.backdrop}>
      <div className={styles.dialog} role="dialog" aria-modal="true" aria-labelledby={`${ids}-title`} aria-describedby={`${ids}-lead`} onKeyDown={trapFocus}>
        <span className={styles.icon} aria-hidden="true">
          <AtSign />
        </span>
        <h2 id={`${ids}-title`} className={styles.title}>
          {text.title}
        </h2>
        <p id={`${ids}-lead`} className={styles.lead}>
          {text.lead}
        </p>
        <form className={styles.form} onSubmit={(event) => void submit(event)} noValidate>
          <label className={styles.field}>
            <span>{text.label}</span>
            <input
              ref={inputRef}
              value={value}
              maxLength={31}
              autoCapitalize="none"
              autoComplete="username"
              spellCheck={false}
              aria-invalid={Boolean(error)}
              aria-describedby={`${ids}-hint`}
              onChange={(event) => setValue(event.target.value)}
            />
          </label>
          <p id={`${ids}-hint`} className={error ? styles.error : styles.hint} role={error ? "alert" : undefined}>
            {error || text.hint}
          </p>
          <button ref={buttonRef} type="submit" className={styles.save} disabled={busy}>
            {busy ? text.saving : text.save}
          </button>
        </form>
      </div>
    </div>
  );
};
