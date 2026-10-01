import { AlertTriangle, Lock, RefreshCw } from "lucide-react";
import { useMemo } from "react";

import { Button } from "@/components/button/button";
import type {
  QuestionnaireAnswers,
  QuestionnaireQuestion
} from "@/services/premium_level1_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./premium_questionnaire_form.module.css";

interface PremiumQuestionnaireFormProps {
  programId: string;
  questions: QuestionnaireQuestion[];
  /** The server's authoritative answer. Editable only while the server says so. */
  answers: QuestionnaireAnswers;
  locked: boolean;
  submitting: boolean;
  /** Field-level reasons from the last rejected submission, keyed by field. */
  fieldErrors: Record<string, string>;
  /** A transport or server failure that is not a field-level rejection. */
  errorMessage: string;
  onChange: (field: string, value: string | number | string[]) => void;
  onSubmit: () => void;
  onRetry: () => void;
}

// renderInput picks the control from the server-declared question type. The
// catalog is the single source of truth, so a new question type never needs a
// client release to be renderable.
const renderInput = (
  question: QuestionnaireQuestion,
  value: string | number | string[] | undefined,
  onChange: (value: string | number | string[]) => void
) => {
  const shared = {
    id: question.field,
    name: question.field,
    required: question.required,
    maxLength: question.max_length,
    "aria-describedby": question.help ? `${question.field}-help` : undefined
  } as const;

  if (question.type === "select") {
    return (
      <select
        {...shared}
        className={styles.control}
        value={typeof value === "string" ? value : ""}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">Select an option…</option>
        {(question.options ?? []).map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    );
  }

  if (question.type === "multi_select") {
    const selected = Array.isArray(value) ? value : [];
    return (
      <div className={styles.options}>
        {(question.options ?? []).map((option) => {
          const isChecked = selected.includes(option);
          return (
            <label
              key={option}
              className={joinClassNames(styles.option, isChecked && styles.optionSelected)}
            >
              <input
                type="checkbox"
                name={question.field}
                value={option}
                checked={isChecked}
                disabled={!isChecked && selected.length >= (question.max_entries ?? Number.MAX_SAFE_INTEGER)}
                onChange={(event) =>
                  onChange(
                    event.target.checked
                      ? [...selected, option]
                      : selected.filter((entry) => entry !== option)
                  )
                }
              />
              <span>{option}</span>
            </label>
          );
        })}
      </div>
    );
  }

  if (question.type === "number") {
    return (
      <input
        {...shared}
        className={styles.control}
        type="number"
        min={question.min}
        max={question.max}
        value={typeof value === "number" ? value : ""}
        onChange={(event) => {
          const parsed = Number(event.target.value);
          onChange(event.target.value === "" ? "" : Number.isNaN(parsed) ? "" : parsed);
        }}
      />
    );
  }

  if (question.type === "textarea") {
    return (
      <textarea
        {...shared}
        className={joinClassNames(styles.control, styles.textarea)}
        value={typeof value === "string" ? value : ""}
        onChange={(event) => onChange(event.target.value)}
      />
    );
  }

  return (
    <input
      {...shared}
      className={styles.control}
      type="text"
      minLength={question.min}
      maxLength={question.max}
      value={typeof value === "string" ? value : ""}
      onChange={(event) => onChange(event.target.value)}
    />
  );
};

// PremiumQuestionnaireForm renders the onboarding questionnaire. It is a pure
// presentational component: the parent owns loading, submission and the
// server-reported locked state, and the component never decides on its own that
// editing is allowed.
export const PremiumQuestionnaireForm = ({
  programId,
  questions,
  answers,
  locked,
  submitting,
  fieldErrors,
  errorMessage,
  onChange,
  onSubmit,
  onRetry
}: PremiumQuestionnaireFormProps) => {
  const incomplete = useMemo(
    () =>
      questions.some((question) => {
        if (!question.required) return false;
        const value = answers[question.field];
        if (Array.isArray(value)) return value.length === 0;
        return value === undefined || value === "";
      }),
    [questions, answers]
  );

  // A locked questionnaire is shown read-only. There is intentionally no
  // "regenerate" or "edit again" action: the intake belongs to a purchased
  // package and changing it would silently change a delivered plan.
  if (locked) {
    return (
      <section className={styles.locked} aria-labelledby={`locked-${programId}`}>
        <p className={styles.lockedTitle}>
          <Lock size={16} aria-hidden="true" />
          <span id={`locked-${programId}`}>Your answers are locked</span>
        </p>
        <p className={styles.lockedText}>
          This questionnaire was completed before your purchase was confirmed, so it is now part of your
          delivered package. Your training plan and nutrition programme are built from exactly these answers.
        </p>
      </section>
    );
  }

  if (errorMessage && questions.length === 0) {
    return (
      <section className={styles.failure}>
        <p className={styles.failureTitle}>
          <AlertTriangle size={16} aria-hidden="true" />
          We could not load the questionnaire
        </p>
        <p className={styles.failureText}>{errorMessage}</p>
        <div className={styles.actions}>
          <Button variant="secondary" size="small" onClick={onRetry} icon={<RefreshCw size={15} />}>
            Retry
          </Button>
        </div>
      </section>
    );
  }

  return (
    <section className={styles.form} aria-labelledby={`questionnaire-${programId}`}>
      <h2 className={styles.title} id={`questionnaire-${programId}`}>
        Nutrition questionnaire
      </h2>
      <p className={styles.intro}>
        These answers personalise the nutrition programme that ships with your package. You can review and
        update them until you complete checkout.
      </p>

      {errorMessage ? (
        <p className={styles.submitError} role="alert">
          {errorMessage}
        </p>
      ) : null}

      <div className={styles.fields}>
        {questions.map((question) => {
          const fieldError = fieldErrors[question.field];
          return (
            <fieldset key={question.field} className={styles.field}>
              <legend className={styles.label}>
                {question.label}
                {question.required ? <span aria-hidden="true"> *</span> : null}
              </legend>
              {question.help ? (
                <p className={styles.help} id={`${question.field}-help`}>
                  {question.help}
                </p>
              ) : null}
              {renderInput(question, answers[question.field], (value) => onChange(question.field, value))}
              {question.sensitive ? (
                <p className={styles.sensitive}>Used only to build your nutrition programme.</p>
              ) : null}
              {fieldError ? (
                <p className={styles.fieldError} role="alert">
                  {fieldError}
                </p>
              ) : null}
            </fieldset>
          );
        })}
      </div>

      <div className={styles.actions}>
        <Button
          variant="primary"
          onClick={onSubmit}
          disabled={submitting || incomplete}
          ariaLabel={submitting ? "Saving your answers" : "Save and continue"}
        >
          {submitting ? "Saving…" : "Save and continue"}
        </Button>
      </div>
    </section>
  );
};