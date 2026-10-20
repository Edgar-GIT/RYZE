import { Pencil, Plus, RefreshCw, Search } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Admin2Modal } from "@/components/admin2/admin2_modal/admin2_modal";
import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Pagination } from "@/components/admin2/admin2_pagination/admin2_pagination";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge, type Admin2BadgeTone } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Button } from "@/components/button/button";
import { formatAdminDate, type AdminPagination } from "@/services/admin_api";
import {
  EXERCISE_STATUS,
  EXERCISE_STATUS_LABELS,
  createAdminExercise,
  fetchAdminExercises,
  isExerciseStatus,
  updateAdminExercise,
  type AdminExercise,
  type AdminExerciseInput,
  type ExerciseStatusValue
} from "@/services/admin2_api";
import {
  DIFFICULTY_LEVELS,
  EXERCISE_BODY_PART_OPTIONS,
  MOVEMENT_CATEGORIES,
  exerciseBodyPartLabel
} from "@/services/exercise_library";
import { ApiError } from "@utils/http_client";

import styles from "./admin2_exercises_page.module.css";

const PAGE_SIZE = 25;
const SEARCH_DEBOUNCE_MS = 250;

type StatusFilter = ExerciseStatusValue | "";

interface EditorState {
  mode: "create" | "edit";
  exercise: AdminExercise | null;
}

interface EditorForm {
  name: string;
  description: string;
  instructions: string;
  steps: string;
  target_muscles: string;
  primary_muscle_group: string;
  secondary_muscle_groups: string;
  equipment: string;
  difficulty: string;
  movement_category: string;
  body_part: string;
  image_url: string;
  animation_url: string;
  source_media_id: string;
  media_attribution: string;
  status: ExerciseStatusValue;
}

const EMPTY_FORM: EditorForm = {
  name: "",
  description: "",
  instructions: "",
  steps: "",
  target_muscles: "",
  primary_muscle_group: "",
  secondary_muscle_groups: "",
  equipment: "",
  difficulty: "",
  movement_category: "",
  body_part: "",
  image_url: "",
  animation_url: "",
  source_media_id: "",
  media_attribution: "",
  status: EXERCISE_STATUS.DRAFT
};

const formFromExercise = (exercise: AdminExercise): EditorForm => ({
  name: exercise.name,
  description: exercise.description,
  instructions: exercise.instructions,
  steps: exercise.instruction_steps.join("\n"),
  target_muscles: exercise.target_muscles,
  primary_muscle_group: exercise.primary_muscle_group,
  secondary_muscle_groups: exercise.secondary_muscle_groups,
  equipment: exercise.equipment,
  difficulty: exercise.difficulty,
  movement_category: exercise.movement_category,
  body_part: exercise.body_part,
  image_url: exercise.image_url,
  animation_url: exercise.animation_url,
  source_media_id: exercise.source_media_id,
  media_attribution: exercise.media_attribution,
  status: isExerciseStatus(exercise.status) ? exercise.status : EXERCISE_STATUS.DRAFT
});

const toInput = (form: EditorForm): AdminExerciseInput => ({
  description: form.description.trim(),
  instructions: form.instructions.trim(),
  instruction_steps: form.steps
    .split("\n")
    .map((step) => step.trim())
    .filter(Boolean),
  target_muscles: form.target_muscles.trim(),
  primary_muscle_group: form.primary_muscle_group.trim(),
  secondary_muscle_groups: form.secondary_muscle_groups.trim(),
  equipment: form.equipment.trim(),
  difficulty: form.difficulty,
  movement_category: form.movement_category,
  body_part: form.body_part,
  image_url: form.image_url.trim(),
  animation_url: form.animation_url.trim(),
  source_media_id: form.source_media_id.trim(),
  media_attribution: form.media_attribution.trim()
});

const editorErrorMessage = (reason: unknown): string => {
  if (reason instanceof ApiError) {
    switch (reason.code) {
      case "EXERCISE_NAME_TAKEN":
        return "An exercise with this name already exists.";
      case "EXERCISE_NOT_FOUND":
        return "This exercise no longer exists. Reload the catalogue.";
      case "VALIDATION_ERROR":
        return "Some fields are invalid. Check the classification values and media paths.";
      default:
        return reason.message || "Unable to save the exercise.";
    }
  }
  return "Unable to save the exercise.";
};

const statusTone = (status: string): Admin2BadgeTone => {
  if (status === EXERCISE_STATUS.PUBLISHED) {
    return "success";
  }
  if (status === EXERCISE_STATUS.DRAFT) {
    return "warning";
  }
  return "neutral";
};

const statusLabel = (status: string): string =>
  isExerciseStatus(status) ? EXERCISE_STATUS_LABELS[status] : status;

interface TextFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  hint?: string;
  disabled?: boolean;
  maxLength?: number;
}

const TextField = ({ label, value, onChange, hint, disabled = false, maxLength }: TextFieldProps) => (
  <label className={styles.field}>
    <span className={styles.fieldLabel}>{label}</span>
    <input
      className={styles.input}
      aria-label={label}
      value={value}
      disabled={disabled}
      maxLength={maxLength}
      onChange={(event) => onChange(event.target.value)}
    />
    {hint ? <span className={styles.fieldHint}>{hint}</span> : null}
  </label>
);

interface TextAreaFieldProps extends TextFieldProps {
  rows?: number;
}

const TextAreaField = ({ label, value, onChange, hint, rows = 3, maxLength }: TextAreaFieldProps) => (
  <label className={styles.field}>
    <span className={styles.fieldLabel}>{label}</span>
    <textarea
      className={styles.textarea}
      aria-label={label}
      rows={rows}
      value={value}
      maxLength={maxLength}
      onChange={(event) => onChange(event.target.value)}
    />
    {hint ? <span className={styles.fieldHint}>{hint}</span> : null}
  </label>
);

interface SelectFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: ReadonlyArray<{ value: string; label: string }>;
}

const SelectField = ({ label, value, onChange, options }: SelectFieldProps) => (
  <label className={styles.field}>
    <span className={styles.fieldLabel}>{label}</span>
    <select
      className={styles.select}
      aria-label={label}
      value={value}
      onChange={(event) => onChange(event.target.value)}
    >
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  </label>
);

const Editor = ({
  state,
  saving,
  error,
  onClose,
  onSubmit
}: {
  state: EditorState;
  saving: boolean;
  error: string;
  onClose: () => void;
  onSubmit: (form: EditorForm) => void;
}) => {
  const [form, setForm] = useState<EditorForm>(() =>
    state.exercise ? formFromExercise(state.exercise) : EMPTY_FORM
  );

  const update = <K extends keyof EditorForm>(key: K, value: EditorForm[K]) =>
    setForm((current) => ({ ...current, [key]: value }));

  const editing = state.mode === "edit";
  const canSubmit = editing || form.name.trim().length > 0;

  return (
    <Admin2Modal
      title={editing ? "Edit exercise" : "New exercise"}
      description={
        editing
          ? "Update the catalogue metadata of this exercise. Published changes are visible immediately."
          : "Create a catalogue entry. New entries start as drafts and are published explicitly."
      }
      onClose={onClose}
    >
      <form
        className={styles.editorForm}
        onSubmit={(event) => {
          event.preventDefault();
          if (canSubmit && !saving) {
            onSubmit(form);
          }
        }}
      >
        <div className={styles.formGrid}>
          <TextField
            label="Name"
            value={form.name}
            onChange={(value) => update("name", value)}
            disabled={editing}
            maxLength={255}
            hint={editing ? "Catalogue names are immutable." : undefined}
          />
          {editing ? (
            <SelectField
              label="Status"
              value={form.status}
              onChange={(value) => update("status", value as ExerciseStatusValue)}
              options={Object.values(EXERCISE_STATUS).map((status) => ({
                value: status,
                label: EXERCISE_STATUS_LABELS[status]
              }))}
            />
          ) : null}
          <SelectField
            label="Body part"
            value={form.body_part}
            onChange={(value) => update("body_part", value)}
            options={[
              { value: "", label: "Unclassified" },
              ...EXERCISE_BODY_PART_OPTIONS.map((option) => ({
                value: option.value,
                label: option.label
              }))
            ]}
          />
          <SelectField
            label="Difficulty"
            value={form.difficulty}
            onChange={(value) => update("difficulty", value)}
            options={[
              { value: "", label: "Unspecified" },
              ...DIFFICULTY_LEVELS.map((level) => ({ value: level, label: level }))
            ]}
          />
          <SelectField
            label="Movement category"
            value={form.movement_category}
            onChange={(value) => update("movement_category", value)}
            options={[
              { value: "", label: "Unspecified" },
              ...MOVEMENT_CATEGORIES.map((category) => ({ value: category, label: category }))
            ]}
          />
          <TextField
            label="Equipment"
            value={form.equipment}
            onChange={(value) => update("equipment", value)}
            maxLength={255}
          />
          <TextField
            label="Target muscles"
            value={form.target_muscles}
            onChange={(value) => update("target_muscles", value)}
            maxLength={255}
          />
          <TextField
            label="Primary muscle group"
            value={form.primary_muscle_group}
            onChange={(value) => update("primary_muscle_group", value)}
            maxLength={80}
          />
          <TextField
            label="Secondary muscle groups"
            value={form.secondary_muscle_groups}
            onChange={(value) => update("secondary_muscle_groups", value)}
            maxLength={255}
          />
        </div>

        <h3 className={styles.formSectionTitle}>Content</h3>
        <TextAreaField
          label="Description"
          value={form.description}
          onChange={(value) => update("description", value)}
          maxLength={2000}
        />
        <TextAreaField
          label="Instructions"
          value={form.instructions}
          onChange={(value) => update("instructions", value)}
          rows={4}
          maxLength={5000}
        />
        <TextAreaField
          label="Instruction steps"
          value={form.steps}
          onChange={(value) => update("steps", value)}
          rows={4}
          hint="One step per line, in execution order."
        />

        <h3 className={styles.formSectionTitle}>Media references</h3>
        <div className={styles.formGrid}>
          <TextField
            label="Image URL"
            value={form.image_url}
            onChange={(value) => update("image_url", value)}
            hint="Relative media path, for example exercises/squat.jpg."
            maxLength={500}
          />
          <TextField
            label="Animation URL"
            value={form.animation_url}
            onChange={(value) => update("animation_url", value)}
            hint="Relative media path, for example exercises/squat.gif."
            maxLength={500}
          />
          <TextField
            label="Source media id"
            value={form.source_media_id}
            onChange={(value) => update("source_media_id", value)}
            maxLength={40}
          />
          <TextField
            label="Media attribution"
            value={form.media_attribution}
            onChange={(value) => update("media_attribution", value)}
            maxLength={255}
          />
        </div>

        {error ? (
          <p className={styles.formError} role="alert">
            {error}
          </p>
        ) : null}

        <div className={styles.modalActions}>
          <Button type="button" variant="ghost" size="small" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="small" disabled={!canSubmit || saving}>
            {saving ? "Saving…" : editing ? "Save changes" : "Create exercise"}
          </Button>
        </div>
      </form>
    </Admin2Modal>
  );
};

export default function Admin2ExercisesPage() {
  const [exercises, setExercises] = useState<AdminExercise[]>([]);
  const [pagination, setPagination] = useState<AdminPagination | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [page, setPage] = useState(1);
  const [searchInput, setSearchInput] = useState("");
  const [appliedQuery, setAppliedQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("");
  const [bodyPartFilter, setBodyPartFilter] = useState("");
  const [reloadToken, setReloadToken] = useState(0);

  const [editor, setEditor] = useState<EditorState | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setAppliedQuery(searchInput.trim());
      setPage(1);
    }, SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");

    fetchAdminExercises(page, PAGE_SIZE, {
      query: appliedQuery,
      status: statusFilter,
      bodyPart: bodyPartFilter
    })
      .then((result) => {
        if (cancelled) {
          return;
        }
        setExercises(result.exercises);
        setPagination(result.pagination);
      })
      .catch(() => {
        if (cancelled) {
          return;
        }
        setError("Unable to load the exercise catalogue.");
        setExercises([]);
        setPagination(null);
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [page, appliedQuery, statusFilter, bodyPartFilter, reloadToken]);

  const openCreate = useCallback(() => {
    setSaveError("");
    setEditor({ mode: "create", exercise: null });
  }, []);

  const openEdit = useCallback((exercise: AdminExercise) => {
    setSaveError("");
    setEditor({ mode: "edit", exercise });
  }, []);

  const closeEditor = useCallback(() => {
    if (!saving) {
      setEditor(null);
      setSaveError("");
    }
  }, [saving]);

  const save = async (form: EditorForm) => {
    if (!editor) {
      return;
    }
    setSaving(true);
    setSaveError("");
    try {
      if (editor.mode === "create") {
        await createAdminExercise({ ...toInput(form), name: form.name.trim() });
      } else if (editor.exercise) {
        await updateAdminExercise(editor.exercise.id, { ...toInput(form), status: form.status });
      }
      setEditor(null);
      setReloadToken((token) => token + 1);
    } catch (reason) {
      setSaveError(editorErrorMessage(reason));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className={styles.page}>
      <Admin2PageHeader
        eyebrow="Catalogue"
        title="Exercises"
        description="Curate the canonical exercise catalogue used by the public library, trainers and programs."
        actions={
          <>
            <Button
              variant="secondary"
              size="small"
              icon={<RefreshCw size={15} />}
              iconPosition="left"
              onClick={() => setReloadToken((token) => token + 1)}
            >
              Refresh
            </Button>
            <Button variant="primary" size="small" icon={<Plus size={15} />} iconPosition="left" onClick={openCreate}>
              New exercise
            </Button>
          </>
        }
      />

      <Admin2Section
        title="Catalogue entries"
        subtitle="Draft and archived entries are visible here only; the public library lists published entries."
      >
        <div className={styles.toolbar}>
          <div className={styles.searchField}>
            <Search size={16} aria-hidden="true" className={styles.searchIcon} />
            <input
              className={styles.searchInput}
              type="search"
              aria-label="Search exercises"
              placeholder="Search by exercise or muscle"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
            />
          </div>
          <select
            className={styles.select}
            aria-label="Filter by status"
            value={statusFilter}
            onChange={(event) => {
              setStatusFilter(event.target.value as StatusFilter);
              setPage(1);
            }}
          >
            <option value="">All statuses</option>
            {Object.values(EXERCISE_STATUS).map((status) => (
              <option key={status} value={status}>
                {EXERCISE_STATUS_LABELS[status]}
              </option>
            ))}
          </select>
          <select
            className={styles.select}
            aria-label="Filter by body part"
            value={bodyPartFilter}
            onChange={(event) => {
              setBodyPartFilter(event.target.value);
              setPage(1);
            }}
          >
            <option value="">All body parts</option>
            {EXERCISE_BODY_PART_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>

        {error ? (
          <p className={styles.errorText} role="alert">
            {error}
          </p>
        ) : null}

        {loading ? (
          <p className={styles.stateText}>Loading catalogue…</p>
        ) : exercises.length === 0 && !error ? (
          <p className={styles.stateText}>No exercises match the current filters.</p>
        ) : (
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Exercise</th>
                  <th scope="col">Classification</th>
                  <th scope="col">Equipment</th>
                  <th scope="col">Status</th>
                  <th scope="col">Updated</th>
                  <th scope="col" aria-label="Actions" />
                </tr>
              </thead>
              <tbody>
                {exercises.map((exercise) => (
                  <tr key={exercise.id}>
                    <td>
                      <span className={styles.exerciseName}>{exercise.name}</span>
                      <span className={styles.exerciseMeta}>
                        {exercise.source}
                        {exercise.source_id ? ` · ${exercise.source_id}` : ""}
                      </span>
                    </td>
                    <td>
                      {exerciseBodyPartLabel(exercise.body_part) || "—"}
                      {exercise.difficulty ? ` · ${exercise.difficulty}` : ""}
                    </td>
                    <td>{exercise.equipment || "—"}</td>
                    <td>
                      <Admin2StatusBadge label={statusLabel(exercise.status)} tone={statusTone(exercise.status)} />
                    </td>
                    <td className={styles.muted}>{formatAdminDate(exercise.updated_at)}</td>
                    <td>
                      <button
                        type="button"
                        className={styles.rowAction}
                        aria-label={`Edit ${exercise.name}`}
                        title="Edit"
                        onClick={() => openEdit(exercise)}
                      >
                        <Pencil size={15} aria-hidden="true" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {pagination ? (
          <Admin2Pagination
            page={pagination.page}
            totalPages={pagination.total_pages}
            pageSize={pagination.limit}
            totalItems={pagination.total}
            onChange={setPage}
          />
        ) : null}
      </Admin2Section>

      {editor ? (
        <Editor state={editor} saving={saving} error={saveError} onClose={closeEditor} onSubmit={(form) => void save(form)} />
      ) : null}
    </div>
  );
}
