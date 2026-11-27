import { Link } from "react-router-dom";

import { LEGAL_PATHS } from "@/constants/legal";
import { useI18n } from "@/i18n/i18n";

import styles from "./health_consent_checkbox.module.css";

const COPY = {
  en: {
    label: "I give my explicit consent for RYZE to process my health data (such as weight, measurements, injuries and health conditions) to prepare and track my training and nutrition.",
    more: "You can withdraw it at any time in Account → Privacy; that erases this data.",
    policy: "Privacy Policy"
  },
  pt: {
    label: "Dou o meu consentimento explícito para a RYZE tratar os meus dados de saúde (como peso, medidas, lesões e condições de saúde) para preparar e acompanhar o meu treino e a minha nutrição.",
    more: "Podes retirá-lo a qualquer momento em Conta → Privacidade; isso apaga estes dados.",
    policy: "Política de Privacidade"
  }
} as const;

// HealthConsentCheckbox is the explicit consent for health data (GDPR art.
// 9(2)(a), owner request 2026-11-24), unticked by default, asked before the
// Premium Level 2 questionnaire or a body measurement is sent.
export const HealthConsentCheckbox = ({
  checked,
  onChange,
  invalid = false
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  invalid?: boolean;
}) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  return (
    <label className={invalid ? `${styles.consent} ${styles.invalid}` : styles.consent}>
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} aria-invalid={invalid} />
      <span>
        {text.label}{" "}
        <small>
          {text.more}{" "}
          <Link to={LEGAL_PATHS.privacy} target="_blank" rel="noopener">
            {text.policy}
          </Link>
        </small>
      </span>
    </label>
  );
};
