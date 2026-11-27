import { useCallback, useEffect, useRef, useState } from "react";
import { useHistory, useLocation } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import type { BuilderExercise } from "@/components/plan_builder/exercise_media";
import { PlanBuilder, type BuilderCategory } from "@/components/plan_builder/plan_builder";
import { type Copy, useI18n } from "@/i18n/i18n";
import {
  createGenericProgram,
  deleteGenericProgram,
  fetchGenericProgram,
  isProgramProductType,
  PROGRAM_PRODUCT_TYPE,
  ProgramStatusEnum,
  updateGenericProgram,
  uploadPlatformCover,
  type GenericProgramDetail,
  type ProgramProductTypeValue
} from "@/services/admin2_api";
import { attachRyzePremiumLevel2TrainingProgram } from "@/services/premium_level_2_api";
import { ApiError } from "@utils/http_client";

import { createPlanDraft, detailToPlanDraft, toGenericProgramInput, type PlanDraft } from "./plan_builder_model";

const COPY: Copy<{
  kicker: string;
  newTitle: string;
  editTitle: string;
  back: string;
  categories: Record<ProgramProductTypeValue, { label: string; description: string }>;
  loadError: string;
  saved: string;
  published: string;
  locked: string;
  errors: Record<string, string>;
  genericError: string;
  unsaved: string;
  requestKicker: string;
  requestTitle: string;
  requestBack: string;
  requestSave: string;
  copySuffix: string;
}> = {
  en: {
    kicker: "Catalogue · plans",
    newTitle: "New plan",
    editTitle: "Edit plan",
    back: "All plans",
    categories: {
      generic: { label: "Generic", description: "A free training plan for the public catalogue." },
      premium_level_1: { label: "Premium Level 1", description: "Training plus nutrition, delivered immediately." },
      premium_level_2: { label: "Premium Level 2", description: "Fully personalised by a trainer after purchase." }
    },
    loadError: "The plan could not be loaded.",
    saved: "Saved as a draft.",
    published: "Published.",
    locked: "This package has completed purchases: its content can no longer change.",
    errors: {
      EXERCISE_NOT_FOUND: "An exercise no longer exists in the catalogue. Remove it and try again.",
      DUPLICATE_EXERCISE: "The same exercise appears twice in one training day.",
      PROGRAM_NOT_FOUND: "The plan no longer exists.",
      PROGRAM_CONTENT_LOCKED: "This package has already been purchased; its content can no longer change.",
      PROGRAM_METADATA_LOCKED: "The objective and nutrition model are fixed once the package exists.",
      VALIDATION_ERROR: "Some details were not accepted. Check the plan and try again."
    },
    genericError: "The plan could not be saved. Try again.",
    unsaved: "You have unsaved changes. Leave anyway?",
    requestKicker: "Premium Level 2 · RYZE personal plan",
    requestTitle: "Client's personal plan",
    requestBack: "Back to the request",
    requestSave: "Save and link to the request",
    copySuffix: "(copy)"
  },
  pt: {
    kicker: "Catálogo · planos",
    newTitle: "Novo plano",
    editTitle: "Editar plano",
    back: "Todos os planos",
    categories: {
      generic: { label: "Genérico", description: "Plano de treino gratuito para o catálogo público." },
      premium_level_1: { label: "Premium Nível 1", description: "Treino e nutrição, entregues de imediato." },
      premium_level_2: { label: "Premium Nível 2", description: "Totalmente personalizado por um treinador após a compra." }
    },
    loadError: "Não foi possível carregar o plano.",
    saved: "Guardado como rascunho.",
    published: "Publicado.",
    locked: "Este pacote já tem compras concluídas: o conteúdo já não pode mudar.",
    errors: {
      EXERCISE_NOT_FOUND: "Um exercício já não existe no catálogo. Remove-o e tenta de novo.",
      DUPLICATE_EXERCISE: "O mesmo exercício aparece duas vezes num dia de treino.",
      PROGRAM_NOT_FOUND: "O plano já não existe.",
      PROGRAM_CONTENT_LOCKED: "Este pacote já foi comprado; o conteúdo já não pode mudar.",
      PROGRAM_METADATA_LOCKED: "O objetivo e o modelo de nutrição ficam fixos depois de o pacote existir.",
      VALIDATION_ERROR: "Alguns dados não foram aceites. Revê o plano e tenta de novo."
    },
    genericError: "Não foi possível guardar o plano. Tenta de novo.",
    unsaved: "Tens alterações por guardar. Sair na mesma?",
    requestKicker: "Premium Nível 2 · plano pessoal RYZE",
    requestTitle: "Plano pessoal do cliente",
    requestBack: "Voltar ao pedido",
    requestSave: "Guardar e ligar ao pedido",
    copySuffix: "(cópia)"
  }
};

// Fixed catalogue prices (shown only; the server applies them).
const CATEGORY_PRICE: Record<ProgramProductTypeValue, number> = {
  generic: 0,
  premium_level_1: 1499,
  premium_level_2: 1949
};

// Premium Level 2 is one platform product made by trainers or RYZE per
// client, never a plan administrators create here (owner request, 2026-11-24).
const FAMILIES: ProgramProductTypeValue[] = [PROGRAM_PRODUCT_TYPE.GENERIC, PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1];

// exercisesOf collects what the builder needs to show the exercises of a
// loaded plan (names and pictures) before any catalogue request.
const exercisesOf = (detail: GenericProgramDetail): BuilderExercise[] =>
  detail.weeks.flatMap((week) =>
    week.workouts.flatMap((workout) =>
      workout.exercises.map((exercise) => ({
        id: exercise.id,
        name: exercise.name,
        primary_muscle_group: exercise.target_muscles,
        body_part: "",
        equipment: exercise.equipment,
        difficulty: exercise.difficulty,
        description: exercise.description,
        instructions: exercise.catalog_instructions,
        instruction_steps: null,
        image_url: exercise.image_url || null,
        animation_url: null,
        video_url: exercise.video_url || null,
        media_attribution: "",
        media_available: Boolean(exercise.image_url || exercise.video_url)
      }))
    )
  );

// Admin2PlanCreatePage creates and edits platform plans (generic, Level 1 and
// Level 2) with the shared plan builder. The family is chosen first and is
// fixed once the plan exists; the price follows it.
export default function Admin2PlanCreatePage() {
  const history = useHistory();
  const location = useLocation();
  const { locale } = useI18n();
  const text = COPY[locale];
  const params = new URLSearchParams(location.search);
  const editId = params.get("edit") ?? "";
  const familyQuery = params.get("family");
  // ?request=<id>: RYZE prepares a Premium Level 2 client's personal plan (a
  // free platform draft linked to the request on save, never published).
  // ?base=<id>&baseFamily=<family>: a new plan starts as a copy of another.
  const requestId = params.get("request") ?? "";
  const baseId = params.get("base") ?? "";
  const baseFamily = params.get("baseFamily");
  const requestPage = requestId ? `/admin2/premium-level-2/${encodeURIComponent(requestId)}` : "";

  const [family, setFamily] = useState<ProgramProductTypeValue>(
    requestId ? PROGRAM_PRODUCT_TYPE.GENERIC : isProgramProductType(familyQuery) && familyQuery !== PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_2 ? familyQuery : PROGRAM_PRODUCT_TYPE.GENERIC
  );
  const [loadingBase, setLoadingBase] = useState(Boolean(!editId && baseId));
  const [programId, setProgramId] = useState(editId);
  const [draft, setDraft] = useState<PlanDraft>(() => createPlanDraft(family));
  const [status, setStatus] = useState<"draft" | "published">("draft");
  const [loading, setLoading] = useState(Boolean(editId));
  const [loadError, setLoadError] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [message, setMessage] = useState<{ tone: "error" | "success"; text: string } | null>(null);
  const [exercises, setExercises] = useState<Record<string, BuilderExercise>>({});
  const [contentLocked, setContentLocked] = useState(false);
  const [redirect, setRedirect] = useState<string | null>(null);
  const loadedId = useRef("");

  const rememberExercises = useCallback((list: BuilderExercise[]) => {
    setExercises((current) => {
      const next = { ...current };
      for (const exercise of list) {
        next[exercise.id] = { ...current[exercise.id], ...exercise };
      }
      return next;
    });
  }, []);

  useEffect(() => {
    if (!editId || editId === loadedId.current) return;
    let cancelled = false;
    setLoading(true);
    fetchGenericProgram(family, editId)
      .then((detail) => {
        if (cancelled) return;
        loadedId.current = detail.id;
        setDraft(detailToPlanDraft(detail));
        setStatus(detail.status === ProgramStatusEnum.PUBLISHED ? "published" : "draft");
        rememberExercises(exercisesOf(detail));
        setLoading(false);
      })
      .catch(() => {
        if (!cancelled) {
          setLoadError(true);
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
    // The family and id come from the URL the plans list built.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editId]);

  // A new plan based on another one copies its training (weeks, days and
  // details), never its cover, nutrition or catalogue family.
  useEffect(() => {
    if (editId || !baseId) return;
    let cancelled = false;
    const sourceFamily = isProgramProductType(baseFamily) ? baseFamily : PROGRAM_PRODUCT_TYPE.GENERIC;
    fetchGenericProgram(sourceFamily, baseId)
      .then((detail) => {
        if (cancelled) return;
        const copied = detailToPlanDraft(detail);
        setDraft({
          ...createPlanDraft(family),
          name: `${detail.name} ${text.copySuffix}`.slice(0, 120),
          description: copied.description,
          level: copied.level,
          training_type: copied.training_type,
          duration_weeks: copied.duration_weeks,
          frequency_per_week: copied.frequency_per_week,
          audience: copied.audience,
          weeks: copied.weeks
        });
        rememberExercises(exercisesOf(detail));
        setDirty(true);
      })
      .catch(() => {
        // The builder starts empty if the base cannot be read.
      })
      .finally(() => {
        if (!cancelled) setLoadingBase(false);
      });
    return () => {
      cancelled = true;
    };
    // The base is read once, when the page opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!dirty) return;
    const unblock = history.block(text.unsaved);
    const onBeforeUnload = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => {
      unblock();
      window.removeEventListener("beforeunload", onBeforeUnload);
    };
  }, [dirty, history, text.unsaved]);

  // Navigation waits for the unsaved-changes guard to be released.
  useEffect(() => {
    if (redirect && !dirty) {
      history.replace(redirect);
    }
  }, [dirty, history, redirect]);

  const change = useCallback((producer: (current: PlanDraft) => PlanDraft) => {
    setDraft((current) => producer(current));
    setDirty(true);
    setMessage(null);
  }, []);

  const changeFamily = (next: string) => {
    if (!isProgramProductType(next) || next === family) return;
    setFamily(next);
    // A new family starts from its own defaults; the name and description
    // carry over. The cover is cleared because covers live per category.
    setDraft((current) => ({ ...createPlanDraft(next), name: current.name, description: current.description }));
    setDirty(true);
  };

  const describe = (error: unknown) =>
    error instanceof ApiError ? text.errors[error.code ?? ""] ?? error.message ?? text.genericError : text.genericError;

  const save = async (publish: boolean) => {
    setSaving(true);
    setMessage(null);
    try {
      const nextStatus = publish && !requestId ? ProgramStatusEnum.PUBLISHED : status;
      const input = toGenericProgramInput({ ...draft, status: nextStatus });
      const saved = programId
        ? await updateGenericProgram(family, programId, input)
        : await createGenericProgram(family, input);
      setDirty(false);
      loadedId.current = saved.id;
      setProgramId(saved.id);
      setStatus(saved.status === ProgramStatusEnum.PUBLISHED ? "published" : "draft");
      setDraft(detailToPlanDraft(saved));
      rememberExercises(exercisesOf(saved));
      setMessage({ tone: "success", text: saved.status === ProgramStatusEnum.PUBLISHED ? text.published : text.saved });
      if (requestId) {
        // The personal plan goes straight onto its request.
        await attachRyzePremiumLevel2TrainingProgram(requestId, saved.id);
        setRedirect(requestPage);
      } else if (publish) {
        // Published (or a published plan saved): back to the plan list, which
        // confirms it in green (owner request, 2026-11-27).
        setRedirect(`/admin2/plans?family=${family}&${status === "published" ? "saved" : "published"}=${encodeURIComponent(saved.name)}`);
      } else if (!programId) {
        setRedirect(`/admin2/plans/create?edit=${saved.id}&family=${family}`);
      }
    } catch (error) {
      if (error instanceof ApiError && error.code === "PROGRAM_CONTENT_LOCKED") {
        setContentLocked(true);
      }
      setMessage({ tone: "error", text: describe(error) });
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!programId) return;
    setDeleting(true);
    try {
      await deleteGenericProgram(family, programId);
      setDirty(false);
      setRedirect("/admin2/plans");
    } catch (error) {
      setMessage({ tone: "error", text: describe(error) });
      setDeleting(false);
    }
  };

  if (loading || loadingBase) {
    return <AppLoading />;
  }

  if (loadError) {
    return <p role="alert">{text.loadError}</p>;
  }

  const categories: BuilderCategory[] = FAMILIES.map((id) => ({
    id,
    label: text.categories[id].label,
    description: text.categories[id].description,
    priceMinor: CATEGORY_PRICE[id]
  }));

  return (
    <PlanBuilder
      heading={{
        kicker: requestId ? text.requestKicker : text.kicker,
        title: programId || baseId ? draft.name || text.editTitle : requestId ? text.requestTitle : text.newTitle,
        backTo: requestPage || "/admin2/plans",
        backLabel: requestId ? text.requestBack : text.back
      }}
      draftOnlyLabel={requestId ? text.requestSave : undefined}
      draft={draft}
      onChange={change}
      categories={categories}
      category={family}
      onCategoryChange={programId || requestId ? undefined : changeFamily}
      showPackageOptions={family === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1}
      packageOptionsLocked={Boolean(programId)}
      hasStructure={family !== PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_2}
      hasNutrition={family === PROGRAM_PRODUCT_TYPE.PREMIUM_LEVEL_1 && draft.nutrition_mode === "curated"}
      exercises={exercises}
      rememberExercises={rememberExercises}
      uploadCover={(file) => uploadPlatformCover(family, file)}
      officialOption={family === PROGRAM_PRODUCT_TYPE.GENERIC && !requestId}
      status={status}
      saving={saving}
      message={message}
      onSave={(publish) => void save(publish)}
      onDelete={programId ? () => void remove() : undefined}
      deleting={deleting}
      lockedNote={contentLocked ? text.locked : undefined}
    />
  );
}
