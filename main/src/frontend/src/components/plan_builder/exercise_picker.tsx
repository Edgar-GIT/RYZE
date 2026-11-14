import { ArrowLeft, Check, Plus, Search, X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import {
  EXERCISE_BODY_PART_OPTIONS,
  EXERCISE_EQUIPMENT_OPTIONS,
  fetchExerciseDetail,
  fetchExerciseLibrary
} from "@/services/exercise_library";
import { joinClassNames } from "@utils/class_names";

import type { BuilderCopy } from "./copy";
import { capitalise, ExerciseAnimation, ExerciseThumb, fromCatalogEntry, type BuilderExercise } from "./exercise_media";
import styles from "./plan_builder.module.css";

const PAGE_SIZE = 24;

interface ExercisePickerProps {
  copy: BuilderCopy;
  dayLabel: string;
  // Exercises already in the day (a day cannot hold the same exercise twice).
  addedIds: string[];
  onAdd: (exercise: BuilderExercise) => void;
  onClose: () => void;
}

// ExercisePicker is the exercise browser of the plan builder, in the style of
// a training app: search and muscle filters on top, a grid of still pictures,
// a one-tap "+" to add, and a tap on the card to preview the movement (the
// animation only plays there).
export const ExercisePicker = ({ copy, dayLabel, addedIds, onAdd, onClose }: ExercisePickerProps) => {
  const [query, setQuery] = useState("");
  const [bodyPart, setBodyPart] = useState("");
  const [equipment, setEquipment] = useState("");
  const [items, setItems] = useState<BuilderExercise[]>([]);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [preview, setPreview] = useState<BuilderExercise | null>(null);
  const [addedNow, setAddedNow] = useState(0);
  const dialogRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  const load = useCallback(
    async (targetPage: number, replace: boolean) => {
      setLoading(true);
      setError(false);
      try {
        const result = await fetchExerciseLibrary({ query, bodyPart, equipment }, targetPage, PAGE_SIZE);
        const mapped = result.exercises.map(fromCatalogEntry);
        setItems((current) => (replace ? mapped : [...current, ...mapped]));
        setPage(result.pagination.page);
        setTotalPages(result.pagination.total_pages);
      } catch {
        setError(true);
      } finally {
        setLoading(false);
      }
    },
    [query, bodyPart, equipment]
  );

  useEffect(() => {
    const timer = window.setTimeout(() => void load(1, true), query ? 250 : 0);
    return () => window.clearTimeout(timer);
  }, [load, query]);

  useEffect(() => {
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    searchRef.current?.focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        if (preview) {
          setPreview(null);
        } else {
          onClose();
        }
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.body.style.overflow = previous;
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [onClose, preview]);

  // The preview loads the full catalogue entry (steps and animation).
  const openPreview = (exercise: BuilderExercise) => {
    setPreview(exercise);
    fetchExerciseDetail(exercise.id)
      .then((detail) => setPreview((current) => (current?.id === exercise.id ? fromCatalogEntry(detail.exercise) : current)))
      .catch(() => undefined);
  };

  const add = (exercise: BuilderExercise) => {
    if (addedIds.includes(exercise.id)) return;
    onAdd(exercise);
    setAddedNow((count) => count + 1);
  };

  return (
    <div className={styles.overlay}>
      <button type="button" className={styles.backdrop} aria-label={copy.close} tabIndex={-1} onClick={onClose} />
      <div ref={dialogRef} className={styles.picker} role="dialog" aria-modal="true" aria-label={copy.pickerTitle(dayLabel)}>
        <header className={styles.pickerHead}>
          {preview ? (
            <button type="button" className={styles.iconButton} onClick={() => setPreview(null)} aria-label={copy.back}>
              <ArrowLeft aria-hidden="true" />
            </button>
          ) : null}
          <h2>{preview ? preview.name : copy.pickerTitle(dayLabel)}</h2>
          <button type="button" className={styles.iconButton} onClick={onClose} aria-label={copy.close}>
            <X aria-hidden="true" />
          </button>
        </header>

        {preview ? (
          <div className={styles.previewBody}>
            <ExerciseAnimation exercise={preview} unavailable={copy.mediaUnavailable} />
            <div className={styles.previewInfo}>
              <dl className={styles.previewFacts}>
                {preview.primary_muscle_group || preview.body_part ? (
                  <div>
                    <dt>{copy.muscles}</dt>
                    <dd>{capitalise(preview.primary_muscle_group || preview.body_part)}</dd>
                  </div>
                ) : null}
                {preview.equipment ? (
                  <div>
                    <dt>{copy.equipment}</dt>
                    <dd>{capitalise(preview.equipment)}</dd>
                  </div>
                ) : null}
              </dl>
              {preview.description ? <p className={styles.previewText}>{preview.description}</p> : null}
              {preview.instruction_steps && preview.instruction_steps.length > 0 ? (
                <>
                  <h3 className={styles.previewSub}>{copy.howTo}</h3>
                  <ol className={styles.howToSteps}>
                    {preview.instruction_steps.map((step, index) => (
                      <li key={index}>{step}</li>
                    ))}
                  </ol>
                </>
              ) : preview.instructions ? (
                <>
                  <h3 className={styles.previewSub}>{copy.howTo}</h3>
                  <p className={styles.previewText}>{preview.instructions}</p>
                </>
              ) : null}
            </div>
            <div className={styles.previewActions}>
              {addedIds.includes(preview.id) ? (
                <span className={styles.addedPill}>
                  <Check aria-hidden="true" /> {copy.added}
                </span>
              ) : (
                <button
                  type="button"
                  className={styles.primaryButton}
                  onClick={() => {
                    add(preview);
                    setPreview(null);
                  }}
                >
                  <Plus aria-hidden="true" /> {copy.addToDay}
                </button>
              )}
            </div>
          </div>
        ) : (
          <>
            <div className={styles.pickerFilters}>
              <label className={styles.searchField}>
                <Search aria-hidden="true" />
                <span className={styles.srOnly}>{copy.search}</span>
                <input
                  ref={searchRef}
                  type="search"
                  value={query}
                  placeholder={copy.search}
                  onChange={(event) => setQuery(event.target.value)}
                />
              </label>
              <select
                className={styles.select}
                value={equipment}
                aria-label={copy.equipment}
                onChange={(event) => setEquipment(event.target.value)}
              >
                <option value="">{copy.anyEquipment}</option>
                {EXERCISE_EQUIPMENT_OPTIONS.map((option) => (
                  <option key={option} value={option}>
                    {option}
                  </option>
                ))}
              </select>
            </div>
            <div className={styles.chips} role="group" aria-label={copy.muscles}>
              <button type="button" aria-pressed={bodyPart === ""} onClick={() => setBodyPart("")}>
                {copy.allMuscles}
              </button>
              {EXERCISE_BODY_PART_OPTIONS.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  aria-pressed={bodyPart === option.value}
                  onClick={() => setBodyPart(option.value)}
                >
                  {copy.bodyParts[option.value] ?? option.label}
                </button>
              ))}
            </div>

            <div className={styles.pickerResults}>
              {error ? (
                <p className={styles.muted}>{copy.noResults}</p>
              ) : items.length === 0 && !loading ? (
                <p className={styles.muted}>{copy.noResults}</p>
              ) : (
                <ul className={styles.exerciseGrid}>
                  {items.map((exercise) => {
                    const isAdded = addedIds.includes(exercise.id);
                    return (
                      <li key={exercise.id} className={joinClassNames(styles.exerciseCard, isAdded && styles.exerciseCardAdded)}>
                        <button type="button" className={styles.exerciseCardOpen} onClick={() => openPreview(exercise)}>
                          <ExerciseThumb exercise={exercise} size="lg" />
                          <span className={styles.exerciseCardText}>
                            <strong>{exercise.name}</strong>
                            <span>{capitalise(exercise.primary_muscle_group || exercise.body_part || "")}</span>
                          </span>
                          <span className={styles.srOnly}>{copy.preview}</span>
                        </button>
                        <button
                          type="button"
                          className={styles.exerciseAdd}
                          aria-label={`${isAdded ? copy.added : copy.add}: ${exercise.name}`}
                          aria-pressed={isAdded}
                          disabled={isAdded}
                          onClick={() => add(exercise)}
                        >
                          {isAdded ? <Check aria-hidden="true" /> : <Plus aria-hidden="true" />}
                        </button>
                      </li>
                    );
                  })}
                </ul>
              )}
              {loading ? <p className={styles.muted}>{copy.loading}</p> : null}
              {!loading && page < totalPages ? (
                <button type="button" className={styles.ghostButton} onClick={() => void load(page + 1, false)}>
                  {copy.loadMore}
                </button>
              ) : null}
            </div>
          </>
        )}

        <footer className={styles.pickerFoot}>
          <button type="button" className={styles.primaryButton} onClick={onClose}>
            {copy.done(addedNow)}
          </button>
        </footer>
      </div>
    </div>
  );
};
