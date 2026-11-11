import { useCallback, useEffect, useState } from "react";

// Desktop-class pointer devices only. Phones and tablets never qualify, so they
// never download Three.js or create a WebGL context.
const IMMERSIVE_QUERY = "(min-width: 1024px) and (min-height: 680px) and (hover: hover) and (pointer: fine)";
const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";

let webglSupport: boolean | null = null;

const supportsWebGL2 = (): boolean => {
  if (webglSupport !== null) {
    return webglSupport;
  }
  try {
    const canvas = document.createElement("canvas");
    const context = canvas.getContext("webgl2");
    webglSupport = context !== null;
    context?.getExtension("WEBGL_lose_context")?.loseContext();
  } catch {
    webglSupport = false;
  }
  return webglSupport;
};

const qualifies = (): boolean =>
  typeof window !== "undefined" &&
  window.matchMedia(IMMERSIVE_QUERY).matches &&
  !window.matchMedia(REDUCED_MOTION_QUERY).matches &&
  supportsWebGL2();

// useImmersiveStory decides whether the Home story renders as the desktop
// Three.js experience or as the lightweight stacked layout. It is evaluated
// synchronously on first render to avoid a layout swap, follows viewport and
// motion-preference changes, and can be forced off when the scene fails.
export const useImmersiveStory = (): { immersive: boolean; disable: () => void } => {
  const [eligible, setEligible] = useState(qualifies);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const queries = [window.matchMedia(IMMERSIVE_QUERY), window.matchMedia(REDUCED_MOTION_QUERY)];
    const update = () => setEligible(qualifies());

    queries.forEach((query) => query.addEventListener("change", update));
    return () => queries.forEach((query) => query.removeEventListener("change", update));
  }, []);

  const disable = useCallback(() => setFailed(true), []);

  return { immersive: eligible && !failed, disable };
};
