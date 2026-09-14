import { Search, SlidersHorizontal } from "lucide-react";
import { useEffect, useState } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./search_bar.module.css";

const SCROLL_THRESHOLD = 24;

interface SearchBarProps {
  onFiltersClick?: () => void;
  totalPrograms?: number;
}

export const SearchBar = ({ onFiltersClick, totalPrograms }: SearchBarProps) => {
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const update = () => setScrolled(window.scrollY > SCROLL_THRESHOLD);
    update();
    window.addEventListener("scroll", update, { passive: true });
    return () => window.removeEventListener("scroll", update);
  }, []);

  return (
    <div className={joinClassNames(styles.search, scrolled && styles.searchScrolled)}>
      <div className={styles.field}>
        <Search className={styles.icon} strokeWidth={2} aria-hidden="true" />
        <input
          type="search"
          className={styles.input}
          placeholder="Search training plans..."
          aria-label="Search training plans"
        />
      </div>

      <div className={styles.actions}>
        {typeof totalPrograms === "number" ? (
          <span className={styles.totalCount}>{totalPrograms} programs</span>
        ) : null}

        {onFiltersClick ? (
          <button type="button" className={styles.filtersButton} onClick={onFiltersClick}>
            <SlidersHorizontal strokeWidth={2} aria-hidden="true" />
            <span>Filters</span>
          </button>
        ) : null}
      </div>
    </div>
  );
};