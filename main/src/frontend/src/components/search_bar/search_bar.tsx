import { Calendar, Repeat, Search, SlidersHorizontal } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";

import {
  searchPrograms,
  type ProgramSearchResult
} from "@/services/program_search";
import { joinClassNames } from "@utils/class_names";

import styles from "./search_bar.module.css";

const SCROLL_THRESHOLD = 24;
const SEARCH_DEBOUNCE_MS = 220;

interface SearchBarProps {
  onFiltersClick?: () => void;
  totalPrograms?: number;
}

type SearchStatus = "idle" | "loading" | "done";

export const SearchBar = ({ onFiltersClick, totalPrograms }: SearchBarProps) => {
  const [scrolled, setScrolled] = useState(false);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [status, setStatus] = useState<SearchStatus>("idle");
  const [results, setResults] = useState<ProgramSearchResult[]>([]);
  const containerRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<number>(0);

  const trimmedQuery = query.trim();
  const dropdownOpen = trimmedQuery.length > 0 && open;

  useEffect(() => {
    const update = () => setScrolled(window.scrollY > SCROLL_THRESHOLD);
    update();
    window.addEventListener("scroll", update, { passive: true });
    return () => window.removeEventListener("scroll", update);
  }, []);

  useEffect(() => {
    if (!dropdownOpen) return;

    const onPointerDown = (event: PointerEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };

    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [dropdownOpen]);

  useEffect(() => () => window.clearTimeout(debounceRef.current), []);

  const handleQueryChange = (value: string) => {
    setQuery(value);
    window.clearTimeout(debounceRef.current);

    const term = value.trim();
    if (!term) {
      setOpen(false);
      setStatus("idle");
      setResults([]);
      return;
    }

    setOpen(true);
    setStatus("loading");
    debounceRef.current = window.setTimeout(async () => {
      let found: ProgramSearchResult[] = [];
      try {
        found = await searchPrograms(term);
      } catch {
        found = [];
      }
      setResults(found);
      setStatus("done");
    }, SEARCH_DEBOUNCE_MS);
  };

  const closeDropdown = () => {
    setOpen(false);
  };

  return (
    <div
      ref={containerRef}
      className={joinClassNames(styles.search, scrolled && styles.searchScrolled)}
    >
      <div className={styles.field}>
        <Search className={styles.icon} strokeWidth={2} aria-hidden="true" />
        <input
          type="search"
          className={styles.input}
          placeholder="Search training plans..."
          aria-label="Search training plans"
          value={query}
          onChange={(event) => handleQueryChange(event.target.value)}
        />
      </div>

      <div className={styles.actions}>
        {typeof totalPrograms === "number" ? (
          <span className={styles.totalCount}>
            {totalPrograms} program{totalPrograms !== 1 ? "s" : ""}
          </span>
        ) : null}

        {onFiltersClick ? (
          <button type="button" className={styles.filtersButton} onClick={onFiltersClick}>
            <SlidersHorizontal strokeWidth={2} aria-hidden="true" />
            <span>Filters</span>
          </button>
        ) : null}
      </div>

      {dropdownOpen ? (
        <div className={styles.dropdown} aria-live="polite">
          {status === "loading" ? (
            <div className={styles.dropdownEmpty}>Searching...</div>
          ) : results.length > 0 ? (
            <ul className={styles.resultList}>
              {results.map((item) => {
                const category = [item.trainingType, item.level].filter(Boolean).join(" · ");

                return (
                  <li key={item.id} className={styles.resultRow}>
                    <Link
                      to={`/services/generic-program/${item.id}`}
                      className={styles.resultLink}
                      onClick={closeDropdown}
                    >
                      <span className={styles.resultTitle}>{item.title}</span>
                      <span className={styles.resultMeta}>
                        {item.durationWeeks !== null ? (
                          <span className={styles.resultStat}>
                            <Calendar strokeWidth={2} aria-hidden="true" />
                            {item.durationWeeks} weeks
                          </span>
                        ) : null}
                        {item.frequencyPerWeek !== null ? (
                          <span className={styles.resultStat}>
                            <Repeat strokeWidth={2} aria-hidden="true" />
                            {item.frequencyPerWeek}/week
                          </span>
                        ) : null}
                        {category ? <span>{category}</span> : null}
                      </span>
                    </Link>
                  </li>
                );
              })}
            </ul>
          ) : (
            <div className={styles.dropdownEmpty}>No results found</div>
          )}
        </div>
      ) : null}
    </div>
  );
};