import type { Copy } from "@/i18n/i18n";

export interface BuilderCopy {
  steps: { basics: string; structure: string; nutrition: string; review: string };
  stepOf: (current: number, total: number) => string;
  back: string;
  next: string;
  // Basics
  categoryTitle: string;
  categoryLocked: string;
  coverTitle: string;
  coverHint: string;
  coverChoose: string;
  coverReplace: string;
  coverRemove: string;
  coverUploading: string;
  coverError: string;
  coverTypes: string;
  name: string;
  namePlaceholder: string;
  description: string;
  descriptionPlaceholder: string;
  goal: string;
  goals: Record<string, string>;
  level: string;
  levels: Record<string, string>;
  weeks: string;
  weeksHint: string;
  daysPerWeek: string;
  daysHint: string;
  decrease: string;
  increase: string;
  objective: string;
  objectives: Record<string, string>;
  nutritionMode: string;
  nutritionCurated: string;
  nutritionCuratedText: string;
  nutritionQuestionnaire: string;
  nutritionQuestionnaireText: string;
  metadataLocked: string;
  level2Note: string;
  // Structure
  week: (n: number) => string;
  weeksCount: (n: number) => string;
  bodyParts: Record<string, string>;
  addWeek: string;
  removeWeek: string;
  copyForward: string;
  copyForwardDone: string;
  day: (n: number) => string;
  dayName: string;
  dayNamePlaceholder: string;
  dayDescription: string;
  dayDescriptionPlaceholder: string;
  dayFocus: string;
  focusNone: string;
  focus: Record<string, string>;
  addDay: string;
  removeDay: string;
  addExercise: string;
  noExercises: string;
  set: string;
  reps: string;
  weight: string;
  rest: string;
  rir: string;
  addSet: string;
  removeSet: (n: number) => string;
  notes: string;
  notesPlaceholder: string;
  moveUp: string;
  moveDown: string;
  removeExercise: string;
  openExercise: (name: string) => string;
  restPlaceholder: string;
  // Picker
  pickerTitle: (day: string) => string;
  search: string;
  allMuscles: string;
  anyEquipment: string;
  loading: string;
  loadMore: string;
  noResults: string;
  add: string;
  added: string;
  addToDay: string;
  done: (n: number) => string;
  close: string;
  preview: string;
  mediaUnavailable: string;
  // Exercise sheet
  sheetPrescription: string;
  setType: string;
  setTypes: Record<string, string>;
  tempo: string;
  rpe: string;
  coachNotes: string;
  instructionsLabel: string;
  instructionsPlaceholder: string;
  howTo: string;
  muscles: string;
  equipment: string;
  // Review
  summary: string;
  price: string;
  free: string;
  fixedPrice: string;
  stats: { weeks: string; days: string; exercises: string; sets: string };
  checklist: string;
  ready: string;
  saveDraft: string;
  publish: string;
  saveChanges: string;
  saving: string;
  published: string;
  draft: string;
  deletePlan: string;
  deleteConfirm: string;
  deleting: string;
  unsaved: string;
  restDay: string;
}

const FOCUS_EN: Record<string, string> = {
  full_body: "Full body",
  upper_body: "Upper body",
  lower_body: "Lower body",
  push: "Push",
  pull: "Pull",
  legs: "Legs",
  chest: "Chest",
  back: "Back",
  shoulders: "Shoulders",
  arms: "Arms",
  core: "Core",
  glutes: "Glutes",
  cardio: "Cardio",
  mobility: "Mobility"
};

const FOCUS_PT: Record<string, string> = {
  full_body: "Corpo inteiro",
  upper_body: "Parte superior",
  lower_body: "Parte inferior",
  push: "Empurrar",
  pull: "Puxar",
  legs: "Pernas",
  chest: "Peito",
  back: "Costas",
  shoulders: "Ombros",
  arms: "Braços",
  core: "Core",
  glutes: "Glúteos",
  cardio: "Cardio",
  mobility: "Mobilidade"
};

export const BUILDER_COPY: Copy<BuilderCopy> = {
  en: {
    steps: { basics: "Plan", structure: "Structure", nutrition: "Nutrition", review: "Review" },
    stepOf: (current, total) => `Step ${current} of ${total}`,
    back: "Back",
    next: "Continue",
    categoryTitle: "Category",
    categoryLocked: "The category is fixed once the plan exists.",
    coverTitle: "Cover image",
    coverHint: "Shown on the plan card and page. A wide picture works best (16:9).",
    coverChoose: "Choose image",
    coverReplace: "Replace",
    coverRemove: "Remove",
    coverUploading: "Uploading…",
    coverError: "The image could not be uploaded. Use a JPEG, PNG or WebP up to 5 MB.",
    coverTypes: "JPEG, PNG or WebP · up to 5 MB",
    name: "Plan name",
    namePlaceholder: "e.g. Strength foundations",
    description: "Description",
    descriptionPlaceholder: "Who the plan is for, what it achieves and how a week looks.",
    goal: "Goal",
    goals: {
      Hypertrophy: "Build muscle",
      Strength: "Strength",
      "Fat Loss": "Lose fat",
      HYROX: "HYROX",
      CrossFit: "CrossFit",
      "At Home": "At home"
    },
    level: "Difficulty",
    levels: { Beginner: "Beginner", Intermediate: "Intermediate", Advanced: "Advanced" },
    weeks: "Weeks",
    weeksHint: "The weeks are created for you in the next step.",
    daysPerWeek: "Training days per week",
    daysHint: "Each new week starts with this many training days.",
    decrease: "Decrease",
    increase: "Increase",
    objective: "Package objective",
    objectives: { gain_muscle: "Gain muscle", lose_fat: "Lose fat", strength_conditioning: "Strength & conditioning" },
    nutritionMode: "Nutrition",
    nutritionCurated: "Curated plan",
    nutritionCuratedText: "You write one nutrition plan that every buyer receives.",
    nutritionQuestionnaire: "Questionnaire",
    nutritionQuestionnaireText: "Each buyer answers a questionnaire and gets a generated plan.",
    metadataLocked: "Objective and nutrition model are fixed once the package exists.",
    level2Note: "A Level 2 offer has no training structure: each client's programme is written by the assigned trainer after purchase.",
    week: (n) => `Week ${n}`,
    weeksCount: (n) => `${n} ${n === 1 ? "week" : "weeks"}`,
    bodyParts: {},
    addWeek: "Add week",
    removeWeek: "Remove this week",
    copyForward: "Copy this week to the next weeks",
    copyForwardDone: "Copied to the following weeks.",
    day: (n) => `Day ${n}`,
    dayName: "Day name",
    dayNamePlaceholder: "e.g. Push — chest & triceps",
    dayDescription: "Description",
    dayDescriptionPlaceholder: "What this session is about (optional)",
    dayFocus: "Focus",
    focusNone: "No focus",
    focus: FOCUS_EN,
    addDay: "Add training day",
    removeDay: "Remove day",
    addExercise: "Add exercises",
    noExercises: "No exercises yet.",
    set: "Set",
    reps: "Reps",
    weight: "kg",
    rest: "Rest",
    rir: "RIR",
    addSet: "Add set",
    removeSet: (n) => `Remove set ${n}`,
    notes: "Notes",
    notesPlaceholder: "Cue or note for this exercise (optional)",
    moveUp: "Move up",
    moveDown: "Move down",
    removeExercise: "Remove exercise",
    openExercise: (name) => `Open ${name}`,
    restPlaceholder: "e.g. 90 or 1:30",
    pickerTitle: (day) => `Add exercises to ${day}`,
    search: "Search exercises",
    allMuscles: "All",
    anyEquipment: "Any equipment",
    loading: "Loading…",
    loadMore: "Show more",
    noResults: "No exercise matches these filters.",
    add: "Add",
    added: "Added",
    addToDay: "Add to the day",
    done: (n) => (n > 0 ? `Done · ${n} added` : "Done"),
    close: "Close",
    preview: "Preview",
    mediaUnavailable: "Demonstration not available",
    sheetPrescription: "Prescription",
    setType: "Type",
    setTypes: { warmup: "Warm-up", working: "Working", drop: "Drop set", backoff: "Back-off", failure: "To failure" },
    tempo: "Tempo",
    rpe: "RPE",
    coachNotes: "Notes",
    instructionsLabel: "Instructions for the client",
    instructionsPlaceholder: "How to perform it in this plan (optional)",
    howTo: "How to do it",
    muscles: "Muscles",
    equipment: "Equipment",
    summary: "Summary",
    price: "Price",
    free: "Free",
    fixedPrice: "Fixed by category",
    stats: { weeks: "Weeks", days: "Training days", exercises: "Exercises", sets: "Sets" },
    checklist: "Before publishing",
    ready: "Everything required is filled in.",
    saveDraft: "Save draft",
    publish: "Publish",
    saveChanges: "Save changes",
    saving: "Saving…",
    published: "Published",
    draft: "Draft",
    deletePlan: "Delete plan",
    deleteConfirm: "Delete this plan? This cannot be undone.",
    deleting: "Deleting…",
    unsaved: "You have unsaved changes. Leave anyway?",
    restDay: "Empty day"
  },
  pt: {
    steps: { basics: "Plano", structure: "Estrutura", nutrition: "Nutrição", review: "Rever" },
    stepOf: (current, total) => `Passo ${current} de ${total}`,
    back: "Voltar",
    next: "Continuar",
    categoryTitle: "Categoria",
    categoryLocked: "A categoria fica fixa depois de o plano existir.",
    coverTitle: "Imagem de capa",
    coverHint: "Aparece no cartão e na página do plano. Uma imagem larga resulta melhor (16:9).",
    coverChoose: "Escolher imagem",
    coverReplace: "Trocar",
    coverRemove: "Remover",
    coverUploading: "A enviar…",
    coverError: "Não foi possível enviar a imagem. Usa JPEG, PNG ou WebP até 5 MB.",
    coverTypes: "JPEG, PNG ou WebP · até 5 MB",
    name: "Nome do plano",
    namePlaceholder: "ex.: Bases de força",
    description: "Descrição",
    descriptionPlaceholder: "Para quem é, o que consegue e como é uma semana.",
    goal: "Objetivo",
    goals: {
      Hypertrophy: "Ganhar músculo",
      Strength: "Força",
      "Fat Loss": "Perder gordura",
      HYROX: "HYROX",
      CrossFit: "CrossFit",
      "At Home": "Em casa"
    },
    level: "Dificuldade",
    levels: { Beginner: "Iniciante", Intermediate: "Intermédio", Advanced: "Avançado" },
    weeks: "Semanas",
    weeksHint: "As semanas são criadas automaticamente no passo seguinte.",
    daysPerWeek: "Dias de treino por semana",
    daysHint: "Cada semana nova começa com este número de dias de treino.",
    decrease: "Diminuir",
    increase: "Aumentar",
    objective: "Objetivo do pacote",
    objectives: { gain_muscle: "Ganhar músculo", lose_fat: "Perder gordura", strength_conditioning: "Força e condição física" },
    nutritionMode: "Nutrição",
    nutritionCurated: "Plano definido",
    nutritionCuratedText: "Escreves um plano de nutrição que todos os compradores recebem.",
    nutritionQuestionnaire: "Questionário",
    nutritionQuestionnaireText: "Cada comprador responde a um questionário e recebe um plano gerado.",
    metadataLocked: "O objetivo e o modelo de nutrição ficam fixos depois de o pacote existir.",
    level2Note: "Uma oferta de Nível 2 não tem estrutura de treino: o programa de cada cliente é escrito pelo treinador atribuído depois da compra.",
    week: (n) => `Semana ${n}`,
    weeksCount: (n) => `${n} ${n === 1 ? "semana" : "semanas"}`,
    bodyParts: {
      chest: "Peito",
      back: "Costas",
      shoulders: "Ombros",
      "upper arms": "Braços",
      "lower arms": "Antebraços",
      "upper legs": "Coxas",
      "lower legs": "Gémeos",
      waist: "Abdómen",
      neck: "Pescoço",
      cardio: "Cardio"
    },
    addWeek: "Adicionar semana",
    removeWeek: "Remover esta semana",
    copyForward: "Copiar esta semana para as seguintes",
    copyForwardDone: "Copiada para as semanas seguintes.",
    day: (n) => `Dia ${n}`,
    dayName: "Nome do dia",
    dayNamePlaceholder: "ex.: Empurrar — peito e tríceps",
    dayDescription: "Descrição",
    dayDescriptionPlaceholder: "Do que trata esta sessão (opcional)",
    dayFocus: "Foco",
    focusNone: "Sem foco",
    focus: FOCUS_PT,
    addDay: "Adicionar dia de treino",
    removeDay: "Remover dia",
    addExercise: "Adicionar exercícios",
    noExercises: "Ainda sem exercícios.",
    set: "Série",
    reps: "Reps",
    weight: "kg",
    rest: "Descanso",
    rir: "RIR",
    addSet: "Adicionar série",
    removeSet: (n) => `Remover série ${n}`,
    notes: "Notas",
    notesPlaceholder: "Dica ou nota para este exercício (opcional)",
    moveUp: "Subir",
    moveDown: "Descer",
    removeExercise: "Remover exercício",
    openExercise: (name) => `Abrir ${name}`,
    restPlaceholder: "ex.: 90 ou 1:30",
    pickerTitle: (day) => `Adicionar exercícios ao ${day}`,
    search: "Pesquisar exercícios",
    allMuscles: "Todos",
    anyEquipment: "Qualquer equipamento",
    loading: "A carregar…",
    loadMore: "Mostrar mais",
    noResults: "Nenhum exercício corresponde a estes filtros.",
    add: "Adicionar",
    added: "Adicionado",
    addToDay: "Adicionar ao dia",
    done: (n) => (n > 0 ? `Concluir · ${n} adicionados` : "Concluir"),
    close: "Fechar",
    preview: "Pré-visualizar",
    mediaUnavailable: "Demonstração indisponível",
    sheetPrescription: "Prescrição",
    setType: "Tipo",
    setTypes: { warmup: "Aquecimento", working: "Normal", drop: "Drop set", backoff: "Back-off", failure: "Até à falha" },
    tempo: "Tempo",
    rpe: "RPE",
    coachNotes: "Notas",
    instructionsLabel: "Instruções para o cliente",
    instructionsPlaceholder: "Como executar neste plano (opcional)",
    howTo: "Como fazer",
    muscles: "Músculos",
    equipment: "Equipamento",
    summary: "Resumo",
    price: "Preço",
    free: "Grátis",
    fixedPrice: "Fixo pela categoria",
    stats: { weeks: "Semanas", days: "Dias de treino", exercises: "Exercícios", sets: "Séries" },
    checklist: "Antes de publicar",
    ready: "Está tudo preenchido.",
    saveDraft: "Guardar rascunho",
    publish: "Publicar",
    saveChanges: "Guardar alterações",
    saving: "A guardar…",
    published: "Publicado",
    draft: "Rascunho",
    deletePlan: "Eliminar plano",
    deleteConfirm: "Eliminar este plano? Não é possível anular.",
    deleting: "A eliminar…",
    unsaved: "Tens alterações por guardar. Sair na mesma?",
    restDay: "Dia vazio"
  }
};
