import { ImagePlus, Plus, Trash2 } from "lucide-react";
import { useRef, useState } from "react";

import { Button } from "@/components/button/button";
import { joinClassNames } from "@utils/class_names";
import { publicFileUrl } from "@utils/http_client";
import { uid, type PlanDraftCuratedItem, type PlanDraftCuratedNutrition } from "./plan_builder_model";

import styles from "./admin2_plan_create_page.module.css";

const emptyItem = (): PlanDraftCuratedItem => ({
  id: uid(),
  food_name: "",
  category: "",
  quantity: "",
  unit: "g",
  calories: "",
  protein_grams: "",
  carbohydrate_grams: "",
  fat_grams: "",
  fiber_grams: "",
  substitution_note: ""
});

const ItemFields = ({
  item,
  onChange,
  onRemove
}: {
  item: PlanDraftCuratedItem;
  onChange: (patch: Partial<PlanDraftCuratedItem>) => void;
  onRemove: () => void;
}) => (
  <div className={styles.nutritionItem}>
    <div className={styles.nutritionItemHead}>
      <span className={styles.nutritionItemTitle}>Food</span>
      <button
        type="button"
        className={joinClassNames(styles.iconButton, styles.iconButtonDanger)}
        onClick={onRemove}
        aria-label="Remove food"
        title="Remove food"
      >
        <Trash2 size={14} aria-hidden="true" />
      </button>
    </div>
    <div className={styles.nutritionGrid}>
      <label className={styles.nutritionField}>
        <span>Name</span>
        <input
          className={styles.formInput}
          value={item.food_name}
          maxLength={120}
          onChange={(event) => onChange({ food_name: event.target.value })}
        />
      </label>
      <label className={styles.nutritionField}>
        <span>Category</span>
        <input
          className={styles.formInput}
          value={item.category}
          maxLength={32}
          placeholder="Protein, carb, vegetable…"
          onChange={(event) => onChange({ category: event.target.value })}
        />
      </label>
      <label className={styles.nutritionField}>
        <span>Quantity</span>
        <input
          className={styles.formInput}
          type="number"
          min="0"
          step="0.01"
          value={item.quantity}
          onChange={(event) => onChange({ quantity: event.target.value })}
        />
      </label>
      <label className={styles.nutritionField}>
        <span>Unit</span>
        <input
          className={styles.formInput}
          value={item.unit}
          maxLength={24}
          onChange={(event) => onChange({ unit: event.target.value })}
        />
      </label>
      {(
        [
          ["calories", "Calories"],
          ["protein_grams", "Protein (g)"],
          ["carbohydrate_grams", "Carbs (g)"],
          ["fat_grams", "Fat (g)"],
          ["fiber_grams", "Fiber (g)"]
        ] as const
      ).map(([key, label]) => (
        <label key={key} className={styles.nutritionField}>
          <span>{label}</span>
          <input
            className={styles.formInput}
            type="number"
            min="0"
            step="1"
            value={item[key]}
            onChange={(event) => onChange({ [key]: event.target.value })}
          />
        </label>
      ))}
    </div>
    <label className={styles.nutritionField}>
      <span>Substitution note (optional)</span>
      <input
        className={styles.formInput}
        value={item.substitution_note}
        maxLength={255}
        onChange={(event) => onChange({ substitution_note: event.target.value })}
      />
    </label>
  </div>
);

const MEAL_IMAGE_TYPES = ["image/jpeg", "image/png", "image/webp"];
const MEAL_IMAGE_MAX_BYTES = 5 * 1024 * 1024;

// MealImageField: an optional picture of the meal (Premium Level 2 plans,
// owner request 2026-11-24).
const MealImageField = ({
  imageUrl,
  upload,
  onChange,
  prompt = "Add a photo of the meal",
  label = "Meal photo"
}: {
  imageUrl: string;
  upload: (file: File) => Promise<{ key: string; url: string }>;
  onChange: (image: { key: string; url: string } | null) => void;
  prompt?: string;
  label?: string;
}) => {
  const inputRef = useRef<HTMLInputElement>(null);
  const [state, setState] = useState<"idle" | "uploading" | "error">("idle");
  const choose = async (file: File | undefined) => {
    if (!file) return;
    if (!MEAL_IMAGE_TYPES.includes(file.type) || file.size > MEAL_IMAGE_MAX_BYTES) {
      setState("error");
      return;
    }
    setState("uploading");
    try {
      onChange(await upload(file));
      setState("idle");
    } catch {
      setState("error");
    }
  };
  return (
    <div className={styles.mealImage}>
      {imageUrl ? (
        <img src={publicFileUrl(imageUrl)} alt="" className={styles.mealImagePreview} />
      ) : (
        <button type="button" className={styles.mealImageEmpty} onClick={() => inputRef.current?.click()}>
          <ImagePlus size={18} aria-hidden="true" />
          <span>{prompt}</span>
        </button>
      )}
      <input
        ref={inputRef}
        type="file"
        accept={MEAL_IMAGE_TYPES.join(",")}
        className={styles.srOnly}
        tabIndex={-1}
        aria-label={label}
        onChange={(event) => {
          void choose(event.target.files?.[0]);
          event.target.value = "";
        }}
      />
      <div className={styles.mealImageActions}>
        {imageUrl ? (
          <>
            <Button type="button" variant="ghost" size="small" onClick={() => inputRef.current?.click()}>
              Replace photo
            </Button>
            <Button type="button" variant="ghost" size="small" onClick={() => onChange(null)}>
              Remove
            </Button>
          </>
        ) : null}
        {state === "uploading" ? <span className={styles.formHint}>Uploading…</span> : null}
        {state === "error" ? <span className={styles.formHint}>Use a JPEG, PNG or WebP picture up to 5 MB.</span> : null}
      </div>
    </div>
  );
};

export const CuratedNutritionEditor = ({
  nutrition,
  onChange,
  intro,
  uploadMealImage,
  showAlternatives = false
}: {
  nutrition: PlanDraftCuratedNutrition;
  onChange: (next: PlanDraftCuratedNutrition) => void;
  intro?: string;
  // Premium Level 1: a line of food alternatives per meal (v2.133.0).
  showAlternatives?: boolean;
  // Premium Level 2: each meal can carry a picture.
  uploadMealImage?: (file: File) => Promise<{ key: string; url: string }>;
}) => {
  const updateMeal = (mealId: string, update: (meal: PlanDraftCuratedNutrition["meals"][number]) => PlanDraftCuratedNutrition["meals"][number]) => {
    onChange({
      ...nutrition,
      meals: nutrition.meals.map((meal) => (meal.id === mealId ? update(meal) : meal))
    });
  };

  return (
    <div className={styles.sectionBody}>
      <p className={styles.formHint}>
        {intro ??
          "This plan is bundled with the package. Every buyer receives exactly this content, so it must be safe, complete and independent of any intake answers."}
      </p>

      <div className={styles.nutritionGridTwo}>
        <label className={styles.nutritionField}>
          <span>Dietary pattern</span>
          <input
            className={styles.formInput}
            value={nutrition.dietary_pattern}
            maxLength={32}
            onChange={(event) => onChange({ ...nutrition, dietary_pattern: event.target.value })}
          />
        </label>
        <label className={styles.nutritionField}>
          <span>Hydration (litres/day)</span>
          <input
            className={styles.formInput}
            type="number"
            min="0.1"
            max="20"
            step="0.1"
            value={nutrition.hydration_litres}
            onChange={(event) => onChange({ ...nutrition, hydration_litres: event.target.value })}
          />
        </label>
      </div>

      <label className={styles.nutritionField}>
        <span>Summary</span>
        <textarea
          className={joinClassNames(styles.formInput, styles.formTextarea)}
          value={nutrition.summary}
          maxLength={255}
          onChange={(event) => onChange({ ...nutrition, summary: event.target.value })}
        />
      </label>
      <label className={styles.nutritionField}>
        <span>Preparation guidance</span>
        <textarea
          className={joinClassNames(styles.formInput, styles.formTextarea)}
          value={nutrition.prep_guidance}
          maxLength={255}
          onChange={(event) => onChange({ ...nutrition, prep_guidance: event.target.value })}
        />
      </label>
      <label className={styles.nutritionField}>
        <span>Hydration note</span>
        <input
          className={styles.formInput}
          value={nutrition.hydration_note}
          maxLength={255}
          onChange={(event) => onChange({ ...nutrition, hydration_note: event.target.value })}
        />
      </label>
      <label className={styles.nutritionField}>
        <span>Cautions (one per line)</span>
        <textarea
          className={joinClassNames(styles.formInput, styles.formTextarea)}
          value={nutrition.cautions}
          onChange={(event) => onChange({ ...nutrition, cautions: event.target.value })}
        />
      </label>

      <div className={styles.nutritionMeals}>
        {nutrition.meals.map((meal, mealIndex) => (
          <div key={meal.id} className={styles.nutritionMeal}>
            <div className={styles.nutritionItemHead}>
              <span className={styles.nutritionItemTitle}>Meal {mealIndex + 1}</span>
              <button
                type="button"
                className={joinClassNames(styles.iconButton, styles.iconButtonDanger)}
                onClick={() =>
                  onChange({
                    ...nutrition,
                    meals: nutrition.meals.filter((entry) => entry.id !== meal.id)
                  })
                }
                aria-label={`Remove meal ${mealIndex + 1}`}
                title="Remove meal"
              >
                <Trash2 size={14} aria-hidden="true" />
              </button>
            </div>
            <div className={styles.nutritionGridTwo}>
              <label className={styles.nutritionField}>
                <span>Label</span>
                <input
                  className={styles.formInput}
                  value={meal.label}
                  maxLength={48}
                  onChange={(event) =>
                    updateMeal(meal.id, (current) => ({ ...current, label: event.target.value }))
                  }
                />
              </label>
              <label className={styles.nutritionField}>
                <span>Kind</span>
                <select
                  className={styles.formInput}
                  value={meal.kind}
                  onChange={(event) =>
                    updateMeal(meal.id, (current) => ({
                      ...current,
                      kind: event.target.value === "snack" ? "snack" : "meal"
                    }))
                  }
                >
                  <option value="meal">Meal</option>
                  <option value="snack">Snack</option>
                </select>
              </label>
            </div>
            <label className={styles.nutritionField}>
              <span>Notes (optional)</span>
              <input
                className={styles.formInput}
                value={meal.notes}
                maxLength={255}
                onChange={(event) =>
                  updateMeal(meal.id, (current) => ({ ...current, notes: event.target.value }))
                }
              />
            </label>
            {uploadMealImage ? (
              <MealImageField
                imageUrl={meal.image_url ?? ""}
                upload={uploadMealImage}
                onChange={(image) =>
                  updateMeal(meal.id, (current) => ({ ...current, image_key: image?.key ?? "", image_url: image?.url ?? "" }))
                }
              />
            ) : null}

            {meal.items.map((item, itemIndex) => (
              <ItemFields
                key={item.id}
                item={item}
                onChange={(patch) =>
                  updateMeal(meal.id, (current) => ({
                    ...current,
                    items: current.items.map((entry) =>
                      entry.id === item.id ? { ...entry, ...patch } : entry
                    )
                  }))
                }
                onRemove={() =>
                  updateMeal(meal.id, (current) => ({
                    ...current,
                    items: current.items.filter((entry) => entry.id !== item.id)
                  }))
                }
              />
            ))}
            <div className={styles.addRow}>
              <Button
                type="button"
                variant="ghost"
                size="small"
                icon={<Plus size={14} />}
                onClick={() =>
                  updateMeal(meal.id, (current) => ({
                    ...current,
                    items: [...current.items, emptyItem()]
                  }))
                }
              >
                Add food
              </Button>
            </div>
            {meal.items.length === 0 ? <p className={styles.formHint}>Add at least one food.</p> : null}

            {/* Food alternatives for the whole meal, with an optional picture
                (v2.133.0, owner request 2026-11-27). */}
            {showAlternatives ? (
            <div className={styles.alternatives}>
              <label className={styles.nutritionField}>
                <span>Alternatives (optional) — what the person can eat instead</span>
                <textarea
                  className={styles.formInput}
                  rows={2}
                  maxLength={500}
                  placeholder="e.g. Rice or potatoes instead of oats; tofu instead of chicken"
                  value={meal.alternatives ?? ""}
                  onChange={(event) =>
                    updateMeal(meal.id, (current) => ({ ...current, alternatives: event.target.value }))
                  }
                />
              </label>
              {uploadMealImage ? (
                <MealImageField
                  imageUrl={meal.alternatives_image_url ?? ""}
                  upload={uploadMealImage}
                  prompt="Add a photo of the alternatives"
                  label="Alternatives photo"
                  onChange={(image) =>
                    updateMeal(meal.id, (current) => ({
                      ...current,
                      alternatives_image_key: image?.key ?? "",
                      alternatives_image_url: image?.url ?? ""
                    }))
                  }
                />
              ) : null}
            </div>
            ) : null}
          </div>
        ))}
      </div>

      <div className={styles.addRow}>
        <Button
          type="button"
          variant="secondary"
          size="small"
          icon={<Plus size={14} />}
          onClick={() =>
            onChange({
              ...nutrition,
              meals: [
                ...nutrition.meals,
                {
                  id: uid(),
                  label: "",
                  kind: "meal",
                  notes: "",
                  items: [emptyItem()]
                }
              ]
            })
          }
        >
          Add meal
        </Button>
      </div>
    </div>
  );
};
