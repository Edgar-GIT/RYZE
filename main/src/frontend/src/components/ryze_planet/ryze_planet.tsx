import { Canvas } from "@react-three/fiber";
import { Component, Suspense, useEffect, useRef, useState, type ReactNode } from "react";

import { joinClassNames } from "@utils/class_names";

import { PlanetScene } from "./planet_scene";
import styles from "./ryze_planet.module.css";

interface RyzePlanetProps {
  className?: string;
}

interface ErrorBoundaryProps {
  children: ReactNode;
  onError: () => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
}

class PlanetErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(): void {
    this.props.onError();
  }

  render() {
    if (this.state.hasError) {
      return null;
    }

    return this.props.children;
  }
}

export const RyzePlanet = ({ className }: RyzePlanetProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isVisible, setIsVisible] = useState(true);
  const [reduceMotion, setReduceMotion] = useState(false);
  const [webglFailed, setWebglFailed] = useState(false);

  useEffect(() => {
    const media = window.matchMedia("(prefers-reduced-motion: reduce)");
    const syncMotion = () => setReduceMotion(media.matches);
    syncMotion();
    media.addEventListener("change", syncMotion);
    return () => media.removeEventListener("change", syncMotion);
  }, []);

  useEffect(() => {
    const node = containerRef.current;
    if (!node) {
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        setIsVisible(entry.isIntersecting);
      },
      { threshold: 0.08, rootMargin: "80px" }
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  return (
    <div
      ref={containerRef}
      className={joinClassNames(styles.planet, className)}
      aria-hidden="true"
    >
      {webglFailed ? (
        <div className={styles.fallback} />
      ) : (
        <PlanetErrorBoundary onError={() => setWebglFailed(true)}>
          <Canvas
            className={styles.canvas}
            dpr={[1, 1.5]}
            gl={{
              antialias: true,
              alpha: true,
              powerPreference: "high-performance",
              stencil: false,
              depth: true
            }}
            camera={{ position: [0, 0.15, 5.4], fov: 38, near: 0.1, far: 50 }}
            frameloop={isVisible ? (reduceMotion ? "demand" : "always") : "never"}
            onCreated={({ gl, invalidate }) => {
              gl.setClearColor(0x000000, 0);
              gl.setClearAlpha(0);
              invalidate();
            }}
          >
            <Suspense fallback={null}>
              <PlanetScene reduceMotion={reduceMotion} />
            </Suspense>
          </Canvas>
        </PlanetErrorBoundary>
      )}

      <p className={joinClassNames(styles.wordmark, reduceMotion && styles.wordmarkStatic)}>
        RYZE
      </p>
    </div>
  );
};
