import { ClipboardList, CreditCard, Dumbbell, FolderOpen, Search } from "lucide-react";

import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { type Copy, useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import styles from "./purchase_journey.module.css";

// The real path from the catalogue to a first logged session (see
// docs/INSTRUCTIONS/project.MD and docs/PAGES/design.MD).
const STEP_ICONS = [Search, ClipboardList, CreditCard, FolderOpen, Dumbbell] as const;

interface JourneyText {
  label: string;
  heading: string;
  headingHighlight: string;
  allPlans: string;
  howItWorks: string;
  step: (index: number) => string;
  steps: { title: string; text: string }[];
  note: string;
}

const COPY: Copy<JourneyText> = {
  en: {
    label: "How it works",
    heading: "From discovery to ",
    headingHighlight: "your first session.",
    allPlans: "See all training plans",
    howItWorks: "See how RYZE works",
    step: (index) => `Step ${index}`,
    steps: [
      { title: "Choose a plan", text: "Browse training plans and Premium Level 1 packages." },
      { title: "Check the details", text: "See the level, duration and weekly frequency before you commit." },
      { title: "Activate or check out", text: "Free plans activate in one step; paid plans use the secure checkout." },
      { title: "Open My Programs", text: "Everything you own is waiting in My Programs." },
      { title: "Train and log", text: "Start a session, log every set and follow your history." }
    ],
    note: "Premium Level 2 adds a detailed intake after purchase; a RYZE trainer then prepares your programme and nutrition before they appear in My Programs."
  },
  pt: {
    label: "Como funciona",
    heading: "Da descoberta ao ",
    headingHighlight: "teu primeiro treino.",
    allPlans: "Ver todos os planos de treino",
    howItWorks: "Ver como a RYZE funciona",
    step: (index) => `Passo ${index}`,
    steps: [
      { title: "Escolhe um plano", text: "Explora planos de treino e pacotes Premium Nível 1." },
      { title: "Vê os detalhes", text: "Confirma o nível, a duração e a frequência semanal antes de decidir." },
      { title: "Ativa ou compra", text: "Os planos gratuitos ativam-se num passo; os pagos usam o pagamento seguro." },
      { title: "Abre Os meus programas", text: "Tudo o que é teu está à tua espera em Os meus programas." },
      { title: "Treina e regista", text: "Inicia um treino, regista cada série e acompanha o teu histórico." }
    ],
    note: "O Premium Nível 2 inclui um questionário detalhado depois da compra; um treinador RYZE prepara depois o teu programa e a tua nutrição, que aparecem em Os meus programas."
  }
};

export const PurchaseJourney = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  return (
  <section
    className={joinClassNames(ui.section, ui.toneNavy, styles.section)}
    aria-labelledby="purchase-journey-heading"
  >
    <div className={joinClassNames(ui.container, styles.layout)}>
      <div className={styles.intro}>
        <p className={ui.label}>{text.label}</p>
        <h2 id="purchase-journey-heading" className={ui.heading}>
          {text.heading}
          <span className={ui.hl}>{text.headingHighlight}</span>
        </h2>
        <div className={joinClassNames(ui.actions, styles.actions)}>
          <PublicAction to="/services/generic-program" variant="ghostOnDark">
            {text.allPlans}
          </PublicAction>
          <PublicAction to="/our-vision" variant="quiet">
            {text.howItWorks}
          </PublicAction>
        </div>
      </div>

      <ol className={styles.steps}>
        {text.steps.map((step, index) => {
          const Icon = STEP_ICONS[index];

          return (
            <li key={step.title} className={styles.step}>
              <span className={styles.node} aria-hidden="true">
                <Icon strokeWidth={1.9} />
              </span>
              <div>
                <p className={styles.index}>{text.step(index + 1)}</p>
                <h3 className={styles.title}>{step.title}</h3>
                <p className={styles.text}>{step.text}</p>
              </div>
            </li>
          );
        })}
      </ol>

      <p className={joinClassNames(ui.caption, styles.note)}>{text.note}</p>
    </div>
  </section>
);
};
