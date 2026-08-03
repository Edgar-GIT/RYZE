import { Plus, X } from "lucide-react";
import { useState } from "react";

import { SectionWrapper } from "@/components/section_wrapper/section_wrapper";

import styles from "./faq_section.module.css";

const faqItems = [
  {
    question: "Do I need a gym membership?",
    answer:
      "No. RYZE plans can be prepared for gym equipment, home training or hybrid routines depending on what the user has available."
  },
  {
    question: "Is the AI writing my program alone?",
    answer:
      "No. The long-term RYZE model combines automation with trainer oversight so recommendations stay useful, explainable and accountable."
  },
  {
    question: "How is nutrition personalised?",
    answer:
      "Nutrition guidance starts from goals, routine, preferences and restrictions, then evolves as progress and feedback become available."
  },
  {
    question: "Can I switch plans later?",
    answer:
      "Yes. The frontend is prepared for different plan levels so users can begin simple and move into more personal support later."
  },
  {
    question: "What happens to my data if I leave?",
    answer:
      "You can export your complete history at any moment, and request full deletion in one action when account tools become available."
  }
] as const;

export const FaqSection = () => {
  const [activeIndex, setActiveIndex] = useState(faqItems.length - 1);

  const toggleItem = (index: number) => {
    setActiveIndex((currentIndex) => (currentIndex === index ? -1 : index));
  };

  return (
    <SectionWrapper className={styles.section} containerClassName={styles.inner} size="narrow">
      <div className={styles.header}>
        <p className={styles.eyebrow}>FAQ</p>
        <h2>Answers, briefly.</h2>
      </div>

      <div className={styles.list}>
        {faqItems.map((item, index) => {
          const isActive = activeIndex === index;
          const answerId = `faq-answer-${index}`;

          return (
            <div className={styles.item} key={item.question}>
              <button
                className={styles.question}
                type="button"
                aria-expanded={isActive}
                aria-controls={answerId}
                onClick={() => toggleItem(index)}
              >
                <span>{item.question}</span>
                {isActive ? <X aria-hidden="true" /> : <Plus aria-hidden="true" />}
              </button>

              {isActive && (
                <p className={styles.answer} id={answerId}>
                  {item.answer}
                </p>
              )}
            </div>
          );
        })}
      </div>
    </SectionWrapper>
  );
};
