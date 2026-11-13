import { useCallback, useEffect, useRef, useState, type PointerEvent } from "react";

import { useGettext } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import { OUR_VISION_PT } from "./our_vision_page.pt";
import { RyzeCard, StandoutVisual } from "./standout_visual";
import styles from "./vision_hero_visual.module.css";
import type { VisionScene } from "./vision_scene";

// Tablets and desktops with full motion and WebGL get the 3D tunnel; phones
// never download Three.js (the same rule as the Home story).
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

// VisionHeroVisual: "Thousands of fitness apps. All the same. RYZE isn't."
// An endless stream of identical app cards rushes past while the RYZE card
// stays still and lit in front, tilting with the pointer like a hologram.
// Decorative: the heading and the comparison section say the same in text.
export const VisionHeroVisual = ({ className }: { className?: string }) => {
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
    return <StandoutVisual className={className} />;
  }
  return <VisionStage className={className} onFailure={fail} />;
};

const VisionStage = ({ className, onFailure }: { className?: string; onFailure: () => void }) => {
  const tr = useGettext(OUR_VISION_PT);
  const stageRef = useRef<HTMLDivElement>(null);
  const hostRef = useRef<HTMLDivElement>(null);
  const sceneRef = useRef<VisionScene | null>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const [ready, setReady] = useState(false);
  const cardLabel = tr("12-week plan");

  useEffect(() => {
    const host = hostRef.current;
    const stage = stageRef.current;
    if (!host || !stage) return;

    const canvas = document.createElement("canvas");
    canvas.className = styles.canvas;
    canvas.setAttribute("aria-hidden", "true");
    host.appendChild(canvas);
    let disposed = false;
    let visible = false;
    let scene: VisionScene | null = null;

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

    import("./vision_scene")
      .then(({ createVisionScene }) => {
        if (disposed) throw new Error("Unmounted before the scene was created");
        return createVisionScene(canvas, {
          cardLabel,
          density: window.matchMedia("(min-width: 1100px)").matches ? "full" : "light"
        });
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
        setReady(true);
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
    // The scene is built once per mount; the cards' caption is a decorative
    // texture, so a later language change does not rebuild it.
  }, [onFailure]);

  // The tilt is written straight to CSS variables: no React render per move.
  const tiltTo = (x: number, y: number) => {
    sceneRef.current?.setPointer(x, y);
    cardRef.current?.style.setProperty("--tilt-x", x.toFixed(3));
    cardRef.current?.style.setProperty("--tilt-y", y.toFixed(3));
  };

  const handlePointerMove = (event: PointerEvent<HTMLDivElement>) => {
    if (event.pointerType !== "mouse") return;
    const rect = event.currentTarget.getBoundingClientRect();
    tiltTo(((event.clientX - rect.left) / rect.width) * 2 - 1, ((event.clientY - rect.top) / rect.height) * 2 - 1);
  };

  const handlePointerLeave = () => tiltTo(0, 0);

  return (
    <div
      ref={stageRef}
      className={joinClassNames(styles.stage, ready && styles.ready, className)}
      onPointerMove={handlePointerMove}
      onPointerLeave={handlePointerLeave}
      aria-hidden="true"
    >
      <div ref={hostRef} className={styles.canvasHost} />
      <div ref={cardRef} className={styles.cardWrap}>
        <RyzeCard className={styles.card} />
        <span className={styles.glare} />
      </div>
    </div>
  );
};
