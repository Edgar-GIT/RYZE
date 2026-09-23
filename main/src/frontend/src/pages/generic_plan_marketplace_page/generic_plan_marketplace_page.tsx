import { Check, PanelLeftClose, PanelLeftOpen, X } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { FilterSidebar } from "@/components/filter_sidebar/filter_sidebar";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ProgramRow } from "@/components/program_row/program_row";
import { SearchBar } from "@/components/search_bar/search_bar";
import {
  DURATION_RANGES,
  MARKETPLACE_FILTER_GROUPS,
  MARKETPLACE_ROWS
} from "@/constants/marketplace";
import {
  fetchAllMarketplacePrograms,
  type MarketplaceProgram
} from "@/services/marketplace_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./generic_plan_marketplace_page.module.css";

type SelectedFilters = Record<string, string[]>;

const EMPTY_SELECTION: SelectedFilters = {};

const byCreatedDesc = (a: MarketplaceProgram, b: MarketplaceProgram) =>
  b.created_at.localeCompare(a.created_at);
const byUpdatedDesc = (a: MarketplaceProgram, b: MarketplaceProgram) =>
  b.updated_at.localeCompare(a.updated_at);

const matchesFilters = (program: MarketplaceProgram, selected: SelectedFilters): boolean => {
  const trainingTypes = selected["training-type"] ?? [];
  if (trainingTypes.length > 0 && !trainingTypes.includes(program.training_type ?? "")) {
    return false;
  }

  const frequencies = selected["frequency"] ?? [];
  if (frequencies.length > 0) {
    const matching = frequencies.some((option) => parseInt(option, 10) === program.frequency_per_week);
    if (!matching) {
      return false;
    }
  }

  const levels = selected["level"] ?? [];
  if (levels.length > 0 && !levels.includes(program.level ?? "")) {
    return false;
  }

  const durations = selected["duration"] ?? [];
  if (durations.length > 0) {
    if (program.duration_weeks === null) {
      return false;
    }
    const matching = durations.some((option) => {
      const range = DURATION_RANGES[option];
      return range && program.duration_weeks !== null && program.duration_weeks >= range[0] && program.duration_weeks <= range[1];
    });
    if (!matching) {
      return false;
    }
  }

  return true;
};

export const GenericPlanMarketplacePage = () => {
  const [selected, setSelected] = useState<SelectedFilters>(EMPTY_SELECTION);
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(
    () => window.matchMedia("(max-width: 1100px)").matches
  );
  const [programs, setPrograms] = useState<MarketplaceProgram[]>([]);
  const [totalPrograms, setTotalPrograms] = useState(0);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");
    try {
      const result = await fetchAllMarketplacePrograms();
      setPrograms(result.programs);
      setTotalPrograms(result.total);
    } catch {
      setPrograms([]);
      setTotalPrograms(0);
      setErrorMessage("Unable to load the training plan catalogue. Please try again.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const activeFilterCount = Object.values(selected).reduce(
    (total, options) => total + options.length,
    0
  );

  const filtered = programs.filter((program) => matchesFilters(program, selected));
  const resultCount = activeFilterCount > 0 ? filtered.length : totalPrograms;

  const rowPrograms = (row: (typeof MARKETPLACE_ROWS)[number]): MarketplaceProgram[] => {
    let candidates: MarketplaceProgram[];
    if (row.source === "recent") {
      candidates = [...filtered].sort(byUpdatedDesc);
    } else if (row.source === "newest") {
      candidates = [...filtered].sort(byCreatedDesc);
    } else {
      candidates = filtered;
    }
    const trainingTypes = row.trainingTypes;
    if (trainingTypes) {
      candidates = candidates.filter((program) =>
        trainingTypes.includes(program.training_type ?? "")
      );
    }
    return candidates;
  };

  const toggleFilter = (groupId: string, option: string) => {
    setSelected((prev) => {
      const current = prev[groupId] ?? [];
      const next = current.includes(option)
        ? current.filter((item) => item !== option)
        : [...current, option];
      return { ...prev, [groupId]: next };
    });
  };

  const closeFilters = () => setFiltersOpen(false);

  return (
    <PageWrapper className={styles.page}>
      <section className={styles.marketplace}>
        <Container size="wide" className={styles.container}>
          <SearchBar
            onFiltersClick={() => setFiltersOpen((open) => !open)}
            totalPrograms={resultCount}
          />

          <div className={styles.contentArea}>
            <header className={styles.heading}>
              <h1 className={styles.title}>Training plans</h1>
              <p className={styles.subtitle}>
                Find a program built around your goals, schedule and experience.
              </p>
            </header>

            {activeFilterCount > 0 ? (
              <div className={styles.filterResults}>
                <Check strokeWidth={2.2} aria-hidden="true" />
                <span>
                  {resultCount} result{resultCount !== 1 ? "s" : ""} for your filters
                  <span className={styles.filterCount}>
                    ({activeFilterCount} filter{activeFilterCount !== 1 ? "s" : ""} active)
                  </span>
                </span>
              </div>
            ) : null}
          </div>

          <div
            className={joinClassNames(
              styles.layout,
              sidebarCollapsed && styles.layoutCollapsed
            )}
          >
            <div className={styles.filtersRail}>
              {sidebarCollapsed ? (
                <button
                  type="button"
                  className={styles.collapsedToggle}
                  onClick={() => setSidebarCollapsed(false)}
                  aria-label="Expand filters"
                >
                  <PanelLeftOpen strokeWidth={2} aria-hidden="true" />
                  <span>Filters</span>
                </button>
              ) : (
                <aside className={styles.panel} aria-label="Filters">
                  <div className={styles.panelHeader}>
                    <span className={styles.panelTitle}>Filters</span>
                    <button
                      type="button"
                      className={styles.collapseButton}
                      onClick={() => setSidebarCollapsed(true)}
                      aria-label="Collapse filters"
                    >
                      <PanelLeftClose strokeWidth={2} aria-hidden="true" />
                    </button>
                  </div>
                  <div className={styles.panelScroll}>
                    <FilterSidebar
                      groups={MARKETPLACE_FILTER_GROUPS}
                      selected={selected}
                      onToggle={toggleFilter}
                      onClear={() => setSelected(EMPTY_SELECTION)}
                    />
                  </div>
                </aside>
              )}
            </div>

            <main className={styles.programs}>
              {loading ? (
                <div className={styles.pageState}>
                  <p>Loading training plans…</p>
                </div>
              ) : errorMessage ? (
                <div className={styles.pageState}>
                  <p>{errorMessage}</p>
                  <div className={styles.retry}>
                    <Button variant="secondary" size="small" onClick={() => void load()}>
                      Retry
                    </Button>
                  </div>
                </div>
              ) : totalPrograms === 0 ? (
                <div className={styles.pageState}>
                  <p>No training plans have been published yet.</p>
                </div>
              ) : (
                <>
                  <div className={styles.rows}>
                    {MARKETPLACE_ROWS.map((row) => {
                      const programsInRow = rowPrograms(row);
                      if (programsInRow.length === 0) {
                        return null;
                      }
                      return (
                        <ProgramRow
                          key={row.id}
                          title={row.title}
                          description={row.description}
                          programs={programsInRow}
                        />
                      );
                    })}
                  </div>

                  <div className={styles.endOfCatalog}>
                    <div className={styles.endDivider} />
                    <p className={styles.endText}>
                      You have seen all {resultCount} training programs
                    </p>
                  </div>
                </>
              )}
            </main>
          </div>
        </Container>

        <div className={styles.mobileFilters}>
          <div
            className={joinClassNames(styles.backdrop, filtersOpen && styles.backdropVisible)}
            onClick={closeFilters}
            aria-hidden="true"
          />
          <aside
            className={joinClassNames(styles.drawer, filtersOpen && styles.drawerOpen)}
            aria-label="Filters"
            aria-hidden={!filtersOpen}
            inert={!filtersOpen}
          >
            <div className={styles.drawerHeader}>
              <span className={styles.drawerTitle}>Filters</span>
              <button
                type="button"
                className={styles.drawerClose}
                onClick={closeFilters}
                aria-label="Close filters"
              >
                <X strokeWidth={2} aria-hidden="true" />
              </button>
            </div>
            <div className={styles.drawerBody}>
              <FilterSidebar
                groups={MARKETPLACE_FILTER_GROUPS}
                selected={selected}
                onToggle={toggleFilter}
                onClear={() => setSelected(EMPTY_SELECTION)}
              />
            </div>
          </aside>
        </div>
      </section>
    </PageWrapper>
  );
};