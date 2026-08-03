import { SectionWrapper } from "@/components/section_wrapper/section_wrapper";

import styles from "./coaches_section.module.css";

const coaches = [
  {
    initials: "MF",
    name: "Marco Ferreira",
    specialty: "Strength & Powerlifting",
    credential: "IPF - 11 yrs"
  },
  {
    initials: "NK",
    name: "Nadia Kaur",
    specialty: "Sports Nutrition",
    credential: "MSc Dietetics"
  },
  {
    initials: "EB",
    name: "Elias Brandt",
    specialty: "Hypertrophy Systems",
    credential: "NSCA-CSCS"
  },
  {
    initials: "SA",
    name: "Sofia Almeida",
    specialty: "Rehab & Longevity",
    credential: "PT, DPT"
  }
] as const;

export const CoachesSection = () => (
  <SectionWrapper className={styles.section} containerClassName={styles.inner}>
    <div className={styles.header}>
      <p className={styles.eyebrow}>Professional coaches</p>
      <h2>Credentialed people, not chat scripts.</h2>
      <p>
        Every RYZE coach is vetted, certified and accountable for the athletes
        on their roster.
      </p>
    </div>

    <div className={styles.grid}>
      {coaches.map((coach) => (
        <article className={styles.card} key={coach.name}>
          <div className={styles.avatar} aria-hidden="true">
            {coach.initials}
          </div>
          <div className={styles.coachCopy}>
            <h3>{coach.name}</h3>
            <p>{coach.specialty}</p>
          </div>
          <span className={styles.credential}>{coach.credential}</span>
        </article>
      ))}
    </div>
  </SectionWrapper>
);
