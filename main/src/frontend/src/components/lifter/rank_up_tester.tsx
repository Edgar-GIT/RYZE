import { Sparkles } from "lucide-react";
import { type CSSProperties, useState } from "react";

import { type Copy, useI18n } from "@/i18n/i18n";
import type { LifterLevelKey } from "@/services/achievements_api";

import { LEVEL_NAMES } from "./lifter";
import { RANK_COLORS, RankUpCelebration } from "./rank_up_celebration";
import styles from "./rank_up_tester.module.css";

const KEYS: LifterLevelKey[] = ["rookie", "beginner", "average", "dedicated", "advanced", "elite", "master", "legend"];

const COPY: Copy<{ open: string; title: string; hint: string; play: (name: string) => string }> = {
  en: {
    open: "Test level up",
    title: "Level-up animation",
    hint: "Preview only: your EXP and level do not change.",
    play: (name) => `Play the level up to ${name}`
  },
  pt: {
    open: "Testar subida de nível",
    title: "Animação de subida de nível",
    hint: "Só pré-visualização: o teu EXP e o teu nível não mudam.",
    play: (name) => `Ver a subida para ${name}`
  }
};

// RankUpTester lets Test Mode (and local development) play the level-up
// animation for every level and colour (owner request, 2026-11-21). It is a
// preview: nothing is saved.
export const RankUpTester = ({ className }: { className?: string }) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const names = LEVEL_NAMES[locale];
  const [open, setOpen] = useState(false);
  const [playing, setPlaying] = useState<number | null>(null);

  return (
    <div className={[styles.tester, className].filter(Boolean).join(" ")}>
      <button type="button" className={styles.toggle} aria-expanded={open} onClick={() => setOpen((value) => !value)}>
        <Sparkles size={14} aria-hidden="true" /> {text.open}
      </button>
      {open ? (
        <div className={styles.panel} role="group" aria-label={text.title}>
          <p className={styles.hint}>{text.hint}</p>
          <ul className={styles.levels}>
            {KEYS.map((key, index) => (
              <li key={key}>
                <button
                  type="button"
                  className={styles.level}
                  style={{ "--rank": RANK_COLORS[key][0], "--rank-2": RANK_COLORS[key][1] } as CSSProperties}
                  aria-label={text.play(names[key])}
                  onClick={() => setPlaying(index)}
                >
                  <b>{index + 1}</b> {names[key]}
                </button>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      {playing !== null ? <RankUpCelebration level={{ rank: playing + 1, key: KEYS[playing] }} onClose={() => setPlaying(null)} /> : null}
    </div>
  );
};
