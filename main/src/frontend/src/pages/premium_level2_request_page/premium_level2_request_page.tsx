import { ArrowLeft, CheckCircle2, ClipboardList, Dumbbell, RefreshCw, Utensils } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useHistory, useParams } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import {
  fetchMyPremiumLevel2Request,
  fetchPremiumLevel2Catalog,
  PREMIUM_LEVEL_2_STATUS_LABELS,
  submitPremiumLevel2Intake,
  type PremiumLevel2Catalog,
  type PremiumLevel2ClientDetail,
  type PremiumLevel2NutritionPlan,
  type PremiumLevel2Question
} from "@/services/premium_level_2_api";
import {
  formatPremiumLevel2Answer,
  isEmptyPremiumLevel2Answer,
  type PremiumLevel2AnswerValue
} from "@/services/premium_level_2_format";
import { COMMERCE_PT } from "@/i18n/catalogs/commerce.pt";
import { PREMIUM_LEVEL2_PT } from "@/i18n/catalogs/premium_level2.pt";
import { QUESTIONNAIRE_PT } from "@/i18n/catalogs/questionnaire.pt";
import { useGettext } from "@/i18n/i18n";
import { ApiError } from "@utils/http_client";
import { joinClassNames } from "@utils/class_names";

import styles from "./premium_level2_request_page.module.css";

type AnswerValue = PremiumLevel2AnswerValue;
type Answers = Record<string, AnswerValue>;

const EDITABLE_STATUSES = new Set(["awaiting_intake", "awaiting_trainer_assignment"]);


const mapFormErrors = (error: unknown): Record<string, string> => {
  if (!(error instanceof ApiError) || error.status !== 400) {
    return {};
  }
  const fields: Record<string, string> = {};
  for (const detail of error.details) {
    const separator = detail.indexOf(": ");
    if (separator === -1) continue;
    const key = detail.slice(0, separator);
    const message = detail.slice(separator + 2);
    if (key && message) fields[key] = message;
  }
  return fields;
};

const statusTone = (status: string): string => {
  switch (status) {
    case "delivered":
      return styles.statusDelivered;
    case "ready_for_delivery":
      return styles.statusReady;
    case "in_progress":
      return styles.statusProgress;
    default:
      return styles.statusWaiting;
  }
};

const NutritionPlanView = ({ plan }: { plan: PremiumLevel2NutritionPlan }) => {
  const tr = useGettext(PREMIUM_LEVEL2_PT);
  return (
  <section className={styles.panel}>
    <div className={styles.sectionHeading}>
      <Utensils size={18} aria-hidden="true" />
      <div>
        <h2 className={styles.sectionTitle}>{tr("Your nutrition plan")}</h2>
        <p className={styles.sectionText}>
          {tr("Version {version} · {pattern}", { version: plan.version, pattern: plan.dietary_pattern.replaceAll("_", " ") })}
        </p>
      </div>
    </div>
    <p className={styles.sectionText}>{plan.summary}</p>
    <p className={styles.sectionText}>
      {tr("Hydration: {litres} litres per day.", { litres: plan.hydration_litres })} {plan.hydration_note}
    </p>
    <p className={styles.sectionText}>{plan.prep_guidance}</p>

    <div className={styles.meals}>
      {plan.meals.map((meal) => (
        <article key={meal.id} className={styles.meal}>
          <h3 className={styles.mealTitle}>
            {meal.label}
            {meal.kind === "snack" ? <span className={styles.mealKind}>{tr("Snack")}</span> : null}
          </h3>
          {meal.notes ? <p className={styles.sectionText}>{meal.notes}</p> : null}
          <div className={styles.foodList}>
            {meal.items.map((food) => (
              <div key={food.id} className={styles.foodRow}>
                <div>
                  <strong>{food.food_name}</strong>
                  <p className={styles.foodMeta}>
                    {food.quantity} {food.unit} · {food.category}
                    {food.substitution_note ? ` · ${food.substitution_note}` : ""}
                  </p>
                </div>
                <p className={styles.foodMacros}>
                  {food.calories} kcal · P {food.protein_grams}g · C {food.carbohydrate_grams}g · F{" "}
                  {food.fat_grams}g
                </p>
              </div>
            ))}
          </div>
        </article>
      ))}
    </div>

    {plan.cautions.length > 0 ? (
      <div className={styles.cautions}>
        <h3 className={styles.mealTitle}>{tr("Cautions")}</h3>
        <ul>
          {plan.cautions.map((caution) => (
            <li key={caution}>{caution}</li>
          ))}
        </ul>
      </div>
    ) : null}
  </section>
  );
};

const QuestionField = ({
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

export const PremiumLevel2RequestPage = () => {
  const tr = useGettext(PREMIUM_LEVEL2_PT, QUESTIONNAIRE_PT, COMMERCE_PT);
  const { requestID } = useParams<{ requestID: string }>();
  const history = useHistory();

  const [catalog, setCatalog] = useState<PremiumLevel2Catalog | null>(null);
  const [detail, setDetail] = useState<PremiumLevel2ClientDetail | null>(null);
  const [answers, setAnswers] = useState<Answers>({});
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError("");
    try {
      const [nextCatalog, nextDetail] = await Promise.all([
        fetchPremiumLevel2Catalog(),
        fetchMyPremiumLevel2Request(requestID)
      ]);
      setCatalog(nextCatalog);
      setDetail(nextDetail);
      if (nextDetail.intake) {
        setAnswers(nextDetail.intake.answers as Answers);
      }
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setLoadError(
        error instanceof ApiError && error.status === 404
          ? "This Premium Level 2 service was not found."
          : "We could not load your service. Please try again."
      );
    } finally {
      setLoading(false);
    }
  }, [history, requestID]);

  useEffect(() => {
    void load();
  }, [load]);

  const editable = detail ? EDITABLE_STATUSES.has(detail.request.status) : false;

  const allQuestions = useMemo(
    () => catalog?.sections.flatMap((section) => section.questions) ?? [],
    [catalog]
  );

  const handleAnswer = useCallback((key: string, value: AnswerValue) => {
    setAnswers((current) => ({ ...current, [key]: value }));
    setFieldErrors((current) => {
      if (!(key in current)) return current;
      const next = { ...current };
      delete next[key];
      return next;
    });
    setSaved(false);
  }, []);

  const handleSubmit = useCallback(async () => {
    if (!catalog) return;
    const nextFieldErrors: Record<string, string> = {};
    for (const section of catalog.sections) {
      for (const question of section.questions) {
        if (question.required && isEmptyPremiumLevel2Answer(answers[question.key])) {
          nextFieldErrors[question.key] = "This answer is required.";
        }
      }
    }
    if (Object.keys(nextFieldErrors).length > 0) {
      setFieldErrors(nextFieldErrors);
      setSubmitError("Complete the required answers before saving.");
      return;
    }

    const payload: Record<string, unknown> = {};
    for (const question of allQuestions) {
      const value = answers[question.key];
      if (isEmptyPremiumLevel2Answer(value)) continue;
      payload[question.key] = typeof value === "string" ? value.trim() : value;
    }

    setSaving(true);
    setSubmitError("");
    setFieldErrors({});
    try {
      await submitPremiumLevel2Intake(requestID, payload);
      setSaved(true);
      await load();
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      if (error instanceof ApiError && error.code === "PREMIUM_LEVEL_2_NOT_EDITABLE") {
        await load();
        return;
      }
      const fields = mapFormErrors(error);
      if (Object.keys(fields).length > 0) {
        setFieldErrors(fields);
      }
      setSubmitError(
        error instanceof ApiError && error.code === "VALIDATION_ERROR"
          ? "Some answers need attention before the intake can be saved."
          : "We could not save your intake. Please try again."
      );
    } finally {
      setSaving(false);
    }
  }, [allQuestions, answers, catalog, history, load, requestID]);

  if (loading) {
    return (
      <PageWrapper className={styles.page}>
        <section className={styles.detail}>
          <Container size="wide">
            <p className={styles.sectionText}>{tr("Loading your service…")}</p>
          </Container>
        </section>
      </PageWrapper>
    );
  }

  if (loadError || !detail || !catalog) {
    return (
      <PageWrapper className={styles.page}>
        <section className={styles.detail}>
          <Container size="wide" className={styles.container}>
            <Button
              to="/services/premium-level-2"
              variant="ghost"
              size="small"
              icon={<ArrowLeft size={15} />}
              iconPosition="left"
            >
              Back to Premium Level 2
            </Button>
            <div className={styles.panel}>
              <p className={styles.errorText}>{tr(loadError || "Your service could not be loaded.")}</p>
              <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => void load()}>
                {tr("Try again")}
              </Button>
            </div>
          </Container>
        </section>
      </PageWrapper>
    );
  }

  const request = detail.request;
  const plan = detail.nutrition_plan;

  return (
    <PageWrapper className={styles.page}>
      <section className={styles.detail}>
        <Container size="wide" className={styles.container}>
          <div className={styles.backRow}>
            <Button
              to="/services/premium-level-2"
              variant="ghost"
              size="small"
              icon={<ArrowLeft size={15} />}
              iconPosition="left"
            >
              Premium Level 2
            </Button>
            <span className={joinClassNames(styles.status, statusTone(request.status))}>
              {tr(PREMIUM_LEVEL_2_STATUS_LABELS[request.status])}
            </span>
          </div>

          <header className={styles.heading}>
            <p className={styles.eyebrow}>
              <ClipboardList size={14} aria-hidden="true" />
              {tr("Your service")}
            </p>
            <h1 className={styles.title}>{request.offer_name || tr("Premium Level 2")}</h1>
            <p className={styles.subtitle}>
              {editable
                ? tr("Complete the intake below. You can keep editing until a trainer is allocated.")
                : tr("Your intake is locked. Your trainer is now building your individual plan.")}
            </p>
          </header>

          <div className={styles.summaryGrid}>
            <div className={styles.summaryCard}>
              <span className={styles.summaryLabel}>{tr("Trainer")}</span>
              <strong>{request.trainer_name || tr("Allocation pending")}</strong>
            </div>
            <div className={styles.summaryCard}>
              <span className={styles.summaryLabel}>{tr("Training programme")}</span>
              <strong>{request.training_program_name || tr("Not attached yet")}</strong>
            </div>
            <div className={styles.summaryCard}>
              <span className={styles.summaryLabel}>{tr("Nutrition plan")}</span>
              <strong>{request.nutrition_plan_status === "delivered" ? tr("Delivered") : tr("Not delivered yet")}</strong>
            </div>
          </div>

          {request.status === "delivered" ? (
            <section className={styles.deliveredPanel}>
              <CheckCircle2 size={18} aria-hidden="true" />
              <div>
                <h2 className={styles.sectionTitle}>{tr("Your service has been delivered")}</h2>
                <p className={styles.sectionText}>
                  Your individual training programme is available in My Programs. The nutrition plan
                  is shown below.
                </p>
                {request.training_program_id ? (
                  <Button
                    variant="primary"
                    size="small"
                    icon={<Dumbbell size={15} />}
                    iconPosition="left"
                    to={`/services/my-programs/${request.training_program_id}`}
                  >
                    {tr("Open my training programme")}
                  </Button>
                ) : null}
              </div>
            </section>
          ) : null}

          {plan && request.status === "delivered" ? <NutritionPlanView plan={plan} /> : null}

          <section className={styles.panel}>
            <div className={styles.sectionHeading}>
              <ClipboardList size={18} aria-hidden="true" />
              <div>
                <h2 className={styles.sectionTitle}>{tr("Your intake")}</h2>
                <p className={styles.sectionText}>
                  {request.intake_version > 0
                    ? tr("Revision {version} saved.", { version: request.intake_version })
                    : tr("Nothing saved yet.")}
                  {editable ? " Saving again creates a new revision for your trainer." : ""}
                </p>
              </div>
            </div>

            {editable ? (
              <>
                {catalog.sections.map((section) => (
                  <fieldset key={section.key} className={styles.formSection}>
                    <legend className={styles.formSectionTitle}>{tr(section.title)}</legend>
                    <p className={styles.sectionText}>{tr(section.description)}</p>
                    <div className={styles.formGrid}>
                      {section.questions.map((question) => (
                        <QuestionField
                          key={question.key}
                          question={question}
                          value={answers[question.key]}
                          error={fieldErrors[question.key] ?? ""}
                          disabled={saving}
                          onChange={(value) => handleAnswer(question.key, value)}
                        />
                      ))}
                    </div>
                  </fieldset>
                ))}

                {submitError ? <p className={styles.errorText}>{tr(submitError)}</p> : null}
                {saved ? (
                  <p className={styles.successText}>
                    <CheckCircle2 size={16} aria-hidden="true" />
                    {tr("Intake saved.")}
                  </p>
                ) : null}

                <div className={styles.actions}>
                  <Button variant="primary" size="small" disabled={saving} onClick={() => void handleSubmit()}>
                    {saving ? tr("Saving…") : request.intake_version > 0 ? tr("Save new revision") : tr("Submit intake")}
                  </Button>
                </div>
              </>
            ) : (
              <div className={styles.answerList}>
                {catalog.sections.map((section) => (
                  <div key={section.key} className={styles.answerSection}>
                    <h3 className={styles.mealTitle}>{tr(section.title)}</h3>
                    <dl className={styles.answerGrid}>
                      {section.questions.map((question) => (
                        <div key={question.key} className={styles.answerRow}>
                          <dt>{question.label}</dt>
                          <dd>{formatPremiumLevel2Answer(question, answers[question.key])}</dd>
                        </div>
                      ))}
                    </dl>
                  </div>
                ))}
              </div>
            )}
          </section>
        </Container>
      </section>
    </PageWrapper>
  );
};
