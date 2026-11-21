import { Plus, RefreshCw, Trash2 } from "lucide-react";
import { type FormEvent, useCallback, useEffect, useMemo, useState } from "react";

import { LineChart, type ChartPoint } from "@/components/training/charts";
import { useI18n } from "@/i18n/i18n";
import {
  deleteMeasurement,
  fetchMeasurements,
  MEASUREMENT_KINDS,
  measurementUnit,
  saveMeasurement,
  type BodyMeasurement,
  type MeasurementKind
} from "@/services/body_stats_api";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";

import styles from "./body_stats.module.css";

const KIND_LABELS: Record<"en" | "pt", Record<MeasurementKind, string>> = {
  en: {
    weight: "Weight",
    body_fat: "Body fat",
    neck: "Neck",
    shoulders: "Shoulders",
    chest: "Chest",
    biceps_left: "Left biceps",
    biceps_right: "Right biceps",
    forearm_left: "Left forearm",
    forearm_right: "Right forearm",
    waist: "Waist",
    hips: "Hips",
    thigh_left: "Left thigh",
    thigh_right: "Right thigh",
    calf_left: "Left calf",
    calf_right: "Right calf"
  },
  pt: {
    weight: "Peso",
    body_fat: "Gordura corporal",
    neck: "Pescoço",
    shoulders: "Ombros",
    chest: "Peito",
    biceps_left: "Bíceps esquerdo",
    biceps_right: "Bíceps direito",
    forearm_left: "Antebraço esquerdo",
    forearm_right: "Antebraço direito",
    waist: "Cintura",
    hips: "Anca",
    thigh_left: "Coxa esquerda",
    thigh_right: "Coxa direita",
    calf_left: "Gémeo esquerdo",
    calf_right: "Gémeo direito"
  }
};

const COPY = {
  en: {
    title: "Body measurements",
    lead: "Log your weight, body fat and measurements to see how your body changes. Logging the same measurement twice on one day keeps the latest value.",
    add: "Log a measurement",
    kind: "Measurement",
    body: "Body",
    circumferences: "Circumferences",
    value: "Value",
    date: "Date",
    save: "Save",
    saving: "Saving…",
    saved: "Saved.",
    latest: "Latest values",
    evolution: "Evolution",
    history: "History",
    needTwo: "Log at least two values on different days to see the evolution.",
    none: "No measurements yet. Start with your weight.",
    remove: "Delete",
    confirmRemove: "Delete this measurement?",
    errValue: "Enter a realistic value for this measurement.",
    errDate: "Choose a date that is not in the future.",
    errSave: "We could not save the measurement. Please try again.",
    errDelete: "We could not delete the measurement.",
    loadError: "We could not load your measurements.",
    retry: "Try again",
    loading: "Loading…"
  },
  pt: {
    title: "Medições corporais",
    lead: "Regista o teu peso, a gordura corporal e as medidas para veres como o teu corpo muda. Registar a mesma medição duas vezes no mesmo dia guarda o valor mais recente.",
    add: "Registar uma medição",
    kind: "Medição",
    body: "Corpo",
    circumferences: "Perímetros",
    value: "Valor",
    date: "Data",
    save: "Guardar",
    saving: "A guardar…",
    saved: "Guardado.",
    latest: "Últimos valores",
    evolution: "Evolução",
    history: "Histórico",
    needTwo: "Regista pelo menos dois valores em dias diferentes para veres a evolução.",
    none: "Ainda não tens medições. Começa pelo teu peso.",
    remove: "Apagar",
    confirmRemove: "Apagar esta medição?",
    errValue: "Indica um valor realista para esta medição.",
    errDate: "Escolhe uma data que não seja no futuro.",
    errSave: "Não foi possível guardar a medição. Tenta de novo.",
    errDelete: "Não foi possível apagar a medição.",
    loadError: "Não foi possível carregar as tuas medições.",
    retry: "Tentar de novo",
    loading: "A carregar…"
  }
} as const;

// today is the member's local date as YYYY-MM-DD.
const today = (): string => {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
};

// MeasurementsPanel logs body measurements and charts each kind over time
// (owner request, 2026-11-20, as in Lyfta).
export const MeasurementsPanel = () => {
  const { locale, formatNumber, formatDate } = useI18n();
  const text = COPY[locale];
  const labels = KIND_LABELS[locale];
  const [entries, setEntries] = useState<BodyMeasurement[] | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [kind, setKind] = useState<MeasurementKind>("weight");
  const [value, setValue] = useState("");
  const [date, setDate] = useState(today);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);
  const [selected, setSelected] = useState<MeasurementKind>("weight");

  const load = useCallback(async () => {
    setLoadError(false);
    try {
      setEntries(await fetchMeasurements());
    } catch {
      setLoadError(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const byKind = useMemo(() => {
    const groups = new Map<MeasurementKind, BodyMeasurement[]>();
    for (const entry of entries ?? []) {
      groups.set(entry.kind, [...(groups.get(entry.kind) ?? []), entry]);
    }
    return groups;
  }, [entries]);

  const number = (amount: number) => formatNumber(amount, { maximumFractionDigits: 1 });
  const withUnit = (amount: number, of: MeasurementKind) => `${number(amount)} ${measurementUnit(of)}`;
  const day = (iso: string) => formatDate(`${iso}T00:00:00`, { day: "numeric", month: "short", year: "numeric" });

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    const amount = Number(value.replace(",", "."));
    if (!value.trim() || !Number.isFinite(amount)) {
      setMessage({ ok: false, text: text.errValue });
      return;
    }
    setSaving(true);
    setMessage(null);
    try {
      const saved = await saveMeasurement(kind, amount, date);
      setEntries((current) =>
        [...(current ?? []).filter((entry) => !(entry.kind === saved.kind && entry.measured_on === saved.measured_on)), saved].sort(
          (a, b) => a.measured_on.localeCompare(b.measured_on) || a.kind.localeCompare(b.kind)
        )
      );
      setValue("");
      setSelected(saved.kind);
      setMessage({ ok: true, text: text.saved });
    } catch (error) {
      const field = error instanceof ApiError && error.code === "VALIDATION_ERROR" ? error.details[0] : "";
      setMessage({ ok: false, text: field === "measured_on" ? text.errDate : field === "value" ? text.errValue : text.errSave });
    } finally {
      setSaving(false);
    }
  };

  const remove = async (entry: BodyMeasurement) => {
    if (!window.confirm(text.confirmRemove)) return;
    try {
      await deleteMeasurement(entry.id);
      setEntries((current) => (current ?? []).filter((item) => item.id !== entry.id));
    } catch {
      setMessage({ ok: false, text: text.errDelete });
    }
  };

  if (loadError) {
    return (
      <div className={styles.state}>
        <p>{text.loadError}</p>
        <button type="button" className={styles.quietButton} onClick={() => void load()}>
          <RefreshCw aria-hidden="true" /> {text.retry}
        </button>
      </div>
    );
  }
  if (!entries) return <p className={styles.muted}>{text.loading}</p>;

  const series = byKind.get(selected) ?? [];
  const points: ChartPoint[] = series.slice(-20).map((entry) => ({
    label: formatDate(`${entry.measured_on}T00:00:00`, { day: "numeric", month: "short" }),
    value: entry.value,
    description: `${day(entry.measured_on)}: ${withUnit(entry.value, entry.kind)}`
  }));
  const loggedKinds = MEASUREMENT_KINDS.filter((item) => byKind.has(item));

  return (
    <section className={styles.measurements} aria-labelledby="measurements-title">
      <header className={styles.panelHead}>
        <h2 id="measurements-title" className={styles.panelTitle}>
          {text.title}
        </h2>
        <p className={styles.muted}>{text.lead}</p>
      </header>

      <form className={styles.measureForm} onSubmit={(event) => void submit(event)}>
        <h3 className={styles.cardTitle}>{text.add}</h3>
        <div className={styles.measureFields}>
          <label>
            <span>{text.kind}</span>
            <select value={kind} onChange={(event) => setKind(event.target.value as MeasurementKind)}>
              <optgroup label={text.body}>
                {MEASUREMENT_KINDS.slice(0, 2).map((item) => (
                  <option key={item} value={item}>
                    {labels[item]} ({measurementUnit(item)})
                  </option>
                ))}
              </optgroup>
              <optgroup label={text.circumferences}>
                {MEASUREMENT_KINDS.slice(2).map((item) => (
                  <option key={item} value={item}>
                    {labels[item]} (cm)
                  </option>
                ))}
              </optgroup>
            </select>
          </label>
          <label>
            <span>
              {text.value} ({measurementUnit(kind)})
            </span>
            <input type="text" inputMode="decimal" value={value} placeholder={kind === "weight" ? "75,5" : kind === "body_fat" ? "18" : "40"} onChange={(event) => setValue(event.target.value)} />
          </label>
          <label>
            <span>{text.date}</span>
            <input type="date" value={date} max={today()} onChange={(event) => setDate(event.target.value)} required />
          </label>
          <button type="submit" className={styles.primaryButton} disabled={saving}>
            <Plus aria-hidden="true" /> {saving ? text.saving : text.save}
          </button>
        </div>
        {message ? (
          <p className={message.ok ? styles.ok : styles.error} role={message.ok ? "status" : "alert"}>
            {message.text}
          </p>
        ) : null}
      </form>

      {loggedKinds.length === 0 ? (
        <p className={styles.state}>{text.none}</p>
      ) : (
        <>
          <div>
            <h3 className={styles.cardTitle}>{text.latest}</h3>
            <div className={styles.latestGrid}>
              {loggedKinds.map((item) => {
                const list = byKind.get(item) ?? [];
                const last = list[list.length - 1];
                const previous = list[list.length - 2];
                const delta = previous ? Math.round((last.value - previous.value) * 10) / 10 : 0;
                return (
                  <button
                    key={item}
                    type="button"
                    className={joinClassNames(styles.latest, selected === item && styles.latestActive)}
                    aria-pressed={selected === item}
                    onClick={() => setSelected(item)}
                  >
                    <span className={styles.latestLabel}>{labels[item]}</span>
                    <span className={styles.latestValue}>{withUnit(last.value, item)}</span>
                    <span className={styles.latestDelta}>
                      {previous && delta !== 0 ? `${delta > 0 ? "▲ +" : "▼ "}${number(delta)}` : day(last.measured_on)}
                    </span>
                  </button>
                );
              })}
            </div>
          </div>

          <div className={styles.measureCharts}>
            <div className={styles.chartCard}>
              <h3 className={styles.cardTitle}>
                {text.evolution}: {labels[selected]} ({measurementUnit(selected)})
              </h3>
              {points.length >= 2 ? (
                <LineChart title={`${text.evolution}: ${labels[selected]}`} points={points} format={number} />
              ) : (
                <p className={styles.muted}>{text.needTwo}</p>
              )}
            </div>
            <div className={styles.chartCard}>
              <h3 className={styles.cardTitle}>
                {text.history}: {labels[selected]}
              </h3>
              <ul className={styles.historyList}>
                {[...series]
                  .reverse()
                  .slice(0, 12)
                  .map((entry) => (
                    <li key={entry.id}>
                      <span>{day(entry.measured_on)}</span>
                      <strong>{withUnit(entry.value, entry.kind)}</strong>
                      <button type="button" className={styles.iconButton} aria-label={`${text.remove} ${labels[entry.kind]} ${day(entry.measured_on)}`} onClick={() => void remove(entry)}>
                        <Trash2 aria-hidden="true" />
                      </button>
                    </li>
                  ))}
              </ul>
            </div>
          </div>
        </>
      )}
    </section>
  );
};
