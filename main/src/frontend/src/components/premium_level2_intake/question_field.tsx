import { PREMIUM_LEVEL2_PT } from "@/i18n/catalogs/premium_level2.pt";
import { QUESTIONNAIRE_PT } from "@/i18n/catalogs/questionnaire.pt";
import { useGettext } from "@/i18n/i18n";
import type { PremiumLevel2Question } from "@/services/premium_level_2_api";
import type { PremiumLevel2AnswerValue } from "@/services/premium_level_2_format";
import { joinClassNames } from "@utils/class_names";

import styles from "./question_field.module.css";

type AnswerValue = PremiumLevel2AnswerValue;

// QuestionField renders one question of the server-owned Premium Level 2
// questionnaire catalogue (select, multi-select, yes/no, number, text, long
// text), shared by the questionnaire before payment and the request page.
export const QuestionField = ({
  question,
  value,
  error,
  disabled,
  onChange
}: {
  question: PremiumLevel2Question;
  value: AnswerValue | undefined;
  error: string;
  disabled: boolean;
  onChange: (value: AnswerValue) => void;
}) => {
  const tr = useGettext(PREMIUM_LEVEL2_PT, QUESTIONNAIRE_PT);
  const describedBy = error ? `${question.key}-error` : undefined;

  if (question.type === "multiselect") {
    const selected = Array.isArray(value) ? value : [];
    return (
      <fieldset className={styles.field} aria-describedby={describedBy} aria-invalid={Boolean(error)}>
        <legend className={styles.label}>
          {tr(question.label)}
          {question.required ? <span className={styles.required}>{tr("Required")}</span> : null}
        </legend>
        {question.help ? <p className={styles.help}>{tr(question.help)}</p> : null}
        <div className={styles.optionGrid}>
          {question.options?.map((option) => (
            <label key={option.value} className={styles.option}>
              <input
                type="checkbox"
                checked={selected.includes(option.value)}
                disabled={disabled}
                onChange={(event) =>
                  onChange(
                    event.target.checked
                      ? [...selected, option.value]
                      : selected.filter((entry) => entry !== option.value)
                  )
                }
              />
              <span>{tr(option.label)}</span>
            </label>
          ))}
        </div>
        {error ? (
          <p className={styles.fieldError} id={describedBy}>
            {tr(error)}
          </p>
        ) : null}
      </fieldset>
    );
  }

  if (question.type === "boolean") {
    return (
      <div className={styles.field}>
        <label className={styles.option}>
          <input
            type="checkbox"
            checked={value === true}
            disabled={disabled}
            onChange={(event) => onChange(event.target.checked)}
          />
          <span>{tr(question.label)}</span>
        </label>
        {question.help ? <p className={styles.help}>{tr(question.help)}</p> : null}
        {error ? (
          <p className={styles.fieldError} id={describedBy}>
            {tr(error)}
          </p>
        ) : null}
      </div>
    );
  }

  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={question.key}>
        {tr(question.label)}
        {question.required ? <span className={styles.required}>{tr("Required")}</span> : null}
      </label>
      {question.help ? <p className={styles.help}>{tr(question.help)}</p> : null}
      {question.type === "textarea" ? (
        <textarea
          id={question.key}
          className={joinClassNames(styles.input, styles.textarea, error && styles.inputError)}
          value={typeof value === "string" ? value : ""}
          placeholder={question.placeholder}
          maxLength={question.max_length || undefined}
          disabled={disabled}
          aria-invalid={Boolean(error)}
          aria-describedby={describedBy}
          onChange={(event) => onChange(event.target.value)}
        />
      ) : question.type === "select" ? (
        <select
          id={question.key}
          className={joinClassNames(styles.input, error && styles.inputError)}
          value={typeof value === "string" ? value : ""}
          disabled={disabled}
          aria-invalid={Boolean(error)}
          aria-describedby={describedBy}
          onChange={(event) => onChange(event.target.value)}
        >
          <option value="">{tr("Choose an option")}</option>
          {question.options?.map((option) => (
            <option key={option.value} value={option.value}>
              {tr(option.label)}
            </option>
          ))}
        </select>
      ) : (
        <input
          id={question.key}
          className={joinClassNames(styles.input, error && styles.inputError)}
          type={question.type === "number" ? "number" : "text"}
          value={value === undefined || value === null ? "" : String(value)}
          placeholder={question.placeholder}
          min={question.min}
          max={question.max}
          step={question.step}
          maxLength={question.max_length || undefined}
          disabled={disabled}
          aria-invalid={Boolean(error)}
          aria-describedby={describedBy}
          onChange={(event) =>
            onChange(
              question.type === "number"
                ? event.target.value === ""
                  ? ""
                  : Number(event.target.value)
                : event.target.value
            )
          }
        />
      )}
      {error ? (
        <p className={styles.fieldError} id={describedBy}>
          {tr(error)}
        </p>
      ) : null}
    </div>
  );
};
