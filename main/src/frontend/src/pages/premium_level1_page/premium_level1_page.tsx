import { ArrowLeft, CheckCircle2, FlaskConical, Sparkles } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ProgramStructure } from "@/components/program_structure/program_structure";
import { useTestMode } from "@/components/test_mode/test_mode_context";
import { useProgramCheckout, type PurchaseState } from "@/hooks/use_program_checkout";
import {
  fetchMarketplaceProgram,
  fetchPremiumLevel1Programs,
  formatMarketplacePrice,
  type MarketplaceProgram,
  type MarketplaceProgramDetail
} from "@/services/marketplace_api";
import {
  fetchNutritionAssignment,
  fetchQuestionnaireCatalog,
  fetchQuestionnaireRequirement,
  generateNutritionAssignment,
  isQuestionnaireLockedError,
  submitQuestionnaire,
  type NutritionAssignmentStatus,
  type QuestionnaireAnswers,
  type QuestionnaireQuestion
} from "@/services/premium_level1_api";
import { ApiError } from "@utils/http_client";

import { PremiumNutritionPlan } from "./premium_nutrition_plan";
import { PremiumQuestionnaireForm } from "./premium_questionnaire_form";
import styles from "./premium_level1_page.module.css";

// The step the client is on. The backend owns every gate that matters — this
// only decides what is worth rendering.
type PremiumStep = "product" | "questionnaire" | "checkout" | "access";

const nutritionUnavailable: NutritionAssignmentStatus = {
  program_id: "",
  status: "pending",
  version: 0,
  questionnaire_version: 0,
  out_of_date: false,
  plan: null
};

// what the client receives, stated plainly. This deliberately promises an
// automated personalised package and nothing more: there is no ongoing human
// coaching and no trainer-authored personalisation.
const PACKAGE_INCLUDES = [
  "A structured training plan you follow week by week",
  "A nutrition programme personalised from your own questionnaire",
  "Daily calorie and macro targets matched to your goal",
  "A meal structure and hydration target you can follow directly",
  "Dietary rules built around the restrictions you reported"
];

export const PremiumLevel1Page = () => {
  const history = useHistory();
  const { isActive: testModeActive } = useTestMode();

  const [product, setProduct] = useState<MarketplaceProgram | null>(null);
  const [detail, setDetail] = useState<MarketplaceProgramDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const [step, setStep] = useState<PremiumStep>("product");

  const [questions, setQuestions] = useState<QuestionnaireQuestion[]>([]);
  const [answers, setAnswers] = useState<QuestionnaireAnswers>({});
  const [requirement, setRequirement] = useState<{ submitted: boolean; locked: boolean } | null>(null);
  const [questionnaireLoading, setQuestionnaireLoading] = useState(false);
  const [questionnaireError, setQuestionnaireError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);

  const [nutrition, setNutrition] = useState<NutritionAssignmentStatus>(nutritionUnavailable);
  const [nutritionLoaded, setNutritionLoaded] = useState(false);
  const [generating, setGenerating] = useState(false);

  const programId = product?.id ?? "";

  const checkout = useProgramCheckout({
    programId,
    free: false,
    testModeActive,
    enabled: programId !== "" && !loading
  });

  // The product metadata comes from the backend: nothing about price, duration
  // or contents is hardcoded here.
  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");
    try {
      const { programs } = await fetchPremiumLevel1Programs();
      if (programs.length === 0) {
        setProduct(null);
        setErrorMessage("Premium Level 1 is not available right now. Please check back soon.");
        return;
      }
      setProduct(programs[0]);
      setDetail(await fetchMarketplaceProgram(programs[0].id));
    } catch {
      setProduct(null);
      setErrorMessage("We could not load Premium Level 1. Please try again.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const loadQuestionnaire = useCallback(async () => {
    if (programId === "") return;
    setQuestionnaireLoading(true);
    setQuestionnaireError("");
    try {
      const [catalog, state] = await Promise.all([
        fetchQuestionnaireCatalog(),
        fetchQuestionnaireRequirement(programId)
      ]);
      setQuestions(catalog.questions);
      setRequirement({ submitted: state.submitted, locked: state.locked });
      // A submitted intake is never read back from the server: the answers are
      // sensitive and are not echoed. The client restores what it holds locally
      // so an in-progress edit is not lost, and an empty form after a reload is
      // expected and honest.
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setQuestionnaireError("We could not load the questionnaire. Please try again.");
    } finally {
      setQuestionnaireLoading(false);
    }
  }, [programId, history]);

  const loadNutrition = useCallback(async () => {
    if (programId === "") return;
    try {
      const status = await fetchNutritionAssignment(programId);
      setNutrition(status);
      setNutritionLoaded(true);
    } catch {
      // A program without a plan, an unowned program and a not-yet-created
      // assignment are indistinguishable server-side. None of them is an error
      // worth showing to a buyer who just paid.
      setNutritionLoaded(false);
    }
  }, [programId]);

  const handleStart = useCallback(async () => {
    setStep("questionnaire");
    await loadQuestionnaire();
  }, [loadQuestionnaire]);

  const handleAnswer = useCallback((field: string, value: string | number | string[]) => {
    setAnswers((previous) => ({ ...previous, [field]: value }));
    setFieldErrors((previous) => {
      if (!(field in previous)) return previous;
      const next = { ...previous };
      delete next[field];
      return next;
    });
  }, []);

  const handleSubmitQuestionnaire = useCallback(async () => {
    setSubmitting(true);
    setQuestionnaireError("");
    setFieldErrors({});
    try {
      const state = await submitQuestionnaire(programId, { answers });
      setRequirement({ submitted: state.submitted, locked: state.locked });
      setStep("checkout");
    } catch (error) {
      if (isQuestionnaireLockedError(error)) {
        // The server says the intake is immutable. Stop editing, re-read the
        // authoritative state and render the locked view.
        await loadQuestionnaire();
        return;
      }
      if (error instanceof ApiError && error.status === 400 && error.details.length > 0) {
        const reasons: Record<string, string> = {};
        for (const detail of error.details) {
          const [field, reason] = detail.split(": ");
          if (field && reason) reasons[field] = reason;
        }
        setFieldErrors(reasons);
        return;
      }
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setQuestionnaireError("We could not save your answers. Please try again.");
    } finally {
      setSubmitting(false);
    }
  }, [answers, history, loadQuestionnaire, programId]);

  const handleGenerate = useCallback(async () => {
    setGenerating(true);
    try {
      setNutrition(await generateNutritionAssignment(programId));
      setNutritionLoaded(true);
    } catch {
      // An absent, unowned or not-yet-created assignment are indistinguishable
      // server-side, so a failure collapses to the same quiet state.
      setNutritionLoaded(false);
    } finally {
      setGenerating(false);
    }
  }, [programId]);

  const cancelled = checkout.state.status === "cancelled" ? checkout.state : null;
  const owned = checkout.state.status === "owned" || checkout.state.status === "success";

  // A client who already owns this package, or who just bought it, should not
  // land on a questionnaire they cannot submit. Ownership is resolved by the
  // shared checkout hook, so this is the same answer the marketplace uses to route
  // an owned program straight to Program Access. Access itself is always decided
  // by the backend; this only decides what is worth rendering.
  useEffect(() => {
    if (!owned || step === "access") return;
    setStep("access");
    void loadNutrition();
  }, [owned, step, loadNutrition]);

  const renderCheckout = () => (
    <section className={styles.checkout} aria-label="Checkout">
      <h2 className={styles.sectionTitle}>Complete your purchase</h2>
      <p className={styles.sectionText}>
        Your questionnaire is saved. Complete the purchase to unlock your training plan and your personalised
        nutrition programme.
      </p>

      {checkout.state.status === "success" ? (
        <div className={styles.stateCard}>
          <p className={styles.stateTitle}>
            <CheckCircle2 size={16} aria-hidden="true" />
            Purchase confirmed
          </p>
          <p className={styles.stateText}>
            Your package is active. We are preparing your nutrition programme now.
          </p>
          <div className={styles.actions}>
            <Button
              variant="primary"
              size="small"
              onClick={() => {
                setStep("access");
                void loadNutrition();
              }}
            >
              Go to my package
            </Button>
          </div>
        </div>
      ) : checkout.state.status === "error" ? (
        <div className={styles.stateCardError}>
          <p className={styles.stateTitle}>{checkout.state.message}</p>
          <div className={styles.actions}>
            <Button variant="secondary" size="small" onClick={() => void checkout.buy()}>
              Try again
            </Button>
          </div>
        </div>
      ) : cancelled ? (
        <div className={styles.stateCard}>
          <p className={styles.stateTitle}>Payment cancelled</p>
          <p className={styles.stateText}>
            No money was taken. Your questionnaire is still saved, so you can resume whenever you are ready.
          </p>
          <div className={styles.actions}>
            <Button variant="primary" size="small" onClick={() => void checkout.retryPurchase(cancelled.purchaseId)}>
              Resume checkout
            </Button>
          </div>
        </div>
      ) : checkout.state.status === "guest" ? (
        <div className={styles.stateCard}>
          <p className={styles.stateTitle}>Sign in to continue</p>
          <div className={styles.actions}>
            <Button variant="primary" size="small" to="/login">
              Sign in
            </Button>
          </div>
        </div>
      ) : (
        <>
          {testModeActive ? (
            <p className={styles.testModeNote}>
              <FlaskConical size={14} aria-hidden="true" />
              Test Mode is active, so this purchase completes without contacting any payment provider.
            </p>
          ) : null}

          {checkout.paymentMethods.length > 0 ? (
            <fieldset className={styles.methods}>
              <legend className={styles.label}>Payment method</legend>
              <div className={styles.methodOptions}>
                {checkout.paymentMethods.map((method) => (
                  <label
                    key={method.method}
                    className={`${styles.method} ${checkout.selectedMethod === method.method ? styles.methodSelected : ""}`}
                  >
                    <input
                      type="radio"
                      name="payment-method"
                      value={method.method}
                      checked={checkout.selectedMethod === method.method}
                      onChange={() => checkout.selectMethod(method.method)}
                    />
                    <span>{method.label}</span>
                  </label>
                ))}
              </div>
            </fieldset>
          ) : (
            <p className={styles.sectionText}>
              Checkout is temporarily unavailable. No payment can be started right now.
            </p>
          )}

          <div className={styles.actions}>
            <Button
              variant="primary"
              size="small"
              disabled={
                checkout.paymentMethods.length === 0 ||
                ["creating", "negotiating", "redirecting", "processing"].includes(checkout.state.status)
              }
              onClick={() => void checkout.buy()}
            >
              {checkout.state.status === "checking" ? "Checking access…" : "Pay and unlock my package"}
            </Button>
          </div>
        </>
      )}
    </section>
  );

  return (
    <PageWrapper className={styles.page}>
      <section className={styles.detail}>
        <Container size="wide" className={styles.container}>
          <div className={styles.back}>
            <Button to="/services" variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
              Back to Services
            </Button>
          </div>

          {loading ? (
            <div className={styles.state}>
              <p>Loading Premium Level 1…</p>
            </div>
          ) : errorMessage || !product ? (
            <div className={styles.state}>
              <p>{errorMessage}</p>
              <div className={styles.actions}>
                <Button variant="secondary" size="small" onClick={() => void load()}>
                  Retry
                </Button>
              </div>
            </div>
          ) : (
            <>
              <header className={styles.heading}>
                <p className={styles.eyebrow}>
                  <Sparkles size={14} aria-hidden="true" />
                  Premium Level 1
                </p>
                <h1 className={styles.title}>{product.name}</h1>
                {product.description ? <p className={styles.subtitle}>{product.description}</p> : null}

                <div className={styles.chips}>
                  <span className={styles.chipPrice}>
                    {testModeActive
                      ? formatMarketplacePrice(0, product.currency, product.type)
                      : formatMarketplacePrice(product.price_minor_units, product.currency, product.type)}
                  </span>
                  {testModeActive ? (
                    <span className={styles.chipTestMode}>
                      <FlaskConical size={12} aria-hidden="true" />
                      Test Mode
                    </span>
                  ) : null}
                  <span className={styles.chip}>Training + nutrition</span>
                  {product.duration_weeks !== null ? (
                    <span className={styles.chip}>
                      {product.duration_weeks} week{product.duration_weeks === 1 ? "" : "s"}
                    </span>
                  ) : null}
                  {product.level ? <span className={styles.chip}>{product.level}</span> : null}
                  {product.frequency_per_week !== null ? (
                    <span className={styles.chip}>
                      {product.frequency_per_week} day{product.frequency_per_week === 1 ? "" : "s"}/week
                    </span>
                  ) : null}
                </div>
              </header>

              <div className={styles.columns}>
                <div className={styles.main}>
                  <section className={styles.panel} aria-label="What you receive">
                    <h2 className={styles.sectionTitle}>One complete package</h2>
                    <p className={styles.sectionText}>
                      Premium Level 1 combines your training plan and your nutrition programme in a single
                      purchase. The nutrition side is personalised automatically from the questionnaire you fill
                      in before checkout, so both halves of the package are built around the same answers.
                    </p>
                    <ul className={styles.includes}>
                      {PACKAGE_INCLUDES.map((item) => (
                        <li key={item} className={styles.includeItem}>
                          <CheckCircle2 size={15} aria-hidden="true" />
                          <span>{item}</span>
                        </li>
                      ))}
                    </ul>
                    <p className={styles.disclaimer}>
                      This is an automated programme, not personal coaching. No trainer reviews your answers and
                      the plan is not a medical or clinical assessment. Speak to a professional before starting
                      if you have a medical condition.
                    </p>
                  </section>

                  {step === "questionnaire" ? (
                    questionnaireLoading ? (
                      <div className={styles.state}>
                        <p>Loading questionnaire…</p>
                      </div>
                    ) : (
                      <PremiumQuestionnaireForm
                        programId={programId}
                        questions={questions}
                        answers={answers}
                        locked={requirement?.locked ?? false}
                        submitting={submitting}
                        fieldErrors={fieldErrors}
                        errorMessage={questionnaireError}
                        onChange={handleAnswer}
                        onSubmit={() => void handleSubmitQuestionnaire()}
                        onRetry={() => void loadQuestionnaire()}
                      />
                    )
                  ) : null}

                  {step === "checkout" ? renderCheckout() : null}

                  {step === "access" && detail ? (
                    <>
                      <section className={styles.panel} aria-label="Your training plan">
                        <h2 className={styles.sectionTitle}>Your training plan</h2>
                        <ProgramStructure weeks={detail.weeks} />
                      </section>

                      {nutritionLoaded ? (
                        <PremiumNutritionPlan
                          status={nutrition}
                          generating={generating}
                          onGenerate={() => void handleGenerate()}
                          onRefresh={() => void loadNutrition()}
                        />
                      ) : null}
                    </>
                  ) : null}
                </div>

                <aside className={styles.aside}>
                  <div className={styles.stickyCard}>
                    <p className={styles.asideTitle}>Premium Level 1</p>
                    <p className={styles.asidePrice}>
                      {testModeActive
                        ? formatMarketplacePrice(0, product.currency, product.type)
                        : formatMarketplacePrice(product.price_minor_units, product.currency, product.type)}
                    </p>
                    <p className={styles.asideText}>
                      A complete training and nutrition package, personalised from your questionnaire.
                    </p>
                    <div className={styles.actions}>
                      {step === "product" ? (
                        <Button variant="primary" size="small" onClick={() => void handleStart()}>
                          Start my questionnaire
                        </Button>
                      ) : step === "questionnaire" && requirement?.locked ? (
                        <Button variant="primary" size="small" onClick={() => setStep("checkout")}>
                          Continue to checkout
                        </Button>
                      ) : step === "questionnaire" && requirement?.submitted ? (
                        <Button variant="primary" size="small" onClick={() => setStep("checkout")}>
                          Continue to checkout
                        </Button>
                      ) : null}
                      {step === "access" ? (
                        <Button variant="primary" size="small" to={`/services/my-programs/${programId}`}>
                          Open Program Access
                        </Button>
                      ) : null}
                    </div>
                    {step === "questionnaire" && requirement?.locked ? (
                      <p className={styles.asideNote}>
                        Your questionnaire is locked because your purchase is complete.
                      </p>
                    ) : null}
                  </div>
                </aside>
              </div>
            </>
          )}
        </Container>
      </section>
    </PageWrapper>
  );
};
