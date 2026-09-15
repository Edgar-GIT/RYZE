import { useEffect, useState } from "react";
import { AlertTriangle, Banknote, Receipt, RotateCcw, ShoppingBag } from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2MetricCard } from "@/components/admin2/admin2_metric_card/admin2_metric_card";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2Table } from "@/components/admin2/admin2_table/admin2_table";
import { fetchAdminPurchases, type AdminPurchase } from "@/services/admin2_api";

import styles from "./admin2_sales_page.module.css";

export default function Admin2SalesPage() {
  const [result, setResult] = useState<Awaited<ReturnType<typeof fetchAdminPurchases>> | null>(null);

  useEffect(() => {
    let cancelled = false;

    fetchAdminPurchases().then((res) => {
      if (!cancelled) {
        setResult(res);
      }
    });

    return () => {
      cancelled = true;
    };
  }, []);

  const unavailable = result === null || !result.available;

  return (
    <>
      <Admin2PageHeader
        eyebrow="Business"
        title="Sales"
        description="Revenue, transactions and refunds across the platform."
      />

      {unavailable ? (
        <div className={styles.notice}>
          <AlertTriangle size={15} className={styles.noticeIcon} aria-hidden="true" />
          <span>
            <strong>{result?.reason ?? "Loading sales data…"}</strong> The dashboard is designed for
            the future purchase analytics API. In the meantime no revenue is fabricated.
          </span>
        </div>
      ) : null}

      <div className={styles.metrics}>
        <Admin2MetricCard label="Revenue" value="—" hint="Awaiting the admin sales API" icon={Banknote} />
        <Admin2MetricCard label="Transactions" value="—" hint="Completed purchases only" icon={Receipt} />
        <Admin2MetricCard label="Refunds" value="—" hint="Amount refunded this period" icon={RotateCcw} />
      </div>

      <Admin2Section title="Recent transactions" subtitle="The purchases model this page will render">
        <Admin2Table<AdminPurchase>
          columns={[
            { key: "order", label: "Order", render: () => <span className={styles.muted}>—</span> },
            { key: "client", label: "Client", render: () => <span className={styles.muted}>—</span> },
            { key: "program", label: "Program", render: () => <span className={styles.muted}>—</span> },
            { key: "amount", label: "Amount", render: () => <span className={styles.muted}>—</span> },
            { key: "status", label: "Status", render: () => <span className={styles.muted}>—</span> }
          ]}
          rows={[]}
          rowKey={(purchase) => purchase.id}
          emptyRow={
            <span className={styles.muted}>
              <ShoppingBag
                size={28}
                aria-hidden="true"
                style={{ display: "block", margin: "0 auto 0.5rem", opacity: 0.5 }}
              />
              No transactions are listed until the admin sales endpoint ships.
            </span>
          }
        />
      </Admin2Section>
    </>
  );
}