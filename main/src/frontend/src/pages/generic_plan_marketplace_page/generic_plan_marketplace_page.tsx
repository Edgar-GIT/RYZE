import { Check, PanelLeftClose, PanelLeftOpen, X } from "lucide-react";
import { useState } from "react";

import { Container } from "@/components/container/container";
import { FilterSidebar } from "@/components/filter_sidebar/filter_sidebar";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ProgramRow } from "@/components/program_row/program_row";
import { SearchBar } from "@/components/search_bar/search_bar";
import { MARKETPLACE_FILTER_GROUPS, MARKETPLACE_ROWS } from "@/constants/marketplace";
import { joinClassNames } from "@utils/class_names";

import styles from "./generic_plan_marketplace_page.module.css";

type SelectedFilters = Record<string, string[]>;

const EMPTY_SELECTION: SelectedFilters = {};

const TOTAL_PROGRAMS = MARKETPLACE_ROWS.reduce(
  (total, row) => total + row.programs.length,
  0
);

export const GenericPlanMarketplacePage = () => {
  const [selected, setSelected] = useState<SelectedFilters>(EMPTY_SELECTION);
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(
    () => window.matchMedia("(max-width: 1100px)").matches
  );

  const activeFilterCount = Object.values(selected).reduce(
    (total, options) => total + options.length,
    0
  );

  const resultCount = activeFilterCount > 0 ? 0 : TOTAL_PROGRAMS;

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
                  0 results for your filters
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
              <div className={styles.rows}>
                {MARKETPLACE_ROWS.map((row) => (
                  <ProgramRow
                    key={row.id}
                    title={row.title}
                    description={row.description}
                    programs={row.programs}
                  />
                ))}
              </div>

              <div className={styles.endOfCatalog}>
                <div className={styles.endDivider} />
                <p className={styles.endText}>
                  You have seen all {TOTAL_PROGRAMS} training programs
                </p>
              </div>
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