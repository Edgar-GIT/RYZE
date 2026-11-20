import { type CSSProperties, useCallback, useEffect, useRef, useState, type PointerEvent } from "react";

import { joinClassNames } from "@utils/class_names";

import styles from "./feed_construction_hologram.module.css";
import type { FeedConstructionScene } from "./feed_construction_scene";

const STAGE_QUERY = "(min-width: 768px)";
const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";

let webglSupport: boolean | null = null;

const supportsWebGL = (): boolean => {
  if (webglSupport !== null) return webglSupport;
  try {
    const probe = document.createElement("canvas");
    const context = probe.getContext("webgl2") ?? probe.getContext("webgl");
    webglSupport = context !== null;
    context?.getExtension("WEBGL_lose_context")?.loseContext();
  } catch {
    webglSupport = false;
  }
  return webglSupport;
};

const qualifies = (): boolean =>
  typeof window !== "undefined" &&
  window.matchMedia(STAGE_QUERY).matches &&
  !window.matchMedia(REDUCED_MOTION_QUERY).matches &&
  supportsWebGL();

// FeedConstructionHologram is deliberately separate from the shared
// under-development illustration: only /feed receives the Three.js scene.
export const FeedConstructionHologram = ({ className }: { className?: string }) => {
  const [stage, setStage] = useState(qualifies);
  const [failed, setFailed] = useState(false);
  const fail = useCallback(() => setFailed(true), []);

  useEffect(() => {
    const queries = [window.matchMedia(STAGE_QUERY), window.matchMedia(REDUCED_MOTION_QUERY)];
    const update = () => setStage(qualifies());
    queries.forEach((query) => query.addEventListener("change", update));
    return () => queries.forEach((query) => query.removeEventListener("change", update));
  }, []);

  if (!stage || failed) {
    return <FeedConstructionFallback className={className} />;
  }

  return <FeedConstructionStage className={className} onFailure={fail} />;
};

// The light version (phones, reduced motion, no WebGL): the Home's drone
// projects the Feed column over a platform and its post cards build into it.
const FALLBACK_SLOTS = [0, 1, 2, 3];

const FeedConstructionFallback = ({ className }: { className?: string }) => (
  <div className={joinClassNames(styles.stage, styles.fallback, className)} aria-hidden="true">
    <svg className={styles.fallbackArt} viewBox="0 0 320 346" focusable="false">
      <defs>
        <linearGradient id="feed-fallback-beam" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#bff6ff" stopOpacity="0.55" />
          <stop offset="1" stopColor="#00cbff" stopOpacity="0.08" />
        </linearGradient>
        <radialGradient id="feed-fallback-top" cx="0.5" cy="0.5" r="0.5">
          <stop offset="0" stopColor="#0c2a66" />
          <stop offset="1" stopColor="#030a22" />
        </radialGradient>
      </defs>
      <path className={styles.beam} d="M146 58 H174 L268 292 H52 Z" fill="url(#feed-fallback-beam)" />
      {/* The platform. */}
      <ellipse cx="160" cy="304" rx="116" ry="22" fill="#0a1534" />
      <ellipse cx="160" cy="296" rx="116" ry="22" fill="url(#feed-fallback-top)" stroke="#3a4f7c" strokeWidth="2" />
      <ellipse cx="160" cy="296" rx="96" ry="17" fill="none" stroke="#7cebff" strokeWidth="1.2" strokeDasharray="6 5" opacity="0.8" />
      <ellipse cx="160" cy="296" rx="52" ry="9" fill="none" stroke="#7cebff" strokeWidth="1" opacity="0.6" />
      {/* The Feed column and its cards. */}
      <g transform="translate(112 92)">
        <rect width="96" height="186" rx="12" fill="rgba(54,214,255,0.08)" stroke="#7cebff" strokeWidth="1.5" />
        <text x="10" y="17" fill="#e8fdff" fontSize="10" fontWeight="800" letterSpacing="1">
          FEED
        </text>
        <circle cx="84" cy="13" r="3" fill="#e8fdff" />
        <path d="M8 24 H88" stroke="#7cebff" strokeOpacity="0.5" />
        {FALLBACK_SLOTS.map((slot) => (
          <g key={slot} transform={`translate(8 ${30 + slot * 38})`}>
            <rect width="80" height="32" rx="6" fill="none" stroke="#7cebff" strokeOpacity="0.35" strokeDasharray="4 3" />
            <g className={styles.card} style={{ "--delay": `${slot * 1.4}s` } as CSSProperties}>
              <rect width="80" height="32" rx="6" fill="rgba(54,214,255,0.22)" stroke="#e8fdff" strokeWidth="1.2" />
              <circle cx="9" cy="8" r="4" fill="#e8fdff" />
              <rect x="17" y="5" width="28" height="3" rx="1.5" fill="#e8fdff" opacity="0.8" />
              <rect x="6" y="15" width="68" height="10" rx="2.5" fill="#e8fdff" opacity="0.3" />
              <rect x="6" y="27.5" width="40" height="2" rx="1" fill="#e8fdff" opacity="0.55" />
            </g>
          </g>
        ))}
      </g>
      <path className={styles.scan} d="M100 96 H220" stroke="#e8fdff" strokeWidth="1.5" strokeLinecap="round" opacity="0.7" />
      {/* The drone, as on the Home page. */}
      <g className={styles.drone} transform="translate(60 14)">
        <line x1="34" y1="22" x2="166" y2="22" stroke="#1c2740" strokeWidth="4" strokeLinecap="round" />
        {[30, 170].map((x) => (
          <g key={x}>
            <ellipse cx={x} cy="21" rx="24" ry="6" fill="none" stroke="#3a4a6e" strokeWidth="3.4" />
            <ellipse className={styles.rotor} cx={x} cy="21" rx="18" ry="3.6" fill="#bff6ff" fillOpacity="0.3" />
          </g>
        ))}
        <path d="M58 26 Q100 8 142 26 Z" fill="#1a2438" />
        <ellipse cx="100" cy="27" rx="42" ry="8" fill="#151c2e" />
        <ellipse cx="100" cy="26" rx="42" ry="8" fill="none" stroke="#7cebff" strokeWidth="1.4" />
        <ellipse cx="100" cy="33" rx="20" ry="4.4" fill="#040914" stroke="#7cebff" strokeWidth="1.1" />
        <ellipse cx="100" cy="34" rx="5" ry="2.2" fill="#e8fdff" />
      </g>
    </svg>
  </div>
);

const FeedConstructionStage = ({ className, onFailure }: { className?: string; onFailure: () => void }) => {
  const stageRef = useRef<HTMLDivElement>(null);
  const hostRef = useRef<HTMLDivElement>(null);
  const sceneRef = useRef<FeedConstructionScene | null>(null);

  useEffect(() => {
    const stage = stageRef.current;
    const host = hostRef.current;
    if (!stage || !host) return;

    const canvas = document.createElement("canvas");
    canvas.className = styles.canvas;
    canvas.setAttribute("aria-hidden", "true");
    host.appendChild(canvas);

    let disposed = false;
    let visible = false;
    let scene: FeedConstructionScene | null = null;

    const handleContextLost = (event: Event) => {
      event.preventDefault();
      onFailure();
    };
    canvas.addEventListener("webglcontextlost", handleContextLost);

    const sync = () => {
      if (visible && !document.hidden) sceneRef.current?.start();
      else sceneRef.current?.stop();
    };
    const observer = new IntersectionObserver(([entry]) => {
      visible = entry.isIntersecting;
      sync();
    });
    observer.observe(stage);
    document.addEventListener("visibilitychange", sync);

    const resizeObserver = new ResizeObserver(([entry]) => {
      sceneRef.current?.resize(entry.contentRect.width, entry.contentRect.height);
    });
    resizeObserver.observe(stage);

    import("./feed_construction_scene")
      .then(({ createFeedConstructionScene }) => {
        if (disposed) throw new Error("Unmounted before the scene was created");
        return createFeedConstructionScene(canvas);
      })
      .then((created) => {
        scene = created;
        if (disposed) {
          created.dispose();
          return;
        }
        sceneRef.current = created;
        const rect = stage.getBoundingClientRect();
        created.resize(rect.width, rect.height);
        sync();
      })
      .catch(() => {
        if (!disposed) onFailure();
      });

    return () => {
      disposed = true;
      observer.disconnect();
      resizeObserver.disconnect();
      document.removeEventListener("visibilitychange", sync);
      canvas.removeEventListener("webglcontextlost", handleContextLost);
      scene?.dispose();
      if (sceneRef.current === scene) sceneRef.current = null;
      canvas.remove();
    };
  }, [onFailure]);

  const handlePointerMove = (event: PointerEvent<HTMLDivElement>) => {
    if (event.pointerType !== "mouse") return;
    const rect = event.currentTarget.getBoundingClientRect();
    sceneRef.current?.setPointer(((event.clientX - rect.left) / rect.width) * 2 - 1, ((event.clientY - rect.top) / rect.height) * 2 - 1);
  };

  const handlePointerLeave = () => sceneRef.current?.setPointer(0, 0);

  return (
    <div
      ref={stageRef}
      className={joinClassNames(styles.stage, className)}
      onPointerMove={handlePointerMove}
      onPointerLeave={handlePointerLeave}
      aria-hidden="true"
    >
      <div ref={hostRef} className={styles.canvasHost} />
    </div>
  );
};
