import { useCallback, useEffect, useState } from "react";
import { Info, RefreshCw, Users } from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2Table } from "@/components/admin2/admin2_table/admin2_table";
import { Admin2Pagination } from "@/components/admin2/admin2_pagination/admin2_pagination";
import { Button } from "@/components/button/button";
import { fetchAdminUsers, fullName, type AdminUser } from "@/services/admin_api";
import { formatAdminDate } from "@/services/admin_api";

import styles from "./admin2_clients_page.module.css";

export default function Admin2ClientsPage() {
  const [clients, setClients] = useState<AdminUser[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const load = useCallback(async (targetPage: number) => {
    setLoading(true);
    setErrorMessage("");
    try {
      const result = await fetchAdminUsers(targetPage, 20);
      setClients(result.users);
      setTotal(result.pagination.total);
      setTotalPages(result.pagination.total_pages);
      setPage(result.pagination.page);
    } catch {
      setErrorMessage("Unable to load clients. Please try again.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load(1);
  }, [load]);

  return (
    <>
      <Admin2PageHeader
        eyebrow="People"
        title="Clients"
        description="The RYZE member base. This is a read-only analytical view."
      />

      <div className={styles.notice}>
        <Info size={15} className={styles.noticeIcon} aria-hidden="true" />
        <span>
          <strong>Read-only view.</strong> Membership tiers, subscription status and lifetime
          purchase value will appear here when the corresponding analytics are implemented.
        </span>
      </div>

      <Admin2Section title="Members" subtitle={`${total} registered client${total === 1 ? "" : "s"}`}>
        {loading ? (
          <div className={styles.pageState}>
            <p className={styles.muted}>Loading clients…</p>
          </div>
        ) : errorMessage ? (
          <div className={styles.pageState}>
            <p className={styles.muted}>{errorMessage}</p>
            <div style={{ marginTop: "0.75rem" }}>
              <Button variant="secondary" size="small" onClick={() => void load(page)} icon={<RefreshCw size={15} />}>
                Retry
              </Button>
            </div>
          </div>
        ) : (
          <Admin2Table
            columns={[
              {
                key: "name",
                label: "Client",
                render: (user: AdminUser) => {
                  const name = fullName(user);
                  return name !== "—" ? name : <span className={styles.muted}>—</span>;
                }
              },
              {
                key: "email",
                label: "Email",
                className: styles.muted,
                render: (user: AdminUser) => user.email
              },
              {
                key: "created",
                label: "Registered",
                className: styles.muted,
                render: (user: AdminUser) => formatAdminDate(user.created_at)
              }
            ]}
            rows={clients}
            rowKey={(user) => user.id}
            emptyRow={
              <span className={styles.muted}>
                <Users
                  size={28}
                  aria-hidden="true"
                  style={{ display: "block", margin: "0 auto 0.5rem", opacity: 0.5 }}
                />
                No clients registered yet.
              </span>
            }
          />
        )}

        {!loading && !errorMessage ? (
          <Admin2Pagination
            page={page}
            totalPages={totalPages}
            pageSize={20}
            totalItems={total}
            onChange={(nextPage) => void load(nextPage)}
          />
        ) : null}
      </Admin2Section>
    </>
  );
}