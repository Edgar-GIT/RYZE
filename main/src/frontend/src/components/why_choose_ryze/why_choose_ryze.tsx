import { Activity, Bot, LineChart, ShieldCheck } from "lucide-react";

import { GlassPanel } from "@/components/glass_panel/glass_panel";
import { SectionTitle } from "@/components/section_title/section_title";
import { SectionWrapper } from "@/components/section_wrapper/section_wrapper";

import styles from "./why_choose_ryze.module.css";

const reasons = [
  {
    title: "Structured from day one",
    description:
      "Plans are organized around clear progression, so users know what to do next instead of guessing.",
    icon: Activity
  },
  {
    title: "Built for automation",
    description:
      "The product foundation is ready to evolve into AI-assisted training and nutrition delivery.",
    icon: Bot
  },
  {
    title: "Premium trainer scale",
    description:
      "RYZE is designed to help certified trainers distribute plans without sacrificing quality.",
    icon: ShieldCheck
  },
  {
    title: "Progress-first interface",
    description:
      "Every interaction is shaped around mobile clarity, fast decisions and long-term consistency.",
    icon: LineChart
  }
];

export const WhyChooseRyze = () => (
  <SectionWrapper className={styles.section} containerClassName={styles.inner}>
    <div className={styles.copy}>
      <SectionTitle
        eyebrow="Why RYZE"
        title="A fitness platform engineered like serious software."
        description="RYZE combines clear plan delivery, automation readiness and a high-end mobile-first experience for users who want structure without friction."
      />
      <GlassPanel className={styles.signalPanel}>
        <span>Launch focus</span>
        <strong>Training + nutrition plan delivery</strong>
        <p>Built as a premium foundation before backend workflows are connected.</p>
      </GlassPanel>
    </div>

    <div className={styles.grid}>
      {reasons.map((reason) => {
        const Icon = reason.icon;

        return (
          <GlassPanel key={reason.title} as="article" className={styles.reason}>
            <span className={styles.icon}>
              <Icon aria-hidden="true" />
            </span>
            <h3>{reason.title}</h3>
            <p>{reason.description}</p>
          </GlassPanel>
        );
      })}
    </div>
  </SectionWrapper>
);
