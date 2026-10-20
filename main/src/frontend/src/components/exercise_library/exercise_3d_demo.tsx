import { Rotate3d } from "lucide-react";
import { Component, Suspense, lazy, useState, type ReactNode } from "react";

import { Button } from "@/components/button/button";

import styles from "./exercise_3d_demo.module.css";

// The keys the procedural viewer implements. Kept here (not in the viewer
// module) so the demo wrapper can validate a stored key without importing the
// Three.js chunk.
export const SUPPORTED_THREE_D_KEYS = ["squat", "push_up", "biceps_curl", "lunge"] as const;

export const isSupportedThreeDKey = (key: string | null): key is (typeof SUPPORTED_THREE_D_KEYS)[number] =>
  Boolean(key) && (SUPPORTED_THREE_D_KEYS as readonly string[]).includes(key as string);

const Exercise3DViewer = lazy(() => import("./exercise_3d_viewer"));

interface ViewerErrorBoundaryState {
  failed: boolean;
}

class ViewerErrorBoundary extends Component<{ children: ReactNode; fallback: ReactNode }, ViewerErrorBoundaryState> {
  state: ViewerErrorBoundaryState = { failed: false };

  static getDerivedStateFromError(): ViewerErrorBoundaryState {
    return { failed: true };
  }

  render(): ReactNode {
    return this.state.failed ? this.props.fallback : this.props.children;
  }
}

interface Exercise3DDemoProps {
  threeDKey: string | null;
  exerciseName: string;
}

// Exercise3DDemo is the on-demand entry point for a procedural 3D
// demonstration. Nothing renders for exercises without an implemented scene,
// and the Three.js chunk is only downloaded after the client opens the demo.
export const Exercise3DDemo = ({ threeDKey, exerciseName }: Exercise3DDemoProps) => {
  const [active, setActive] = useState(false);

  if (!isSupportedThreeDKey(threeDKey)) {
    return null;
  }

  return (
    <section className={styles.demo} aria-label="3D demonstration">
      <div className={styles.header}>
        <div className={styles.copy}>
          <span className={styles.eyebrow}>3D demonstration</span>
          <h2 className={styles.title}>Study the movement in 3D</h2>
          <p className={styles.description}>
            A procedural, animated 3D preview of this exercise. It loads only when opened and works
            without any catalogue media.
          </p>
        </div>
        {!active ? (
          <Button variant="secondary" size="small" icon={<Rotate3d size={15} />} iconPosition="left" onClick={() => setActive(true)}>
            Open 3D view
          </Button>
        ) : null}
      </div>

      {active ? (
        <ViewerErrorBoundary
          fallback={
            <p className={styles.fallback} role="status">
              The 3D demonstration could not be loaded. The written instructions describe the full movement.
            </p>
          }
        >
          <Suspense fallback={<p className={styles.loading}>Loading 3D engine…</p>}>
            <Exercise3DViewer threeDKey={threeDKey} exerciseName={exerciseName} />
          </Suspense>
        </ViewerErrorBoundary>
      ) : null}
    </section>
  );
};
