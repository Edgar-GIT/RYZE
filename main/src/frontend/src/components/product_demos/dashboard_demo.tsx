import { GlassPanel } from "@/components/glass_panel/glass_panel";

import styles from "./product_demos.module.css";

const weekBars = [42, 68, 55, 80, 72, 90, 64];
const days = ["M", "T", "W", "T", "F", "S", "S"];

export const DashboardDemo = () => (
  <GlassPanel className={styles.window} as="article">
    <div className={styles.windowChrome}>
      <span />
      <span />
      <span />
      <p>RYZE / Dashboard</p>
    </div>

    <div className={styles.dashboardGrid}>
      <div className={styles.statCard}>
        <span>Weekly volume</span>
        <strong>12.4k kg</strong>
        <em>+18%</em>
      </div>
      <div className={styles.statCard}>
        <span>Adherence</span>
        <strong>94%</strong>
        <em>On track</em>
      </div>
      <div className={styles.statCard}>
        <span>Recovery</span>
        <strong>Good</strong>
        <em>Ready</em>
      </div>

      <div className={styles.chartCard}>
        <div className={styles.chartHeader}>
          <span>Activity</span>
          <span>7 days</span>
        </div>
        <div className={styles.bars} aria-hidden="true">
          {weekBars.map((value, index) => (
            <div key={`${days[index]}-${value}`} className={styles.barColumn}>
              <div className={styles.barTrack}>
                <div className={styles.barFill} style={{ height: `${value}%` }} />
              </div>
              <span>{days[index]}</span>
            </div>
          ))}
        </div>
      </div>

      <div className={styles.sideCard}>
        <span>Today</span>
        <strong>Upper strength</strong>
        <p>4 exercises · 62 min</p>
        <div className={styles.progressRow}>
          <div className={styles.progressFill} style={{ width: "72%" }} />
        </div>
      </div>
    </div>
  </GlassPanel>
);
