import { Canvas, useFrame, useLoader } from "@react-three/fiber";
import { Suspense, useMemo, useRef } from "react";
import type { Group, Mesh } from "three";
import { MathUtils, SRGBColorSpace, TextureLoader } from "three";

import { BRAND_ASSETS } from "@/constants/brand_assets";
import { usePrefersReducedMotion } from "@/hooks/use_prefers_reduced_motion";

import styles from "./hero_scene.module.css";

interface SceneContentProps {
  reducedMotion: boolean;
}

interface FloatingPanelConfig {
  position: [number, number, number];
  rotation: [number, number, number];
  scale: [number, number, number];
}

const floatingPanels: FloatingPanelConfig[] = [
  {
    position: [-1.15, 0.78, -0.16],
    rotation: [0.24, -0.32, -0.1],
    scale: [1.18, 0.7, 0.035]
  },
  {
    position: [1.18, -0.48, -0.1],
    rotation: [-0.16, 0.28, 0.14],
    scale: [1.28, 0.78, 0.035]
  },
  {
    position: [0.3, 1.3, -0.36],
    rotation: [0.16, 0.2, 0.08],
    scale: [1.02, 0.48, 0.028]
  }
];

const createParticlePositions = () =>
  Array.from({ length: 34 }, (_, index) => {
    const angle = index * 0.78;
    const radius = 1.95 + (index % 7) * 0.12;
    const y = ((index % 9) - 4) * 0.25;

    return {
      key: `particle-${index}`,
      position: [
        Math.cos(angle) * radius,
        y,
        Math.sin(angle) * 0.54 - 0.55
      ] as [number, number, number],
      scale: 0.018 + (index % 4) * 0.007
    };
  });

const BrandPlate = ({ reducedMotion }: SceneContentProps) => {
  const meshRef = useRef<Mesh>(null);
  const texture = useLoader(TextureLoader, BRAND_ASSETS.set);

  texture.colorSpace = SRGBColorSpace;

  useFrame(({ clock }) => {
    if (!meshRef.current || reducedMotion) {
      return;
    }

    meshRef.current.position.y = Math.sin(clock.elapsedTime * 0.85) * 0.045;
    meshRef.current.rotation.z = Math.sin(clock.elapsedTime * 0.45) * 0.018;
  });

  return (
    <group position={[0, 0.05, 0.2]}>
      <mesh ref={meshRef}>
        <planeGeometry args={[3.25, 2.17]} />
        <meshBasicMaterial map={texture} transparent opacity={0.86} />
      </mesh>
      <mesh position={[0, 0, -0.08]}>
        <boxGeometry args={[3.42, 2.2, 0.035]} />
        <meshStandardMaterial
          color="#599DB1"
          emissive="#105773"
          emissiveIntensity={0.42}
          metalness={0.24}
          opacity={0.16}
          roughness={0.18}
          transparent
        />
      </mesh>
    </group>
  );
};

const ProceduralScene = ({ reducedMotion }: SceneContentProps) => {
  const groupRef = useRef<Group>(null);
  const orbitRef = useRef<Group>(null);
  const particles = useMemo(createParticlePositions, []);

  useFrame(({ clock, pointer }) => {
    if (!groupRef.current || reducedMotion) {
      return;
    }

    groupRef.current.rotation.x = MathUtils.lerp(
      groupRef.current.rotation.x,
      pointer.y * 0.12,
      0.035
    );
    groupRef.current.rotation.y = MathUtils.lerp(
      groupRef.current.rotation.y,
      pointer.x * 0.18,
      0.035
    );

    if (orbitRef.current) {
      orbitRef.current.rotation.z = clock.elapsedTime * 0.08;
      orbitRef.current.rotation.y = clock.elapsedTime * 0.045;
    }
  });

  return (
    <group ref={groupRef}>
      <ambientLight intensity={0.72} />
      <directionalLight color="#FFFFFF" intensity={1.25} position={[2.6, 2.2, 3.2]} />
      <pointLight color="#599DB1" intensity={44} position={[-1.8, 1.5, 2.2]} />
      <pointLight color="#105773" intensity={34} position={[2.3, -1.2, 2.4]} />

      <group ref={orbitRef}>
        <mesh rotation={[1.18, 0.35, 0.2]}>
          <torusGeometry args={[1.72, 0.018, 16, 128]} />
          <meshBasicMaterial color="#599DB1" transparent opacity={0.82} />
        </mesh>
        <mesh rotation={[1.38, -0.42, -0.12]}>
          <torusGeometry args={[1.28, 0.012, 16, 128]} />
          <meshBasicMaterial color="#DEDEDE" transparent opacity={0.42} />
        </mesh>
        <mesh rotation={[1.58, 0.08, 0.64]}>
          <torusGeometry args={[2.08, 0.008, 12, 128]} />
          <meshBasicMaterial color="#5699A1" transparent opacity={0.38} />
        </mesh>
      </group>

      {floatingPanels.map((panel) => (
        <mesh
          key={panel.position.join("-")}
          position={panel.position}
          rotation={panel.rotation}
          scale={panel.scale}
        >
          <boxGeometry args={[1, 1, 1]} />
          <meshStandardMaterial
            color="#FFFFFF"
            emissive="#105773"
            emissiveIntensity={0.18}
            metalness={0.18}
            opacity={0.16}
            roughness={0.14}
            transparent
          />
        </mesh>
      ))}

      <BrandPlate reducedMotion={reducedMotion} />

      <mesh position={[0, -1.58, -0.12]} rotation={[0, 0, Math.PI / 2]}>
        <cylinderGeometry args={[0.012, 0.012, 3.45, 16]} />
        <meshBasicMaterial color="#599DB1" transparent opacity={0.56} />
      </mesh>

      {particles.map((particle) => (
        <mesh key={particle.key} position={particle.position} scale={particle.scale}>
          <sphereGeometry args={[1, 10, 10]} />
          <meshBasicMaterial color="#DEDEDE" transparent opacity={0.82} />
        </mesh>
      ))}
    </group>
  );
};

export const HeroScene = () => {
  const reducedMotion = usePrefersReducedMotion();

  return (
    <div className={styles.scene} aria-hidden="true">
      <Canvas
        camera={{ position: [0, 0, 5.4], fov: 42 }}
        dpr={[1, 1.5]}
        gl={{ alpha: true, antialias: true, powerPreference: "high-performance" }}
      >
        <Suspense fallback={null}>
          <ProceduralScene reducedMotion={reducedMotion} />
        </Suspense>
      </Canvas>
    </div>
  );
};
