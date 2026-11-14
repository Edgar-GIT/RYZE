import { CalendarDays, CreditCard, FlaskConical, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useHistory } from "react-router-dom";

import { Button } from "@/components/button/button";
import { AppLoading } from "@/components/app_shell/app_loading";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useI18n } from "@/i18n/i18n";
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
const STATUS_KEY = {
  pending: "account.purchases.pending",
  completed: "account.purchases.completed",
  failed: "account.purchases.failed"
} as const;

// Brand names stay as they are; only "Card" is translated.
const METHOD_LABEL: Record<string, string> = {
  mbway: "MB WAY",
  paypal: "PayPal"
};

export const AccountPurchaseHistoryPage = () => {
  const { t, formatDate, formatMoney } = useI18n();
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

      <div className={styles.main}>
        {state.status === "loading" ? <AppLoading /> : null}

        {state.status === "error" ? (
          <section className={styles.stateCard}>
            <p>{t("account.purchases.loadError")}</p>
            <Button
              variant="secondary"
              size="small"
              onClick={() => setRetryCount((count) => count + 1)}
              icon={<RefreshCw size={15} />}
            >
              {t("common.retry")}
            </Button>
          </section>
        ) : null}

        {state.status === "empty" ? (
          <section className={styles.stateCard}>
            <p className={styles.stateTitle}>{t("account.purchases.emptyTitle")}</p>
            <p className={styles.stateText}>
              {t("account.purchases.emptyText")}
            </p>
            <Button to="/services/generic-program" variant="primary" size="small">
              {t("account.purchases.browse")}
            </Button>
          </section>
        ) : null}

        {state.status === "ready" ? (
          <section className={styles.content}>
            <header className={styles.heading}>
              <h1 className={styles.title}>{t("account.purchases.title")}</h1>
              <p className={styles.subtitle}>{t("account.purchases.subtitle")}</p>
            </header>

            <ul className={styles.list}>
              {state.entries.map((entry) => {
                const statusKey = STATUS_KEY[entry.status as keyof typeof STATUS_KEY];
                const statusLabel = statusKey ? t(statusKey) : entry.status;
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
                        {formatDate(entry.created_at, { year: "numeric", month: "short", day: "numeric" })}
                      </span>
                    </div>

                    <div className={styles.entrySide}>
                      <span className={styles.entryPrice}>
                        <CreditCard size={13} aria-hidden="true" />
                        {formatMoney(entry.price_minor_units, entry.currency || "EUR")}
                      </span>

                      {!entry.test && entry.payment_method ? (
                        <span className={styles.methodBadge}>
                          {entry.payment_method === "card"
                            ? t("account.purchases.card")
                            : (METHOD_LABEL[entry.payment_method] ?? entry.payment_method)}
                        </span>
                      ) : null}

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
                          {t("account.purchases.test")}
                        </span>
                      ) : null}

                      {canOpen ? (
                        <Button
                          to={`/services/my-programs/${entry.program_id}`}
                          variant="secondary"
                          size="small"
                        >
                          {t("account.purchases.open")}
                        </Button>
                      ) : null}

                      {programUnavailable ? (
                        <span className={styles.unavailable}>{t("account.purchases.unavailable")}</span>
                      ) : null}
                    </div>
                  </li>
                );
              })}
            </ul>
          </section>
        ) : null}
      </div>
    </PageWrapper>
  );
};