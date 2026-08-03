import { GlassPanel } from "@/components/glass_panel/glass_panel";

import styles from "./product_demos.module.css";

const macros = [
  { label: "Protein", value: 142, target: 160, unit: "g" },
  { label: "Carbs", value: 210, target: 240, unit: "g" },
  { label: "Fat", value: 58, target: 65, unit: "g" }
] as const;

export const NutritionDemo = () => (
  <GlassPanel className={styles.window} as="article">
    <div className={styles.windowChrome}>
      <span />
      <span />
      <span />
      <p>Nutrition · Today</p>
    </div>

    <div className={styles.nutritionLayout}>
      <div className={styles.calorieRing} aria-hidden="true">
        <svg viewBox="0 0 120 120">
          <circle cx="60" cy="60" r="48" className={styles.ringTrack} />
          <circle cx="60" cy="60" r="48" className={styles.ringValue} />
        </svg>
        <div className={styles.ringLabel}>
          <strong>1,840</strong>
          <span>/ 2,200 kcal</span>
        </div>
      </div>

      <div className={styles.macroList}>
        {macros.map((macro) => (
          <div key={macro.label} className={styles.macroRow}>
            <div className={styles.macroMeta}>
              <span>{macro.label}</span>
              <strong>
                {macro.value}
                {macro.unit}
                <em>
                  {" "}
                  / {macro.target}
                  {macro.unit}
                </em>
              </strong>
            </div>
            <div className={styles.progressRow}>
              <div
                className={styles.progressFill}
                style={{ width: `${(macro.value / macro.target) * 100}%` }}
              />
            </div>
          </div>
        ))}
      </div>

      <div className={styles.aiCard}>
        <span>Recommendation</span>
        <p>Shift 30g carbs earlier on training days to protect evening recovery.</p>
      </div>
    </div>
  </GlassPanel>
);
