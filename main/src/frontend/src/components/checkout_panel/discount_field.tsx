import { Check, Loader2, Tag, X } from "lucide-react";
import { useState } from "react";

import { useI18n } from "@/i18n/i18n";
import { DISCOUNT_REASON, previewDiscountCode, type DiscountPreview } from "@/services/discount_codes_api";
import { ApiError } from "@utils/http_client";

import styles from "./checkout_panel.module.css";

const COPY = {
  en: { label: "Discount code", placeholder: "Enter your code", apply: "Apply", remove: "Remove code", off: (pct: number) => `${pct}% off` },
  pt: { label: "Código de desconto", placeholder: "Escreve o teu código", apply: "Aplicar", remove: "Retirar código", off: (pct: number) => `${pct}% de desconto` }
} as const;

// DiscountField checks a discount code against the plan (owner request,
// 2026-11-24); the server applies it again when the payment opens.
export const DiscountField = ({
  programId,
  disabled,
  formatPrice,
  onChange
}: {
  programId: string;
  disabled?: boolean;
  formatPrice: (minor: number) => string;
  onChange: (preview: DiscountPreview | null) => void;
}) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const [code, setCode] = useState("");
  const [applied, setApplied] = useState<DiscountPreview | null>(null);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState("");

  const apply = async () => {
    if (!code.trim()) return;
    setChecking(true);
    setError("");
    try {
      const preview = await previewDiscountCode(code, programId);
      setApplied(preview);
      onChange(preview);
    } catch (caught) {
      const reason = caught instanceof ApiError && caught.code === "DISCOUNT_INVALID" ? caught.details[0] ?? "" : "";
      setError(DISCOUNT_REASON[locale][reason] ?? DISCOUNT_REASON[locale].generic);
    } finally {
      setChecking(false);
    }
  };

  const remove = () => {
    setApplied(null);
    setCode("");
    onChange(null);
  };

  if (applied) {
    return (
      <div className={styles.discountApplied}>
        <Tag aria-hidden="true" />
        <span>
          <strong>{applied.code}</strong> · {text.off(applied.percent_off)} (−{formatPrice(applied.discount_minor_units)})
        </span>
        <button type="button" onClick={remove} disabled={disabled} aria-label={text.remove}>
          <X aria-hidden="true" />
        </button>
      </div>
    );
  }
  return (
    <div className={styles.discount}>
      <label htmlFor="checkout-discount">{text.label}</label>
      <div className={styles.discountRow}>
        <input
          id="checkout-discount"
          value={code}
          maxLength={32}
          autoCapitalize="characters"
          autoComplete="off"
          spellCheck={false}
          placeholder={text.placeholder}
          disabled={disabled || checking}
          onChange={(event) => {
            setCode(event.target.value.toUpperCase());
            setError("");
          }}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              void apply();
            }
          }}
          aria-invalid={Boolean(error)}
          aria-describedby={error ? "checkout-discount-error" : undefined}
        />
        <button type="button" onClick={() => void apply()} disabled={disabled || checking || !code.trim()}>
          {checking ? <Loader2 className={styles.spin} aria-hidden="true" /> : <Check aria-hidden="true" />} {text.apply}
        </button>
      </div>
      {error ? (
        <p id="checkout-discount-error" className={styles.discountError} role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
};
