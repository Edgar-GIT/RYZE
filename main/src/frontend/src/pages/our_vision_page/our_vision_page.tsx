import { motion, useReducedMotion } from "framer-motion";
import {
  Activity,
  ArrowRight,
  BarChart3,
  Brain,
  ChartNoAxesCombined,
  ChevronRight,
  Crosshair,
  Dumbbell,
  Gauge,
  Globe2,
  Heart,
  Hexagon,
  Layers3,
  Leaf,
  Link2,
  MonitorSmartphone,
  Network,
  Play,
  Rocket,
  ShieldCheck,
  Sprout,
  User,
  UserPlus,
  Users,
  Utensils,
  Zap
} from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import continuousEvolutionBackground from "@resources/img/hero/conti_bg.png";
import fitnessOsBackground from "@resources/img/hero/evo_man_bg.png";
import futureBackground from "@resources/img/hero/future_bg.png";
import howItWorksBackground from "@resources/img/hero/how_bg.png";
import joinBackground from "@resources/img/hero/join_bg.png";
import ourVisionBackground from "@resources/img/hero/our_view.png";

import styles from "./our_vision_page.module.css";

const fadeUp = {
  hidden: { opacity: 0, y: 34 },
  visible: { opacity: 1, y: 0 }
};

const howItWorksSteps = [
  {
    number: "01",
    icon: UserPlus,
    title: "Sign up",
    text: "Create your account in seconds."
  },
  {
    number: "02",
    icon: Crosshair,
    title: "Tell us your goals",
    text: "Share your goals and preferences."
  },
  {
    number: "03",
    icon: Brain,
    title: "AI builds your program",
    text: "We create a training and nutrition program just for you."
  },
  {
    number: "04",
    icon: BarChart3,
    title: "Track progress",
    text: "Monitor your performance in real time."
  },
  {
    number: "05",
    icon: Rocket,
    title: "Get better every week",
    text: "Smart adjustments keep you moving forward."
  }
] as const;

const whyCards = [
  {
    icon: Hexagon,
    title: "Adaptive Programs",
    text: "Programs that adapt in real-time based on your progress, recovery and goals."
  },
  {
    icon: Brain,
    title: "AI-Powered Coaching",
    text: "Advanced AI that analyzes your data and gives you smarter recommendations."
  },
  {
    icon: Utensils,
    title: "Smart Nutrition",
    text: "Nutrition programs that fit your lifestyle and evolve with your training."
  },
  {
    icon: ChartNoAxesCombined,
    title: "Progress Intelligence",
    text: "Track what matters with powerful analytics and actionable insights."
  },
  {
    icon: MonitorSmartphone,
    title: "All Your Devices",
    text: "Seamless sync across phone, watch, desktop and more."
  },
  {
    icon: ShieldCheck,
    title: "Built-In Accountability",
    text: "Stay consistent with built-in reminders, check-ins and streak tracking."
  },
  {
    icon: Users,
    title: "Coach & Community",
    text: "Connect with elite coaches and a community that pushes you forward."
  },
  {
    icon: Zap,
    title: "Results, Faster",
    text: "Everything working together so you get results in less time."
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

const joinFeatures = [
  {
    icon: Users,
    title: "Strong community",
    text: "Together we go further."
  },
  {
    icon: Dumbbell,
    title: "Smarter training",
    text: "Train with purpose."
  },
  {
    icon: Heart,
    title: "Built with passion",
    text: "For people who never stop."
  },
  {
    icon: ChartNoAxesCombined,
    title: "Continuous evolution",
    text: "Always improving."
  },
  {
    icon: Zap,
    title: "Your journey",
    text: "Your goals, our mission."
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

const HowWorksPanel = () => {
  const reduceMotion = useReducedMotion();

  return (
    <motion.div
      className={styles.howWorks}
      initial={reduceMotion ? false : { opacity: 0, y: 28 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, amount: 0.35 }}
      transition={{ duration: 0.65, ease: "easeOut", delay: 0.1 }}
    >
      <motion.div
        className={styles.howWorksTitle}
        initial={reduceMotion ? false : { opacity: 0 }}
        whileInView={{ opacity: 1 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5, ease: "easeOut", delay: 0.22 }}
      >
        <span className={styles.howWorksLine} aria-hidden="true" />
        <h3>How RYZE works</h3>
        <span className={styles.howWorksLine} aria-hidden="true" />
      </motion.div>

      <ol className={styles.howWorksSteps}>
        {howItWorksSteps.map((step, index) => {
          const Icon = step.icon;

          return (
            <motion.li
              key={step.number}
              className={styles.howWorksStep}
              initial={reduceMotion ? false : { opacity: 0, y: 18, scale: 0.92 }}
              whileInView={{ opacity: 1, y: 0, scale: 1 }}
              viewport={{ once: true, amount: 0.4 }}
              transition={{
                duration: 0.48,
                ease: "easeOut",
                delay: 0.28 + index * 0.09
              }}
            >
              <span className={styles.howWorksNumber}>{step.number}</span>
              <span className={styles.howWorksIcon}>
                <Icon aria-hidden="true" strokeWidth={1.6} />
              </span>
              <h4>{step.title}</h4>
              <p>{step.text}</p>
            </motion.li>
          );
        })}
      </ol>
    </motion.div>
  );
};

const WhyFeatureGrid = () => {
  const reduceMotion = useReducedMotion();

  return (
    <div className={styles.whyGrid}>
      {whyCards.map((card, index) => {
        const Icon = card.icon;

        return (
          <motion.article
            key={card.title}
            className={styles.whyCard}
            initial={reduceMotion ? false : { opacity: 0, y: 16 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.35 }}
            transition={{ duration: 0.45, ease: "easeOut", delay: 0.08 + index * 0.05 }}
          >
            <span className={styles.whyIcon}>
              <Icon aria-hidden="true" strokeWidth={1.6} />
            </span>
            <div className={styles.whyCopy}>
              <h3>{card.title}</h3>
              <p>{card.text}</p>
            </div>
          </motion.article>
        );
      })}
    </div>
  );
};

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
            Modern training should adapt to you, not force you to adapt to static programs.
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
      <img
        className={styles.missionBackground}
        src={howItWorksBackground}
        alt=""
        aria-hidden="true"
      />

      <Container className={styles.missionLayout}>
        <Reveal className={styles.missionIntro}>
          <p className={styles.missionEyebrow}>Get started</p>
          <h2>
            Start your journey in <span className={styles.accentPhrase}>minutes.</span>
            <br />
            Transform for life.
          </h2>
          <p className={styles.sectionLead}>
            RYZE makes it easy to start. Set your goals, follow your program and let technology handle
            the rest.
          </p>
        </Reveal>

        <HowWorksPanel />
      </Container>
    </section>

    <section className={`${styles.section} ${styles.sectionWhy}`}>
      <img
        className={styles.whyBackground}
        src={fitnessOsBackground}
        alt=""
        aria-hidden="true"
      />

      <Container className={styles.whyLayout}>
        <Reveal className={styles.whyIntro}>
          <p className={styles.eyebrow}>Built different. For real results.</p>
          <h2>
            A fitness <span className={styles.accentPhrase}>operating system</span>, not another
            program.
          </h2>
          <p className={styles.sectionLead}>
            Training, nutrition, recovery and progress in one connected system — built to adapt with
            you, not force you into a static template.
          </p>
          <a className={styles.whyCta} href="#vision-roadmap">
            Explore RYZE
            <ChevronRight aria-hidden="true" />
          </a>
        </Reveal>

        <Reveal className={styles.whyPanel} delay={0.12}>
          <WhyFeatureGrid />
        </Reveal>
      </Container>
    </section>

    <section className={`${styles.section} ${styles.sectionRoadmap}`} id="vision-roadmap">
      <img
        className={styles.roadmapBackground}
        src={futureBackground}
        alt=""
        aria-hidden="true"
      />

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

    <section className={`${styles.section} ${styles.sectionJoin}`}>
      <img
        className={styles.joinBackground}
        src={joinBackground}
        alt=""
        aria-hidden="true"
      />

      <Container className={styles.joinLayout}>
        <div className={styles.joinContent}>
          <Reveal className={styles.joinIntro}>
            <p className={styles.joinEyebrow}>The journey continues</p>
            <h2 className={styles.joinTitle}>
              Grow together
              <br />
              with RYZE, <span className={styles.joinAccent}>join us now.</span>
            </h2>
            <p className={styles.joinLead}>
              We're building the future of fitness, education and technology.
              <br />
              Be part of a community that evolves every day.
            </p>

            <Button
              className={styles.joinCta}
              to="/register"
              variant="secondary"
              size="medium"
              icon={<ArrowRight aria-hidden="true" />}
            >
              JOIN RYZE
            </Button>
          </Reveal>

          <Reveal className={styles.joinFeaturesPanel} delay={0.12}>
            <ol className={styles.joinFeatures}>
              {joinFeatures.map((feature, index) => {
                const Icon = feature.icon;

                return (
                  <li key={feature.title} className={styles.joinFeature}>
                    <span className={styles.joinFeatureIcon}>
                      <Icon aria-hidden="true" />
                    </span>
                    <span className={styles.joinFeatureCopy}>
                      <span className={styles.joinFeatureHeader}>
                        <span className={styles.joinFeatureNumber}>
                          {String(index + 1).padStart(2, "0")}
                        </span>
                        <strong>{feature.title}</strong>
                      </span>
                      <p>{feature.text}</p>
                    </span>
                  </li>
                );
              })}
            </ol>
          </Reveal>

          <div className={styles.joinFooter}>
            <p className={styles.joinStatement}>
              <span className={styles.joinStatementLine} aria-hidden="true" />
              RYZE is more than a platform.
              <span className={styles.joinStatementLine} aria-hidden="true" />
            </p>
            <p className={styles.joinMovement}>It's a movement.</p>
          </div>
        </div>
      </Container>
    </section>
  </PageWrapper>
);
