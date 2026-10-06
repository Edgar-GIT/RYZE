import {
  AlertTriangle,
  CheckCircle2,
  Droplets,
  RefreshCw,
  UtensilsCrossed,
} from "lucide-react";

import { Button } from "@/components/button/button";
import type {
  NutritionAssignmentStatus,
  NutritionExclusion,
  NutritionMealItem,
} from "@/services/premium_level1_api";

import styles from "./premium_nutrition_plan.module.css";

interface PremiumNutritionPlanProps {
  status: NutritionAssignmentStatus;
  /** True while the generation request is in flight. */
  generating: boolean;
  /** Client-owned wording for the most recent generation failure, if any. */
  errorMessage?: string | null;
  onGenerate: () => void;
  onRefresh: () => void;
}

// exclusionLabels maps the server's controlled reason tokens onto wording the
// client owns. The token is a normalised vocabulary entry, never raw questionnaire
// text, so it is safe to render here.
const exclusionLabels: Record<NutritionExclusion["reason_code"], (token: string) => string> = {
  allergy: (token: string) => `No ${token}`,
  intolerance: (token: string) => `No ${token}`,
  diet: (token: string) => `${token} only`,
  excluded_food: (token: string) => `Excludes ${token}`,
  disliked_food: (token: string) => `Without ${token}`,
  insufficient_alternatives: (token: string) => `Limited options with ${token}`,
};

const exclusionLabel = (exclusion: NutritionExclusion) => {
  const label = exclusionLabels[exclusion.reason_code];
  return label ? label(exclusion.token) : exclusion.token;
};

const mealItemKey = (item: NutritionMealItem) => `${item.position}-${item.food_name}`;

// PremiumNutritionPlan renders the server-owned assignment lifecycle. The
// component maps the backend state directly and never invents readiness: a plan
// is only ever presented when the backend reports `completed`.
export const PremiumNutritionPlan = ({ status, generating, errorMessage, onGenerate, onRefresh }: PremiumNutritionPlanProps) => {
  if (status.status === "failed") {
    return (
      <section className={styles.panel} aria-label="Nutrition programme">
        <p className={styles.heading}>
          <AlertTriangle size={16} aria-hidden="true" />
          We could not build your nutrition programme yet
        </p>
        <p className={styles.body}>
          {errorMessage ??
            "Nothing is wrong with your purchase and no action is required from you. You can try building it again — your package and your access are unchanged."}
        </p>
        <div className={styles.actions}>
          <Button variant="primary" size="small" onClick={onGenerate} disabled={generating}>
            {generating ? "Building…" : "Try again"}
          </Button>
        </div>
      </section>
    );
  }

  if (status.status === "pending" || status.status === "processing") {
    return (
      <section className={styles.panel} aria-label="Nutrition programme">
        <p className={styles.heading}>
          <RefreshCw size={16} aria-hidden="true" />
          Preparing your nutrition programme
        </p>
        <p className={styles.body}>
          {status.status === "processing"
            ? "RYZE is building your personalised nutrition programme now. This only takes a moment."
            : "Your package is confirmed. Your personalised nutrition programme has not been built yet."}
        </p>
        {errorMessage ? <p className={styles.notice}>{errorMessage}</p> : null}
        <div className={styles.actions}>
          {status.status === "pending" ? (
            <Button variant="primary" size="small" onClick={onGenerate} disabled={generating}>
              {generating ? "Building…" : "Build my nutrition programme"}
            </Button>
          ) : null}
          <Button variant="secondary" size="small" onClick={onRefresh}>
            Refresh status
          </Button>
        </div>
      </section>
    );
  }

  const plan = status.plan;
  if (!plan) {
    // The backend reported completion but sent no plan. Treating that as
    // anything other than an unavailable plan would be inventing state.
    return (
      <section className={styles.panel} aria-label="Nutrition programme">
        <p className={styles.heading}>
          <AlertTriangle size={16} aria-hidden="true" />
          Nutrition programme unavailable
        </p>
        <p className={styles.body}>
          Your training plan is ready. The nutrition programme could not be displayed right now. Please refresh
          to try again.
        </p>
        <div className={styles.actions}>
          <Button variant="secondary" size="small" onClick={onRefresh}>
            Refresh
          </Button>
        </div>
      </section>
    );
  }

  return (
    <section className={styles.panel} aria-label="Nutrition programme">
      <p className={styles.heading}>
        <CheckCircle2 size={16} aria-hidden="true" />
        Your nutrition programme is ready
      </p>
      <p className={styles.summary}>{plan.summary}</p>

      {status.out_of_date ? (
        // The delivered plan is still shown while it is stale: it is what the
        // client is following today, and hiding it would leave them without one.
        // Regeneration stays an explicit action because it supersedes a version
        // they may have already started following.
        <div className={styles.stale} role="status">
          <p className={styles.body}>
            Your answers changed after this plan was built, so it no longer reflects your latest questionnaire.
            Your current plan stays available below until you update it.
          </p>
          {errorMessage ? <p className={styles.notice}>{errorMessage}</p> : null}
          <div className={styles.actions}>
            <Button variant="primary" size="small" onClick={onGenerate} disabled={generating}>
              {generating ? "Rebuilding…" : "Update to my latest answers"}
            </Button>
          </div>
        </div>
      ) : null}

      <div className={styles.tiles}>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Daily target</span>
          <span className={styles.tileValue}>{plan.target_calories} kcal</span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Maintenance</span>
          <span className={styles.tileValue}>{plan.maintenance_calories} kcal</span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Protein</span>
          <span className={styles.tileValue}>
            {plan.daily.protein_grams} g
            <span className={styles.tileNote}>{plan.protein_percent}%</span>
          </span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Carbohydrates</span>
          <span className={styles.tileValue}>
            {plan.daily.carbs_grams} g
            <span className={styles.tileNote}>{plan.carbohydrate_percent}%</span>
          </span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Fat</span>
          <span className={styles.tileValue}>
            {plan.daily.fat_grams} g
            <span className={styles.tileNote}>{plan.fat_percent}%</span>
          </span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Fibre</span>
          <span className={styles.tileValue}>{plan.daily.fiber_grams} g</span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>
            <Droplets size={12} aria-hidden="true" /> Fluids
          </span>
          <span className={styles.tileValue}>{plan.hydration.daily_litres} L</span>
        </div>
      </div>

      <h3 className={styles.subheading}>
        <UtensilsCrossed size={15} aria-hidden="true" />
        What to eat today
      </h3>
      <p className={styles.body}>
        {plan.meals_per_day} meals and {plan.snacks_per_day} snack
        {plan.snacks_per_day === 1 ? "" : "s"} per day.
      </p>

      <ul className={styles.meals}>
        {plan.meals.map((meal) => (
          <li key={`${meal.position}-${meal.label}`} className={styles.meal}>
            <span className={styles.mealLabel}>{meal.label}</span>
            <span className={styles.mealMeta}>
              {meal.percent_of_daily}% of day · {meal.macros.calories} kcal · {meal.macros.protein_grams} g protein
            </span>
            <ul className={styles.items}>
              {meal.items.map((item) => (
                <li key={mealItemKey(item)} className={styles.item}>
                  <span className={styles.itemName}>{item.food_name}</span>
                  <span className={styles.itemAmount}>
                    {item.quantity} {item.unit}
                  </span>
                  <span className={styles.itemMacros}>
                    {item.macros.calories} kcal · {item.macros.protein_grams} P · {item.macros.carbs_grams} C ·{" "}
                    {item.macros.fat_grams} F
                  </span>
                  {item.substitution_note ? (
                    <span className={styles.itemSwap}>{item.substitution_note}</span>
                  ) : null}
                </li>
              ))}
            </ul>
            {meal.notes ? <span className={styles.mealNotes}>{meal.notes}</span> : null}
          </li>
        ))}
      </ul>

      {plan.prep_guidance ? <p className={styles.body}>{plan.prep_guidance}</p> : null}
      {plan.hydration.note ? <p className={styles.body}>{plan.hydration.note}</p> : null}

      {plan.exclusions.length > 0 ? (
        <>
          <h3 className={styles.subheading}>Built around your answers</h3>
          <div className={styles.tags}>
            {plan.exclusions.map((exclusion) => (
              <span
                key={`${exclusion.reason_code}-${exclusion.token}`}
                className={styles.tag}
              >
                {exclusionLabel(exclusion)}
              </span>
            ))}
          </div>
        </>
      ) : null}

      {plan.cautions.length > 0 ? (
        <div className={styles.cautions} role="note">
          {plan.cautions.map((caution) => (
            <p key={caution} className={styles.caution}>
              <AlertTriangle size={14} aria-hidden="true" />
              <span>{caution}</span>
            </p>
          ))}
        </div>
      ) : null}

      {/* A delivered plan is part of a purchased package. It is deliberately
          read-only: the only way it changes is an explicit refresh when the
          questionnaire moved on, never an edit. */}
      <p className={styles.immutable}>
        This nutrition programme is part of your package and is built from the answers you submitted before
        checkout.
      </p>
    </section>
  );
};
