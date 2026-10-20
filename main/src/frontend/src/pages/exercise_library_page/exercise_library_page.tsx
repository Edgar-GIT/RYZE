import { ChevronLeft, ChevronRight, Search, SlidersHorizontal, X } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import {
  countActiveLibraryFilters,
  DIFFICULTY_LEVELS,
  exerciseBodyPartLabel,
  EXERCISE_BODY_PART_OPTIONS,
  EXERCISE_EQUIPMENT_OPTIONS,
  EXERCISE_MUSCLE_GROUPS,
  exerciseMediaView,
  exerciseMetaLine,
  fetchExerciseLibrary,
  MOVEMENT_CATEGORIES,
  type ExerciseCatalogEntry
} from "@/services/exercise_library";
import { joinClassNames } from "@utils/class_names";

import styles from "./exercise_library_page.module.css";

const PAGE_LIMIT = 24;
const SEARCH_DEBOUNCE_MS = 250;

interface LibraryFilters {
  bodyPart: string;
  muscle: string;
  equipment: string;
  difficulty: string;
  category: string;
}

const EMPTY_FILTERS: LibraryFilters = {
  bodyPart: "",
  muscle: "",
  equipment: "",
  difficulty: "",
  category: ""
};

// ExerciseMediaThumb renders the card media only while the backend reports it
// available, and always keeps the Gym visual attribution visible next to it.
const ExerciseMediaThumb = ({ exercise }: { exercise: ExerciseCatalogEntry }) => {
  const media = exerciseMediaView(exercise);

  if (media.kind === "none") {
    return (
      <div className={styles.thumbPlaceholder} aria-hidden="true">
        <span className={styles.thumbPlaceholderText}>Media unavailable</span>
      </div>
    );
  }

  return (
    <div className={styles.thumb}>
      {media.kind === "video" ? (
        <video
          src={media.url}
          muted
          loop
          playsInline
          preload="metadata"
          className={styles.thumbMedia}
          aria-label={`${exercise.name} demonstration`}
        />
      ) : (
        <img
          src={media.url}
          alt={`${exercise.name} demonstration`}
          className={styles.thumbMedia}
          loading="lazy"
        />
      )}
      {media.attribution ? <span className={styles.thumbAttribution}>{media.attribution}</span> : null}
    </div>
  );
};

export const ExerciseLibraryPage = () => {
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [filters, setFilters] = useState<LibraryFilters>(EMPTY_FILTERS);
  const [page, setPage] = useState(1);
  const [reloadToken, setReloadToken] = useState(0);

  const [exercises, setExercises] = useState<ExerciseCatalogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    const timer = window.setTimeout(() => setDebouncedQuery(query.trim()), SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [query]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    fetchExerciseLibrary(
      {
        query: debouncedQuery,
        bodyPart: filters.bodyPart,
        muscle: filters.muscle,
        equipment: filters.equipment,
        difficulty: filters.difficulty,
        category: filters.category
      },
      page,
      PAGE_LIMIT
    )
      .then((result) => {
        if (cancelled) {
          return;
        }
        setExercises(result.exercises);
        setTotal(result.pagination?.total ?? result.exercises.length);
        setTotalPages(Math.max(1, result.pagination?.total_pages ?? 1));
      })
      .catch((reason: unknown) => {
        if (!cancelled) {
          setExercises([]);
          setTotal(0);
          setTotalPages(1);
          setError(reason instanceof Error ? reason.message : "Could not load the exercise catalogue.");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [debouncedQuery, filters, page, reloadToken]);

  const activeFilterCount = countActiveLibraryFilters({ ...filters, query: debouncedQuery });

  const updateFilter = (key: keyof LibraryFilters, value: string): void => {
    setFilters((current) => ({ ...current, [key]: value }));
    setPage(1);
  };

  const clearAll = (): void => {
    setFilters(EMPTY_FILTERS);
    setQuery("");
    setDebouncedQuery("");
    setPage(1);
  };

  const firstResult = total === 0 ? 0 : (page - 1) * PAGE_LIMIT + 1;
  const lastResult = Math.min(page * PAGE_LIMIT, total);

  return (
    <PageWrapper className={styles.page}>
      <section className={styles.library}>
        <Container size="wide" className={styles.container}>
          <header className={styles.heading}>
            <span className={styles.eyebrow}>Exercise Library</span>
            <h1 className={styles.title}>Every movement, one catalogue</h1>
            <p className={styles.subtitle}>
              Browse the full RYZE exercise catalogue by body part, muscle, equipment and level.
              Open any exercise for step-by-step instructions and alternatives.
            </p>
          </header>

          <div className={styles.toolbar}>
            <div className={styles.searchField}>
              <Search size={16} aria-hidden="true" className={styles.searchIcon} />
              <input
                type="search"
                value={query}
                onChange={(event) => {
                  setQuery(event.target.value);
                  setPage(1);
                }}
                placeholder="Search exercises by name…"
                aria-label="Search exercises"
                className={styles.searchInput}
              />
              {query ? (
                <button
                  type="button"
                  onClick={() => {
                    setQuery("");
                    setPage(1);
                  }}
                  aria-label="Clear search"
                  className={styles.searchClear}
                >
                  <X size={14} />
                </button>
              ) : null}
            </div>

            <div className={styles.filters}>
              <label className={styles.filterItem}>
                <span className={styles.filterLabel}>Body part</span>
                <select
                  value={filters.bodyPart}
                  onChange={(event) => updateFilter("bodyPart", event.target.value)}
                  className={styles.select}
                  aria-label="Filter by body part"
                >
                  <option value="">All body parts</option>
                  {EXERCISE_BODY_PART_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </label>

              <label className={styles.filterItem}>
                <span className={styles.filterLabel}>Muscle</span>
                <select
                  value={filters.muscle}
                  onChange={(event) => updateFilter("muscle", event.target.value)}
                  className={styles.select}
                  aria-label="Filter by muscle group"
                >
                  <option value="">All muscles</option>
                  {EXERCISE_MUSCLE_GROUPS.map((group) => (
                    <option key={group} value={group}>
                      {group}
                    </option>
                  ))}
                </select>
              </label>

              <label className={styles.filterItem}>
                <span className={styles.filterLabel}>Equipment</span>
                <select
                  value={filters.equipment}
                  onChange={(event) => updateFilter("equipment", event.target.value)}
                  className={styles.select}
                  aria-label="Filter by equipment"
                >
                  <option value="">All equipment</option>
                  {EXERCISE_EQUIPMENT_OPTIONS.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              </label>

              <label className={styles.filterItem}>
                <span className={styles.filterLabel}>Difficulty</span>
                <select
                  value={filters.difficulty}
                  onChange={(event) => updateFilter("difficulty", event.target.value)}
                  className={styles.select}
                  aria-label="Filter by difficulty"
                >
                  <option value="">All levels</option>
                  {DIFFICULTY_LEVELS.map((level) => (
                    <option key={level} value={level}>
                      {level}
                    </option>
                  ))}
                </select>
              </label>

              <label className={styles.filterItem}>
                <span className={styles.filterLabel}>Category</span>
                <select
                  value={filters.category}
                  onChange={(event) => updateFilter("category", event.target.value)}
                  className={styles.select}
                  aria-label="Filter by movement category"
                >
                  <option value="">All categories</option>
                  {MOVEMENT_CATEGORIES.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              </label>

              {activeFilterCount > 0 ? (
                <button type="button" onClick={clearAll} className={styles.clearFilters}>
                  <SlidersHorizontal size={13} aria-hidden="true" />
                  Clear {activeFilterCount} filter{activeFilterCount === 1 ? "" : "s"}
                </button>
              ) : null}
            </div>
          </div>

          <div className={styles.resultBar} aria-live="polite">
            {loading
              ? "Loading catalogue…"
              : error
                ? "Catalogue unavailable"
                : total === 0
                  ? "No exercises match the current search."
                  : `Showing ${firstResult}–${lastResult} of ${total} exercise${total === 1 ? "" : "s"}`}
          </div>

          {loading ? (
            <div className={styles.pageState}>
              <p>Loading exercises…</p>
            </div>
          ) : error ? (
            <div className={styles.pageState}>
              <p>{error}</p>
              <div className={styles.retry}>
                <Button variant="secondary" size="small" onClick={() => setReloadToken((current) => current + 1)}>
                  Retry
                </Button>
              </div>
            </div>
          ) : exercises.length === 0 ? (
            <div className={styles.pageState}>
              <p>No exercises match the current search or filters.</p>
              {activeFilterCount > 0 ? (
                <div className={styles.retry}>
                  <Button variant="secondary" size="small" onClick={clearAll}>
                    Clear search and filters
                  </Button>
                </div>
              ) : null}
            </div>
          ) : (
            <ul className={styles.grid}>
              {exercises.map((exercise) => (
                <li key={exercise.id} className={styles.cardItem}>
                  <Link to={`/exercises/${exercise.id}`} className={styles.card}>
                    <ExerciseMediaThumb exercise={exercise} />
                    <div className={styles.cardBody}>
                      <h2 className={styles.cardName}>{exercise.name}</h2>
                      <p className={styles.cardMeta}>{exerciseMetaLine(exercise)}</p>
                      {exercise.body_part ? (
                        <span className={styles.cardTag}>{exerciseBodyPartLabel(exercise.body_part)}</span>
                      ) : null}
                    </div>
                  </Link>
                </li>
              ))}
            </ul>
          )}

          {!loading && !error && total > PAGE_LIMIT ? (
            <nav className={styles.pager} aria-label="Catalogue pagination">
              <Button
                variant="secondary"
                size="small"
                icon={<ChevronLeft size={15} />}
                iconPosition="left"
                disabled={page <= 1}
                onClick={() => setPage((current) => Math.max(1, current - 1))}
              >
                Previous
              </Button>
              <span className={styles.pagerStatus}>
                Page {page} of {totalPages}
              </span>
              <Button
                variant="secondary"
                size="small"
                icon={<ChevronRight size={15} />}
                disabled={page >= totalPages}
                onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
              >
                Next
              </Button>
            </nav>
          ) : null}
        </Container>
      </section>
    </PageWrapper>
  );
};
