import { Crown, Medal, Trophy, Users } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { type Copy, useI18n } from "@/i18n/i18n";
import { fetchLiftRankings, type LiftRanking } from "@/services/achievements_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./lift_rankings.module.css";

const COPY: Copy<{
  title: string;
  lead: string;
  exercise: string;
  top: (percent: number) => string;
  verdict: (name: string, percent: number) => string;
  beats: (percent: number, count: number) => string;
  alone: string;
  best: (weight: string, reps: number, e1rm: string) => string;
  rank: (rank: number, total: number) => string;
  empty: string;
  error: string;
  how: string;
}> = {
  en: {
    title: "Compare with RYZE",
    lead: "Pick an exercise and see where your best lift stands among every lifter on RYZE.",
    exercise: "Exercise",
    top: (percent) => `Top ${percent}%`,
    verdict: (name, percent) =>
      percent <= 1 ? `${name}: outstanding — you are in the top 1% of RYZE!` : percent <= 10 ? `${name}: great — top ${percent}% of RYZE.` : percent <= 50 ? `${name}: above most lifters — top ${percent}%.` : `${name}: top ${percent}%. Every workout moves you up.`,
    beats: (percent, count) => `You lift more than ${percent}% of the other ${count} ${count === 1 ? "lifter" : "lifters"} who trained it.`,
    alone: "You are the first on RYZE to log this exercise. Others will join the board soon.",
    best: (weight, reps, e1rm) => `Your best: ${weight} kg × ${reps} (estimated 1RM ${e1rm} kg)`,
    rank: (rank, total) => `#${rank} of ${total}`,
    empty: "Log a working set with weight on any exercise and your ranking appears here.",
    error: "We could not load the rankings.",
    how: "Ranked by estimated one-rep max (Epley). Only your own numbers are shown; nobody sees yours."
  },
  pt: {
    title: "Compara-te com a RYZE",
    lead: "Escolhe um exercício e vê onde está a tua melhor marca entre todos os lifters da RYZE.",
    exercise: "Exercício",
    top: (percent) => `Top ${percent}%`,
    verdict: (name, percent) =>
      percent <= 1 ? `${name}: boa! Pertences ao top 1% da plataforma!` : percent <= 10 ? `${name}: excelente — top ${percent}% da RYZE.` : percent <= 50 ? `${name}: acima da maioria — top ${percent}%.` : `${name}: top ${percent}%. Cada treino faz-te subir.`,
    beats: (percent, count) => `Levantas mais do que ${percent}% ${count === 1 ? "do outro lifter que o treinou" : `dos outros ${count} lifters que o treinaram`}.`,
    alone: "És o primeiro na RYZE a registar este exercício. Em breve terás com quem te comparar.",
    best: (weight, reps, e1rm) => `A tua melhor: ${weight} kg × ${reps} (1RM estimada ${e1rm} kg)`,
    rank: (rank, total) => `#${rank} de ${total}`,
    empty: "Regista uma série com peso em qualquer exercício e o teu ranking aparece aqui.",
    error: "Não foi possível carregar os rankings.",
    how: "Ordenado pela 1RM estimada (Epley). Só vês os teus números; ninguém vê os teus."
  }
};

const capitalise = (value: string) => value.replace(/\b\p{L}/gu, (letter) => letter.toUpperCase());

const tier = (percent: number) => (percent <= 1 ? styles.tierElite : percent <= 10 ? styles.tierGold : percent <= 50 ? styles.tierSilver : styles.tierBase);

// LiftRankings compares the member's best lift on an exercise with every
// member of RYZE (owner request, 2026-11-22, like Lyfta's "Lyftoff").
export const LiftRankings = () => {
  const { locale, formatNumber } = useI18n();
  const text = COPY[locale];
  const [rankings, setRankings] = useState<LiftRanking[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [selected, setSelected] = useState("");

  useEffect(() => {
    let cancelled = false;
    fetchLiftRankings()
      .then((list) => {
        if (cancelled) return;
        setRankings(list);
        setSelected(list[0]?.exercise_id ?? "");
      })
      .catch(() => {
        if (!cancelled) setFailed(true);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const current = useMemo(() => rankings?.find((ranking) => ranking.exercise_id === selected) ?? null, [rankings, selected]);
  const kg = (value: number) => formatNumber(value, { maximumFractionDigits: 1 });

  return (
    <section className={styles.rankings} aria-labelledby="lift-rankings">
      <header className={styles.head}>
        <h2 id="lift-rankings" className={styles.title}>
          <Users aria-hidden="true" /> {text.title}
        </h2>
        <p className={styles.lead}>{text.lead}</p>
      </header>

      {failed ? <p className={styles.muted}>{text.error}</p> : null}
      {!failed && rankings === null ? <div className={styles.skeleton} aria-hidden="true" /> : null}
      {rankings && rankings.length === 0 ? <p className={styles.muted}>{text.empty}</p> : null}

      {rankings && rankings.length > 0 ? (
        <>
          <label className={styles.picker}>
            <span>{text.exercise}</span>
            <select value={selected} onChange={(event) => setSelected(event.target.value)}>
              {rankings.map((ranking) => (
                <option key={ranking.exercise_id} value={ranking.exercise_id}>
                  {capitalise(ranking.exercise_name)}
                </option>
              ))}
            </select>
          </label>

          {current ? (
            <div className={joinClassNames(styles.result, tier(current.top_percent))} aria-live="polite">
              <div className={styles.badge}>
                {current.top_percent <= 1 ? <Crown aria-hidden="true" /> : current.top_percent <= 10 ? <Trophy aria-hidden="true" /> : <Medal aria-hidden="true" />}
                <strong>{text.top(current.top_percent)}</strong>
                <span>{text.rank(current.rank, current.participants)}</span>
              </div>
              <div className={styles.copy}>
                <p className={styles.verdict}>{text.verdict(capitalise(current.exercise_name), current.top_percent)}</p>
                <p className={styles.muted}>{current.participants > 1 ? text.beats(current.beats_percent, current.participants - 1) : text.alone}</p>
                <p className={styles.best}>{text.best(kg(current.best_weight_kg), current.best_reps, kg(current.best_e1rm_kg))}</p>
                <div
                  className={styles.scale}
                  role="img"
                  aria-label={text.rank(current.rank, current.participants)}
                >
                  <span className={styles.marker} style={{ left: `${Math.max(2, Math.min(98, 100 - current.top_percent))}%` }} />
                </div>
              </div>
            </div>
          ) : null}
          <p className={styles.how}>{text.how}</p>
        </>
      ) : null}
    </section>
  );
};
