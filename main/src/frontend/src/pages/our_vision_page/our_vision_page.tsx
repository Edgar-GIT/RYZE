import { motion, useReducedMotion } from "framer-motion";
import {
  Activity,
  Brain,
  ChartNoAxesCombined,
  Cpu,
  Crosshair,
  Dumbbell,
  Gauge,
  Globe2,
  Layers3,
  Leaf,
  MonitorSmartphone,
  Play,
  Radar,
  RefreshCw,
  ShieldCheck,
  Sparkles,
  Timer,
  Trophy,
  Utensils,
  WandSparkles,
  Zap
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import ourVisionBackground from "@resources/img/hero/our_view.png";

import styles from "./our_vision_page.module.css";

const fadeUp = {
  hidden: { opacity: 0, y: 28 },
  visible: { opacity: 1, y: 0 }
};

const missionCards = [
  {
    icon: Globe2,
    title: "Accessible",
    text: "Professional guidance without waiting rooms or gatekeeping."
  },
  {
    icon: Zap,
    title: "Affordable",
    text: "Technology lowers cost without lowering the quality of the plan."
  },
  {
    icon: ShieldCheck,
    title: "Professional",
    text: "Structure, progression and review — not random workout lists."
  },
  {
    icon: Crosshair,
    title: "Simple",
    text: "Clear weekly actions. Open the app and know what to do."
  },
  {
    icon: Cpu,
    title: "Technology first",
    text: "Built like modern software, not a static fitness brochure."
  }
] as const;

const whyCards = [
  {
    icon: Dumbbell,
    title: "Free Programs",
    text: "Start today with ready-made structure and zero waiting."
  },
  {
    icon: Sparkles,
    title: "Premium Programs",
    text: "Plans that adapt around your goals, schedule and phase."
  },
  {
    icon: Brain,
    title: "AI-Assisted Planning",
    text: "Automation that protects quality while removing busywork."
  },
  {
    icon: Utensils,
    title: "Nutrition Integration",
    text: "Fuel that moves with training — not a disconnected PDF."
  },
  {
    icon: ChartNoAxesCombined,
    title: "Progress Tracking",
    text: "Proof of adaptation through calm, readable analytics."
  },
  {
    icon: Trophy,
    title: "Personal Records",
    text: "Strength milestones stay visible as the plan evolves."
  },
  {
    icon: ShieldCheck,
    title: "Coach Reviewed Plans",
    text: "Human review when you need exclusive, elite precision."
  },
  {
    icon: RefreshCw,
    title: "Constant Evolution",
    text: "New capabilities ship continuously into the same product."
  },
  {
    icon: MonitorSmartphone,
    title: "Cross Device Experience",
    text: "One system that stays clear from phone to desktop."
  },
  {
    icon: Timer,
    title: "Fast Delivery",
    text: "From questionnaire to plan in minutes, not weeks."
  }
] as const;

const roadmap = [
  { label: "Today", title: "Platform Launch" },
  { label: "Next", title: "AI Training" },
  { label: "Next", title: "AI Nutrition" },
  { label: "Soon", title: "Recovery Intelligence" },
  { label: "Later", title: "Coach Marketplace" },
  { label: "Later", title: "Worldwide Trainers" },
  { label: "Horizon", title: "Complete Fitness Ecosystem" }
] as const;

const evolutionCards = [
  { icon: WandSparkles, title: "New AI capabilities" },
  { icon: Gauge, title: "Performance improvements" },
  { icon: Layers3, title: "New trainer tools" },
  { icon: Radar, title: "Better personalization" },
  { icon: ChartNoAxesCombined, title: "Better analytics" },
  { icon: Leaf, title: "New integrations" }
] as const;

const founders = [
  {
    initials: "E",
    name: "Edgar",
    role: "Technology",
    text: "Focused on building technology that makes professional fitness accessible."
  },
  {
    initials: "S",
    name: "Sandro",
    role: "Fitness & Strategy",
    text: "Focused on keeping every plan practical, human and worth following."
  }
] as const;

const heroBadges = [
  { icon: Brain, label: "AI" },
  { icon: Activity, label: "Training" },
  { icon: Utensils, label: "Nutrition" },
  { icon: Leaf, label: "Recovery" },
  { icon: ChartNoAxesCombined, label: "Progress" }
] as const;

interface RevealProps {
  children: ReactNode;
  className?: string;
  delay?: number;
}

const Reveal = ({ children, className, delay = 0 }: RevealProps) => {
  const reduceMotion = useReducedMotion();

  return (
    <motion.div
      className={className}
      initial={reduceMotion ? false : "hidden"}
      whileInView="visible"
      viewport={{ once: true, amount: 0.24 }}
      variants={fadeUp}
      transition={{ duration: 0.55, ease: "easeOut", delay }}
    >
      {children}
    </motion.div>
  );
};

const IconCard = ({
  icon: Icon,
  title,
  text
}: {
  icon: LucideIcon;
  title: string;
  text: string;
}) => (
  <article className={styles.iconCard}>
    <span className={styles.iconWrap}>
      <Icon aria-hidden="true" />
    </span>
    <h3>{title}</h3>
    <p>{text}</p>
  </article>
);

const TechVisual = () => (
  <div className={styles.techVisual} aria-hidden="true">
    <div className={styles.techCore}>
      <Cpu />
      <strong>RYZE OS</strong>
    </div>
    <div className={`${styles.techModule} ${styles.techA}`}>
      <Dumbbell />
      <span>Training</span>
    </div>
    <div className={`${styles.techModule} ${styles.techB}`}>
      <Utensils />
      <span>Nutrition</span>
    </div>
    <div className={`${styles.techModule} ${styles.techC}`}>
      <Brain />
      <span>AI Layer</span>
    </div>
    <div className={`${styles.techModule} ${styles.techD}`}>
      <ChartNoAxesCombined />
      <span>Progress</span>
    </div>
    <div className={`${styles.techModule} ${styles.techE}`}>
      <ShieldCheck />
      <span>Coach</span>
    </div>
  </div>
);

export const OurVisionPage = () => (
  <PageWrapper className={styles.page}>
    <section className={styles.hero}>
      <img
        className={styles.heroBackground}
        src={ourVisionBackground}
        alt=""
        aria-hidden="true"
      />

      <Container className={styles.heroLayout}>
        <Reveal className={styles.heroCopy}>
          <p className={styles.eyebrow}>The future of fitness</p>
          <h1>
            Fitness deserves better{" "}
            <span className={styles.accentPhrase}>software.</span>
          </h1>
          <p className={styles.lead}>
            Modern training should adapt to you, not force you to adapt to static plans.
          </p>

          <ul className={styles.heroBadges}>
            {heroBadges.map((badge) => {
              const Icon = badge.icon;

              return (
                <li key={badge.label} className={styles.heroBadge}>
                  <span className={styles.heroBadgeIcon}>
                    <Icon aria-hidden="true" />
                  </span>
                  <span>{badge.label}</span>
                </li>
              );
            })}
          </ul>

          <a className={styles.visionLink} href="#vision-roadmap">
            <span className={styles.visionPlay}>
              <Play aria-hidden="true" />
            </span>
            Our vision
          </a>
        </Reveal>
      </Container>
    </section>

    <section className={styles.section}>
      <Container>
        <Reveal className={styles.sectionIntro}>
          <p className={styles.eyebrow}>Our mission</p>
          <h2>Professional guidance without the friction.</h2>
          <p className={styles.sectionLead}>
            Quality training and nutrition should not be expensive or complicated. Technology lets
            us keep the standard high and the path simple.
          </p>
        </Reveal>
        <div className={styles.missionGrid}>
          {missionCards.map((card, index) => (
            <Reveal key={card.title} delay={index * 0.05}>
              <IconCard {...card} />
            </Reveal>
          ))}
        </div>
      </Container>
    </section>

    <section className={`${styles.section} ${styles.sectionWhy}`}>
      <Container>
        <Reveal className={styles.sectionIntro}>
          <p className={styles.eyebrow}>Why RYZE</p>
          <h2>A fitness operating system, not another plan PDF.</h2>
          <p className={styles.sectionLead}>
            Every surface exists to make starting easier and staying consistent measurable.
          </p>
        </Reveal>
        <div className={styles.whyGrid}>
          {whyCards.map((card, index) => (
            <Reveal key={card.title} delay={(index % 5) * 0.04}>
              <article className={styles.whyCard}>
                <span className={styles.whyIcon}>
                  <card.icon aria-hidden="true" />
                </span>
                <h3>{card.title}</h3>
                <p>{card.text}</p>
                <span className={styles.whyAccent} />
              </article>
            </Reveal>
          ))}
        </div>
      </Container>
    </section>

    <section className={styles.section} id="vision-roadmap">
      <Container className={styles.roadmapLayout}>
        <Reveal className={styles.sectionIntro}>
          <p className={styles.eyebrow}>Our vision</p>
          <h2>Where RYZE is going.</h2>
          <p className={styles.sectionLead}>
            The roadmap is the story. Each step expands the ecosystem without abandoning the core.
          </p>
        </Reveal>
        <ol className={styles.roadmap}>
          {roadmap.map((item, index) => (
            <Reveal key={item.title} delay={index * 0.06}>
              <li className={styles.roadmapItem}>
                <span className={styles.roadmapIndex}>{String(index + 1).padStart(2, "0")}</span>
                <div>
                  <p>{item.label}</p>
                  <strong>{item.title}</strong>
                </div>
              </li>
            </Reveal>
          ))}
        </ol>
      </Container>
    </section>

    <section className={`${styles.section} ${styles.sectionEvolution}`}>
      <Container>
        <Reveal className={styles.sectionIntro}>
          <p className={styles.eyebrow}>Continuous evolution</p>
          <h2>Software is never finished.</h2>
          <p className={styles.sectionLead}>
            RYZE keeps receiving new capability while the product stays one coherent system.
          </p>
        </Reveal>
        <div className={styles.evolutionGrid}>
          {evolutionCards.map((card, index) => (
            <Reveal key={card.title} delay={index * 0.05}>
              <article className={styles.evolutionCard}>
                <card.icon aria-hidden="true" />
                <h3>{card.title}</h3>
              </article>
            </Reveal>
          ))}
        </div>
      </Container>
    </section>

    <section className={styles.section}>
      <Container className={styles.techLayout}>
        <Reveal className={styles.sectionIntro}>
          <p className={styles.eyebrow}>Technology first</p>
          <h2>Built like modern software.</h2>
          <p className={styles.sectionLead}>
            Connected modules. Clear interfaces. Automation where it helps — humans where it
            matters.
          </p>
        </Reveal>
        <Reveal delay={0.1}>
          <TechVisual />
        </Reveal>
      </Container>
    </section>

    <section className={`${styles.section} ${styles.sectionPeople}`}>
      <Container>
        <Reveal className={styles.sectionIntro}>
          <p className={styles.eyebrow}>The people behind RYZE</p>
          <h2>Builders of the system.</h2>
          <p className={styles.sectionLead}>The platform stays the hero. We build it.</p>
        </Reveal>
        <div className={styles.peopleGrid}>
          {founders.map((founder, index) => (
            <Reveal key={founder.name} delay={index * 0.08}>
              <article className={styles.personCard}>
                <span className={styles.avatar}>{founder.initials}</span>
                <div>
                  <h3>{founder.name}</h3>
                  <p className={styles.personRole}>{founder.role}</p>
                  <p>{founder.text}</p>
                </div>
              </article>
            </Reveal>
          ))}
        </div>
      </Container>
    </section>
  </PageWrapper>
);
