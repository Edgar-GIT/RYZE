import { ArrowLeft, CheckCircle2, ClipboardList, Dumbbell, RefreshCw, Send, Utensils } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useHistory, useParams } from "react-router-dom";

import { Admin2StatusBadge, type Admin2BadgeTone } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Button } from "@/components/button/button";
import { PremiumLevel2IntakeAnswers } from "@/components/premium_level2_intake/intake_answers";
import { CuratedNutritionEditor } from "@/pages/admin2/admin2_plan_create_page/curated_nutrition_editor";
import {
  emptyCuratedNutritionDraft,
  nutritionDraftToInput,
  nutritionPlanToDraft,
  validateNutritionDraft
} from "@/pages/admin2/admin2_plan_create_page/l2_nutrition_mapping";
import {
  attachPremiumLevel2TrainingProgram,
  deliverPremiumLevel2Request,
  fetchTrainerPremiumLevel2Catalog,
  fetchTrainerPremiumLevel2Request,
  PREMIUM_LEVEL_2_STATUS_LABELS,
  savePremiumLevel2Nutrition,
  type PremiumLevel2Catalog,
  type PremiumLevel2NutritionPlan,
  type PremiumLevel2RequestStatus,
  type PremiumLevel2StaffDetail,
  type StaffPremiumLevel2Request
} from "@/services/premium_level_2_api";
import type { PlanDraftCuratedNutrition } from "@/pages/admin2/admin2_plan_create_page/plan_builder_model";
import { fetchTrainerPrograms, type TrainerProgram } from "@/services/trainer_api";
import { formatAdminDate } from "@/services/admin_api";
import { ApiError } from "@utils/http_client";

import styles from "./trainer_premium_level2_detail_page.module.css";

const statusTone = (status: PremiumLevel2RequestStatus): Admin2BadgeTone => {
  if (status === "delivered") return "success";
  if (status === "ready_for_delivery") return "warning";
  if (status === "cancelled") return "danger";
  return "neutral";
};

const clientName = (request: StaffPremiumLevel2Request): string =>
  [request.client_first_name, request.client_last_name].filter(Boolean).join(" ") || request.client_email;

const editableStatus = (status: PremiumLevel2RequestStatus): boolean =>
  status !== "delivered" && status !== "cancelled";

const errorMessage = (error: unknown, fallback: string): string => {
  if (error instanceof ApiError) {
    if (error.details.length > 0) return error.details.join(" ");
    if (error.code === "PREMIUM_LEVEL_2_NOT_EDITABLE" || error.code === "INVALID_STATE") {
      return "This service can no longer be edited in its current state.";
    }
  }
  return fallback;
};

export const TrainerPremiumLevel2DetailPage = () => {
  const { requestID } = useParams<{ requestID: string }>();
  const history = useHistory();

  const [detail, setDetail] = useState<PremiumLevel2StaffDetail | null>(null);
  const [catalog, setCatalog] = useState<PremiumLevel2Catalog | null>(null);
  const [programs, setPrograms] = useState<TrainerProgram[]>([]);
  const [nutrition, setNutrition] = useState<PlanDraftCuratedNutrition>(() =>
    emptyCuratedNutritionDraft()
  );
  const [programId, setProgramId] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");

  const [savingNutrition, setSavingNutrition] = useState(false);
  const [nutritionError, setNutritionError] = useState("");
  const [nutritionSaved, setNutritionSaved] = useState(false);

  const [attaching, setAttaching] = useState(false);
  const [attachError, setAttachError] = useState("");
  const [attachSuccess, setAttachSuccess] = useState("");

  const [delivering, setDelivering] = useState(false);
  const [deliverError, setDeliverError] = useState("");
  const [deliverSuccess, setDeliverSuccess] = useState("");

  const load = useCallback(
    async (resetNutrition: boolean) => {
      setLoadError("");
      try {
        const loaded = await fetchTrainerPremiumLevel2Request(requestID);
        setDetail(loaded);
        setProgramId((current) => current || loaded.request.training_program_id || "");
        if (resetNutrition) {
          setNutrition(
            loaded.nutrition_plan
              ? nutritionPlanToDraft(loaded.nutrition_plan)
              : emptyCuratedNutritionDraft()
          );
        }
      } catch {
        setLoadError("This service could not be loaded, or it is not allocated to you.");
      } finally {
        setLoading(false);
      }
    },
    [requestID]
  );

  useEffect(() => {
    void load(true);
  }, [load]);

  useEffect(() => {
    let cancelled = false;
    fetchTrainerPremiumLevel2Catalog()
      .then((value) => {
        if (!cancelled) setCatalog(value);
      })
      .catch(() => {
        // Answers still render with humanised keys.
      });
    fetchTrainerPrograms(1, 100)
      .then((result) => {
        if (!cancelled) setPrograms(result.programs);
      })
      .catch(() => {
        // The attach control stays unavailable if programmes cannot be loaded.
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const attach = async () => {
    if (!detail || !programId) return;
    setAttaching(true);
    setAttachError("");
    setAttachSuccess("");
    try {
      await attachPremiumLevel2TrainingProgram(detail.request.id, programId);
      await load(false);
      setAttachSuccess("Training programme attached.");
    } catch (error) {
      setAttachError(
        errorMessage(
          error,
          "Unable to attach the programme. It must be your draft programme with at least one exercise."
        )
      );
    } finally {
      setAttaching(false);
    }
  };

  const saveNutrition = async () => {
    if (!detail) return;
    const messages = validateNutritionDraft(nutrition);
    if (messages.length > 0) {
      setNutritionError(messages[0]);
      return;
    }
    setSavingNutrition(true);
    setNutritionError("");
    setNutritionSaved(false);
    try {
      const plan = await savePremiumLevel2Nutrition(detail.request.id, nutritionDraftToInput(nutrition));
      setDetail((current) => (current ? { ...current, nutrition_plan: plan } : current));
      setNutrition(nutritionPlanToDraft(plan));
      setNutritionSaved(true);
      await load(false);
    } catch (error) {
      setNutritionError(errorMessage(error, "Unable to save the nutrition plan."));
    } finally {
      setSavingNutrition(false);
    }
  };

  const deliver = async () => {
    if (!detail) return;
    if (!window.confirm("Deliver this service? The client will receive the programme and nutrition plan immediately.")) {
      return;
    }
    setDelivering(true);
    setDeliverError("");
    setDeliverSuccess("");
    try {
      await deliverPremiumLevel2Request(detail.request.id);
      await load(false);
      setDeliverSuccess("Service delivered. The client can now access both plans.");
    } catch (error) {
      setDeliverError(errorMessage(error, "Unable to deliver the service."));
    } finally {
      setDelivering(false);
    }
  };

  if (loading) {
    return <p className={styles.muted}>Loading service…</p>;
  }

  if (loadError || !detail) {
    return (
      <div className={styles.stateBlock}>
        <p className={styles.error}>{loadError || "Service unavailable."}</p>
        <Button
          to="/trainer/premium-level-2"
          variant="secondary"
          size="small"
          icon={<ArrowLeft size={15} />}
          iconPosition="left"
        >
          Back to client services
        </Button>
      </div>
    );
  }

  const request = detail.request;
  const editable = editableStatus(request.status);
  const currentProgram = programs.find((program) => program.id === request.training_program_id);
  const drafts = programs.filter((program) => program.status === "draft");

  return (
    <div className={styles.page}>
      <div className={styles.topRow}>
        <Button
          to="/trainer/premium-level-2"
          variant="ghost"
          size="small"
          icon={<ArrowLeft size={15} />}
          iconPosition="left"
          onClick={() => history.push("/trainer/premium-level-2")}
        >
          Client services
        </Button>
        <Admin2StatusBadge
          label={PREMIUM_LEVEL_2_STATUS_LABELS[request.status]}
          tone={statusTone(request.status)}
        />
      </div>

      <header className={styles.header}>
        <p className={styles.eyebrow}>Premium Level 2 · {clientName(request)}</p>
        <h1 className={styles.title}>{request.offer_name}</h1>
        <p className={styles.subtitle}>
          {request.client_email} · intake revision {request.intake_version || "not submitted"} ·
          allocated {formatAdminDate(request.created_at)}
        </p>
      </header>

      <div className={styles.columns}>
        <div className={styles.main}>
          <section className={styles.panel}>
            <div className={styles.sectionHeading}>
              <Dumbbell size={17} aria-hidden="true" />
              <h2>Training programme</h2>
            </div>
            <p className={styles.muted}>
              Author a draft programme in your trainer workspace with at least one exercise, then
              attach it here. The programme stays private until you deliver the service.
            </p>
            <div className={styles.attachRow}>
              <select
                className={styles.select}
                value={programId}
                disabled={!editable || attaching}
                onChange={(event) => setProgramId(event.target.value)}
                aria-label="Select training programme"
              >
                <option value="">Choose a draft programme</option>
                {drafts.map((program) => (
                  <option key={program.id} value={program.id}>
                    {program.name}
                  </option>
                ))}
              </select>
              <Button
                variant="secondary"
                size="small"
                disabled={!editable || !programId || attaching || programId === request.training_program_id}
                onClick={() => void attach()}
              >
                {attaching ? "Attaching…" : request.training_program_id ? "Replace programme" : "Attach programme"}
              </Button>
            </div>
            {request.training_program_id ? (
              <p className={styles.success}>
                <CheckCircle2 size={15} aria-hidden="true" />
                Attached:{" "}
                <Link to={`/trainer/programs/${request.training_program_id}`} className={styles.link}>
                  {request.training_program_name || currentProgram?.name || "Open programme"}
                </Link>
              </p>
            ) : null}
            {attachError ? <p className={styles.error}>{attachError}</p> : null}
            {attachSuccess ? (
              <p className={styles.success}>
                <CheckCircle2 size={15} aria-hidden="true" />
                {attachSuccess}
              </p>
            ) : null}
            <Button
              to="/trainer/programs"
              variant="ghost"
              size="small"
              icon={<Dumbbell size={15} />}
              iconPosition="left"
            >
              Open my programmes
            </Button>
          </section>

          <section className={styles.panel}>
            <div className={styles.sectionHeading}>
              <Utensils size={17} aria-hidden="true" />
              <h2>Nutrition plan</h2>
            </div>
            {detail.nutrition_plan ? (
              <p className={styles.muted}>
                Current saved version: {detail.nutrition_plan.version} ·{" "}
                {detail.nutrition_plan.status}
              </p>
            ) : (
              <p className={styles.muted}>No nutrition plan has been authored yet.</p>
            )}

            {editable ? (
              <CuratedNutritionEditor
                nutrition={nutrition}
                onChange={(next) => {
                  setNutrition(next);
                  setNutritionSaved(false);
                  setNutritionError("");
                }}
                intro="Build the client's individual nutrition plan from the intake below. Every saved revision replaces the previous draft until the service is delivered."
              />
            ) : detail.nutrition_plan ? (
              <p className={styles.muted}>The delivered plan is locked and cannot be edited.</p>
            ) : null}

            {editable ? (
              <div className={styles.actions}>
                {nutritionError ? <p className={styles.error}>{nutritionError}</p> : null}
                {nutritionSaved ? (
                  <p className={styles.success}>
                    <CheckCircle2 size={15} aria-hidden="true" />
                    Nutrition plan saved.
                  </p>
                ) : null}
                <Button
                  variant="primary"
                  size="small"
                  disabled={savingNutrition}
                  onClick={() => void saveNutrition()}
                >
                  {savingNutrition ? "Saving…" : "Save nutrition plan"}
                </Button>
              </div>
            ) : null}
          </section>

          <section className={styles.panel}>
            <div className={styles.sectionHeading}>
              <Send size={17} aria-hidden="true" />
              <h2>Delivery</h2>
            </div>
            <p className={styles.muted}>
              Delivery creates the client's access to the training programme and releases the
              nutrition plan. Both must be saved first.
            </p>
            {deliverError ? <p className={styles.error}>{deliverError}</p> : null}
            {deliverSuccess ? (
              <p className={styles.success}>
                <CheckCircle2 size={15} aria-hidden="true" />
                {deliverSuccess}
              </p>
            ) : null}
            {request.status === "delivered" ? (
              <p className={styles.success}>
                <CheckCircle2 size={15} aria-hidden="true" />
                Delivered {request.delivered_at ? formatAdminDate(request.delivered_at) : ""}
              </p>
            ) : (
              <Button
                variant="primary"
                size="small"
                disabled={request.status !== "ready_for_delivery" || delivering}
                onClick={() => void deliver()}
              >
                {delivering ? "Delivering…" : "Deliver service"}
              </Button>
            )}
          </section>
        </div>

        <aside className={styles.aside}>
          <section className={styles.panel}>
            <div className={styles.sectionHeading}>
              <ClipboardList size={17} aria-hidden="true" />
              <h2>Client intake</h2>
            </div>
            <PremiumLevel2IntakeAnswers intake={detail.intake} catalog={catalog} />
          </section>

          <section className={styles.panel}>
            <div className={styles.sectionHeading}>
              <RefreshCw size={17} aria-hidden="true" />
              <h2>History</h2>
            </div>
            {detail.events.length === 0 ? (
              <p className={styles.muted}>No events recorded yet.</p>
            ) : (
              <ol className={styles.timeline}>
                {detail.events.map((event, index) => (
                  <li key={`${event.event_type}-${event.created_at}-${index}`}>
                    <span className={styles.timelineType}>{event.event_type.replaceAll("_", " ")}</span>
                    <span className={styles.muted}>
                      {formatAdminDate(event.created_at)}
                      {event.to_status
                        ? ` · ${PREMIUM_LEVEL_2_STATUS_LABELS[event.to_status as PremiumLevel2RequestStatus] ?? event.to_status}`
                        : ""}
                    </span>
                  </li>
                ))}
              </ol>
            )}
          </section>
        </aside>
      </div>
    </div>
  );
};
