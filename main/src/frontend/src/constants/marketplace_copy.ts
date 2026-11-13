import { type Copy, useI18n } from "@/i18n/i18n";

// Text of the training-plan catalogue (page, filters, search, rows and cards).
// Filter options and program metadata stay in English as API values; only
// the label shown to the visitor is translated (see valueLabel).

interface MarketplaceText {
  title: string;
  subtitle: string;
  results: (count: number) => string;
  activeFilters: (count: number) => string;
  filters: string;
  expandFilters: string;
  collapseFilters: string;
  closeFilters: string;
  clearAll: string;
  loading: string;
  loadError: string;
  retry: string;
  noMatch: string;
  noneYet: string;
  endOfCatalog: (perSection: number, count: number, filtered: boolean) => string;
  searchPlaceholder: string;
  searchLabel: string;
  programCount: (count: number) => string;
  searching: string;
  noResults: string;
  weeks: (count: number) => string;
  perWeek: (count: number) => string;
  daysPerWeek: (count: number) => string;
  previous: (row: string) => string;
  next: (row: string) => string;
  open: (name: string) => string;
  view: (name: string) => string;
  free: string;
  owned: string;
  featuredTitle: string;
  featuredDescription: string;
  coachLink: (name: string) => string;
  groups: Record<string, string>;
  rows: Record<string, { title: string; description?: string }>;
  values: Record<string, string>;
}

const plural = (count: number, one: string, many: string) => (count === 1 ? one : many);

const en: MarketplaceText = {
  title: "Training plans",
  subtitle: "Find a program built around your goals, schedule and experience.",
  results: (count) => `${count} ${plural(count, "result", "results")} for your filters`,
  activeFilters: (count) => `(${count} ${plural(count, "filter", "filters")} active)`,
  filters: "Filters",
  expandFilters: "Expand filters",
  collapseFilters: "Collapse filters",
  closeFilters: "Close filters",
  clearAll: "Clear all",
  loading: "Loading training plans…",
  loadError: "Unable to load the training plan catalogue. Please try again.",
  retry: "Retry",
  noMatch: "No training plans match your filters.",
  noneYet: "No training plans have been published yet.",
  endOfCatalog: (perSection, count, filtered) =>
    `Showing up to ${perSection} programs per section · ${count} ${plural(count, "program", "programs")} match${filtered ? " your filters" : ""}`,
  searchPlaceholder: "Search training plans...",
  searchLabel: "Search training plans",
  programCount: (count) => `${count} ${plural(count, "program", "programs")}`,
  searching: "Searching...",
  noResults: "No results found",
  weeks: (count) => `${count} weeks`,
  perWeek: (count) => `${count}/week`,
  daysPerWeek: (count) => `${count} days/week`,
  previous: (row) => `Previous ${row} programs`,
  next: (row) => `Next ${row} programs`,
  open: (name) => `Open ${name}`,
  view: (name) => `View ${name}`,
  free: "Free",
  owned: "Owned",
  featuredTitle: "Featured coach programs",
  featuredDescription: "The most viewed plans from RYZE coaches. Tap a coach to see all their programs.",
  coachLink: (name) => `See ${name}'s coach profile`,
  groups: {},
  rows: {},
  values: {}
};

const pt: MarketplaceText = {
  title: "Planos de treino",
  subtitle: "Encontra um programa feito para os teus objetivos, horário e experiência.",
  results: (count) => `${count} ${plural(count, "resultado", "resultados")} para os teus filtros`,
  activeFilters: (count) => `(${count} ${plural(count, "filtro ativo", "filtros ativos")})`,
  filters: "Filtros",
  expandFilters: "Mostrar filtros",
  collapseFilters: "Esconder filtros",
  closeFilters: "Fechar filtros",
  clearAll: "Limpar tudo",
  loading: "A carregar os planos de treino…",
  loadError: "Não foi possível carregar o catálogo de planos de treino. Tenta novamente.",
  retry: "Tentar novamente",
  noMatch: "Nenhum plano de treino corresponde aos teus filtros.",
  noneYet: "Ainda não foram publicados planos de treino.",
  endOfCatalog: (perSection, count, filtered) =>
    `Até ${perSection} programas por secção · ${count} ${plural(count, "programa corresponde", "programas correspondem")}${filtered ? " aos teus filtros" : ""}`,
  searchPlaceholder: "Pesquisar planos de treino...",
  searchLabel: "Pesquisar planos de treino",
  programCount: (count) => `${count} ${plural(count, "programa", "programas")}`,
  searching: "A pesquisar...",
  noResults: "Sem resultados",
  weeks: (count) => `${count} ${plural(count, "semana", "semanas")}`,
  perWeek: (count) => `${count}/semana`,
  daysPerWeek: (count) => `${count} ${plural(count, "dia", "dias")}/semana`,
  previous: (row) => `Programas anteriores de ${row}`,
  next: (row) => `Programas seguintes de ${row}`,
  open: (name) => `Abrir ${name}`,
  view: (name) => `Ver ${name}`,
  free: "Grátis",
  owned: "Já é teu",
  featuredTitle: "Programas de treinadores em destaque",
  featuredDescription: "Os planos mais vistos dos treinadores RYZE. Toca num treinador para ver todos os seus programas.",
  coachLink: (name) => `Ver o perfil de treinador de ${name}`,
  groups: {
    "training-type": "Tipo de treino",
    frequency: "Frequência de treino",
    level: "Nível",
    duration: "Duração do plano"
  },
  rows: {
    "all-programs": { title: "Todos os planos de treino", description: "Explora todos os planos publicados no catálogo." },
    "recently-updated": { title: "Atualizados recentemente", description: "Planos com as atualizações mais recentes." },
    "new-noteworthy": { title: "Novidades", description: "Planos publicados recentemente." },
    strength: { title: "Ganhar força" },
    hypertrophy: { title: "Ganhar massa muscular" },
    "fat-loss": { title: "Perda de gordura" },
    "at-home": { title: "Treinar em casa" },
    performance: { title: "Performance e condição física" }
  },
  values: {
    Hypertrophy: "Hipertrofia",
    Strength: "Força",
    "Fat Loss": "Perda de gordura",
    "At Home": "Em casa",
    Beginner: "Iniciante",
    Intermediate: "Intermédio",
    Advanced: "Avançado",
    "1 day / week": "1 dia / semana",
    "2 days / week": "2 dias / semana",
    "3 days / week": "3 dias / semana",
    "4 days / week": "4 dias / semana",
    "5 days / week": "5 dias / semana",
    "6 days / week": "6 dias / semana",
    "7 days / week": "7 dias / semana",
    "1–2 weeks": "1–2 semanas",
    "3–4 weeks": "3–4 semanas",
    "5–8 weeks": "5–8 semanas",
    "9–12 weeks": "9–12 semanas"
  }
};

export const MARKETPLACE_COPY: Copy<MarketplaceText> = { en, pt };

// useMarketplaceCopy returns the catalogue text plus helpers that translate
// API values (training types, levels, filter options), group titles and row
// titles, falling back to the English value.
export const useMarketplaceCopy = () => {
  const { locale } = useI18n();
  const text = MARKETPLACE_COPY[locale];
  return {
    text,
    locale,
    valueLabel: (value: string | null | undefined) => (value ? (text.values[value] ?? value) : ""),
    groupTitle: (id: string, fallback: string) => text.groups[id] ?? fallback,
    row: (id: string, fallback: { title: string; description?: string }) => text.rows[id] ?? fallback
  };
};
