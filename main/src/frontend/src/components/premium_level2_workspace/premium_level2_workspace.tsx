import { ArrowLeft, CheckCircle2, ClipboardList, Copy, Dumbbell, PencilLine, Plus, RefreshCw, Send, Utensils } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link, useHistory } from "react-router-dom";

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
import type { PlanDraftCuratedNutrition } from "@/pages/admin2/admin2_plan_create_page/plan_builder_model";
import { formatAdminDate } from "@/services/admin_api";
import {
  PREMIUM_LEVEL_2_STATUS_LABELS,
  type PremiumLevel2Catalog,
  type PremiumLevel2NutritionInput,
  type PremiumLevel2NutritionPlan,
  type PremiumLevel2RequestStatus,
  type PremiumLevel2StaffDetail,
  type StaffPremiumLevel2Request
} from "@/services/premium_level_2_api";
import { ApiError } from "@utils/http_client";

import styles from "./premium_level2_workspace.module.css";

// A plan the person working on the request can start from or link.
export interface WorkspacePlan {
  id: string;
  name: string;
  status: string;
  // The catalogue family (RYZE's plans); empty for a trainer's own plans.
  family?: string;
}

// WorkspaceAdapter connects the workspace to the trainer's or to RYZE's
// endpoints and pages; the server decides who may do what.
export interface WorkspaceAdapter {
  backTo: string;
  backLabel: string;
  eyebrow: string;
  load: (requestId: string) => Promise<PremiumLevel2StaffDetail>;
  loadCatalog: () => Promise<PremiumLevel2Catalog>;
  loadPlans: () => Promise<WorkspacePlan[]>;
  attach: (requestId: string, programId: string) => Promise<unknown>;
  saveNutrition: (requestId: string, input: PremiumLevel2NutritionInput) => Promise<PremiumLevel2NutritionPlan>;
  deliver: (requestId: string) => Promise<unknown>;
  uploadMealImage: (file: File) => Promise<{ key: string; url: string }>;
  // The plan builder, for a new personal plan (optionally a copy of base) or
  // to edit the linked one.
  newPlanLink: (requestId: string, base?: WorkspacePlan) => string;
  editPlanLink: (requestId: string, programId: string) => string;
  // Whether this side works on the request (RYZE only on requests it took).
  canWork?: (request: StaffPremiumLevel2Request) => boolean;
}

const statusTone = (status: PremiumLevel2RequestStatus): Admin2BadgeTone => {
  if (status === "delivered") return "success";
  if (status === "ready_for_delivery") return "warning";
  if (status === "cancelled") return "danger";
  return "neutral";
};

const clientName = (request: StaffPremiumLevel2Request): string =>
  [request.client_first_name, request.client_last_name].filter(Boolean).join(" ") || request.client_email;

const editableStatus = (status: PremiumLevel2RequestStatus): boolean =>
  status === "in_progress" || status === "ready_for_delivery";

const errorMessage = (error: unknown, fallback: string): string => {
  if (error instanceof ApiError) {
    if (error.details.length > 0) return error.details.join(" ");
    if (error.code === "PREMIUM_LEVEL_2_NOT_EDITABLE" || error.code === "INVALID_STATE") {
      return "This service can no longer be edited in its current state.";
    }
  }
  return fallback;
};

// PremiumLevel2Workspace is the screen where the trainer who accepted a
// Premium Level 2 request — or RYZE itself — prepares it (owner request,
// 2026-11-24): the client's answers, the personal training plan (created here,
// from scratch or from another plan), the nutrition plan with meal pictures,
// and the delivery to the client's personal area.
export const PremiumLevel2Workspace = ({ requestId, adapter }: { requestId: string; adapter: WorkspaceAdapter }) => {
  const history = useHistory();

  const [detail, setDetail] = useState<PremiumLevel2StaffDetail | null>(null);
  const [catalog, setCatalog] = useState<PremiumLevel2Catalog | null>(null);
  const [plans, setPlans] = useState<WorkspacePlan[]>([]);
  const [nutrition, setNutrition] = useState<PlanDraftCuratedNutrition>(() => emptyCuratedNutritionDraft());
  const [baseKey, setBaseKey] = useState("");
  const [linkId, setLinkId] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");

  const [savingNutrition, setSavingNutrition] = useState(false);
  const [nutritionError, setNutritionError] = useState("");
  const [nutritionSaved, setNutritionSaved] = useState(false);

  const [attaching, setAttaching] = useState(false);
  const [attachError, setAttachError] = useState("");

  const [delivering, setDelivering] = useState(false);
  const [deliverError, setDeliverError] = useState("");
  const [deliverSuccess, setDeliverSuccess] = useState("");

  const load = useCallback(
    async (resetNutrition: boolean) => {
      setLoadError("");
      try {
        const loaded = await adapter.load(requestId);
        setDetail(loaded);
        if (resetNutrition) {
          setNutrition(loaded.nutrition_plan ? nutritionPlanToDraft(loaded.nutrition_plan) : emptyCuratedNutritionDraft());
        }
      } catch {
        setLoadError("This service could not be loaded, or it is not yours to prepare.");
      } finally {
        setLoading(false);
      }
    },
    [adapter, requestId]
  );

  useEffect(() => {
    void load(true);
  }, [load]);

  useEffect(() => {
    let cancelled = false;
    adapter
      .loadCatalog()
      .then((value) => {
        if (!cancelled) setCatalog(value);
      })
      .catch(() => {
        // Answers still render with humanised keys.
      });
    adapter
      .loadPlans()
      .then((value) => {
        if (!cancelled) setPlans(value);
      })
      .catch(() => {
        // Creating from scratch still works without the plan list.
      });
    return () => {
      cancelled = true;
    };
  }, [adapter]);

  const planKey = (plan: WorkspacePlan) => `${plan.family ?? ""}:${plan.id}`;

  const attach = async () => {
    if (!detail || !linkId) return;
    setAttaching(true);
    setAttachError("");
    try {
      await adapter.attach(detail.request.id, linkId);
      setLinkId("");
      await load(false);
    } catch (error) {
      setAttachError(errorMessage(error, "Unable to link the plan. It must be a draft with at least one exercise, not used by another request."));
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
      const plan = await adapter.saveNutrition(detail.request.id, nutritionDraftToInput(nutrition));
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
    if (!window.confirm("Deliver to the client? They are notified and both plans appear in their personal area right away.")) {
      return;
    }
    setDelivering(true);
    setDeliverError("");
    setDeliverSuccess("");
    try {
      await adapter.deliver(detail.request.id);
      await load(false);
      setDeliverSuccess("Delivered. The client was notified and can open both plans.");
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
        <Button to={adapter.backTo} variant="secondary" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
          {adapter.backLabel}
        </Button>
      </div>
    );
  }

  const request = detail.request;
  const editable = editableStatus(request.status) && (adapter.canWork?.(request) ?? true);
  const drafts = plans.filter((plan) => plan.status === "draft" && plan.id !== request.training_program_id);
  const base = plans.find((plan) => planKey(plan) === baseKey);

  return (
    <div className={styles.page}>
      <div className={styles.topRow}>
        <Button
          to={adapter.backTo}
          variant="ghost"
          size="small"
          icon={<ArrowLeft size={15} />}
          iconPosition="left"
          onClick={() => history.push(adapter.backTo)}
        >
          {adapter.backLabel}
        </Button>
        <Admin2StatusBadge label={PREMIUM_LEVEL_2_STATUS_LABELS[request.status]} tone={statusTone(request.status)} />
      </div>

      <header className={styles.header}>
        <p className={styles.eyebrow}>
          {adapter.eyebrow} · {clientName(request)}
        </p>
        <h1 className={styles.title}>{request.offer_name}</h1>
        <p className={styles.subtitle}>
          {request.client_email} · intake revision {request.intake_version || "not submitted"} · requested{" "}
          {formatAdminDate(request.created_at)}
        </p>
      </header>

      <ol className={styles.steps} aria-label="Steps">
        <li className={request.training_program_id ? styles.stepDone : undefined}>1. Training plan</li>
        <li className={detail.nutrition_plan ? styles.stepDone : undefined}>2. Nutrition plan</li>
        <li className={request.status === "delivered" ? styles.stepDone : undefined}>3. Deliver to the client</li>
      </ol>

      <div className={styles.columns}>
        <div className={styles.main}>
          <section className={styles.panel}>
            <div className={styles.sectionHeading}>
              <Dumbbell size={17} aria-hidden="true" />
              <h2>1. Training plan</h2>
            </div>
            {request.training_program_id ? (
              <div className={styles.linkedPlan}>
                <CheckCircle2 size={18} aria-hidden="true" />
                <span>
                  <strong>{request.training_program_name || "Personal plan"}</strong>
                  <small>Linked to this request · private until delivery</small>
                </span>
                {editable ? (
                  <Button
                    to={adapter.editPlanLink(request.id, request.training_program_id)}
                    variant="secondary"
                    size="small"
                    icon={<PencilLine size={15} />}
                    iconPosition="left"
                  >
                    Edit plan
                  </Button>
                ) : null}
              </div>
            ) : (
              <p className={styles.muted}>No plan yet. Build the client's plan here, from scratch or starting from one of your plans.</p>
            )}

            {editable ? (
              <>
                <div className={styles.createBox}>
                  <label className={styles.field}>
                    <span>{request.training_program_id ? "Start a new plan from" : "Start from"}</span>
                    <select className={styles.select} value={baseKey} onChange={(event) => setBaseKey(event.target.value)}>
                      <option value="">A blank plan</option>
                      {plans.map((plan) => (
                        <option key={planKey(plan)} value={planKey(plan)}>
                          {plan.name}
                          {plan.status === "draft" ? " (draft)" : ""}
                        </option>
                      ))}
                    </select>
                  </label>
                  <Button
                    to={adapter.newPlanLink(request.id, base)}
                    variant="primary"
                    size="small"
                    icon={base ? <Copy size={15} /> : <Plus size={15} />}
                    iconPosition="left"
                  >
                    {base ? "Create from this plan" : "Create the plan"}
                  </Button>
                </div>

                {drafts.length > 0 ? (
                  <details className={styles.linkExisting}>
                    <summary>Or link a draft you already made</summary>
                    <div className={styles.attachRow}>
                      <select
                        className={styles.select}
                        value={linkId}
                        disabled={attaching}
                        onChange={(event) => setLinkId(event.target.value)}
                        aria-label="Draft to link"
                      >
                        <option value="">Choose a draft</option>
                        {drafts.map((plan) => (
                          <option key={planKey(plan)} value={plan.id}>
                            {plan.name}
                          </option>
                        ))}
                      </select>
                      <Button variant="secondary" size="small" disabled={!linkId || attaching} onClick={() => void attach()}>
                        {attaching ? "Linking…" : "Link"}
                      </Button>
                    </div>
                  </details>
                ) : null}
                {attachError ? <p className={styles.error}>{attachError}</p> : null}
              </>
            ) : null}
          </section>

          <section className={styles.panel}>
            <div className={styles.sectionHeading}>
              <Utensils size={17} aria-hidden="true" />
              <h2>2. Nutrition plan</h2>
            </div>
            {detail.nutrition_plan ? (
              <p className={styles.muted}>
                Saved version {detail.nutrition_plan.version} · {detail.nutrition_plan.status}
              </p>
            ) : (
              <p className={styles.muted}>No nutrition plan yet.</p>
            )}

            {editable ? (
              <CuratedNutritionEditor
                nutrition={nutrition}
                onChange={(next) => {
                  setNutrition(next);
                  setNutritionSaved(false);
                  setNutritionError("");
                }}
                uploadMealImage={adapter.uploadMealImage}
                intro="Build the client's own nutrition plan from their answers. Add a photo to each meal so it is easy to follow. Each save replaces the previous draft until delivery."
              />
            ) : detail.nutrition_plan ? (
              <p className={styles.muted}>The delivered plan is locked.</p>
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
                <Button variant="primary" size="small" disabled={savingNutrition} onClick={() => void saveNutrition()}>
                  {savingNutrition ? "Saving…" : "Save nutrition plan"}
                </Button>
              </div>
            ) : null}
          </section>

          <section className={styles.panel}>
            <div className={styles.sectionHeading}>
              <Send size={17} aria-hidden="true" />
              <h2>3. Deliver to the client</h2>
            </div>
            <p className={styles.muted}>
              Delivering gives the client the training plan and releases the nutrition plan in their personal area, and
              notifies them. Both plans must be saved first.
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
                icon={<Send size={15} />}
                iconPosition="left"
                disabled={request.status !== "ready_for_delivery" || delivering}
                onClick={() => void deliver()}
              >
                {delivering ? "Delivering…" : "Deliver to the client"}
              </Button>
            )}
          </section>
        </div>

        <aside className={styles.aside}>
          <section className={styles.panel}>
            <div className={styles.sectionHeading}>
              <ClipboardList size={17} aria-hidden="true" />
              <h2>Client's answers</h2>
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
          <p className={styles.muted}>
            <Link to={adapter.backTo} className={styles.link}>
              {adapter.backLabel}
            </Link>
          </p>
        </aside>
      </div>
    </div>
  );
};
