import { Info, Settings } from "lucide-react";

import { Admin2PageHeader } from "@/components/admin2/admin2_page_header/admin2_page_header";
import { Admin2Section } from "@/components/admin2/admin2_section/admin2_section";
import { Admin2StatusBadge } from "@/components/admin2/admin2_status_badge/admin2_status_badge";

import styles from "./admin2_settings_page.module.css";

const CONFIG_GROUPS = ["Currency", "Pricing floor", "Commission default", "Marketplace regions"];

export default function Admin2SettingsPage() {
  return (
    <>
      <Admin2PageHeader
        eyebrow="Configuration"
        title="Business settings"
        description="The commercial parameters that govern the RYZE marketplace."
      />

      <div className={styles.notice}>
        <Info size={15} className={styles.noticeIcon} aria-hidden="true" />
        <span>
          <strong>Read-only configuration.</strong> These parameters are enforced server-side and
          are not editable from the console yet.
        </span>
      </div>

      <Admin2Section title="Commercial configuration" subtitle="Platform-wide business rules">
        <div className={styles.list}>
          <div className={styles.row}>
            <p className={styles.rowLabel}>Marketplace currency</p>
            <p className={styles.rowValue}>EUR</p>
          </div>
          <div className={styles.row}>
            <p className={styles.rowLabel}>Minimum paid program price</p>
            <p className={styles.rowValue}>€1.00</p>
          </div>
          <div className={styles.row}>
            <p className={styles.rowLabel}>Default platform commission</p>
            <p className={styles.rowValue}>20% (2000 bps)</p>
          </div>
          <div className={styles.row}>
            <p className={styles.rowLabel}>Commission cap</p>
            <p className={styles.rowValue}>100% (10000 bps)</p>
          </div>
          <div className={styles.row}>
            <p className={styles.rowLabel}>Payment region</p>
            <div className={styles.rowValue}>
              <Admin2StatusBadge label="Europe" tone="success" withDot={false} />
            </div>
          </div>
        </div>
      </Admin2Section>

      <div style={{ height: "0.75rem" }} />

      <Admin2Section title="Configurable groups" subtitle="Identifiers for future settings screens">
        <div className={styles.pills}>
          {CONFIG_GROUPS.map((group) => (
            <span key={group} className={styles.pill}>
              {group}
            </span>
          ))}
        </div>
      </Admin2Section>

      <p className={styles.notice} style={{ marginTop: "0.75rem", flexDirection: "row" }}>
        <Settings size={15} className={styles.noticeIcon} aria-hidden="true" />
        <span>Settings persistence will connect to a dedicated configuration endpoint in a later milestone.</span>
      </p>
    </>
  );
}