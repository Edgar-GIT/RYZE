import { Check, RotateCcw } from "lucide-react";

import { joinClassNames } from "@utils/class_names";
import type { FilterGroup } from "@/constants/marketplace";

import styles from "./filter_sidebar.module.css";

interface FilterSidebarProps {
  groups: FilterGroup[];
  selected: Record<string, string[]>;
  onToggle: (groupId: string, option: string) => void;
  onClear: () => void;
  className?: string;
}

export const FilterSidebar = ({
  groups,
  selected,
  onToggle,
  onClear,
  className
}: FilterSidebarProps) => {
  const hasSelection = Object.values(selected).some((options) => options.length > 0);

  return (
    <div className={joinClassNames(styles.sidebar, className)}>
      {groups.map((group) => (
        <fieldset key={group.id} className={styles.group}>
          <legend className={styles.groupTitle}>{group.title}</legend>
          <div className={styles.options}>
            {group.options.map((option) => {
              const isChecked = (selected[group.id] ?? []).includes(option);

              return (
                <label key={option} className={styles.option}>
                  <input
                    type="checkbox"
                    className={styles.input}
                    checked={isChecked}
                    onChange={() => onToggle(group.id, option)}
                  />
                  <span
                    className={joinClassNames(styles.checkbox, isChecked && styles.checkboxChecked)}
                  >
                    {isChecked ? (
                      <Check strokeWidth={2.6} aria-hidden="true" />
                    ) : null}
                  </span>
                  <span className={styles.label}>{option}</span>
                </label>
              );
            })}
          </div>
        </fieldset>
      ))}

      {hasSelection ? (
        <button type="button" className={styles.clear} onClick={onClear}>
          <RotateCcw strokeWidth={1.8} aria-hidden="true" />
          <span>Clear all</span>
        </button>
      ) : null}
    </div>
  );
};