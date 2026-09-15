import { BRAND_ASSETS } from "@/constants/brand_assets";
import { Admin2StatusBadge } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { formatAdminPrice, programTypeShortLabel, ProgramType } from "@/services/admin2_api";
import {
  countDays,
  countExercises,
  countWeeks,
  type PlanDraft
} from "./plan_builder_model";

import styles from "./plan_preview.module.css";

export const PlanPreview = ({ draft }: { draft: PlanDraft }) => {
  const paidPrice = Math.round((Number(draft.price) || 0) * 100);
  const priceLabel =
    draft.type === ProgramType.FREE ? "Free" : formatAdminPrice(paidPrice, draft.currency);
  const hasWeeks = countWeeks(draft) > 0;

  return (
    <div className={styles.card}>
      <div className={styles.cover}>
        <img src={BRAND_ASSETS.icon} alt="" className={styles.logo} />
      </div>
      <div className={styles.body}>
        <h3 className={styles.title}>
          {draft.name.trim() ? draft.name : <span className={styles.titlePlaceholder}>Untitled plan</span>}
        </h3>
        <p className={styles.description}>
          {draft.description.trim() ? (
            draft.description
          ) : (
            <span className={styles.descPlaceholder}>
              Describe what this plan helps members achieve.
            </span>
          )}
        </p>
        <div className={styles.meta}>
          <Admin2StatusBadge label={programTypeShortLabel(draft.type)} tone="success" withDot={false} />
          <Admin2StatusBadge
            label={draft.status === "published" ? "Published" : "Draft"}
            tone={draft.status === "published" ? "warning" : "neutral"}
          />
          <Admin2StatusBadge label={priceLabel} tone="neutral" withDot={false} />
        </div>
      </div>
      <div className={styles.stats}>
        {hasWeeks ? (
          <>
            <span className={styles.stat}>{countWeeks(draft)} week{countWeeks(draft) === 1 ? "" : "s"}</span>
            <span className={styles.stat}>{countDays(draft)} day{countDays(draft) === 1 ? "" : "s"}</span>
            <span className={styles.stat}>
              {countExercises(draft)} exercise{countExercises(draft) === 1 ? "" : "s"}
            </span>
          </>
        ) : (
          <span className={styles.stat}>No structure yet</span>
        )}
      </div>
    </div>
  );
};