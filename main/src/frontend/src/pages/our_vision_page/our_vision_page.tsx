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
  Globe2,
  Heart,
  Link2,
  Network,
  Rocket,
  Sprout,
  User,
  UserPlus,
  Users,
  Zap
} from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { CONTACT_WHATSAPP_URL } from "@/constants/contact";
import continuousEvolutionBackground from "@resources/img/hero/conti_bg.png";
import ryzeIconBanner from "@resources/img/logo/ryze_icon2.png";
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

const roadmap = [
  { icon: Rocket, label: "Today", title: "Platform Launch" },
  { icon: Brain, label: "Next", title: "AI Training" },
  { icon: Sprout, label: "Next", title: "AI Nutrition" },
  { icon: Activity, label: "Soon", title: "Recovery Intelligence" },
  { icon: User, label: "Later", title: "Coach Marketplace" },
  { icon: Globe2, label: "Later", title: "Worldwide Trainers" },
  { icon: Network, label: "Horizon", title: "Complete Fitness Ecosystem" }
] as const;

const leftEvolution = [
  {
    icon: Brain,
    title: "New AI capabilities",
    text: "Smarter models, deeper insights, endless potential."
  },
  {
    icon: Zap,
    title: "Performance improvements",
    text: "Faster, lighter, stronger. Built for what's next."
  },
  {
    icon: User,
    title: "Better personalization",
    text: "Tailored experiences that adapt to you."
  }
] as const;

const rightEvolution = [
  {
    icon: BarChart3,
    title: "Better analytics",
    text: "More data, clearer insights, smarter decisions."
  },
  {
    icon: Dumbbell,
    title: "New trainer tools",
    text: "Powerful tools to train smarter and scale faster."
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

const mulberry32 = (seed: number) => {
  let state = seed;

  return () => {
    state |= 0;
    state = (state + 0x6d2b79f5) | 0;
    let t = Math.imul(state ^ (state >>> 15), 1 | state);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
};

interface StarPoint {
  top: number;
  left: number;
  size: number;
  delay: number;
  duration: number;
  glow: string;
}

const starRandom = mulberry32(20260920);

const starPoints: StarPoint[] = Array.from({ length: 150 }, () => {
  const size = [1, 1.4, 1.8, 2.4][Math.floor(starRandom() * 4)];

  return {
    top: +(starRandom() * 100).toFixed(2),
    left: +(starRandom() * 100).toFixed(2),
    size,
    delay: +(starRandom() * 3).toFixed(2),
    duration: 4.5 + Math.floor(starRandom() * 4),
    glow:
      size >= 1.8
        ? "0 0 8px rgba(24, 216, 255, 0.85)"
        : "0 0 4px rgba(24, 216, 255, 0.45)"
  };
});

const StarField = () => (
  <div className={styles.starField} aria-hidden="true">
    {starPoints.map((star, index) => (
      <span
        key={index}
        style={{
          top: `${star.top}%`,
          left: `${star.left}%`,
          width: `${star.size}px`,
          height: `${star.size}px`,
          boxShadow: star.glow,
          animationDelay: `${star.delay}s`,
          animationDuration: `${star.duration}s`
        }}
      />
    ))}
  </div>
);

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
        </Reveal>
      </Container>
    </section>

    <section className={`${styles.section} ${styles.sectionMission}`}>
      <StarField />

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

    <section className={`${styles.section} ${styles.sectionRoadmap}`} id="vision-roadmap">
      <StarField />

      <Container className={styles.roadmapLayout}>
        <Reveal className={styles.roadmapIntro}>
          <p className={styles.eyebrow}>Our vision</p>
          <h2>
            Where <span className={styles.accentPhrase}>RYZE</span> is going.
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
          <p className={styles.evolutionEyebrow}>
            <span className={styles.evolutionEyebrowLine} aria-hidden="true" />
            Continuous evolution
          </p>
          <h2>
            <span className={styles.evolutionLine}>Software is</span>
            <span className={`${styles.evolutionLine} ${styles.accentPhrase}`}>
              never finished.
            </span>
          </h2>
          <p className={styles.evolutionLead}>
            RYZE keeps receiving new capabilities while the product stays one coherent system.
          </p>
          <a className={styles.evolutionCta} href="#vision-roadmap">
            See what's next
            <ArrowRight aria-hidden="true" />
          </a>
        </Reveal>

        <div className={styles.evolutionSystem}>
          <div className={styles.evolutionSide} data-side="left">
            {leftEvolution.map((card, index) => {
              const Icon = card.icon;

              return (
                <Reveal key={card.title} delay={index * 0.06}>
                  <article className={styles.evolutionItem}>
                    <span className={styles.evolutionIcon}>
                      <Icon aria-hidden="true" strokeWidth={1.6} />
                    </span>
                    <div className={styles.evolutionCopy}>
                      <h3>{card.title}</h3>
                      <p>{card.text}</p>
                    </div>
                  </article>
                </Reveal>
              );
            })}
          </div>

          <div className={styles.evolutionCenter} aria-hidden="true" />

          <div className={styles.evolutionSide} data-side="right">
            {rightEvolution.map((card, index) => {
              const Icon = card.icon;

              return (
                <Reveal key={card.title} delay={index * 0.06}>
                  <article className={styles.evolutionItem}>
                    <span className={styles.evolutionIcon}>
                      <Icon aria-hidden="true" strokeWidth={1.6} />
                    </span>
                    <div className={styles.evolutionCopy}>
                      <h3>{card.title}</h3>
                      <p>{card.text}</p>
                    </div>
                  </article>
                </Reveal>
              );
            })}
          </div>
        </div>

        <Reveal className={styles.evolutionBanner}>
          <img
            className={styles.evolutionBannerMark}
            src={ryzeIconBanner}
            alt=""
            aria-hidden="true"
          />
          <div className={styles.evolutionBannerCopy}>
            <h3>
              The Future of Fitness is{" "}
              <span className={styles.accentPhrase}>Adaptive</span>
            </h3>
            <p>We're building more than a platform - We're building the Future.</p>
          </div>
          <a className={styles.evolutionBannerCta} href="#vision-roadmap">
            Join the Journey
          </a>
        </Reveal>
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
              href={CONTACT_WHATSAPP_URL}
              variant="secondary"
              size="medium"
              icon={<ArrowRight aria-hidden="true" />}
            >
              JOIN RYZE
            </Button>
          </Reveal>

          <Reveal className={styles.joinFeaturesPanel} delay={0.12}>
            <ol className={styles.joinFeatures}>
              {joinFeatures.map((feature) => {
                const Icon = feature.icon;

                return (
                  <li key={feature.title} className={styles.joinFeature}>
                    <span className={styles.joinFeatureIcon}>
                      <Icon aria-hidden="true" />
                    </span>
                    <span className={styles.joinFeatureCopy}>
                      <strong>{feature.title}</strong>
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
