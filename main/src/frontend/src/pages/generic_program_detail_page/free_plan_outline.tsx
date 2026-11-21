import { CalendarDays, Dumbbell, ListChecks } from "lucide-react";
import { useEffect, useState } from "react";

import { type Catalog, useGettext, useI18n } from "@/i18n/i18n";
import { fetchProgramOutline, type ProgramOutline } from "@/services/marketplace_api";

import styles from "./free_plan_outline.module.css";

const OUTLINE_PT: Catalog = {
  "Plan overview": "Visão geral do plano",
  "Every week and training day of this plan. The exercises open once you get it.":
    "Todas as semanas e dias de treino deste plano. Os exercícios ficam disponíveis quando o obtiveres.",
  "Week {number}": "Semana {number}",
  "Day {number}": "Dia {number}",
  "{count} exercises": "{count} exercícios",
  "1 exercise": "1 exercício",
  "{count} weeks": "{count} semanas",
  "{count} training days": "{count} dias de treino",
  "Loading the plan overview…": "A carregar a visão geral do plano…",
  "The plan overview is not available right now.": "A visão geral do plano não está disponível de momento.",
  "This plan has no training days yet.": "Este plano ainda não tem dias de treino."
};

// The day focus values (migration 000043), in both languages.
const FOCUS: Record<string, { en: string; pt: string }> = {
  full_body: { en: "Full body", pt: "Corpo inteiro" },
  upper_body: { en: "Upper body", pt: "Parte superior" },
  lower_body: { en: "Lower body", pt: "Parte inferior" },
  push: { en: "Push", pt: "Empurrar" },
  pull: { en: "Pull", pt: "Puxar" },
  legs: { en: "Legs", pt: "Pernas" },
  chest: { en: "Chest", pt: "Peito" },
  back: { en: "Back", pt: "Costas" },
  shoulders: { en: "Shoulders", pt: "Ombros" },
  arms: { en: "Arms", pt: "Braços" },
  core: { en: "Core", pt: "Core" },
  glutes: { en: "Glutes", pt: "Glúteos" },
  cardio: { en: "Cardio", pt: "Cardio" },
  mobility: { en: "Mobility", pt: "Mobilidade" }
};

// FreePlanOutline shows a free plan's weeks and training days — each day's
// title, focus and number of exercises — without the exercises themselves
// (owner request, 2026-11-20).
export const FreePlanOutline = ({ programId }: { programId: string }) => {
  const tr = useGettext(OUTLINE_PT);
  const { locale } = useI18n();
  const [outline, setOutline] = useState<ProgramOutline | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");

  useEffect(() => {
    let cancelled = false;
    setState("loading");
    fetchProgramOutline(programId)
      .then((result) => {
        if (cancelled) return;
        setOutline(result);
        setState("ready");
      })
      .catch(() => {
        if (!cancelled) setState("error");
      });
    return () => {
      cancelled = true;
    };
  }, [programId]);

  const exercises = (count: number) => (count === 1 ? tr("1 exercise") : tr("{count} exercises", { count }));

  return (
    <section className={styles.outline} aria-labelledby="plan-overview">
      <header className={styles.head}>
        <h2 id="plan-overview" className={styles.title}>
          {tr("Plan overview")}
        </h2>
        <p className={styles.lead}>{tr("Every week and training day of this plan. The exercises open once you get it.")}</p>
        {outline && outline.total_days > 0 ? (
          <ul className={styles.totals}>
            <li>
              <CalendarDays aria-hidden="true" /> {tr("{count} weeks", { count: outline.weeks.length })}
            </li>
            <li>
              <ListChecks aria-hidden="true" /> {tr("{count} training days", { count: outline.total_days })}
            </li>
            <li>
              <Dumbbell aria-hidden="true" /> {exercises(outline.total_exercises)}
            </li>
          </ul>
        ) : null}
      </header>

      {state === "loading" ? <p className={styles.state}>{tr("Loading the plan overview…")}</p> : null}
      {state === "error" ? <p className={styles.state}>{tr("The plan overview is not available right now.")}</p> : null}
      {state === "ready" && outline && outline.weeks.length === 0 ? (
        <p className={styles.state}>{tr("This plan has no training days yet.")}</p>
      ) : null}

      {state === "ready" && outline ? (
        <ol className={styles.weeks}>
          {outline.weeks.map((week, weekIndex) => (
            <li key={week.week_number} className={styles.week}>
              <details open={weekIndex === 0}>
                <summary className={styles.weekSummary}>
                  <span>{tr("Week {number}", { number: week.week_number })}</span>
                  <span className={styles.weekMeta}>
                    {tr("{count} training days", { count: week.days.length })} ·{" "}
                    {exercises(week.days.reduce((sum, day) => sum + day.exercise_count, 0))}
                  </span>
                </summary>
                <ol className={styles.days}>
                  {week.days.map((day, dayIndex) => (
                    <li key={`${week.week_number}-${day.position}`} className={styles.day}>
                      <span className={styles.dayNumber}>{tr("Day {number}", { number: dayIndex + 1 })}</span>
                      <span className={styles.dayTitle}>
                        {day.title || tr("Day {number}", { number: dayIndex + 1 })}
                        {day.focus && FOCUS[day.focus] ? <em className={styles.focus}>{FOCUS[day.focus][locale]}</em> : null}
                      </span>
                      <span className={styles.dayCount}>{exercises(day.exercise_count)}</span>
                    </li>
                  ))}
                </ol>
              </details>
            </li>
          ))}
        </ol>
      ) : null}
    </section>
  );
};
