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
  /** The client's current answers. Never sent back by the server, so this is
   *  the only place they exist. */
  answers: QuestionnaireAnswers;
  /** The server's authoritative lock state. */
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

// The server answers with short, stable reason codes so the API never carries
// presentation copy. The client owns the wording, and an unknown code still
// renders as a readable message instead of leaking a token to the user.
const REASON_TEXT: Record<string, string> = {
  required: "This answer is required.",
  "unsupported value": "Choose one of the listed options.",
  "out of range": "This value is outside the accepted range.",
  "too many entries": "Too many options selected.",
  "entry too long": "This answer is too long.",
  "too long": "This answer is too long."
};

const reasonText = (reason: string): string => REASON_TEXT[reason] ?? "Please check this answer.";

// renderInput picks the control from the server-declared question type. The
// catalog is the single source of truth, so a new question type never needs a
// client release to be renderable.
const renderInput = (
  question: QuestionnaireQuestion,
  value: string | number | string[] | undefined,
  describedBy: string | undefined,
  onChange: (value: string | number | string[]) => void
) => {
  // Only the attributes that are meaningful for the chosen control are
  // forwarded. A server-provided max_length must not leak onto a <select> or a
  // checkbox group, where it would be invalid markup.
  const described = describedBy ? { "aria-describedby": describedBy } : {};

  if (question.type === "select") {
    return (
      <select
        id={question.field}
        name={question.field}
        required={question.required}
        className={styles.control}
        value={typeof value === "string" ? value : ""}
        onChange={(event) => onChange(event.target.value)}
        {...described}
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
    const capped = selected.length >= (question.max_entries ?? Number.MAX_SAFE_INTEGER);
    return (
      <div className={styles.options} {...described}>
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
                // Unchecked options stop being selectable once the server-declared
                // entry cap is reached. Selecting one still requires an explicit
                // click; nothing is ever selected on the buyer's behalf.
                disabled={!isChecked && capped}
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
        id={question.field}
        name={question.field}
        required={question.required}
        className={styles.control}
        type="number"
        min={question.min}
        max={question.max}
        value={typeof value === "number" ? value : ""}
        onChange={(event) => {
          const raw = event.target.value;
          if (raw === "") {
            onChange("");
            return;
          }
          const parsed = Number(raw);
          onChange(Number.isNaN(parsed) ? "" : parsed);
        }}
        {...described}
      />
    );
  }

  if (question.type === "list") {
    // List questions are free-text entries (allergies, excluded foods, medical
    // conditions...). One item per line is the simplest unambiguous mapping of
    // text into the array the server contract expects; blank lines are dropped
    // and every entry is trimmed, mirroring the server-side normalisation.
    const entries = Array.isArray(value) ? value : [];
    return (
      <textarea
        id={question.field}
        name={question.field}
        className={joinClassNames(styles.control, styles.textarea)}
        rows={4}
        placeholder="One item per line"
        value={entries.join("\n")}
        onChange={(event) => {
          const lines = event.target.value
            .split("\n")
            .map((line) => line.trim())
            .filter((line) => line !== "");
          onChange(lines);
        }}
        {...described}
      />
    );
  }

  if (question.type === "textarea") {
    return (
      <textarea
        id={question.field}
        name={question.field}
        required={question.required}
        className={joinClassNames(styles.control, styles.textarea)}
        maxLength={question.max_length}
        value={typeof value === "string" ? value : ""}
        onChange={(event) => onChange(event.target.value)}
        {...described}
      />
    );
  }

  return (
    <input
      id={question.field}
      name={question.field}
      required={question.required}
      className={styles.control}
      type="text"
      minLength={question.min}
      maxLength={question.max_length}
      value={typeof value === "string" ? value : ""}
      onChange={(event) => onChange(event.target.value)}
      {...described}
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
          Your questionnaire is part of the package you purchased, so it can no longer be edited. Your training
          plan and your nutrition programme are built from exactly these answers.
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
          const errorId = `${question.field}-error`;
          // The control is described by its help text, its sensitivity note and
          // its error, so a screen reader reaches the reason it was rejected.
          const describedBy =
            [question.help ? `${question.field}-help` : "", question.sensitive ? `${question.field}-sensitive` : "", fieldError ? errorId : ""]
              .filter(Boolean)
              .join(" ") || undefined;

          return (
            <fieldset key={question.field} className={styles.field}>
              <legend className={styles.label}>
                {question.label}
                {question.required ? (
                  <>
                    <span aria-hidden="true"> *</span>
                    <span className={styles.visuallyHidden}> (required)</span>
                  </>
                ) : null}
              </legend>
              {question.help ? (
                <p className={styles.help} id={`${question.field}-help`}>
                  {question.help}
                </p>
              ) : null}
              {renderInput(
                question,
                answers[question.field],
                describedBy,
                (value) => onChange(question.field, value)
              )}
              {question.sensitive ? (
                <p className={styles.sensitive} id={`${question.field}-sensitive`}>
                  Used only to build your nutrition programme.
                </p>
              ) : null}
              {fieldError ? (
                <p className={styles.fieldError} id={errorId} role="alert">
                  {reasonText(fieldError)}
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