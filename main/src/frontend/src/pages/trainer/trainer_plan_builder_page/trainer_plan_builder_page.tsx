import { useCallback, useEffect, useRef, useState } from "react";
import { useHistory, useParams } from "react-router-dom";

import { AppLoading } from "@/components/app_shell/app_loading";
import type { BuilderExercise } from "@/components/plan_builder/exercise_media";
import { PlanBuilder, type BuilderCategory } from "@/components/plan_builder/plan_builder";
import { type Copy, useI18n } from "@/i18n/i18n";
import {
  createPlanDraft,
  detailToPlanDraft,
  toGenericProgramInput,
  type PlanDraft
} from "@/pages/admin2/admin2_plan_create_page/plan_builder_model";
import {
  PROGRAM_PRODUCT_TYPE,
  type GenericProgramDetail,
  type GenericProgramWeek,
  type ProgramTypeValue
} from "@/services/admin2_api";
import {
  createTrainerProgram,
  deleteTrainerProgram,
  fetchTrainerProgram,
  fetchTrainerProgramStructure,
  publishTrainerProgram,
  saveTrainerProgramStructure,
  TrainerProgramStatus,
  TrainerProgramType,
  updateTrainerProgram,
  uploadTrainerCover,
  type TrainerProgram,
  type TrainerProgramTypeValue
} from "@/services/trainer_api";
import { ApiError } from "@utils/http_client";

const COPY: Copy<{
  kicker: string;
  newTitle: string;
  editTitle: string;
  back: string;
  categories: Record<TrainerProgramTypeValue, { label: string; description: string }>;
  loadError: string;
  saved: string;
  published: string;
  locked: string;
  errors: Record<string, string>;
  genericError: string;
  unsaved: string;
}> = {
  en: {
    kicker: "Trainer · programmes",
    newTitle: "New programme",
    editTitle: "Edit programme",
    back: "My programmes",
    categories: {
      free: { label: "Free", description: "Free on your coach profile and in the store." },
      premium: { label: "Premium", description: "Sold at the Level 1 price; you receive it minus the RYZE commission." },
      personalized: { label: "Personalized", description: "Sold at the Level 2 price, for more tailored plans." }
    },
    loadError: "The programme could not be loaded.",
    saved: "Saved as a draft.",
    published: "Published.",
    locked: "This programme was delivered through Premium Level 2 and can no longer change.",
    errors: {
      EXERCISE_NOT_FOUND: "An exercise no longer exists in the catalogue. Remove it and try again.",
      DUPLICATE_EXERCISE: "The same exercise appears twice in one training day.",
      PROGRAM_NOT_FOUND: "The programme no longer exists.",
      PROGRAM_LOCKED: "This programme was delivered through Premium Level 2 and can no longer change.",
      VALIDATION_ERROR: "Some details were not accepted. Check the programme and try again."
    },
    genericError: "The programme could not be saved. Try again.",
    unsaved: "You have unsaved changes. Leave anyway?"
  },
  pt: {
    kicker: "Treinador · programas",
    newTitle: "Novo programa",
    editTitle: "Editar programa",
    back: "Os meus programas",
    categories: {
      free: { label: "Grátis", description: "Grátis no teu perfil de treinador e na loja." },
      premium: { label: "Premium", description: "Vendido ao preço do Nível 1; recebes o valor menos a comissão da RYZE." },
      personalized: { label: "Personalizado", description: "Vendido ao preço do Nível 2, para planos mais à medida." }
    },
    loadError: "Não foi possível carregar o programa.",
    saved: "Guardado como rascunho.",
    published: "Publicado.",
    locked: "Este programa foi entregue num Premium Nível 2 e já não pode mudar.",
    errors: {
      EXERCISE_NOT_FOUND: "Um exercício já não existe no catálogo. Remove-o e tenta de novo.",
      DUPLICATE_EXERCISE: "O mesmo exercício aparece duas vezes num dia de treino.",
      PROGRAM_NOT_FOUND: "O programa já não existe.",
      PROGRAM_LOCKED: "Este programa foi entregue num Premium Nível 2 e já não pode mudar.",
      VALIDATION_ERROR: "Alguns dados não foram aceites. Revê o programa e tenta de novo."
    },
    genericError: "Não foi possível guardar o programa. Tenta de novo.",
    unsaved: "Tens alterações por guardar. Sair na mesma?"
  }
};

const TYPE_PRICE: Record<TrainerProgramTypeValue, number> = { free: 0, premium: 1499, personalized: 1949 };
const TYPES: TrainerProgramTypeValue[] = [TrainerProgramType.FREE, TrainerProgramType.PREMIUM, TrainerProgramType.PERSONALIZED];

// toDetail presents a trainer programme and its structure in the shape the
// shared plan draft mapping reads.
const toDetail = (program: TrainerProgram, weeks: GenericProgramWeek[]): GenericProgramDetail => ({
  id: program.id,
  name: program.name,
  description: program.description,
  type: program.type as ProgramTypeValue,
  product_type: PROGRAM_PRODUCT_TYPE.GENERIC,
  objective: null,
  nutrition_mode: null,
  content_version: 1,
  status: program.status,
  level: (program.level ?? null) as GenericProgramDetail["level"],
  duration_weeks: program.duration_weeks ?? (weeks.length || null),
  frequency_per_week: program.frequency_per_week ?? null,
  training_type: (program.training_type ?? null) as GenericProgramDetail["training_type"],
  price_minor_units: program.price_minor_units,
  currency: program.currency,
  cover_image: program.cover_image ?? null,
  cover_image_url: program.cover_image_url ?? "",
  created_at: program.created_at,
  updated_at: program.updated_at,
  curated_nutrition: null,
  weeks
});

const exercisesOf = (weeks: GenericProgramWeek[]): BuilderExercise[] =>
  weeks.flatMap((week) =>
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

const metadataOf = (draft: PlanDraft) => ({
  name: draft.name.trim(),
  description: draft.description.trim(),
  type: draft.type as TrainerProgramTypeValue,
  level: draft.level,
  training_type: draft.training_type,
  duration_weeks: Number(draft.duration_weeks) || 0,
  frequency_per_week: Number(draft.frequency_per_week) || 0
});

// TrainerPlanBuilderPage creates and edits a trainer's own programme with the
// shared plan builder: details first, then the weeks and days, then review.
// The programme metadata and its whole structure are saved together; the
// price follows the chosen type.
export default function TrainerPlanBuilderPage() {
  const history = useHistory();
  const { programId: routeId } = useParams<{ programId?: string }>();
  const { locale } = useI18n();
  const text = COPY[locale];

  const [programId, setProgramId] = useState(routeId && routeId !== "new" ? routeId : "");
  const [draft, setDraft] = useState<PlanDraft>(() => ({
    ...createPlanDraft(PROGRAM_PRODUCT_TYPE.GENERIC),
    type: TrainerProgramType.FREE
  }));
  const [status, setStatus] = useState<"draft" | "published">("draft");
  const [loading, setLoading] = useState(Boolean(programId));
  const [loadError, setLoadError] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [message, setMessage] = useState<{ tone: "error" | "success"; text: string } | null>(null);
  const [exercises, setExercises] = useState<Record<string, BuilderExercise>>({});
  const [locked, setLocked] = useState(false);
  const [redirect, setRedirect] = useState<string | null>(null);
  const savedCover = useRef("");
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
    if (!programId || programId === loadedId.current) return;
    let cancelled = false;
    setLoading(true);
    Promise.all([fetchTrainerProgram(programId), fetchTrainerProgramStructure(programId)])
      .then(([program, weeks]) => {
        if (cancelled) return;
        loadedId.current = program.id;
        setDraft(detailToPlanDraft(toDetail(program, weeks)));
        setStatus(program.status === TrainerProgramStatus.PUBLISHED ? "published" : "draft");
        savedCover.current = program.cover_image ?? "";
        rememberExercises(exercisesOf(weeks));
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
  }, [programId, rememberExercises]);

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

  const describe = (error: unknown) =>
    error instanceof ApiError ? text.errors[error.code ?? ""] ?? error.message ?? text.genericError : text.genericError;

  const save = async (publish: boolean) => {
    setSaving(true);
    setMessage(null);
    try {
      const metadata = metadataOf(draft);
      // The cover is sent only when it changed: a cover keeps the category
      // folder it was uploaded to even if the type changes later.
      const coverChanged = draft.cover_image !== savedCover.current;
      let id = programId;
      if (!id) {
        const created = await createTrainerProgram({
          ...metadata,
          status: TrainerProgramStatus.DRAFT,
          cover_image: draft.cover_image
        });
        id = created.id;
        loadedId.current = id;
        setProgramId(id);
      } else {
        await updateTrainerProgram(id, { ...metadata, ...(coverChanged ? { cover_image: draft.cover_image } : {}) });
      }
      savedCover.current = draft.cover_image;
      const weeks = await saveTrainerProgramStructure(id, toGenericProgramInput(draft).weeks);
      let program = await fetchTrainerProgram(id);
      if (publish && program.status === TrainerProgramStatus.DRAFT) {
        program = await publishTrainerProgram(id);
      }
      setDraft(detailToPlanDraft(toDetail(program, weeks)));
      rememberExercises(exercisesOf(weeks));
      setStatus(program.status === TrainerProgramStatus.PUBLISHED ? "published" : "draft");
      setDirty(false);
      setMessage({ tone: "success", text: program.status === TrainerProgramStatus.PUBLISHED ? text.published : text.saved });
      if (!routeId || routeId === "new") {
        setRedirect(`/trainer/programs/${id}`);
      }
    } catch (error) {
      if (error instanceof ApiError && error.code === "PROGRAM_LOCKED") {
        setLocked(true);
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
      await deleteTrainerProgram(programId);
      setDirty(false);
      setRedirect("/trainer/programs");
    } catch (error) {
      setMessage({ tone: "error", text: describe(error) });
      setDeleting(false);
    }
  };

  if (loading) {
    return <AppLoading />;
  }

  if (loadError) {
    return <p role="alert">{text.loadError}</p>;
  }

  const categories: BuilderCategory[] = TYPES.map((id) => ({
    id,
    label: text.categories[id].label,
    description: text.categories[id].description,
    priceMinor: TYPE_PRICE[id]
  }));

  return (
    <PlanBuilder
      heading={{
        kicker: text.kicker,
        title: programId ? draft.name || text.editTitle : text.newTitle,
        backTo: "/trainer/programs",
        backLabel: text.back
      }}
      draft={draft}
      onChange={change}
      categories={categories}
      category={draft.type}
      onCategoryChange={(id) => change((current) => ({ ...current, type: id as ProgramTypeValue }))}
      showPackageOptions={false}
      packageOptionsLocked
      hasStructure
      hasNutrition={false}
      exercises={exercises}
      rememberExercises={rememberExercises}
      uploadCover={(file) => uploadTrainerCover(draft.type as TrainerProgramTypeValue, file)}
      status={status}
      saving={saving}
      message={message}
      onSave={(publish) => void save(publish)}
      onDelete={programId ? () => void remove() : undefined}
      deleting={deleting}
      lockedNote={locked ? text.locked : undefined}
    />
  );
}
