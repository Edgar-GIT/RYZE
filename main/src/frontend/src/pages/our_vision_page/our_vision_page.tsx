import { motion, useReducedMotion } from "framer-motion";
import {
  Activity,
  BarChart3,
  Boxes,
  Brain,
  ChartNoAxesCombined,
  ChevronRight,
  Cpu,
  Crosshair,
  Dumbbell,
  Gauge,
  Globe2,
  Layers3,
  Leaf,
  Link2,
  MonitorSmartphone,
  Network,
  Play,
  RefreshCw,
  Rocket,
  ShieldCheck,
  Sparkles,
  Sprout,
  Timer,
  Trophy,
  User,
  Utensils,
  Zap
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import continuousEvolutionBackground from "@resources/img/hero/conti_bg.png";
import ourVisionBackground from "@resources/img/hero/our_view.png";
import technologyBackground from "@resources/img/hero/tech_bg.png";

import styles from "./our_vision_page.module.css";

const fadeUp = {
  hidden: { opacity: 0, y: 34 },
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
  { icon: Rocket, label: "Today", title: "Platform Launch" },
  { icon: Brain, label: "Next", title: "AI Training" },
  { icon: Sprout, label: "Next", title: "AI Nutrition" },
  { icon: Activity, label: "Soon", title: "Recovery Intelligence" },
  { icon: User, label: "Later", title: "Coach Marketplace" },
  { icon: Globe2, label: "Later", title: "Worldwide Trainers" },
  { icon: Network, label: "Horizon", title: "Complete Fitness Ecosystem" }
] as const;

const evolutionCards = [
  {
    icon: Brain,
    title: "New AI capabilities",
    text: "Smarter models, deeper insights, endless potential."
  },
  {
    icon: Gauge,
    title: "Performance improvements",
    text: "Faster, lighter, stronger. Built for what's next."
  },
  {
    icon: Layers3,
    title: "New trainer tools",
    text: "Powerful tools to train smarter and scale faster."
  },
  {
    icon: User,
    title: "Better personalization",
    text: "Tailored experiences that adapt to you."
  },
  {
    icon: BarChart3,
    title: "Better analytics",
    text: "More data, clearer insights, smarter decisions."
  },
  {
    icon: Link2,
    title: "New integrations",
    text: "Seamless connections with the tools you love."
  }
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
  { icon: Dumbbell, label: "Training" },
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
      transition={{ duration: 0.62, ease: "easeOut", delay }}
    >
      {children}
    </motion.div>
  );
};

const HeroBadges = () => {
  const reduceMotion = useReducedMotion();

  return (
    <ul className={styles.heroBadges}>
      {heroBadges.map((badge, index) => {
        const Icon = badge.icon;

        return (
          <motion.li
            key={badge.label}
            className={styles.heroBadge}
            initial={reduceMotion ? false : { opacity: 0, y: 12 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.6 }}
            transition={{ duration: 0.4, ease: "easeOut", delay: 0.1 + index * 0.05 }}
          >
            <span className={styles.heroBadgeIcon}>
              <Icon aria-hidden="true" />
            </span>
            <span className={styles.heroBadgeLabel}>{badge.label}</span>
          </motion.li>
        );
      })}
    </ul>
  );
};

const techFeatures = [
  {
    icon: Boxes,
    title: "Connected Modules",
    text: "Everything works together. Nothing works alone."
  },
  {
    icon: Zap,
    title: "Clear Interfaces",
    text: "Designed for people. Built for precision."
  },
  {
    icon: Sparkles,
    title: "Smart Automation",
    text: "Less manual work. More human impact."
  }
] as const;

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

          <HeroBadges />

          <a className={styles.visionLink} href="#vision-roadmap">
            <span className={styles.visionPlay}>
              <Play aria-hidden="true" />
            </span>
            Our vision
          </a>
        </Reveal>
      </Container>
    </section>

    <section className={`${styles.section} ${styles.sectionMission}`}>
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

    <section className={`${styles.section} ${styles.sectionRoadmap}`} id="vision-roadmap">
      <Container className={styles.roadmapLayout}>
        <Reveal className={styles.roadmapIntro}>
          <p className={styles.eyebrow}>Our vision</p>
          <h2>
            Where RYZE is going<span className={styles.accentPhrase}>.</span>
          </h2>
          <p className={styles.sectionLead}>
            The roadmap is the story. Each step expands the ecosystem without abandoning the core.
          </p>
          <ul className={styles.roadmapPhases}>
            {["Today", "Next", "Soon", "Later", "Horizon"].map((phase) => (
              <li key={phase}>{phase}</li>
            ))}
          </ul>
        </Reveal>

        <ol className={styles.roadmap}>
          {roadmap.map((item, index) => {
            const Icon = item.icon;

            return (
              <li key={item.title} className={styles.roadmapEntry}>
                <span className={styles.roadmapNode} aria-hidden="true" />
                <Reveal delay={index * 0.05} className={styles.roadmapCardReveal}>
                  <article className={styles.roadmapItem}>
                    <span className={styles.roadmapIcon}>
                      <Icon aria-hidden="true" />
                    </span>
                    <span className={styles.roadmapIndex}>
                      {String(index + 1).padStart(2, "0")}
                    </span>
                    <div className={styles.roadmapCopy}>
                      <p>{item.label}</p>
                      <strong>{item.title}</strong>
                    </div>
                    <ChevronRight className={styles.roadmapChevron} aria-hidden="true" />
                  </article>
                </Reveal>
              </li>
            );
          })}
        </ol>
      </Container>
    </section>

    <section className={`${styles.section} ${styles.sectionEvolution}`}>
      <img
        className={styles.evolutionBackground}
        src={continuousEvolutionBackground}
        alt=""
        aria-hidden="true"
      />

      <Container className={styles.evolutionLayout}>
        <Reveal className={styles.evolutionIntro}>
          <p className={styles.eyebrow}>Continuous evolution</p>
          <h2>
            Software is never finished<span className={styles.accentPhrase}>.</span>
          </h2>
          <p className={styles.sectionLead}>
            RYZE keeps receiving new capability while the product stays one coherent system.
          </p>
        </Reveal>

        <div className={styles.evolutionGrid}>
          {evolutionCards.map((card, index) => {
            const Icon = card.icon;

            return (
              <Reveal key={card.title} delay={index * 0.05}>
                <article className={styles.evolutionCard}>
                  <span className={styles.evolutionIcon}>
                    <Icon aria-hidden="true" />
                  </span>
                  <div className={styles.evolutionCopy}>
                    <h3>{card.title}</h3>
                    <p>{card.text}</p>
                  </div>
                  <ChevronRight className={styles.evolutionChevron} aria-hidden="true" />
                </article>
              </Reveal>
            );
          })}
        </div>
      </Container>
    </section>

    <section className={`${styles.section} ${styles.sectionTech}`}>
      <img
        className={styles.techBackground}
        src={technologyBackground}
        alt=""
        aria-hidden="true"
      />

      <Container className={styles.techLayout}>
        <Reveal className={styles.techIntro}>
          <p className={styles.techEyebrow}>Technology first</p>
          <h2>
            Built like <span className={styles.accentPhrase}>modern</span> software.
          </h2>
          <p className={styles.sectionLead}>
            Connected modules. Clear interfaces.
            <br />
            Automation where it helps. Intelligence where it matters.
          </p>

          <ul className={styles.techFeatures}>
            {techFeatures.map((feature) => {
              const Icon = feature.icon;

              return (
                <li key={feature.title} className={styles.techFeature}>
                  <span className={styles.techFeatureIcon}>
                    <Icon aria-hidden="true" />
                  </span>
                  <div>
                    <strong>{feature.title}</strong>
                    <p>{feature.text}</p>
                  </div>
                </li>
              );
            })}
          </ul>
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
