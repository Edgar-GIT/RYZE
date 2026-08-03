import { GlassPanel } from "@/components/glass_panel/glass_panel";

import styles from "./product_demos.module.css";

const trend = [62, 64, 63, 67, 70, 69, 74, 78, 76, 82, 85, 88];
const measurements = [
  { label: "Body weight", value: "74.2 kg", delta: "-2.1" },
  { label: "Bench PR", value: "92.5 kg", delta: "+7.5" },
  { label: "Waist", value: "78 cm", delta: "-4" }
] as const;

export const ProgressDemo = () => {
  const max = Math.max(...trend);
  const points = trend
    .map((value, index) => {
      const x = (index / (trend.length - 1)) * 100;
      const y = 100 - (value / max) * 78 - 10;
      return `${x},${y}`;
    })
    .join(" ");

  return (
    <GlassPanel className={styles.window} as="article">
      <div className={styles.windowChrome}>
        <span />
        <span />
        <span />
        <p>Progress · 12 weeks</p>
      </div>

      <div className={styles.progressLayout}>
        <div className={styles.trendCard}>
          <div className={styles.chartHeader}>
            <span>Strength index</span>
            <em>+26%</em>
          </div>
          <svg className={styles.trendChart} viewBox="0 0 100 100" preserveAspectRatio="none">
            <polygon className={styles.trendArea} points={`0,100 ${points} 100,100`} />
            <polyline className={styles.trendLine} fill="none" points={points} />
          </svg>
        </div>

        <div className={styles.measureGrid}>
          {measurements.map((item) => (
            <div key={item.label} className={styles.measureCard}>
              <span>{item.label}</span>
              <strong>{item.value}</strong>
              <em>{item.delta}</em>
            </div>
          ))}
        </div>
      </div>
    </GlassPanel>
  );
};
