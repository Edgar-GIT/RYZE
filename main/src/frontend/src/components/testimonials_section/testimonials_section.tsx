import { SectionWrapper } from "@/components/section_wrapper/section_wrapper";

import styles from "./testimonials_section.module.css";

const testimonials = [
  {
    quote:
      "I stopped guessing. The plan changes before I notice I need it to and there is a real coach signing off on it.",
    name: "Tomas R.",
    result: "Deadlift 180 -> 232 kg in 9 months"
  },
  {
    quote:
      "The nutrition side is the difference. It moves with my training week instead of punishing me for it.",
    name: "Hanna L.",
    result: "-11% body fat, strength maintained"
  },
  {
    quote:
      "It feels like software built by people who actually lift. Everything is where it should be.",
    name: "Dan M.",
    result: "Third year on the platform"
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
        <article className={styles.card} key={testimonial.name}>
          <blockquote>"{testimonial.quote}"</blockquote>
          <div className={styles.divider} />
          <div className={styles.person}>
            <strong>{testimonial.name}</strong>
            <span>{testimonial.result}</span>
          </div>
        </article>
      ))}
    </div>
  </SectionWrapper>
);
