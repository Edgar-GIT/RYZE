import { Sparkles, X } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";

import { type Copy, useI18n } from "@/i18n/i18n";
import type { Sex } from "@/services/plan_preferences_api";

import styles from "./plan_audience_prompt.module.css";

const COPY: Copy<{
  kicker: string;
  askSex: string;
  sexes: Record<Sex, string>;
  askFilter: Record<Sex, string>;
  yes: string;
  no: string;
  later: string;
  note: string;
  back: string;
}> = {
  en: {
    kicker: "Plans that suit you",
    askSex: "Are you a woman or a man?",
    sexes: { female: "Woman", male: "Man" },
    askFilter: {
      female: "Show the plans designed for women first?",
      male: "Show the plans designed for men first?"
    },
    yes: "Yes, show them",
    no: "No, show every plan",
    later: "Not now",
    note: "Plans for everyone always appear. You can change this at any time.",
    back: "Change answer"
  },
  pt: {
    kicker: "Planos à tua medida",
    askSex: "És mulher ou homem?",
    sexes: { female: "Mulher", male: "Homem" },
    askFilter: {
      female: "Queres ver primeiro os planos pensados para mulheres?",
      male: "Queres ver primeiro os planos pensados para homens?"
    },
    yes: "Sim, mostrar",
    no: "Não, mostrar todos",
    later: "Agora não",
    note: "Os planos para todos aparecem sempre. Podes mudar isto quando quiseres.",
    back: "Mudar resposta"
  }
};

export interface PlanAudienceAnswer {
  sex: Sex | null;
  filter: boolean;
}

interface PlanAudiencePromptProps {
  initialSex: Sex | null;
  onAnswer: (answer: PlanAudienceAnswer) => void;
}

// PlanAudiencePrompt asks, the first time someone opens the training plans,
// whether they are a woman or a man and whether they prefer to see the plans
// designed for them first (owner request, 2026-11-14). It sits above the
// catalogue and never blocks it; "Not now" shows every plan.
export const PlanAudiencePrompt = ({ initialSex, onAnswer }: PlanAudiencePromptProps) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const titleId = useId();
  const [sex, setSex] = useState<Sex | null>(initialSex);
  const [step, setStep] = useState<"sex" | "filter">(initialSex ? "filter" : "sex");
  const questionRef = useRef<HTMLHeadingElement>(null);
  const started = useRef(false);

  // Move focus to the new question after an answer (not on first render).
  useEffect(() => {
    if (started.current) questionRef.current?.focus();
    started.current = true;
  }, [step]);

  return (
    <section className={styles.prompt} aria-labelledby={titleId}>
      <Sparkles className={styles.icon} aria-hidden="true" />
      <div className={styles.body}>
        <p className={styles.kicker}>{text.kicker}</p>
        <h2 id={titleId} ref={questionRef} tabIndex={-1} className={styles.question} aria-live="polite">
          {step === "sex" || !sex ? text.askSex : text.askFilter[sex]}
        </h2>
        {step === "sex" || !sex ? (
          <div className={styles.options}>
            {(["female", "male"] as const).map((option) => (
              <button
                key={option}
                type="button"
                className={styles.option}
                aria-pressed={sex === option}
                onClick={() => {
                  setSex(option);
                  setStep("filter");
                }}
              >
                {text.sexes[option]}
              </button>
            ))}
          </div>
        ) : (
          <>
            <div className={styles.options}>
              <button type="button" className={styles.primary} onClick={() => onAnswer({ sex, filter: true })}>
                {text.yes}
              </button>
              <button type="button" className={styles.option} onClick={() => onAnswer({ sex, filter: false })}>
                {text.no}
              </button>
            </div>
            <button type="button" className={styles.link} onClick={() => setStep("sex")}>
              {text.sexes[sex]} · {text.back}
            </button>
          </>
        )}
        <p className={styles.note}>{text.note}</p>
      </div>
      <button type="button" className={styles.close} aria-label={text.later} onClick={() => onAnswer({ sex, filter: false })}>
        <X aria-hidden="true" />
      </button>
    </section>
  );
};
