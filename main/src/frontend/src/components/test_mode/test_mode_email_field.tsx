import { useId } from "react";

import { useI18n } from "@/i18n/i18n";

import { isValidTestEmail } from "./test_mode_email";
import styles from "./test_mode_email_field.module.css";

interface TestModeEmailFieldProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

const COPY = {
  en: {
    label: "Test email recipient",
    hint: "The purchase confirmation, the purchase receipt and the rest of the Test Mode emails are sent to this address. Use a real inbox you can open.",
    invalid: "Enter a valid email address, for example you@example.com."
  },
  pt: {
    label: "Email de teste",
    hint: "A confirmação da compra, o comprovativo e os restantes emails do Modo de Teste são enviados para este endereço. Usa um email real que consigas abrir.",
    invalid: "Introduz um email válido, por exemplo you@example.com."
  }
} as const;

// TestModeEmailField is the single Test Mode recipient input used by every
// checkout surface. It never decides whether a purchase may start: the shared
// checkout hook and the backend both re-check the address.
export const TestModeEmailField = ({ value, onChange, disabled = false }: TestModeEmailFieldProps) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const inputId = useId();
  const hintId = `${inputId}-hint`;
  const invalid = value.trim().length > 0 && !isValidTestEmail(value);

  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={inputId}>
        {text.label}
      </label>
      <input
        id={inputId}
        className={styles.input}
        type="email"
        inputMode="email"
        autoComplete="email"
        spellCheck={false}
        required
        value={value}
        disabled={disabled}
        aria-describedby={hintId}
        aria-invalid={invalid}
        placeholder="you@example.com"
        onChange={(event) => onChange(event.target.value)}
      />
      <p id={hintId} className={invalid ? styles.error : styles.hint} aria-live="polite">
        {invalid ? text.invalid : text.hint}
      </p>
    </div>
  );
};
