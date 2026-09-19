import { useEffect, useState } from "react";
import { Check, Loader2, Plus, Search, SlidersHorizontal, X } from "lucide-react";

import { Button } from "@/components/button/button";
import { Admin2Modal } from "@/components/admin2/admin2_modal/admin2_modal";
import { joinClassNames } from "@utils/class_names";

import {
  countActiveLibraryFilters,
  DIFFICULTY_LEVELS,
  EXERCISE_EQUIPMENT_OPTIONS,
  EXERCISE_MUSCLE_GROUPS,
  fetchExerciseDetail,
  fetchExerciseLibrary,
  MOVEMENT_CATEGORIES,
  type ExerciseAlternative,
  type ExerciseCatalogEntry
} from "@/services/exercise_library";
import { ExercisePreviewBody } from "./exercise_preview_body";

import styles from "./exercise_library_dialog.module.css";

interface ExerciseLibraryDialogProps {
  title: string;
  description?: string;
  addedExerciseIds: string[];
  onClose: () => void;
  onAdd: (exercise: ExerciseCatalogEntry) => void;
  actionLabel?: string;
  addedLabel?: string;
}

interface DetailBag {
  [exerciseId: string]: { alternatives: ExerciseAlternative[] };
}

// ExerciseLibraryDialog is the reusable two-pane exercise picker: a
// searchable, filterable catalogue list on the left and a live preview with
// the primary action on the right. It never mutates state itself; every
// selection is delegated to the onAdd callback.
export const ExerciseLibraryDialog = ({
  title,
  description,
  addedExerciseIds,
  onClose,
  onAdd,
  actionLabel = "Add to day",
  addedLabel = "Added"
}: ExerciseLibraryDialogProps) => {
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [muscle, setMuscle] = useState("");
  const [equipment, setEquipment] = useState("");
  const [difficulty, setDifficulty] = useState("");
  const [category, setCategory] = useState("");

  const [exercises, setExercises] = useState<ExerciseCatalogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [selectedId, setSelectedId] = useState<string>("");
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailBag, setDetailBag] = useState<DetailBag>({});

  useEffect(() => {
    const timer = window.setTimeout(() => setDebouncedQuery(query.trim()), 250);
    return () => window.clearTimeout(timer);
  }, [query]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    fetchExerciseLibrary({ query: debouncedQuery, muscle, equipment, difficulty, category }, 1, 100)
      .then((result) => {
        if (!cancelled) {
          setExercises(result.exercises);
          setTotal(result.pagination?.total ?? result.exercises.length);
        }
      })
      .catch((reason: unknown) => {
        if (!cancelled) {
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
  }, [debouncedQuery, muscle, equipment, difficulty, category]);

  useEffect(() => {
    if (exercises.length > 0 && !exercises.some((exercise) => exercise.id === selectedId)) {
      setSelectedId(exercises[0].id);
    }
  }, [exercises, selectedId]);

  useEffect(() => {
    if (!selectedId || detailBag[selectedId]) {
      return;
    }
    let cancelled = false;
    setDetailLoading(true);
    fetchExerciseDetail(selectedId)
      .then((result) => {
        if (!cancelled) {
          setDetailBag((current) => ({ ...current, [selectedId]: { alternatives: result.alternatives } }));
        }
      })
      .catch(() => {
        // The catalogue remains usable without alternatives loaded.
      })
      .finally(() => {
        if (!cancelled) {
          setDetailLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [selectedId, detailBag]);

  const selected = exercises.find((exercise) => exercise.id === selectedId) ?? null;
  const alternatives = selectedId ? (detailBag[selectedId]?.alternatives ?? []) : [];
  const activeFilterCount = countActiveLibraryFilters({ muscle, equipment, difficulty, category });

  const clearFilters = (): void => {
    setMuscle("");
    setEquipment("");
    setDifficulty("");
    setCategory("");
    setQuery("");
  };

  return (
    <Admin2Modal title={title} description={description} onClose={onClose} className={styles.libraryModal}>
      <div className={styles.toolbar}>
        <div className={styles.searchBox}>
          <Search size={15} className={styles.searchIcon} aria-hidden="true" />
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search exercises…"
            aria-label="Search exercises"
            className={styles.searchInput}
          />
          {query ? (
            <button type="button" onClick={() => setQuery("")} aria-label="Clear search" className={styles.searchClear}>
              <X size={14} />
            </button>
          ) : null}
        </div>

        <div className={styles.filters}>
          <label className={styles.filterItem}>
            <span className={styles.filterLabel}>Muscle</span>
            <select value={muscle} onChange={(event) => setMuscle(event.target.value)} className={styles.select} aria-label="Filter by muscle group">
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
            <select value={equipment} onChange={(event) => setEquipment(event.target.value)} className={styles.select} aria-label="Filter by equipment">
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
            <select value={difficulty} onChange={(event) => setDifficulty(event.target.value)} className={styles.select} aria-label="Filter by difficulty">
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
            <select value={category} onChange={(event) => setCategory(event.target.value)} className={styles.select} aria-label="Filter by movement category">
              <option value="">All categories</option>
              {MOVEMENT_CATEGORIES.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
          </label>

          {activeFilterCount > 0 ? (
            <button type="button" onClick={clearFilters} className={styles.clearFilters}>
              <SlidersHorizontal size={13} />
              Clear {activeFilterCount} filter{activeFilterCount === 1 ? "" : "s"}
            </button>
          ) : null}
        </div>
      </div>

      <div className={styles.layout}>
        <section className={styles.listColumn} aria-label="Exercise catalogue">
          <p className={styles.resultCount}>
            {loading ? "Loading catalogue…" : `${exercises.length} returned${total > exercises.length ? ` of ${total}` : ""}`}
          </p>

          {error ? <p className={styles.error}>{error}</p> : null}

          <div className={styles.list}>
            {loading ? (
              <p className={styles.empty}>
                <Loader2 size={18} className={styles.spinning} />
                Loading…
              </p>
            ) : null}

            {!loading && exercises.length === 0 ? (
              <p className={styles.empty}>No exercises match the current filters.</p>
            ) : null}

            {exercises.map((exercise) => {
              const active = exercise.id === selectedId;
              return (
                <button
                  type="button"
                  key={exercise.id}
                  onClick={() => setSelectedId(exercise.id)}
                  className={joinClassNames(styles.row, active && styles.rowActive)}
                >
                  <span className={styles.rowName}>{exercise.name}</span>
                  <span className={styles.rowMeta}>
                    {[exercise.primary_muscle_group, exercise.equipment, exercise.difficulty]
                      .map((value) => value?.trim())
                      .filter(Boolean)
                      .join(" · ")}
                  </span>
                </button>
              );
            })}
          </div>
        </section>

        <section className={styles.previewColumn} aria-label="Exercise preview">
          {selected ? (
            <>
              <ExercisePreviewBody
                exercise={selected}
                alternatives={alternatives}
                onSelectAlternative={(exerciseId) => setSelectedId(exerciseId)}
              />
              <div className={styles.footer}>
                <Button
                  variant="primary"
                  icon={addedExerciseIds.includes(selected.id) ? <Check size={15} /> : <Plus size={15} />}
                  iconPosition="left"
                  disabled={addedExerciseIds.includes(selected.id)}
                  onClick={() => onAdd(selected)}
                >
                  {addedExerciseIds.includes(selected.id) ? addedLabel : actionLabel}
                </Button>
                <Button variant="ghost" size="small" onClick={onClose}>
                  Cancel
                </Button>
              </div>
            </>
          ) : (
            <p className={styles.empty}>Select an exercise to preview it.</p>
          )}
          {detailLoading ? (
            <p className={styles.detailLoading}>
              <Loader2 size={14} className={styles.spinning} />
              Loading details…
            </p>
          ) : null}
        </section>
      </div>
    </Admin2Modal>
  );
};