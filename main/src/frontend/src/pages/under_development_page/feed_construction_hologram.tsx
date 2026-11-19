import { useCallback, useEffect, useRef, useState, type PointerEvent } from "react";

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

const FeedConstructionFallback = ({ className }: { className?: string }) => (
  <div className={joinClassNames(styles.stage, styles.fallback, className)} aria-hidden="true">
    <div className={styles.fallbackBeam} />
    <div className={styles.fallbackPlatform} />
    <div className={styles.fallbackWorker}>
      <span className={styles.fallbackHelmet} />
      <span className={styles.fallbackHead} />
      <span className={styles.fallbackBody} />
      <span className={styles.fallbackArm} />
      <span className={styles.fallbackHammer} />
      <span className={styles.fallbackLegLeft} />
      <span className={styles.fallbackLegRight} />
    </div>
    <div className={styles.fallbackBeamLine} />
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
