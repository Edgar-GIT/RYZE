import { useEffect, useId, useRef, useState } from "react";

import styles from "./admin_charts.module.css";

export interface TrendPoint {
  date: string;
  value: number;
}

interface TrendChartProps {
  title: string;
  points: TrendPoint[];
  format: (value: number) => string;
  // Text for one point, read by screen readers and shown on hover.
  describe: (point: TrendPoint) => string;
  tone?: "cyan" | "mint" | "violet";
}

const HEIGHT = 170;
const PAD_LEFT = 8;
const PAD_RIGHT = 8;
const PAD_TOP = 18;
const PAD_BOTTOM = 26;

const shortDate = (iso: string) => {
  const [, month, day] = iso.split("-");
  return `${day}/${month}`;
};

// TrendChart draws one value per day as slim bars with a soft area line, a
// top gridline with the period maximum and a few date labels. No library; the
// values are also listed for assistive technology.
export const TrendChart = ({ title, points, format, describe, tone = "cyan" }: TrendChartProps) => {
  const id = useId();
  const ref = useRef<HTMLElement>(null);
  const [width, setWidth] = useState(600);

  useEffect(() => {
    const element = ref.current;
    if (!element) return;
    const measure = () => setWidth(Math.max(260, Math.round(element.clientWidth)));
    measure();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const max = Math.max(1, ...points.map((point) => point.value));
  const plotWidth = width - PAD_LEFT - PAD_RIGHT;
  const plotHeight = HEIGHT - PAD_TOP - PAD_BOTTOM;
  const slot = plotWidth / Math.max(points.length, 1);
  const barWidth = Math.max(2, Math.min(18, slot * 0.58));
  const base = HEIGHT - PAD_BOTTOM;
  const labelEvery = Math.max(1, Math.ceil(points.length / (width < 420 ? 4 : 7)));
  const total = points.reduce((sum, point) => sum + point.value, 0);

  const line = points
    .map((point, index) => {
      const x = PAD_LEFT + slot * index + slot / 2;
      const y = base - (point.value / max) * plotHeight;
      return `${index === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");

  return (
    <figure ref={ref} className={styles.trend} data-tone={tone}>
      <svg viewBox={`0 0 ${width} ${HEIGHT}`} role="img" aria-labelledby={`${id}-t`}>
        <title id={`${id}-t`}>{`${title}: ${format(total)} in total`}</title>
        <defs>
          <linearGradient id={`${id}-fill`} x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" className={styles.fillStop} stopOpacity="0.32" />
            <stop offset="100%" className={styles.fillStop} stopOpacity="0" />
          </linearGradient>
        </defs>
        <line x1={PAD_LEFT} x2={width - PAD_RIGHT} y1={PAD_TOP} y2={PAD_TOP} className={styles.grid} />
        <text x={width - PAD_RIGHT} y={PAD_TOP - 6} textAnchor="end" className={styles.gridLabel}>
          {format(max)}
        </text>
        <line x1={PAD_LEFT} x2={width - PAD_RIGHT} y1={base} y2={base} className={styles.axis} />
        {points.length > 1 ? (
          <path d={`${line} L${(PAD_LEFT + slot * (points.length - 1) + slot / 2).toFixed(1)},${base} L${(PAD_LEFT + slot / 2).toFixed(1)},${base} Z`} fill={`url(#${id}-fill)`} />
        ) : null}
        {points.map((point, index) => {
          const height = (point.value / max) * plotHeight;
          const x = PAD_LEFT + slot * index + (slot - barWidth) / 2;
          return (
            <g key={point.date}>
              <title>{describe(point)}</title>
              <rect
                x={x}
                y={point.value > 0 ? base - Math.max(height, 2) : base - 2}
                width={barWidth}
                height={point.value > 0 ? Math.max(height, 2) : 2}
                rx={Math.min(4, barWidth / 2)}
                className={point.value > 0 ? styles.bar : styles.barEmpty}
              />
              {index % labelEvery === 0 || index === points.length - 1 ? (
                <text x={x + barWidth / 2} y={HEIGHT - 8} textAnchor="middle" className={styles.axisLabel}>
                  {shortDate(point.date)}
                </text>
              ) : null}
            </g>
          );
        })}
        {points.length > 1 ? <path d={line} className={styles.line} /> : null}
      </svg>
      <ul className={styles.srOnly}>
        {points.map((point) => (
          <li key={point.date}>{describe(point)}</li>
        ))}
      </ul>
    </figure>
  );
};

export interface ShareItem {
  label: string;
  value: number;
  detail?: string;
}

// ShareBars lists labelled values as horizontal bars relative to the largest.
export const ShareBars = ({ items, format, empty }: { items: ShareItem[]; format: (value: number) => string; empty: string }) => {
  const max = Math.max(0, ...items.map((item) => item.value));
  if (items.length === 0 || max === 0) {
    return <p className={styles.empty}>{empty}</p>;
  }
  return (
    <ul className={styles.shares}>
      {items.map((item) => (
        <li key={item.label} className={styles.share}>
          <div className={styles.shareHead}>
            <span>{item.label}</span>
            <strong>{format(item.value)}</strong>
          </div>
          <span className={styles.shareTrack} aria-hidden="true">
            <span className={styles.shareFill} style={{ width: `${Math.max(3, (item.value / max) * 100)}%` }} />
          </span>
          {item.detail ? <span className={styles.shareDetail}>{item.detail}</span> : null}
        </li>
      ))}
    </ul>
  );
};
