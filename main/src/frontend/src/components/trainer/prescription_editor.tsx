import { useState } from "react";
import { ArrowDown, ArrowUp, Plus, Trash2, X } from "lucide-react";

import { Button } from "@/components/button/button";
import {
  WorkoutSetType,
  type TrainerPrescriptionInput,
  type TrainerWorkoutExercise,
  type TrainerWorkoutSet,
  type TrainerWorkoutSetInput,
  type WorkoutSetTypeValue
} from "@/services/trainer_api";

import styles from "./prescription_editor.module.css";

const MAX_SETS = 50;
const MAX_TEMPO_LENGTH = 20;

type DraftSet = {
  set_type: WorkoutSetTypeValue;
  reps: string;
  weight_kg: string;
  rir: string;
  rpe: string;
  rest_seconds: string;
  tempo: string;
};

const SET_TYPE_OPTIONS: Array<{ value: WorkoutSetTypeValue; label: string }> = [
  { value: WorkoutSetType.WARMUP, label: "Warm-up" },
  { value: WorkoutSetType.WORKING, label: "Working" },
  { value: WorkoutSetType.DROP, label: "Drop" },
  { value: WorkoutSetType.BACKOFF, label: "Back-off" },
  { value: WorkoutSetType.FAILURE, label: "Failure" }
];

const toDraft = (set: TrainerWorkoutSet): DraftSet => ({
  set_type: set.set_type,
  reps: set.reps === null ? "" : String(set.reps),
  weight_kg: set.weight_kg === null ? "" : String(set.weight_kg),
  rir: set.rir === null ? "" : String(set.rir),
  rpe: set.rpe === null ? "" : String(set.rpe),
  rest_seconds: set.rest_seconds === null ? "" : String(set.rest_seconds),
  tempo: set.tempo
});

const emptySet = (): DraftSet => ({
  set_type: WorkoutSetType.WORKING,
  reps: "",
  weight_kg: "",
  rir: "",
  rpe: "",
  rest_seconds: "",
  tempo: ""
});

const moveSet = (sets: DraftSet[], from: number, to: number): DraftSet[] => {
  const next = [...sets];
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return next;
};

const parseInteger = (value: string, label: string, minimum: number): number | null | string => {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }
  if (!/^\d+$/.test(trimmed)) {
    return `${label} must be a whole number.`;
  }
  const parsed = Number(trimmed);
  if (parsed < minimum) {
    return `${label} must be at least ${minimum}.`;
  }
  return parsed;
};

const parseDecimal = (value: string, label: string, minimum: number, maximum?: number): number | null | string => {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }
  if (!/^\d+(?:[.,]\d+)?$/.test(trimmed)) {
    return `${label} must be a number.`;
  }
  const parsed = Number(trimmed.replace(",", "."));
  if (parsed < minimum || (maximum !== undefined && parsed > maximum)) {
    return maximum === undefined
      ? `${label} must be at least ${minimum}.`
      : `${label} must be between ${minimum} and ${maximum}.`;
  }
  return parsed;
};

const buildInput = (
  instructions: string,
  notes: string,
  sets: DraftSet[]
): { input: TrainerPrescriptionInput } | { error: string } => {
  if (sets.length > MAX_SETS) {
    return { error: `An exercise can have at most ${MAX_SETS} sets.` };
  }

  const parsedSets: TrainerWorkoutSetInput[] = [];
  for (let index = 0; index < sets.length; index += 1) {
    const draft = sets[index];
    const label = `Set ${index + 1}`;
    const reps = parseInteger(draft.reps, `${label} reps`, 1);
    if (typeof reps === "string") {
      return { error: reps };
    }
    const weight = parseDecimal(draft.weight_kg, `${label} weight`, 0);
    if (typeof weight === "string") {
      return { error: weight };
    }
    const rir = parseInteger(draft.rir, `${label} RIR`, 0);
    if (typeof rir === "string") {
      return { error: rir };
    }
    const rpe = parseDecimal(draft.rpe, `${label} RPE`, 1, 10);
    if (typeof rpe === "string") {
      return { error: rpe };
    }
    const rest = parseInteger(draft.rest_seconds, `${label} rest`, 0);
    if (typeof rest === "string") {
      return { error: rest };
    }
    if (draft.tempo.trim().length > MAX_TEMPO_LENGTH) {
      return { error: `${label} tempo cannot exceed ${MAX_TEMPO_LENGTH} characters.` };
    }
    parsedSets.push({
      set_type: draft.set_type,
      reps,
      weight_kg: weight,
      rir,
      rpe,
      rest_seconds: rest,
      tempo: draft.tempo.trim()
    });
  }

  return {
    input: {
      instructions: instructions.trim(),
      notes: notes.trim(),
      sets: parsedSets
    }
  };
};

interface PrescriptionEditorProps {
  entry: TrainerWorkoutExercise;
  busy: boolean;
  onSave: (input: TrainerPrescriptionInput) => void;
  onCancel: () => void;
}

export const PrescriptionEditor = ({ entry, busy, onSave, onCancel }: PrescriptionEditorProps) => {
  const [instructions, setInstructions] = useState(entry.instructions);
  const [notes, setNotes] = useState(entry.notes);
  const [sets, setSets] = useState<DraftSet[]>(() => entry.sets.map(toDraft));
  const [error, setError] = useState("");

  const updateSet = (index: number, field: keyof DraftSet, value: string) => {
    setSets((current) =>
      current.map((set, position) =>
        position === index
          ? { ...set, [field]: field === "set_type" ? (value as WorkoutSetTypeValue) : value }
          : set
      )
    );
  };

  const submit = () => {
    const result = buildInput(instructions, notes, sets);
    if ("error" in result) {
      setError(result.error);
      return;
    }
    setError("");
    onSave(result.input);
  };

  return (
    <section className={styles.editor} aria-label={`Prescription for ${entry.exercise.name}`}>
      <header className={styles.header}>
        <div>
          <h3 className={styles.title}>{entry.exercise.name}</h3>
          <p className={styles.subtitle}>Prescribe the sets your client will follow.</p>
        </div>
        <button
          type="button"
          className={styles.closeButton}
          aria-label={`Close prescription for ${entry.exercise.name}`}
          onClick={onCancel}
        >
          <X size={16} />
        </button>
      </header>

      <div className={styles.textFields}>
        <label className={styles.field}>
          <span className={styles.label}>Instructions</span>
          <textarea
            className={styles.textarea}
            value={instructions}
            placeholder="How to perform this exercise."
            onChange={(event) => setInstructions(event.target.value)}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Notes</span>
          <textarea
            className={styles.textarea}
            value={notes}
            placeholder="Cues, substitutions or reminders."
            onChange={(event) => setNotes(event.target.value)}
          />
        </label>
      </div>

      <div className={styles.setsHeader}>
        <span className={styles.label}>Sets</span>
        <span className={styles.hint}>{sets.length} / {MAX_SETS}</span>
      </div>

      {sets.length === 0 ? (
        <p className={styles.empty}>No sets yet. Add the first set to build the prescription.</p>
      ) : (
        <div className={styles.setList}>
          {sets.map((set, index) => (
            <div key={index} className={styles.setRow}>
              <span className={styles.setNumber}>{index + 1}</span>
              <label className={styles.setField}>
                <span className={styles.srOnly}>Set {index + 1} type</span>
                <select
                  className={styles.select}
                  value={set.set_type}
                  aria-label={`Set ${index + 1} type`}
                  onChange={(event) => updateSet(index, "set_type", event.target.value)}
                >
                  {SET_TYPE_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </label>
              <label className={styles.setField}>
                <span className={styles.srOnly}>Set {index + 1} reps</span>
                <input
                  className={styles.input}
                  inputMode="numeric"
                  placeholder="Reps"
                  value={set.reps}
                  aria-label={`Set ${index + 1} reps`}
                  onChange={(event) => updateSet(index, "reps", event.target.value)}
                />
              </label>
              <label className={styles.setField}>
                <span className={styles.srOnly}>Set {index + 1} weight in kilograms</span>
                <input
                  className={styles.input}
                  inputMode="decimal"
                  placeholder="kg"
                  value={set.weight_kg}
                  aria-label={`Set ${index + 1} weight in kilograms`}
                  onChange={(event) => updateSet(index, "weight_kg", event.target.value)}
                />
              </label>
              <label className={styles.setField}>
                <span className={styles.srOnly}>Set {index + 1} RIR</span>
                <input
                  className={styles.input}
                  inputMode="numeric"
                  placeholder="RIR"
                  value={set.rir}
                  aria-label={`Set ${index + 1} RIR`}
                  onChange={(event) => updateSet(index, "rir", event.target.value)}
                />
              </label>
              <label className={styles.setField}>
                <span className={styles.srOnly}>Set {index + 1} RPE</span>
                <input
                  className={styles.input}
                  inputMode="decimal"
                  placeholder="RPE"
                  value={set.rpe}
                  aria-label={`Set ${index + 1} RPE`}
                  onChange={(event) => updateSet(index, "rpe", event.target.value)}
                />
              </label>
              <label className={styles.setField}>
                <span className={styles.srOnly}>Set {index + 1} rest in seconds</span>
                <input
                  className={styles.input}
                  inputMode="numeric"
                  placeholder="Rest (s)"
                  value={set.rest_seconds}
                  aria-label={`Set ${index + 1} rest in seconds`}
                  onChange={(event) => updateSet(index, "rest_seconds", event.target.value)}
                />
              </label>
              <label className={styles.setField}>
                <span className={styles.srOnly}>Set {index + 1} tempo</span>
                <input
                  className={styles.input}
                  placeholder="Tempo"
                  maxLength={MAX_TEMPO_LENGTH}
                  value={set.tempo}
                  aria-label={`Set ${index + 1} tempo`}
                  onChange={(event) => updateSet(index, "tempo", event.target.value)}
                />
              </label>
              <div className={styles.setActions}>
                <button
                  type="button"
                  className={styles.iconButton}
                  aria-label={`Move set ${index + 1} up`}
                  disabled={index === 0 || busy}
                  onClick={() => setSets((current) => moveSet(current, index, index - 1))}
                >
                  <ArrowUp size={14} />
                </button>
                <button
                  type="button"
                  className={styles.iconButton}
                  aria-label={`Move set ${index + 1} down`}
                  disabled={index === sets.length - 1 || busy}
                  onClick={() => setSets((current) => moveSet(current, index, index + 1))}
                >
                  <ArrowDown size={14} />
                </button>
                <button
                  type="button"
                  className={styles.iconButton}
                  aria-label={`Remove set ${index + 1}`}
                  disabled={busy}
                  onClick={() => setSets((current) => current.filter((_, position) => position !== index))}
                >
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {error ? <p className={styles.error} role="alert">{error}</p> : null}

      <div className={styles.actions}>
        <Button
          variant="secondary"
          size="small"
          icon={<Plus size={15} />}
          iconPosition="left"
          disabled={sets.length >= MAX_SETS || busy}
          onClick={() => setSets((current) => [...current, emptySet()])}
        >
          Add set
        </Button>
        <div className={styles.actionsEnd}>
          <Button variant="ghost" size="small" disabled={busy} onClick={onCancel}>
            Cancel
          </Button>
          <Button variant="primary" size="small" disabled={busy} onClick={submit}>
            {busy ? "Saving…" : "Save prescription"}
          </Button>
        </div>
      </div>
    </section>
  );
};
