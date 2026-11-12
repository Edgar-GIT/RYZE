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
import { useI18n } from "@/i18n/i18n";

interface PremiumNutritionPlanProps {
  status: NutritionAssignmentStatus;
  /** True while the generation request is in flight. */
  generating: boolean;
  /** Curated packages ship a fixed plan; no answers or generation exist. */
  curated?: boolean;
  /** Client-owned wording for the most recent generation failure, if any. */
  errorMessage?: string | null;
  onGenerate: () => void;
  onRefresh: () => void;
}

// exclusionLabels maps the server's controlled reason tokens onto wording the
// client owns. The token is a normalised vocabulary entry, never raw questionnaire
// text, so it is safe to render here.
const exclusionKeys = {
  allergy: "account.nutrition.exclusionNo",
  intolerance: "account.nutrition.exclusionNo",
  diet: "account.nutrition.exclusionOnly",
  excluded_food: "account.nutrition.exclusionExcludes",
  disliked_food: "account.nutrition.exclusionWithout",
  insufficient_alternatives: "account.nutrition.exclusionLimited"
} as const satisfies Record<NutritionExclusion["reason_code"], string>;

const exclusionLabel = (exclusion: NutritionExclusion, t: ReturnType<typeof useI18n>["t"]) => {
  const key = exclusionKeys[exclusion.reason_code];
  return key ? t(key, { token: exclusion.token }) : exclusion.token;
};

const mealItemKey = (item: NutritionMealItem) => `${item.position}-${item.food_name}`;

// PremiumNutritionPlan renders the server-owned assignment lifecycle. The
// component maps the backend state directly and never invents readiness: a plan
// is only ever presented when the backend reports `completed`.
export const PremiumNutritionPlan = ({
  status,
  generating,
  curated = false,
  errorMessage,
  onGenerate,
  onRefresh
}: PremiumNutritionPlanProps) => {
  const { t } = useI18n();
  if (status.status === "failed") {
    return (
      <section className={styles.panel} aria-label={t("account.nutrition.label")}>
        <p className={styles.heading}>
          <AlertTriangle size={16} aria-hidden="true" />
          {curated ? t("account.nutrition.failedCurated") : t("account.nutrition.failedBuilt")}
        </p>
        <p className={styles.body}>
          {errorMessage ?? t("account.nutrition.failedBody")}
        </p>
        <div className={styles.actions}>
          <Button variant="primary" size="small" onClick={onGenerate} disabled={generating}>
            {generating ? t("account.nutrition.loading") : t("account.nutrition.tryAgain")}
          </Button>
        </div>
      </section>
    );
  }

  if (status.status === "pending" || status.status === "processing") {
    return (
      <section className={styles.panel} aria-label={t("account.nutrition.label")}>
        <p className={styles.heading}>
          <RefreshCw size={16} aria-hidden="true" />
          {curated ? t("account.nutrition.preparingCurated") : t("account.nutrition.preparingBuilt")}
        </p>
        <p className={styles.body}>
          {curated
            ? t("account.nutrition.preparingCuratedBody")
            : status.status === "processing"
              ? t("account.nutrition.processingBody")
              : t("account.nutrition.pendingBody")}
        </p>
        {errorMessage ? <p className={styles.notice}>{errorMessage}</p> : null}
        <div className={styles.actions}>
          {status.status === "pending" ? (
            <Button variant="primary" size="small" onClick={onGenerate} disabled={generating}>
              {generating
                ? t("account.nutrition.loading")
                : curated
                  ? t("account.nutrition.openCurated")
                  : t("account.nutrition.build")}
            </Button>
          ) : null}
          <Button variant="secondary" size="small" onClick={onRefresh}>
            {t("account.nutrition.refreshStatus")}
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
      <section className={styles.panel} aria-label={t("account.nutrition.label")}>
        <p className={styles.heading}>
          <AlertTriangle size={16} aria-hidden="true" />
          {t("account.nutrition.unavailable")}
        </p>
        <p className={styles.body}>
          {curated ? t("account.nutrition.unavailableCurated") : t("account.nutrition.unavailableBuilt")}
        </p>
        <div className={styles.actions}>
          <Button variant="secondary" size="small" onClick={onRefresh}>
            {t("account.nutrition.refresh")}
          </Button>
        </div>
      </section>
    );
  }

  return (
    <section className={styles.panel} aria-label={t("account.nutrition.label")}>
      <p className={styles.heading}>
        <CheckCircle2 size={16} aria-hidden="true" />
        {curated ? t("account.nutrition.readyCurated") : t("account.nutrition.readyBuilt")}
      </p>
      <p className={styles.summary}>{plan.summary}</p>

      {status.out_of_date && !curated ? (
        // The delivered plan is still shown while it is stale: it is what the
        // client is following today, and hiding it would leave them without one.
        // Regeneration stays an explicit action because it supersedes a version
        // they may have already started following.
        <div className={styles.stale} role="status">
          <p className={styles.body}>
            {t("account.nutrition.stale")}
          </p>
          {errorMessage ? <p className={styles.notice}>{errorMessage}</p> : null}
          <div className={styles.actions}>
            <Button variant="primary" size="small" onClick={onGenerate} disabled={generating}>
              {generating ? t("account.nutrition.rebuilding") : t("account.nutrition.update")}
            </Button>
          </div>
        </div>
      ) : null}

      <div className={styles.tiles}>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>{t("account.nutrition.dailyTarget")}</span>
          <span className={styles.tileValue}>{plan.target_calories} kcal</span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>{t("account.nutrition.maintenance")}</span>
          <span className={styles.tileValue}>{plan.maintenance_calories} kcal</span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>{t("account.nutrition.protein")}</span>
          <span className={styles.tileValue}>
            {plan.daily.protein_grams} g
            <span className={styles.tileNote}>{plan.protein_percent}%</span>
          </span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>{t("account.nutrition.carbs")}</span>
          <span className={styles.tileValue}>
            {plan.daily.carbs_grams} g
            <span className={styles.tileNote}>{plan.carbohydrate_percent}%</span>
          </span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>{t("account.nutrition.fat")}</span>
          <span className={styles.tileValue}>
            {plan.daily.fat_grams} g
            <span className={styles.tileNote}>{plan.fat_percent}%</span>
          </span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>{t("account.nutrition.fibre")}</span>
          <span className={styles.tileValue}>{plan.daily.fiber_grams} g</span>
        </div>
        <div className={styles.tile}>
          <span className={styles.tileLabel}>
            <Droplets size={12} aria-hidden="true" /> {t("account.nutrition.fluids")}
          </span>
          <span className={styles.tileValue}>{plan.hydration.daily_litres} L</span>
        </div>
      </div>

      <h3 className={styles.subheading}>
        <UtensilsCrossed size={15} aria-hidden="true" />
        {t("account.nutrition.today")}
      </h3>
      <p className={styles.body}>
        {plan.snacks_per_day === 1
          ? t("account.nutrition.mealsPerDayOneSnack", { meals: plan.meals_per_day })
          : t("account.nutrition.mealsPerDay", { meals: plan.meals_per_day, snacks: plan.snacks_per_day })}
      </p>

      <ul className={styles.meals}>
        {plan.meals.map((meal) => (
          <li key={`${meal.position}-${meal.label}`} className={styles.meal}>
            <span className={styles.mealLabel}>{meal.label}</span>
            <span className={styles.mealMeta}>
              {t("account.nutrition.mealMeta", {
                percent: meal.percent_of_daily,
                kcal: meal.macros.calories,
                protein: meal.macros.protein_grams
              })}
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
          <h3 className={styles.subheading}>
            {curated ? t("account.nutrition.notesCurated") : t("account.nutrition.notesBuilt")}
          </h3>
          <div className={styles.tags}>
            {plan.exclusions.map((exclusion) => (
              <span
                key={`${exclusion.reason_code}-${exclusion.token}`}
                className={styles.tag}
              >
                {exclusionLabel(exclusion, t)}
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
          read-only: the only way a questionnaire-derived plan changes is an
          explicit refresh when the intake moved on, never an edit. */}
      <p className={styles.immutable}>
        {curated ? t("account.nutrition.immutableCurated") : t("account.nutrition.immutableBuilt")}
      </p>
    </section>
  );
};
