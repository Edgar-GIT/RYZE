import { Search, SlidersHorizontal, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
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
  exerciseMetaLine,
  exerciseThumbnailView,
  fetchExerciseLibrary,
  MOVEMENT_CATEGORIES,
  type ExerciseCatalogEntry
} from "@/services/exercise_library";
import { joinClassNames } from "@utils/class_names";

import styles from "./exercise_library_page.module.css";

const PAGE_LIMIT = 10;
const SEARCH_DEBOUNCE_MS = 250;
const LOAD_MORE_ROOT_MARGIN = "120px 0px";

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

// ExerciseMediaThumb renders only the static catalogue image. Animated GIFs
// and videos are reserved for the detail page, so the library never plays
// media while browsing. The image keeps its intrinsic size instead of being
// stretched into the card.
const ExerciseMediaThumb = ({ exercise }: { exercise: ExerciseCatalogEntry }) => {
  const media = exerciseThumbnailView(exercise);
  const [failedURL, setFailedURL] = useState("");

  if (media.kind === "none" || (media.url !== "" && media.url === failedURL)) {
    return (
      <div className={styles.thumbPlaceholder} aria-hidden="true">
        <span className={styles.thumbPlaceholderText}>Media unavailable</span>
      </div>
    );
  }

  return (
    <div className={styles.thumb}>
      <img
        src={media.url}
        alt={`${exercise.name} demonstration`}
        className={styles.thumbMedia}
        loading="lazy"
        onError={() => setFailedURL(media.url)}
      />
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
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState("");
  const [loadMoreError, setLoadMoreError] = useState("");
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const timer = window.setTimeout(() => setDebouncedQuery(query.trim()), SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [query]);

  useEffect(() => {
    let cancelled = false;
    const firstPage = page === 1;

    if (firstPage) {
      setLoading(true);
    } else {
      setLoadingMore(true);
    }
    setError("");
    setLoadMoreError("");

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
        setExercises((current) => {
          if (firstPage) {
            return result.exercises;
          }
          const seen = new Set(current.map((exercise) => exercise.id));
          return [...current, ...result.exercises.filter((exercise) => !seen.has(exercise.id))];
        });
        const totalCount = result.pagination?.total ?? result.exercises.length;
        const totalPages = Math.max(1, result.pagination?.total_pages ?? 1);
        setTotal(totalCount);
        setHasMore(page < totalPages);
      })
      .catch((reason: unknown) => {
        if (cancelled) {
          return;
        }
        const message = reason instanceof Error ? reason.message : "Could not load the exercise catalogue.";
        if (firstPage) {
          setExercises([]);
          setTotal(0);
          setHasMore(false);
          setError(message);
        } else {
          setLoadMoreError(message);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
          setLoadingMore(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [debouncedQuery, filters, page, reloadToken]);

  useEffect(() => {
    const node = sentinelRef.current;
    if (
      !node ||
      loading ||
      loadingMore ||
      error ||
      loadMoreError ||
      !hasMore ||
      typeof IntersectionObserver === "undefined"
    ) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          observer.disconnect();
          setPage((current) => current + 1);
        }
      },
      { rootMargin: LOAD_MORE_ROOT_MARGIN }
    );
    observer.observe(node);

    return () => observer.disconnect();
  }, [loading, loadingMore, error, loadMoreError, hasMore, exercises.length]);

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

  const firstResult = exercises.length === 0 ? 0 : 1;
  const lastResult = exercises.length;

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

          {!loading && !error && exercises.length > 0 ? (
            <div ref={sentinelRef} className={styles.loadMore} aria-live="polite">
              {loadingMore ? (
                <span>Loading more exercises…</span>
              ) : loadMoreError ? (
                <>
                  <span>{loadMoreError}</span>
                  <Button
                    variant="secondary"
                    size="small"
                    onClick={() => setReloadToken((current) => current + 1)}
                  >
                    Retry
                  </Button>
                </>
              ) : hasMore ? (
                <span>Scroll to load more exercises.</span>
              ) : (
                <span>You have reached the end of the library.</span>
              )}
            </div>
          ) : null}
        </Container>
      </section>
    </PageWrapper>
  );
};
