import { Check, PanelLeftClose, PanelLeftOpen, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { FilterSidebar } from "@/components/filter_sidebar/filter_sidebar";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ProgramRow } from "@/components/program_row/program_row";
import { SearchBar } from "@/components/search_bar/search_bar";
import { useTestMode } from "@/components/test_mode/test_mode_context";
import {
  DURATION_RANGES,
  MARKETPLACE_FILTER_GROUPS,
  MARKETPLACE_ROWS
} from "@/constants/marketplace";
import {
  fetchMarketplacePrograms,
  type MarketplaceFilters,
  type MarketplaceProgram
} from "@/services/marketplace_api";
import { fetchEntitlements } from "@/services/purchases_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./generic_plan_marketplace_page.module.css";

type SelectedFilters = Record<string, string[]>;

const EMPTY_SELECTION: SelectedFilters = {};

// Each section shows a bounded page instead of loading the whole catalogue.
// The backend applies the filters and ordering, so the client never has to
// fetch every program to discover a match.
const ROW_LIMIT = 12;

const filtersFromSelection = (selected: SelectedFilters): MarketplaceFilters => {
  const frequencies = (selected["frequency"] ?? [])
    .map((option) => parseInt(option, 10))
    .filter((value) => Number.isFinite(value));
  const durationRanges: Array<[number, number]> = [];
  for (const option of selected["duration"] ?? []) {
    const range = DURATION_RANGES[option];
    if (range) {
      durationRanges.push(range);
    }
  }
  return {
    trainingTypes: selected["training-type"] ?? [],
    levels: selected["level"] ?? [],
    frequencies,
    durationRanges
  };
};

export const GenericPlanMarketplacePage = () => {
  const { isActive: testModeActive } = useTestMode();
  const [selected, setSelected] = useState<SelectedFilters>(EMPTY_SELECTION);
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(
    () => window.matchMedia("(max-width: 1100px)").matches
  );
  const [rows, setRows] = useState<Record<string, MarketplaceProgram[]>>({});
  const [totalPrograms, setTotalPrograms] = useState(0);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [ownedProgramIds, setOwnedProgramIds] = useState<ReadonlySet<string>>(
    () => new Set()
  );

  useEffect(() => {
    let cancelled = false;
    void fetchEntitlements()
      .then((entitlements) => {
        if (!cancelled) {
          setOwnedProgramIds(new Set(entitlements.map((entry) => entry.program_id)));
        }
      })
      .catch(() => {
        // Not signed in (or fetching entitlements failed) means no owned
        // programs. The marketplace stays fully usable without them.
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const filters = useMemo(() => filtersFromSelection(selected), [selected]);

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");
    try {
      const results = await Promise.all(
        MARKETPLACE_ROWS.map(async (row) => {
          const selectedTypes = filters.trainingTypes ?? [];
          let trainingTypes = filters.trainingTypes;
          if (row.trainingTypes) {
            const intersection = selectedTypes.length
              ? row.trainingTypes.filter((type) => selectedTypes.includes(type))
              : row.trainingTypes;
            if (intersection.length === 0) {
              return { row, programs: [], total: null };
            }
            trainingTypes = intersection;
          }

          const result = await fetchMarketplacePrograms({
            ...filters,
            trainingTypes,
            sort: row.source === "recent" ? "updated_at" : "created_at",
            order: "desc",
            page: 1,
            limit: ROW_LIMIT
          });
          return { row, programs: result.programs, total: result.total };
        })
      );

      const nextRows: Record<string, MarketplaceProgram[]> = {};
      for (const result of results) {
        nextRows[result.row.id] = result.programs;
      }
      setRows(nextRows);
      // The first row has no training-type restriction, so its total is the
      // number of programs matching every active filter.
      setTotalPrograms(results[0]?.total ?? 0);
    } catch {
      setRows({});
      setTotalPrograms(0);
      setErrorMessage("Unable to load the training plan catalogue. Please try again.");
    } finally {
      setLoading(false);
    }
  }, [filters]);

  useEffect(() => {
    void load();
  }, [load]);

  const activeFilterCount = Object.values(selected).reduce(
    (total, options) => total + options.length,
    0
  );

  const resultCount = totalPrograms;

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
                  <p>
                    {activeFilterCount > 0
                      ? "No training plans match your filters."
                      : "No training plans have been published yet."}
                  </p>
                </div>
              ) : (
                <>
                  <div className={styles.rows}>
                    {MARKETPLACE_ROWS.map((row) => {
                      const programsInRow = rows[row.id] ?? [];
                      if (programsInRow.length === 0) {
                        return null;
                      }
                      return (
                        <ProgramRow
                          key={row.id}
                          title={row.title}
                          description={row.description}
                          programs={programsInRow}
                          forceFree={testModeActive}
                          ownedProgramIds={ownedProgramIds}
                        />
                      );
                    })}
                  </div>

                  <div className={styles.endOfCatalog}>
                    <div className={styles.endDivider} />
                    <p className={styles.endText}>
                      Showing up to {ROW_LIMIT} programs per section · {resultCount} program
                      {resultCount !== 1 ? "s" : ""} match
                      {activeFilterCount > 0 ? " your filters" : ""}
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