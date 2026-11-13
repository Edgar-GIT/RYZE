import { useEffect, useId, useRef, useState, type RefObject } from "react";

import styles from "./charts.module.css";

export interface ChartPoint {
  label: string;
  value: number;
  // Text read by screen readers and shown on hover, e.g. "12 Nov: 2 workouts".
  description: string;
}

interface ChartProps {
  title: string;
  points: ChartPoint[];
  // Formats axis maxima and highlighted values.
  format?: (value: number) => string;
  // Highlight the last point (this week, latest session).
  highlightLast?: boolean;
}

const DEFAULT_WIDTH = 320;
const HEIGHT = 150;
const PAD_X = 8;
const PAD_TOP = 20;
const PAD_BOTTOM = 24;

// useChartWidth measures the figure so the SVG is drawn at its real pixel
// width: text keeps its size on a phone and on a wide desktop card instead of
// being scaled with the drawing.
const useChartWidth = (ref: RefObject<HTMLElement | null>) => {
  const [width, setWidth] = useState(DEFAULT_WIDTH);
  useEffect(() => {
    const element = ref.current;
    if (!element) return;
    const measure = () => {
      const next = Math.round(element.clientWidth);
      if (next > 0) setWidth(Math.max(240, next));
    };
    measure();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    return () => observer.disconnect();
  }, [ref]);
  return width;
};

// A compact bar chart (SVG, no library). The values are also listed for
// assistive technology, so the picture never carries the only copy.
export const BarChart = ({ title, points, format = String, highlightLast = true }: ChartProps) => {
  const id = useId();
  const figureRef = useRef<HTMLElement>(null);
  const width = useChartWidth(figureRef);
  const max = Math.max(1, ...points.map((point) => point.value));
  const slot = (width - PAD_X * 2) / Math.max(points.length, 1);
  const barWidth = Math.min(32, slot * 0.62);
  const plotHeight = HEIGHT - PAD_TOP - PAD_BOTTOM;
  return (
    <figure ref={figureRef} className={styles.chart}>
      <svg viewBox={`0 0 ${width} ${HEIGHT}`} role="img" aria-labelledby={`${id}-title`}>
        <title id={`${id}-title`}>{title}</title>
        <line x1={PAD_X} x2={width - PAD_X} y1={HEIGHT - PAD_BOTTOM} y2={HEIGHT - PAD_BOTTOM} className={styles.axis} />
        {points.map((point, index) => {
          const height = (point.value / max) * plotHeight;
          const x = PAD_X + slot * index + (slot - barWidth) / 2;
          const y = HEIGHT - PAD_BOTTOM - height;
          const last = highlightLast && index === points.length - 1;
          return (
            <g key={`${point.label}-${index}`}>
              <title>{point.description}</title>
              <rect
                x={x}
                y={point.value > 0 ? y : HEIGHT - PAD_BOTTOM - 2}
                width={barWidth}
                height={point.value > 0 ? Math.max(height, 2) : 2}
                rx={4}
                className={last ? styles.barLast : point.value > 0 ? styles.bar : styles.barEmpty}
              />
              {point.value > 0 ? (
                <text x={x + barWidth / 2} y={y - 5} textAnchor="middle" className={styles.value}>
                  {format(point.value)}
                </text>
              ) : null}
              <text x={x + barWidth / 2} y={HEIGHT - 6} textAnchor="middle" className={styles.label}>
                {point.label}
              </text>
            </g>
          );
        })}
      </svg>
      <ul className={styles.srOnly}>
        {points.map((point, index) => (
          <li key={`${point.label}-${index}`}>{point.description}</li>
        ))}
      </ul>
    </figure>
  );
};

// A line chart with dots, for progress over sessions.
export const LineChart = ({ title, points, format = String }: ChartProps) => {
  const id = useId();
  const figureRef = useRef<HTMLElement>(null);
  const width = useChartWidth(figureRef);
  const values = points.map((point) => point.value);
  const max = Math.max(...values, 1);
  const min = Math.min(...values, max);
  const spread = max - min || max * 0.2 || 1;
  const top = max + spread * 0.15;
  const bottom = Math.max(0, min - spread * 0.25);
  const plotHeight = HEIGHT - PAD_TOP - PAD_BOTTOM;
  const step = points.length > 1 ? (width - PAD_X * 6) / (points.length - 1) : 0;
  const coords = points.map((point, index) => ({
    x: points.length > 1 ? PAD_X * 3 + step * index : width / 2,
    y: PAD_TOP + (1 - (point.value - bottom) / (top - bottom)) * plotHeight
  }));
  const path = coords.map((coord, index) => `${index === 0 ? "M" : "L"}${coord.x.toFixed(1)},${coord.y.toFixed(1)}`).join(" ");
  const area = coords.length > 1 ? `${path} L${coords[coords.length - 1].x.toFixed(1)},${HEIGHT - PAD_BOTTOM} L${coords[0].x.toFixed(1)},${HEIGHT - PAD_BOTTOM} Z` : "";
  const lastIndex = points.length - 1;
  return (
    <figure ref={figureRef} className={styles.chart}>
      <svg viewBox={`0 0 ${width} ${HEIGHT}`} role="img" aria-labelledby={`${id}-title`}>
        <title id={`${id}-title`}>{title}</title>
        <defs>
          <linearGradient id={`${id}-fill`} x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor="#00cbff" stopOpacity="0.28" />
            <stop offset="100%" stopColor="#00cbff" stopOpacity="0" />
          </linearGradient>
        </defs>
        <line x1={PAD_X} x2={width - PAD_X} y1={HEIGHT - PAD_BOTTOM} y2={HEIGHT - PAD_BOTTOM} className={styles.axis} />
        {area ? <path d={area} fill={`url(#${id}-fill)`} /> : null}
        {coords.length > 1 ? <path d={path} className={styles.line} /> : null}
        {coords.map((coord, index) => (
          <g key={`${points[index].label}-${index}`}>
            <title>{points[index].description}</title>
            <circle cx={coord.x} cy={coord.y} r={index === lastIndex ? 4.5 : 3} className={index === lastIndex ? styles.dotLast : styles.dot} />
          </g>
        ))}
        {lastIndex >= 0 ? (
          <text
            x={Math.min(coords[lastIndex].x, width - 28)}
            y={coords[lastIndex].y - 9}
            textAnchor="middle"
            className={styles.value}
          >
            {format(points[lastIndex].value)}
          </text>
        ) : null}
        {points.length > 0 ? (
          <>
            <text x={coords[0].x} y={HEIGHT - 6} textAnchor="start" className={styles.label}>
              {points[0].label}
            </text>
            {lastIndex > 0 ? (
              <text x={coords[lastIndex].x} y={HEIGHT - 6} textAnchor="end" className={styles.label}>
                {points[lastIndex].label}
              </text>
            ) : null}
          </>
        ) : null}
      </svg>
      <ul className={styles.srOnly}>
        {points.map((point, index) => (
          <li key={`${point.label}-${index}`}>{point.description}</li>
        ))}
      </ul>
    </figure>
  );
};
