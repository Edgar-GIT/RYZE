import { ClipboardList, CreditCard, Dumbbell, FolderOpen, Search } from "lucide-react";

import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { joinClassNames } from "@utils/class_names";

import styles from "./purchase_journey.module.css";

// The real path from the catalogue to a first logged session (see
// docs/INSTRUCTIONS/project.MD and docs/PAGES/design.MD).
const STEPS = [
  { icon: Search, title: "Choose a plan", text: "Browse training plans and Premium Level 1 packages." },
  { icon: ClipboardList, title: "Check the details", text: "See the level, duration and weekly frequency before you commit." },
  { icon: CreditCard, title: "Activate or check out", text: "Free plans activate in one step; paid plans use the secure checkout." },
  { icon: FolderOpen, title: "Open My Programs", text: "Everything you own is waiting in My Programs." },
  { icon: Dumbbell, title: "Train and log", text: "Start a session, log every set and follow your history." }
] as const;

export const PurchaseJourney = () => (
  <section
    className={joinClassNames(ui.section, ui.toneNavy, styles.section)}
    aria-labelledby="purchase-journey-heading"
  >
    <div className={joinClassNames(ui.container, styles.layout)}>
      <div className={styles.intro}>
        <p className={ui.label}>How it works</p>
        <h2 id="purchase-journey-heading" className={ui.heading}>
          From discovery to <span className={ui.hl}>your first session.</span>
        </h2>
        <div className={joinClassNames(ui.actions, styles.actions)}>
          <PublicAction to="/services/generic-program" variant="ghostOnDark">
            See all training plans
          </PublicAction>
          <PublicAction to="/our-vision" variant="quiet">
            See how RYZE works
          </PublicAction>
        </div>
      </div>

      <ol className={styles.steps}>
        {STEPS.map((step, index) => {
          const Icon = step.icon;

          return (
            <li key={step.title} className={styles.step}>
              <span className={styles.node} aria-hidden="true">
                <Icon strokeWidth={1.9} />
              </span>
              <div>
                <p className={styles.index}>Step {index + 1}</p>
                <h3 className={styles.title}>{step.title}</h3>
                <p className={styles.text}>{step.text}</p>
              </div>
            </li>
          );
        })}
      </ol>

      <p className={joinClassNames(ui.caption, styles.note)}>
        Premium Level 2 adds a detailed intake after purchase; a RYZE trainer then prepares your
        programme and nutrition before they appear in My Programs.
      </p>
    </div>
  </section>
);
