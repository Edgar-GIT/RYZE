import type { PremiumLevel2Catalog, PremiumLevel2Intake } from "@/services/premium_level_2_api";
import {
  formatPremiumLevel2Answer,
  humanizeAnswerKey,
  type PremiumLevel2AnswerValue
} from "@/services/premium_level_2_format";

import styles from "./intake_answers.module.css";

export const PremiumLevel2IntakeAnswers = ({
  intake,
  catalog
}: {
  intake: PremiumLevel2Intake | null;
  catalog: PremiumLevel2Catalog | null;
}) => {
  if (!intake) {
    return <p className={styles.muted}>The intake has not been submitted yet.</p>;
  }

  const answers = intake.answers;
  const sections = catalog?.sections ?? [];
  const catalogKeys = new Set(
    sections.flatMap((section) => section.questions.map((question) => question.key))
  );

  return (
    <div className={styles.sections}>
      {sections.map((section) => {
        const answered = section.questions.filter((question) => question.key in answers);
        if (answered.length === 0) return null;
        return (
          <div key={section.key} className={styles.section}>
            <h4 className={styles.sectionTitle}>{section.title}</h4>
            <dl className={styles.grid}>
              {answered.map((question) => (
                <div key={question.key} className={styles.row}>
                  <dt>{question.label}</dt>
                  <dd>{formatPremiumLevel2Answer(question, answers[question.key])}</dd>
                </div>
              ))}
            </dl>
          </div>
        );
      })}

      {Object.entries(answers)
        .filter(([key]) => !catalogKeys.has(key))
        .map(([key, value]) => {
          const question = catalog?.sections
            .flatMap((section) => section.questions)
            .find((entry) => entry.key === key);
          return (
            <div key={key} className={styles.row}>
              <dt>{question?.label ?? humanizeAnswerKey(key)}</dt>
              <dd>
                {question
                  ? formatPremiumLevel2Answer(question, value)
                  : typeof value === "object"
                    ? JSON.stringify(value)
                    : String(value as PremiumLevel2AnswerValue)}
              </dd>
            </div>
          );
        })}
    </div>
  );
};
