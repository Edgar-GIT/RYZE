import { SectionWrapper } from "@/components/section_wrapper/section_wrapper";

import styles from "./testimonials_section.module.css";

const testimonials = [
  {
    quote: "I stopped guessing. The plan changes before I notice I need it to.",
    name: "Tomas R.",
    result: "Deadlift 180 → 232 kg"
  },
  {
    quote: "Nutrition moves with my training week instead of fighting it.",
    name: "Hanna L.",
    result: "−11% body fat"
  },
  {
    quote: "It feels like software built by people who actually lift.",
    name: "Dan M.",
    result: "Third year on RYZE"
  }
] as const;

export const TestimonialsSection = () => (
  <SectionWrapper className={styles.section} containerClassName={styles.inner}>
    <div className={styles.header}>
      <p className={styles.eyebrow}>Athletes</p>
      <h2>Progress, on record.</h2>
    </div>

    <div className={styles.grid}>
      {testimonials.map((testimonial) => (
        <article className={styles.item} key={testimonial.name}>
          <blockquote>{testimonial.quote}</blockquote>
          <div className={styles.person}>
            <strong>{testimonial.name}</strong>
            <span>{testimonial.result}</span>
          </div>
        </article>
      ))}
    </div>
  </SectionWrapper>
);
