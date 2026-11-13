import {
  Activity,
  ArrowDown,
  ArrowRight,
  BarChart3,
  Brain,
  ChartNoAxesCombined,
  Check,
  Dumbbell,
  Globe2,
  Heart,
  Link2,
  Network,
  Rocket,
  Sprout,
  User,
  Users,
  X,
  Zap
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { Reveal } from "@/components/reveal/reveal";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { BrandStop, PublicAction, Waves } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { BRAND_ASSETS } from "@/constants/brand_assets";
import { CONTACT_WHATSAPP_URL } from "@/constants/contact";
import { useGettext } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";
import ryzeIconBanner from "@resources/img/logo/ryze_icon2.png";

import styles from "./our_vision_page.module.css";
import { OUR_VISION_PT } from "./our_vision_page.pt";
import { StandoutVisual } from "./standout_visual";

interface IconItem {
  icon: LucideIcon;
  title: string;
  text: string;
}

const pageSections = [
  { href: "#why-ryze", label: "Why RYZE is different" },
  { href: "#how-it-works", label: "How RYZE works" },
  { href: "#vision-roadmap", label: "Where RYZE is going" },
  { href: "#vision-evolution", label: "Continuous evolution" }
] as const;

// How most fitness apps work, against what RYZE does today (and next).
const TYPICAL = [
  "One template for everyone",
  "Training or nutrition, rarely both",
  "A plan you follow on your own",
  "No record of what you actually did"
] as const;

const RYZE_WAY: Array<{ text: string; next?: boolean }> = [
  { text: "Programs filtered by goal, level, duration and weekly frequency" },
  { text: "Training and nutrition together in Premium plans" },
  { text: "Premium Level 2 plans prepared for you by a RYZE trainer" },
  { text: "Every session logged, with your history and progress" },
  { text: "AI-generated programs", next: true }
];

// Existing Our Vision copy, rendered as numbered steps.
const howItWorksSteps: Array<{ title: string; text: string }> = [
  { title: "Sign up", text: "Create your account in seconds." },
  { title: "Tell us your goals", text: "Share your goals and preferences." },
  { title: "AI builds your program", text: "We create a training and nutrition program just for you." },
  { title: "Track progress", text: "Monitor your performance in real time." },
  { title: "Get better every week", text: "Smart adjustments keep you moving forward." }
];

const roadmapPhases: Array<{ phase: string; milestones: Array<{ icon: LucideIcon; title: string }> }> = [
  { phase: "Today", milestones: [{ icon: Rocket, title: "Platform Launch" }] },
  { phase: "Next", milestones: [{ icon: Brain, title: "AI Training" }, { icon: Sprout, title: "AI Nutrition" }] },
  { phase: "Soon", milestones: [{ icon: Activity, title: "Recovery Intelligence" }] },
  { phase: "Later", milestones: [{ icon: User, title: "Coach Marketplace" }, { icon: Globe2, title: "Worldwide Trainers" }] },
  { phase: "Horizon", milestones: [{ icon: Network, title: "Complete Fitness Ecosystem" }] }
];

const evolutionItems: IconItem[] = [
  { icon: Brain, title: "New AI capabilities", text: "Smarter models, deeper insights, endless potential." },
  { icon: Zap, title: "Performance improvements", text: "Faster, lighter, stronger. Built for what's next." },
  { icon: User, title: "Better personalization", text: "Tailored experiences that adapt to you." },
  { icon: BarChart3, title: "Better analytics", text: "More data, clearer insights, smarter decisions." },
  { icon: Dumbbell, title: "New trainer tools", text: "Powerful tools to train smarter and scale faster." },
  { icon: Link2, title: "New integrations", text: "Seamless connections with the tools you love." }
];

const joinFeatures: IconItem[] = [
  { icon: Users, title: "Strong community", text: "Together we go further." },
  { icon: Dumbbell, title: "Smarter training", text: "Train with purpose." },
  { icon: Heart, title: "Built with passion", text: "For people who never stop." },
  { icon: ChartNoAxesCombined, title: "Continuous evolution", text: "Always improving." },
  { icon: Zap, title: "Your journey", text: "Your goals, our mission." }
];

const IconTile = ({ icon: Icon }: { icon: LucideIcon }) => (
  <span className={styles.iconTile} aria-hidden="true">
    <Icon strokeWidth={1.9} />
  </span>
);

export const OurVisionPage = () => {
  const tr = useGettext(OUR_VISION_PT);
  return (
  <PageWrapper className={ui.page}>
    <section className={joinClassNames(ui.toneNavy, styles.hero)} aria-labelledby="vision-hero-heading">
      <div className={joinClassNames(ui.container, styles.heroLayout)}>
        <div className={styles.heroCopy}>
          <p className={ui.label}>{tr("Our vision")}</p>
          <h1 id="vision-hero-heading" className={ui.display}>
            {tr("Thousands of fitness apps. All the same.")} <span className={ui.hl}>{tr("RYZE isn't.")}</span>
          </h1>
          <p className={ui.lead}>
            {tr(
              "Most fitness platforms sell the same static plans and stop there. RYZE brings training, nutrition and real trainers into one platform — and keeps evolving, with AI-generated programs next on our roadmap."
            )}
          </p>
          <nav aria-label={tr("On this page")}>
            <ul className={styles.pageNav}>
              {pageSections.map((section) => (
                <li key={section.href}>
                  <a className={joinClassNames(styles.pageNavLink, ui.focusRing)} href={section.href}>
                    {tr(section.label)}
                    <ArrowDown aria-hidden="true" />
                  </a>
                </li>
              ))}
            </ul>
          </nav>
        </div>
        <StandoutVisual className={styles.heroVisual} />
      </div>
    </section>

    <section
      id="why-ryze"
      className={joinClassNames(ui.section, ui.toneNavy, ui.anchorTarget)}
      aria-labelledby="why-ryze-heading"
    >
      <div className={joinClassNames(ui.container, styles.stack)}>
        <div className={styles.intro}>
          <p className={ui.label}>{tr("Why RYZE")}</p>
          <h2 id="why-ryze-heading" className={ui.heading}>
            {tr("Others stop at a plan.")} <span className={ui.hl}>{tr("RYZE keeps going.")}</span>
          </h2>
        </div>

        <div className={styles.compare}>
          <div className={joinClassNames(styles.compareColumn, styles.compareOthers)}>
            <h3>{tr("Typical fitness apps")}</h3>
            <ul>
              {TYPICAL.map((item) => (
                <li key={item}>
                  <X aria-hidden="true" strokeWidth={2.2} />
                  {tr(item)}
                </li>
              ))}
            </ul>
          </div>
          <div className={joinClassNames(ui.surface, styles.compareColumn, styles.compareRyze)}>
            <h3>
              <img src={BRAND_ASSETS.icon} alt="" width="600" height="658" />
              RYZE
            </h3>
            <ul>
              {RYZE_WAY.map((item) => (
                <li key={item.text}>
                  <Check aria-hidden="true" strokeWidth={2.6} />
                  <span>
                    {tr(item.text)}
                    {item.next ? <em className={styles.nextTag}>{tr("Next")}</em> : null}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </section>

    <section
      id="how-it-works"
      className={joinClassNames(ui.section, ui.toneNavy, ui.anchorTarget)}
      aria-labelledby="how-it-works-heading"
    >
      <div className={joinClassNames(ui.container, styles.stack)}>
        <div className={styles.intro}>
          <p className={ui.label}>{tr("Get started")}</p>
          <h2 id="how-it-works-heading" className={ui.heading}>
            {tr("Start your journey in")} <span className={ui.hl}>{tr("minutes.")}</span> {tr("Transform for life")}
            <BrandStop />
          </h2>
          <p className={ui.lead}>
            {tr("RYZE makes it easy to start. Set your goals, follow your program and let technology handle the rest.")}
          </p>
        </div>

        <ol className={styles.steps} aria-label={tr("How RYZE works")}>
          {howItWorksSteps.map((step, index) => (
            <li key={step.title}>
              <Reveal className={joinClassNames(ui.surface, styles.step)} delay={index * 0.06} y={20}>
                <span className={styles.stepNumber} aria-hidden="true">
                  {String(index + 1).padStart(2, "0")}
                </span>
                <h3>{tr(step.title)}</h3>
                <p>{tr(step.text)}</p>
              </Reveal>
            </li>
          ))}
        </ol>
      </div>
    </section>

    <section
      id="vision-roadmap"
      className={joinClassNames(ui.section, ui.toneNavy, ui.anchorTarget)}
      aria-labelledby="roadmap-heading"
    >
      <div className={joinClassNames(ui.container, styles.stack)}>
        <div className={styles.intro}>
          <p className={ui.label}>{tr("Roadmap")}</p>
          <h2 id="roadmap-heading" className={ui.heading}>
            {tr("Where")} <span className={ui.hl}>RYZE</span> {tr("is going.")}
          </h2>
          <p className={ui.lead}>
            {tr("The roadmap is the story. Each step expands the ecosystem without abandoning the core.")}
          </p>
        </div>

        <ol className={styles.roadmap} aria-label={tr("RYZE roadmap")}>
          {roadmapPhases.map((phase, index) => (
            <li key={phase.phase} className={joinClassNames(styles.phase, index === 0 && styles.phaseCurrent)}>
              <span className={styles.phaseNode} aria-hidden="true" />
              <h3 className={styles.phaseName}>{tr(phase.phase)}</h3>
              <ul className={styles.milestones}>
                {phase.milestones.map((milestone) => {
                  const Icon = milestone.icon;

                  return (
                    <li key={milestone.title} className={styles.milestone}>
                      <span className={styles.milestoneIcon} aria-hidden="true">
                        <Icon strokeWidth={1.9} />
                      </span>
                      {tr(milestone.title)}
                    </li>
                  );
                })}
              </ul>
            </li>
          ))}
        </ol>
      </div>
    </section>

    <section
      id="vision-evolution"
      className={joinClassNames(ui.section, ui.toneNavy, ui.anchorTarget)}
      aria-labelledby="evolution-heading"
    >
      <div className={joinClassNames(ui.container, styles.stack)}>
        <div className={styles.intro}>
          <p className={ui.label}>{tr("Continuous evolution")}</p>
          <h2 id="evolution-heading" className={ui.heading}>
            {tr("Software is")} <span className={ui.hl}>{tr("never finished.")}</span>
          </h2>
          <div className={styles.introAside}>
            <p className={ui.lead}>
              {tr("RYZE keeps receiving new capabilities while the product stays one coherent system.")}
            </p>
            <PublicAction href="#vision-roadmap" variant="quiet">
              {tr("See what's next")}
            </PublicAction>
          </div>
        </div>

        <ul className={styles.featureGrid}>
          {evolutionItems.map((item, index) => (
            <li key={item.title}>
              <Reveal className={joinClassNames(ui.surface, styles.feature)} delay={(index % 3) * 0.06} y={20}>
                <IconTile icon={item.icon} />
                <h3>{tr(item.title)}</h3>
                <p>{tr(item.text)}</p>
              </Reveal>
            </li>
          ))}
        </ul>

        <div className={joinClassNames(ui.toneLight, ui.lightPanel, styles.banner)}>
          <Waves />
          <img
            className={styles.bannerMark}
            src={ryzeIconBanner}
            alt=""
            width="96"
            height="100"
            loading="lazy"
            decoding="async"
          />
          <div className={styles.bannerCopy}>
            <h3>
              {tr("The future of fitness is")} <span className={ui.hl}>{tr("adaptive.")}</span>
            </h3>
            <p>{tr("We're building more than a platform — we're building the future.")}</p>
          </div>
          <PublicAction href="#vision-roadmap" variant="ink" icon={<ArrowRight />}>
            {tr("Join the journey")}
          </PublicAction>
        </div>
      </div>
    </section>

    <section className={joinClassNames(ui.section, ui.toneNavy)} aria-labelledby="join-heading">
      <div className={joinClassNames(ui.container, styles.stack)}>
        <div className={styles.intro}>
          <p className={ui.label}>{tr("The journey continues")}</p>
          <h2 id="join-heading" className={ui.heading}>
            {tr("Grow together with RYZE,")} <span className={ui.hl}>{tr("join us now.")}</span>
          </h2>
          <div className={styles.introAside}>
            <p className={ui.lead}>
              {tr("We're building the future of fitness, education and technology. Be part of a community that evolves every day.")}
            </p>
            <div className={styles.joinAction}>
              <PublicAction href={CONTACT_WHATSAPP_URL} icon={<ArrowRight />}>
                {tr("Join RYZE")}
              </PublicAction>
              <p className={ui.caption}>{tr("Opens the RYZE community on WhatsApp.")}</p>
            </div>
          </div>
        </div>

        <ul className={styles.joinList}>
          {joinFeatures.map((feature) => (
            <li key={feature.title} className={styles.joinItem}>
              <IconTile icon={feature.icon} />
              <div>
                <h3>{tr(feature.title)}</h3>
                <p>{tr(feature.text)}</p>
              </div>
            </li>
          ))}
        </ul>

        <p className={styles.statement}>
          {tr("RYZE is more than a platform.")} <span className={ui.hl}>{tr("It's a movement.")}</span>
        </p>
      </div>
    </section>
  </PageWrapper>
  );
};
