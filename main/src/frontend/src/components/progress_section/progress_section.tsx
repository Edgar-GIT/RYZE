import { motion, useReducedMotion } from "framer-motion";
import {
  Activity,
  CalendarDays,
  ChartNoAxesCombined,
  ChevronDown,
  Crosshair,
  Dumbbell,
  HeartPulse,
  Scale,
  Target,
  TrendingUp,
  Trophy
} from "lucide-react";

import { Container } from "@/components/container/container";
import { Reveal } from "@/components/reveal/reveal";

import styles from "./progress_section.module.css";

const featureItems = [
  {
    icon: ChartNoAxesCombined,
    title: "Strength trends",
    description: "See load climb week after week."
  },
  {
    icon: Scale,
    title: "Body metrics",
    description: "Weight and composition in one place."
  },
  {
    icon: Trophy,
    title: "Personal records",
    description: "Every PR logged automatically."
  },
  {
    icon: Target,
    title: "Real consistency",
    description: "Proof you showed up — not vibes."
  }
] as const;

const summaryStats = [
  { icon: Trophy, label: "New PRs", value: "3", note: "This period" },
  { icon: Crosshair, label: "Consistency", value: "91%", note: "Excellent" },
  { icon: TrendingUp, label: "Trend", value: "Upward", note: "Keep it up" }
] as const;

const metricCards = [
  { icon: Dumbbell, label: "Bench PR", value: "92.5 kg", note: "+7.5 kg" },
  { icon: Scale, label: "Body weight", value: "74.2 kg", note: "-2.1 kg" },
  { icon: CalendarDays, label: "Sessions", value: "48", note: "+6" },
  { icon: HeartPulse, label: "Recovery", value: "91%", note: "Excellent" }
] as const;

const chartPoints = [
  { x: 4, y: 78 },
  { x: 22, y: 62 },
  { x: 40, y: 54 },
  { x: 58, y: 42 },
  { x: 76, y: 30 },
  { x: 94, y: 18 }
] as const;

const floatTransition = {
  duration: 5.8,
  repeat: Infinity,
  repeatType: "mirror" as const,
  ease: "easeInOut" as const
};

export const ProgressSection = () => {
  const reduceMotion = useReducedMotion();
  const linePoints = chartPoints.map((point) => `${point.x},${point.y}`).join(" ");
  const areaPoints = `4,88 ${linePoints} 94,88`;

  return (
    <section className={styles.section} aria-labelledby="progress-heading">
      <div className={styles.atmosphere} aria-hidden="true">
        <div className={styles.nebula} />
        <div className={styles.grid} />
        <div className={styles.stars}>
          {Array.from({ length: 16 }, (_, index) => (
            <span key={index} style={{ ["--i" as string]: index }} />
          ))}
        </div>
      </div>

      <Container className={styles.layout}>
        <Reveal className={styles.copy}>
          <p className={styles.eyebrow}>Progress</p>
          <h2 id="progress-heading">
            Proof you can
            <br />
            actually <span className={styles.accentPhrase}>see.</span>
          </h2>
          <p className={styles.description}>
            Strength trends, body metrics and PRs in one calm analytics surface — not motivation
            posters.
          </p>

          <ul className={styles.features}>
            {featureItems.map((item, index) => {
              const Icon = item.icon;

              return (
                <Reveal
                  key={item.title}
                  className={styles.feature}
                  delay={0.1 + index * 0.07}
                  y={18}
                >
                  <span className={styles.featureIcon}>
                    <Icon aria-hidden="true" />
                  </span>
                  <span className={styles.featureCopy}>
                    <strong>{item.title}</strong>
                    <span>{item.description}</span>
                  </span>
                </Reveal>
              );
            })}
          </ul>
        </Reveal>

        <Reveal className={styles.visual} delay={0.14} y={32}>
          <motion.article
            className={styles.card}
            animate={reduceMotion ? undefined : { y: [0, -8, 0] }}
            transition={floatTransition}
          >
            <header className={styles.cardHeader}>
              <div className={styles.cardTitle}>
                <Activity aria-hidden="true" />
                <span>Progress · 12 weeks</span>
              </div>
              <button type="button" className={styles.rangePill}>
                12 weeks
                <ChevronDown aria-hidden="true" />
              </button>
            </header>

            <div className={styles.chartBlock}>
              <div className={styles.chartMeta}>
                <span>Strength index</span>
                <strong>
                  +26%
                  <em>↑</em>
                </strong>
                <p>vs last 12 weeks</p>
              </div>

              <div className={styles.chartWrap}>
                <svg className={styles.chart} viewBox="0 0 100 100" preserveAspectRatio="none">
                  {[20, 40, 60, 80].map((y) => (
                    <line key={y} x1="0" y1={y} x2="100" y2={y} className={styles.gridLine} />
                  ))}
                  <polygon className={styles.trendArea} points={areaPoints} />
                  <polyline className={styles.trendLine} points={linePoints} fill="none" />
                  {chartPoints.map((point) => (
                    <circle
                      key={`${point.x}-${point.y}`}
                      cx={point.x}
                      cy={point.y}
                      r="1.6"
                      className={styles.trendDot}
                    />
                  ))}
                </svg>

                <div className={styles.yAxis}>
                  <span>+30%</span>
                  <span>+20%</span>
                  <span>+10%</span>
                  <span>0%</span>
                  <span>-10%</span>
                </div>

                <div className={styles.xAxis}>
                  {["Week 1", "Week 3", "Week 6", "Week 9", "Week 12"].map((label) => (
                    <span key={label}>{label}</span>
                  ))}
                </div>
              </div>
            </div>

            <div className={styles.summaryRow}>
              {summaryStats.map((stat) => {
                const Icon = stat.icon;

                return (
                  <div key={stat.label} className={styles.summaryItem}>
                    <span className={styles.summaryIcon}>
                      <Icon aria-hidden="true" />
                    </span>
                    <div className={styles.summaryCopy}>
                      <span className={styles.summaryLabel}>{stat.label}</span>
                      <strong className={styles.summaryValue}>{stat.value}</strong>
                      <em className={styles.summaryNote}>{stat.note}</em>
                    </div>
                  </div>
                );
              })}
            </div>

            <div className={styles.metricGrid}>
              {metricCards.map((metric) => {
                const Icon = metric.icon;

                return (
                  <div key={metric.label} className={styles.metricCard}>
                    <Icon aria-hidden="true" />
                    <span>{metric.label}</span>
                    <strong>{metric.value}</strong>
                    <em>{metric.note}</em>
                  </div>
                );
              })}
            </div>
          </motion.article>
        </Reveal>
      </Container>
    </section>
  );
};
