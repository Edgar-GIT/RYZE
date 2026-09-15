import { RefreshCw, ShieldOff } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/button/button";
import {
  disableAdminTrainer,
  fetchAdminDeletedTrainers,
  fetchAdminTrainers,
  fullName,
  formatAdminDate,
  reactivateAdminTrainer,
  type AdminTrainer
} from "@/services/admin_api";
import { joinClassNames } from "@utils/class_names";

import styles from "./admin_trainers_page.module.css";

const TRAINERS_PER_PAGE = 20;

type ActiveTab = "active" | "deleted";

export const AdminTrainersPage = () => {
  const [activeTab, setActiveTab] = useState<ActiveTab>("active");
  const [page, setPage] = useState(1);
  const [trainers, setTrainers] = useState<AdminTrainer[]>([]);
  const [pagination, setPagination] = useState<{ page: number; total_pages: number; total: number } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (targetPage: number, tab: ActiveTab) => {
    setLoading(true);
    setError(null);
    try {
      const result = tab === "active"
        ? await fetchAdminTrainers(targetPage, TRAINERS_PER_PAGE)
        : await fetchAdminDeletedTrainers(targetPage, TRAINERS_PER_PAGE);
      setTrainers(result.trainers);
      setPagination(result.pagination);
      setPage(result.pagination.page);
    } catch {
      setError("Unable to load trainers.");
      setTrainers([]);
      setPagination(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(page, activeTab);
  }, [activeTab, page, load]);

  const handleDisable = async (id: string) => {
    try {
      await disableAdminTrainer(id);
      await load(page, activeTab);
    } catch {
      setError("Unable to disable trainer.");
    }
  };

  const handleReactivate = async (id: string) => {
    try {
      await reactivateAdminTrainer(id);
      await load(page, activeTab);
    } catch {
      setError("Unable to reactivate trainer.");
    }
  };

  return (
    <div className={styles.trainersPage}>
      <header className={styles.header}>
        <div className={styles.headerText}>
          <p className={styles.eyebrow}>Management</p>
          <h1 className={styles.title}>Trainers</h1>
          <p className={styles.description}>
            Active trainers and disabled trainer identities.
          </p>
        </div>
      </header>

      <div className={styles.tabs} role="tablist" aria-label="Trainer tabs">
        {(["active", "deleted"] as const).map((tab) => (
          <button
            key={tab}
            type="button"
            role="tab"
            aria-selected={activeTab === tab}
            className={joinClassNames(styles.tab, activeTab === tab && styles.tabActive)}
            onClick={() => { setActiveTab(tab); setPage(1); }}
          >
            {tab === "active" ? "Active" : "Deleted"}
          </button>
        ))}
      </div>

      {error ? (
        <p className={styles.errorBanner} role="alert">{error}</p>
      ) : null}

      {loading ? (
        <div className={styles.tableCard}>
          <p className={styles.loadingBanner}>Loading trainers…</p>
        </div>
      ) : trainers.length === 0 ? (
        <div className={styles.tableCard}>
          <div className={styles.emptyState}>
            <p className={styles.emptyStateEmphasis}>
              {activeTab === "deleted" ? "No disabled trainers." : "No trainers yet."}
            </p>
          </div>
        </div>
      ) : (
        <>
          <div className={styles.tableCard}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Name</th>
                  <th scope="col">Email</th>
                  <th scope="col">Status</th>
                  <th scope="col">Created</th>
                  <th scope="col" aria-label="Actions" style={{ width: "5rem" }} />
                </tr>
              </thead>
              <tbody>
                {trainers.map((trainer) => (
                  <tr key={trainer.id}>
                    <td className={styles.nameCell}>{fullName(trainer)}</td>
                    <td className={styles.emailCell}>{trainer.email}</td>
                    <td>
                      <span className={joinClassNames(styles.statusBadge, trainer.status === "active" ? styles.statusActive : styles.statusDisabled)}>
                        {trainer.status === "active" ? "Active" : "Disabled"}
                      </span>
                    </td>
                    <td>{formatAdminDate(trainer.created_at)}</td>
                    <td>
                      <div className={styles.actionsCell}>
                        {activeTab === "active" ? (
                          <button
                            type="button"
                            className={joinClassNames(styles.iconButton, styles.iconButtonDanger)}
                            aria-label={`Disable ${trainer.email}`}
                            title="Disable trainer"
                            onClick={() => handleDisable(trainer.id)}
                          >
                            <ShieldOff aria-hidden="true" />
                          </button>
                        ) : (
                          <button
                            type="button"
                            className={styles.iconButton}
                            aria-label={`Reactivate ${trainer.email}`}
                            title="Reactivate trainer"
                            onClick={() => handleReactivate(trainer.id)}
                          >
                            <RefreshCw aria-hidden="true" />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {pagination && pagination.total_pages > 1 ? (
            <div className={styles.pagination}>
              <p className={styles.paginationInfo}>Page {pagination.page} of {pagination.total_pages}</p>
              {pagination.page > 1 ? (
                <Button type="button" variant="ghost" onClick={() => setPage((p) => Math.max(1, p - 1))}>
                  Previous
                </Button>
              ) : null}
              {pagination.page < pagination.total_pages ? (
                <Button type="button" variant="ghost" onClick={() => setPage((p) => p + 1)}>
                  Next
                </Button>
              ) : null}
            </div>
          ) : null}
        </>
      )}
    </div>
  );
};