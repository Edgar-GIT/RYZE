import { CalendarDays, CreditCard, FlaskConical, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory } from "react-router-dom";

import { AccountNav } from "@/components/account_nav/account_nav";
import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Button } from "@/components/button/button";
import { LoadingScreen } from "@/components/loading_screen/loading_screen";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { fetchMyPurchases, type PurchaseHistoryEntry } from "@/services/purchases_api";
import { ApiError } from "@utils/http_client";
import { joinClassNames } from "@utils/class_names";

import styles from "./account_purchase_history_page.module.css";

type State =
  | { status: "loading" }
  | { status: "error" }
  | { status: "empty" }
  | { status: "ready"; entries: PurchaseHistoryEntry[] };

// The backend only ever emits pending, completed and failed. No other status
// can appear; cancelled/missing payments stay pending until resolved or marked
// failed by the backend.
const STATUS_LABEL: Record<string, string> = {
  pending: "Pending",
  completed: "Completed",
  failed: "Failed"
};

const formatPrice = (minorUnits: number, currency: string): string =>
  new Intl.NumberFormat("en-GB", {
    style: "currency",
    currency: currency || "EUR"
  }).format(minorUnits / 100);

const formatDate = (isoDate: string): string =>
  new Date(isoDate).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric"
  });

export const AccountPurchaseHistoryPage = () => {
  const history = useHistory();
  const [state, setState] = useState<State>({ status: "loading" });
  const [retryCount, setRetryCount] = useState(0);

  const load = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const entries = await fetchMyPurchases();
      setState(
        entries.length === 0 ? { status: "empty" } : { status: "ready", entries }
      );
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        history.replace("/login");
        return;
      }
      setState({ status: "error" });
    }
  }, [history]);

  useEffect(() => {
    void load();
  }, [load, retryCount]);

  return (
    <PageWrapper className={styles.page}>
      <AnimatedBackground />
      <AccountNav />

      <main className={styles.main}>
        {state.status === "loading" ? <LoadingScreen /> : null}

        {state.status === "error" ? (
          <section className={styles.stateCard}>
            <p>Unable to load your purchase history.</p>
            <Button
              variant="secondary"
              size="small"
              onClick={() => setRetryCount((count) => count + 1)}
              icon={<RefreshCw size={15} />}
            >
              Try again
            </Button>
          </section>
        ) : null}

        {state.status === "empty" ? (
          <section className={styles.stateCard}>
            <p className={styles.stateTitle}>You have not purchased any programs yet.</p>
            <p className={styles.stateText}>
              Browse the marketplace to find your next training plan.
            </p>
            <Button to="/services/generic-program" variant="primary" size="small">
              Browse training plans
            </Button>
          </section>
        ) : null}

        {state.status === "ready" ? (
          <section className={styles.content}>
            <header className={styles.heading}>
              <h1 className={styles.title}>Purchase History</h1>
              <p className={styles.subtitle}>Your training plan purchases, most recent first.</p>
            </header>

            <ul className={styles.list}>
              {state.entries.map((entry) => {
                const statusLabel = STATUS_LABEL[entry.status] ?? entry.status;
                const canOpen = entry.status === "completed" && entry.access;
                const programUnavailable = entry.status === "completed" && !entry.access;
                return (
                  <li key={entry.id} className={styles.entry}>
                    <div className={styles.entryMain}>
                      <span className={styles.entryProgram}>{entry.program.name}</span>
                      {entry.program.description ? (
                        <span className={styles.entryDescription}>{entry.program.description}</span>
                      ) : null}
                      <span className={styles.entryMeta}>
                        <CalendarDays size={13} aria-hidden="true" />
                        {formatDate(entry.created_at)}
                      </span>
                    </div>

                    <div className={styles.entrySide}>
                      <span className={styles.entryPrice}>
                        <CreditCard size={13} aria-hidden="true" />
                        {formatPrice(entry.price_minor_units, entry.currency)}
                      </span>

                      <span
                        className={joinClassNames(
                          styles.statusBadge,
                          entry.status === "completed" && styles.statusCompleted,
                          entry.status === "pending" && styles.statusPending,
                          entry.status === "failed" && styles.statusFailed
                        )}
                      >
                        {statusLabel}
                      </span>

                      {entry.test ? (
                        <span className={styles.testBadge}>
                          <FlaskConical size={12} aria-hidden="true" />
                          Test purchase
                        </span>
                      ) : null}

                      {canOpen ? (
                        <Button
                          to={`/services/my-programs/${entry.program_id}`}
                          variant="secondary"
                          size="small"
                        >
                          Open program
                        </Button>
                      ) : null}

                      {programUnavailable ? (
                        <span className={styles.unavailable}>No longer available</span>
                      ) : null}
                    </div>
                  </li>
                );
              })}
            </ul>
          </section>
        ) : null}
      </main>
    </PageWrapper>
  );
};