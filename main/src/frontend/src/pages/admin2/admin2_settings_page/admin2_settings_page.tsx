import { Info, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge } from "@/components/admin2/admin2_status_badge/admin2_status_badge";
import { Button } from "@/components/button/button";
import { apiGet } from "@utils/http_client";

import styles from "./admin2_settings_page.module.css";

interface CommerceSettings {
  currency: string;
  min_program_price_minor_units: number;
  default_platform_commission_bps: number;
  max_commission_bps: number;
  payment_methods: string[];
}

const METHOD_LABEL: Record<string, string> = { card: "Card", mbway: "MB WAY", paypal: "PayPal" };

const formatMoney = (minorUnits: number, currency: string) =>
  new Intl.NumberFormat("en-GB", { style: "currency", currency }).format(minorUnits / 100);

const formatBps = (bps: number) => `${(bps / 100).toLocaleString("en-GB", { maximumFractionDigits: 2 })}% (${bps} bps)`;

// Admin2SettingsPage shows the commercial parameters the server enforces,
// read from the deployment configuration (they are changed there, not here).
export default function Admin2SettingsPage() {
  const [settings, setSettings] = useState<CommerceSettings | null>(null);
  const [error, setError] = useState(false);

  const load = useCallback(async () => {
    setError(false);
    try {
      setSettings(await apiGet<CommerceSettings>("/admin/settings/commerce"));
    } catch {
      setError(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <>
      <Admin2PageHeader
        eyebrow="Configuration"
        title="Business settings"
        description="The commercial parameters the server enforces for every sale."
      />

      <div className={styles.notice}>
        <Info size={15} className={styles.noticeIcon} aria-hidden="true" />
        <span>
          <strong>Read from the server configuration.</strong> They change through the deployment settings, so
          every sale uses exactly these values. Trainer-specific commissions are set on each trainer.
        </span>
      </div>

      <Admin2Section title="Commercial configuration" subtitle="Platform-wide business rules in force">
        {error ? (
          <div className={styles.list}>
            <p className={styles.rowLabel}>Settings could not be loaded.</p>
            <Button variant="secondary" size="small" icon={<RefreshCw size={15} />} onClick={() => void load()}>
              Try again
            </Button>
          </div>
        ) : !settings ? (
          <div className={styles.list}>
            <p className={styles.rowLabel}>Loading settings…</p>
          </div>
        ) : (
          <div className={styles.list}>
            <div className={styles.row}>
              <p className={styles.rowLabel}>Marketplace currency</p>
              <p className={styles.rowValue}>{settings.currency}</p>
            </div>
            <div className={styles.row}>
              <p className={styles.rowLabel}>Minimum paid program price</p>
              <p className={styles.rowValue}>{formatMoney(settings.min_program_price_minor_units, settings.currency)}</p>
            </div>
            <div className={styles.row}>
              <p className={styles.rowLabel}>Default platform commission</p>
              <p className={styles.rowValue}>{formatBps(settings.default_platform_commission_bps)}</p>
            </div>
            <div className={styles.row}>
              <p className={styles.rowLabel}>Commission cap</p>
              <p className={styles.rowValue}>{formatBps(settings.max_commission_bps)}</p>
            </div>
            <div className={styles.row}>
              <p className={styles.rowLabel}>Payment methods enabled</p>
              <div className={styles.rowValue}>
                {settings.payment_methods.length === 0 ? (
                  <Admin2StatusBadge label="None configured" tone="warning" withDot={false} />
                ) : (
                  settings.payment_methods.map((method) => (
                    <Admin2StatusBadge key={method} label={METHOD_LABEL[method] ?? method} tone="success" withDot={false} />
                  ))
                )}
              </div>
            </div>
          </div>
        )}
      </Admin2Section>
    </>
  );
}
