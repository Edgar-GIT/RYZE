import { useId } from "react";

import { useI18n } from "@/i18n/i18n";

import styles from "./checkout_tax_id_field.module.css";
import { isValidOptionalTaxId } from "./tax_id";

interface CheckoutTaxIdFieldProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

const COPY = {
  en: {
    label: "Taxpayer number (NIF), optional",
    hint: "Add it to show it on your purchase receipt. Without it, the receipt is issued to a final consumer.",
    invalid: "This NIF is not valid. Check the nine digits or leave the field empty."
  },
  pt: {
    label: "NIF (opcional)",
    hint: "Indica-o para constar no comprovativo da compra. Sem NIF, o comprovativo é emitido a consumidor final.",
    invalid: "Este NIF não é válido. Confirma os nove dígitos ou deixa o campo vazio."
  }
} as const;

// CheckoutTaxIdField is the optional NIF input shared by every checkout
// surface (real and Test Mode). It never blocks a purchase on its own: the
// checkout keeps the pay button disabled while a typed NIF is invalid, and the
// backend validates it again.
export const CheckoutTaxIdField = ({ value, onChange, disabled = false }: CheckoutTaxIdFieldProps) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const inputId = useId();
  const hintId = `${inputId}-hint`;
  const invalid = !isValidOptionalTaxId(value);

  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={inputId}>
        {text.label}
      </label>
      <input
        id={inputId}
        className={styles.input}
        type="text"
        inputMode="numeric"
        autoComplete="off"
        spellCheck={false}
        maxLength={14}
        value={value}
        disabled={disabled}
        aria-describedby={hintId}
        aria-invalid={invalid}
        placeholder="123456789"
        onChange={(event) => onChange(event.target.value)}
      />
      <p id={hintId} className={invalid ? styles.error : styles.hint} aria-live="polite">
        {invalid ? text.invalid : text.hint}
      </p>
    </div>
  );
};
