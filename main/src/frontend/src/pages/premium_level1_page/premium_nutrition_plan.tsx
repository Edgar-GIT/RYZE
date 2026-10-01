import { AlertTriangle, CheckCircle2, Droplets, RefreshCw, UtensilsCrossed } from "lucide-react";

import { Button } from "@/components/button/button";
import type { NutritionAssignmentStatus } from "@/services/premium_level1_api";

import styles from "./premium_nutrition_plan.module.css";

interface PremiumNutritionPlanProps {
  status: NutritionAssignmentStatus;
  /** True while the generation request is in flight. */
  generating: boolean;
  onGenerate: () => void;
  onRefresh: () => void;
}

// PremiumNutritionPlan renders the server-owned assignment lifecycle. The
// component maps the backend state directly and never invents readiness: a plan
// is only ever presented when the backend reports `completed`.
export const PremiumNutritionPlan = ({ status, generating, onGenerate, onRefresh }: PremiumNutritionPlanProps) => {
  if (status.status === "failed") {
    return (
      <section className={styles.panel} aria-label="Nutrition programme">
        <p className={styles.heading}>
          <AlertTriangle size={16} aria-hidden="true" />
          We could not build your nutrition programme yet
        </p>
        <p className={styles.body}>
          Nothing is wrong with your purchase and no action is required from you. You can try building it again
          — your package and your access are unchanged.
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

  const energy = plan.energy_targets;
  const macros = plan.macronutrients;

  return (
    <section className={styles.panel} aria-label="Nutrition programme">
      <p className={styles.heading}>
        <CheckCircle2 size={16} aria-hidden="true" />
        Your nutrition programme is ready
      </p>
      <p className={styles.summary}>{plan.summary}</p>

      <div className={styles.tiles}>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Daily target</span>
          <span className={styles.tileValue}>{energy.target_calories} kcal</span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Maintenance</span>
          <span className={styles.tileValue}>{energy.maintenance_calories} kcal</span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Protein</span>
          <span className={styles.tileValue}>{macros.protein_grams} g</span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Carbohydrates</span>
          <span className={styles.tileValue}>{macros.carbohydrate_grams} g</span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>Fat</span>
          <span className={styles.tileValue}>{macros.fat_grams} g</span>
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
        Daily structure
      </h3>
      <p className={styles.body}>
        {plan.meal_plan.meals_per_day} meals and {plan.meal_plan.snacks_per_day} snack
        {plan.meal_plan.snacks_per_day === 1 ? "" : "s"} per day.
      </p>
      <ul className={styles.meals}>
        {plan.meal_plan.distribution.map((meal) => (
          <li key={meal.label} className={styles.meal}>
            <span className={styles.mealLabel}>{meal.label}</span>
            <span className={styles.mealMeta}>
              {meal.percent_of_daily}% · {meal.approx_calories} kcal · {meal.approx_protein_grams} g protein
            </span>
          </li>
        ))}
      </ul>

      {plan.meal_plan.prep_guidance ? (
        <p className={styles.body}>{plan.meal_plan.prep_guidance}</p>
      ) : null}
      {plan.hydration.note ? <p className={styles.body}>{plan.hydration.note}</p> : null}

      {plan.dietary_rules.excluded_foods.length > 0 || plan.dietary_rules.allergies.length > 0 ? (
        <>
          <h3 className={styles.subheading}>Built around your answers</h3>
          <div className={styles.tags}>
            {plan.dietary_rules.allergies.map((item) => (
              <span key={`allergy-${item}`} className={styles.tag}>
                No {item}
              </span>
            ))}
            {plan.dietary_rules.excluded_foods.map((item) => (
              <span key={`excluded-${item}`} className={styles.tag}>
                Excludes {item}
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
          read-only: there is no regeneration action anywhere in this flow. */}
      <p className={styles.immutable}>
        This nutrition programme is part of your package and is fixed from the answers you submitted before
        checkout.
      </p>
    </section>
  );
};