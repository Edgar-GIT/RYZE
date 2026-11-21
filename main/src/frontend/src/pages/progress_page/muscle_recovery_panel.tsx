import { RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { MuscleMap } from "@/components/training/muscle_map";
import { useI18n } from "@/i18n/i18n";
import { fetchMuscleRecovery, type MuscleName, type MuscleRecoveryReport, type RecoveryStatus } from "@/services/body_stats_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./body_stats.module.css";

export const MUSCLE_LABELS: Record<"en" | "pt", Record<MuscleName, string>> = {
  en: {
    chest: "Chest",
    shoulders: "Shoulders",
    biceps: "Biceps",
    triceps: "Triceps",
    forearms: "Forearms",
    abs: "Abs",
    obliques: "Obliques",
    traps: "Traps",
    lats: "Lats",
    upper_back: "Upper back",
    lower_back: "Lower back",
    glutes: "Glutes",
    quads: "Quads",
    hamstrings: "Hamstrings",
    adductors: "Adductors",
    calves: "Calves"
  },
  pt: {
    chest: "Peito",
    shoulders: "Ombros",
    biceps: "Bíceps",
    triceps: "Tríceps",
    forearms: "Antebraços",
    abs: "Abdominais",
    obliques: "Oblíquos",
    traps: "Trapézio",
    lats: "Dorsais",
    upper_back: "Costas (parte superior)",
    lower_back: "Lombar",
    glutes: "Glúteos",
    quads: "Quadríceps",
    hamstrings: "Isquiotibiais",
    adductors: "Adutores",
    calves: "Gémeos"
  }
};

const COPY = {
  en: {
    title: "Muscle recovery",
    lead: "Estimated from the sets you completed in the last 7 days. Secondary muscles count half a set.",
    summary: (ready: number, total: number) => `${ready} of ${total} muscle groups ready to train`,
    status: { ready: "Ready", recovering: "Recovering", fatigued: "Fatigued" } as Record<RecoveryStatus, string>,
    front: "Front",
    back: "Back",
    mapLabel: (ready: string) => `Body map of muscle recovery. Ready to train: ${ready || "none"}.`,
    fullAt: "fully recovered",
    trained: "trained",
    notTrained: "Not trained in the last 7 days",
    sets: (value: string) => `${value} sets this week`,
    groups: { fatigued: "Fatigued", recovering: "Recovering", ready: "Ready to train" } as Record<RecoveryStatus, string>,
    selected: "Selected muscle",
    loadError: "We could not load your muscle recovery.",
    retry: "Try again",
    loading: "Loading…",
    estimate: "This is an estimate, not medical advice: listen to your body."
  },
  pt: {
    title: "Recuperação muscular",
    lead: "Estimada a partir das séries que completaste nos últimos 7 dias. Os músculos secundários contam meia série.",
    summary: (ready: number, total: number) => `${ready} de ${total} grupos musculares prontos a treinar`,
    status: { ready: "Pronto", recovering: "A recuperar", fatigued: "Cansado" } as Record<RecoveryStatus, string>,
    front: "Frente",
    back: "Costas",
    mapLabel: (ready: string) => `Mapa do corpo com a recuperação muscular. Prontos a treinar: ${ready || "nenhum"}.`,
    fullAt: "recuperado",
    trained: "treinado",
    notTrained: "Sem treino nos últimos 7 dias",
    sets: (value: string) => `${value} séries esta semana`,
    groups: { fatigued: "Cansados", recovering: "A recuperar", ready: "Prontos a treinar" } as Record<RecoveryStatus, string>,
    selected: "Músculo selecionado",
    loadError: "Não foi possível carregar a tua recuperação muscular.",
    retry: "Tentar de novo",
    loading: "A carregar…",
    estimate: "É uma estimativa, não um conselho médico: ouve o teu corpo."
  }
} as const;

// relative formats a moment as "in 14 hours" / "2 days ago".
const relative = (iso: string, intlLocale: string, now: number): string => {
  const format = new Intl.RelativeTimeFormat(intlLocale, { numeric: "auto" });
  const hours = (new Date(iso).getTime() - now) / 3_600_000;
  if (Math.abs(hours) < 1) return format.format(Math.round(hours * 60), "minute");
  if (Math.abs(hours) < 36) return format.format(Math.round(hours), "hour");
  return format.format(Math.round(hours / 24), "day");
};

// MuscleRecoveryPanel shows the body map and, beside it, every muscle group
// with its recovery percentage (owner request, 2026-11-20, as in Lyfta).
export const MuscleRecoveryPanel = () => {
  const { locale, intlLocale, formatNumber } = useI18n();
  const text = COPY[locale];
  const names = MUSCLE_LABELS[locale];
  const [report, setReport] = useState<MuscleRecoveryReport | null>(null);
  const [error, setError] = useState(false);
  const [active, setActive] = useState<MuscleName | null>(null);

  const load = useCallback(async () => {
    setError(false);
    try {
      setReport(await fetchMuscleRecovery());
    } catch {
      setError(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (error) {
    return (
      <div className={styles.state}>
        <p>{text.loadError}</p>
        <button type="button" className={styles.quietButton} onClick={() => void load()}>
          <RefreshCw aria-hidden="true" /> {text.retry}
        </button>
      </div>
    );
  }
  if (!report) return <p className={styles.muted}>{text.loading}</p>;

  const now = new Date(report.generated_at).getTime();
  const sorted = [...report.muscles].sort((a, b) => a.recovery - b.recovery || names[a.muscle].localeCompare(names[b.muscle]));
  const ready = sorted.filter((muscle) => muscle.status === "ready");
  // The detail card shows the chosen muscle, or the least recovered one.
  const shown = sorted.find((muscle) => muscle.muscle === active) ?? sorted[0];
  const toggle = (muscle: MuscleName) => setActive(active === muscle ? null : muscle);
  const metaOf = (muscle: (typeof sorted)[number]) =>
    `${
      muscle.recovered_at
        ? `${text.fullAt} ${relative(muscle.recovered_at, intlLocale, now)}`
        : muscle.last_trained_at
          ? `${text.trained} ${relative(muscle.last_trained_at, intlLocale, now)}`
          : text.notTrained
    }${muscle.weekly_sets > 0 ? ` · ${text.sets(formatNumber(muscle.weekly_sets, { maximumFractionDigits: 1 }))}` : ""}`;

  return (
    <section className={styles.recovery} aria-labelledby="muscle-recovery-title">
      <header className={styles.panelHead}>
        <h2 id="muscle-recovery-title" className={styles.panelTitle}>
          {text.title}
        </h2>
        <p className={styles.muted}>{text.lead}</p>
        <p className={styles.summary}>{text.summary(ready.length, report.muscles.length)}</p>
      </header>

      <div className={styles.recoveryGrid}>
        <div className={styles.mapCard}>
          <MuscleMap
            muscles={report.muscles}
            label={text.mapLabel(ready.map((muscle) => names[muscle.muscle]).join(", "))}
            frontLabel={text.front}
            backLabel={text.back}
            active={active}
            onSelect={toggle}
          />
          <ul className={styles.legend} aria-label={text.title}>
            {(["ready", "recovering", "fatigued"] as const).map((status) => (
              <li key={status}>
                <span className={joinClassNames(styles.swatch, styles[status])} aria-hidden="true" />
                {text.status[status]}
              </li>
            ))}
          </ul>
        </div>

        <div className={styles.recoverySide}>
          {shown ? (
            <div className={joinClassNames(styles.focusCard, styles[`focus_${shown.status}`])} aria-live="polite">
              <span className={styles.focusKicker}>{text.selected}</span>
              <div className={styles.focusTop}>
                <strong className={styles.focusName}>{names[shown.muscle]}</strong>
                <span className={joinClassNames(styles.focusPercent, styles[`text_${shown.status}`])}>{shown.recovery}%</span>
              </div>
              <span className={styles.bar} aria-hidden="true">
                <span className={styles[shown.status]} style={{ width: `${Math.max(2, shown.recovery)}%` }} />
              </span>
              <span className={styles.muscleMeta}>
                <b className={styles[`text_${shown.status}`]}>{text.status[shown.status]}</b> · {metaOf(shown)}
              </span>
            </div>
          ) : null}

          {(["fatigued", "recovering", "ready"] as const).map((status) => {
            const group = sorted.filter((muscle) => muscle.status === status);
            if (group.length === 0) return null;
            return (
              <div key={status} className={styles.muscleGroup}>
                <h3 className={styles.groupTitle}>
                  <span className={joinClassNames(styles.swatch, styles[status])} aria-hidden="true" />
                  {text.groups[status]} <span className={styles.groupCount}>{group.length}</span>
                </h3>
                <ul className={styles.muscleChips}>
                  {group.map((muscle) => (
                    <li key={muscle.muscle}>
                      <button
                        type="button"
                        className={joinClassNames(styles.muscleChip, active === muscle.muscle && styles.muscleRowActive)}
                        aria-pressed={active === muscle.muscle}
                        aria-label={`${names[muscle.muscle]}: ${muscle.recovery}%, ${text.status[muscle.status]}. ${metaOf(muscle)}`}
                        onClick={() => toggle(muscle.muscle)}
                      >
                        <span className={styles.chipName}>{names[muscle.muscle]}</span>
                        <span className={joinClassNames(styles.chipPercent, styles[`text_${muscle.status}`])}>{muscle.recovery}%</span>
                        <span className={styles.chipBar} aria-hidden="true">
                          <span className={styles[muscle.status]} style={{ width: `${Math.max(2, muscle.recovery)}%` }} />
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            );
          })}
        </div>
      </div>
      <p className={styles.muted}>{text.estimate}</p>
    </section>
  );
};
