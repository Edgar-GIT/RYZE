import { useFrame, useThree } from "@react-three/fiber";
import { useMemo, useRef } from "react";
import {
  AdditiveBlending,
  BufferAttribute,
  BufferGeometry,
  Color,
  DoubleSide,
  Group,
  type Mesh,
  type MeshBasicMaterial,
  Points,
  ShaderMaterial,
  Vector4
} from "three";

import {
  atmosphereFragmentShader,
  atmosphereVertexShader,
  planetFragmentShader,
  planetVertexShader
} from "./planet_shaders";

const PLANET_RADIUS = 1.35;
const ROTATION_SECONDS = 48;
const GLOW = "#A78BFA";
const ACCENT = "#8B5CF6";
const HIGHLIGHT = "#C4B5FD";
const GOLD = "#F5B942";
const GOLD_HOT = "#FFE9A8";
const BASE = "#14101F";
const SURFACE = "#2A1B45";

interface SceneProps {
  reduceMotion: boolean;
}

interface MeteorSlot {
  active: boolean;
  cooldown: number;
  progress: number;
  duration: number;
  offset: number;
}

const createMeteorSlots = (): MeteorSlot[] => [
  { active: false, cooldown: 0.6, progress: 0, duration: 0.9, offset: 0.25 },
  { active: false, cooldown: 2.4, progress: 0, duration: 0.9, offset: 0.6 },
  { active: false, cooldown: 4.0, progress: 0, duration: 0.9, offset: 0.4 },
  { active: false, cooldown: 5.5, progress: 0, duration: 0.9, offset: 0.75 }
];

const PlanetCore = ({ reduceMotion }: SceneProps) => {
  const groupRef = useRef<Group>(null);
  const materialRef = useRef<ShaderMaterial>(null);
  const meteors = useRef(createMeteorSlots());
  const { camera } = useThree();

  const uniforms = useMemo(
    () => ({
      uTime: { value: 0 },
      uMeteorActive: { value: new Vector4(0, 0, 0, 0) },
      uMeteorProgress: { value: new Vector4(0, 0, 0, 0) },
      uMeteorOffset: { value: new Vector4(0.25, 0.6, 0.4, 0.75) },
      uBaseColor: { value: new Color(BASE) },
      uSurfaceColor: { value: new Color(SURFACE) },
      uGlowColor: { value: new Color(GLOW) },
      uHighlight: { value: new Color(HIGHLIGHT) },
      uGoldColor: { value: new Color(GOLD) },
      uCameraPosition: { value: camera.position.clone() }
    }),
    [camera]
  );

  useFrame((_, delta) => {
    const material = materialRef.current;
    if (!material) {
      return;
    }

    material.uniforms.uTime.value += delta;
    material.uniforms.uCameraPosition.value.copy(camera.position);

    const active = material.uniforms.uMeteorActive.value as Vector4;
    const progress = material.uniforms.uMeteorProgress.value as Vector4;
    const offset = material.uniforms.uMeteorOffset.value as Vector4;

    if (reduceMotion) {
      active.set(0, 0, 0, 0);
      progress.set(0, 0, 0, 0);
    } else {
      meteors.current.forEach((slot, index) => {
        if (slot.active) {
          slot.progress += delta / slot.duration;
          if (slot.progress >= 1) {
            slot.active = false;
            slot.cooldown = 2.8 + Math.random() * 4.5;
            slot.progress = 0;
          }
        } else {
          slot.cooldown -= delta;
          if (slot.cooldown <= 0) {
            // Prefer one meteor at a time for a cleaner shooting-star read
            const othersActive = meteors.current.some((entry, i) => i !== index && entry.active);
            if (!othersActive) {
              slot.active = true;
              slot.progress = 0;
              slot.duration = 0.7 + Math.random() * 0.35;
              slot.offset = Math.random();
            } else {
              slot.cooldown = 0.4 + Math.random() * 0.8;
            }
          }
        }

        const key = (["x", "y", "z", "w"] as const)[index];
        active[key] = slot.active ? 1 : 0;
        progress[key] = slot.progress;
        offset[key] = slot.offset;
      });
    }

    if (!reduceMotion && groupRef.current) {
      groupRef.current.rotation.y += (Math.PI * 2 * delta) / ROTATION_SECONDS;
    }
  });

  return (
    <group ref={groupRef} rotation={[0.12, 0, 0.08]}>
      <mesh>
        <sphereGeometry args={[PLANET_RADIUS, 128, 128]} />
        <shaderMaterial
          ref={materialRef}
          vertexShader={planetVertexShader}
          fragmentShader={planetFragmentShader}
          uniforms={uniforms}
        />
      </mesh>
    </group>
  );
};

const Atmosphere = () => {
  const materialRef = useRef<ShaderMaterial>(null);
  const { camera } = useThree();

  const uniforms = useMemo(
    () => ({
      uGlowColor: { value: new Color(ACCENT) },
      uHighlight: { value: new Color(HIGHLIGHT) },
      uCameraPosition: { value: camera.position.clone() }
    }),
    [camera]
  );

  useFrame(() => {
    if (materialRef.current) {
      materialRef.current.uniforms.uCameraPosition.value.copy(camera.position);
    }
  });

  return (
    <mesh scale={1.065}>
      <sphereGeometry args={[PLANET_RADIUS, 64, 64]} />
      <shaderMaterial
        ref={materialRef}
        vertexShader={atmosphereVertexShader}
        fragmentShader={atmosphereFragmentShader}
        uniforms={uniforms}
        transparent
        depthWrite={false}
        side={DoubleSide}
        blending={AdditiveBlending}
      />
    </mesh>
  );
};

const ShootingStar = ({
  reduceMotion,
  delay,
  tilt,
  radiusScale,
  speed
}: {
  reduceMotion: boolean;
  delay: number;
  tilt: [number, number, number];
  radiusScale: number;
  speed: number;
}) => {
  const groupRef = useRef<Group>(null);
  const trailRef = useRef<Mesh>(null);
  const headRef = useRef<Mesh>(null);
  const life = useRef(-delay);

  useFrame((_, delta) => {
    if (reduceMotion || !groupRef.current || !trailRef.current || !headRef.current) {
      return;
    }

    life.current += delta * speed;

    if (life.current > 1.15) {
      life.current = -(1.8 + Math.random() * 3.5);
      groupRef.current.rotation.z = (Math.random() - 0.5) * 0.35;
    }

    const progress = Math.min(Math.max(life.current, 0), 1);
    const fade =
      progress < 0.1 ? progress / 0.1 : progress > 0.75 ? Math.max(0, (1 - progress) / 0.25) : 1;

    groupRef.current.rotation.y = progress * Math.PI * 1.65;

    const trailMat = trailRef.current.material as MeshBasicMaterial;
    const headMat = headRef.current.material as MeshBasicMaterial;
    trailMat.opacity = fade * 0.95;
    headMat.opacity = fade;
  });

  return (
    <group ref={groupRef} rotation={tilt}>
      <mesh ref={trailRef}>
        <torusGeometry args={[PLANET_RADIUS * radiusScale, 0.018, 8, 96, Math.PI * 0.55]} />
        <meshBasicMaterial
          color={GOLD_HOT}
          transparent
          opacity={0}
          depthWrite={false}
          blending={AdditiveBlending}
          toneMapped={false}
        />
      </mesh>
      <mesh ref={headRef} rotation={[0, 0.02, 0]}>
        <torusGeometry args={[PLANET_RADIUS * radiusScale, 0.032, 8, 48, Math.PI * 0.12]} />
        <meshBasicMaterial
          color="#FFF6D0"
          transparent
          opacity={0}
          depthWrite={false}
          blending={AdditiveBlending}
          toneMapped={false}
        />
      </mesh>
    </group>
  );
};

const AmbientParticles = ({ reduceMotion }: SceneProps) => {
  const pointsRef = useRef<Points>(null);

  const geometry = useMemo(() => {
    const count = 80;
    const positions = new Float32Array(count * 3);

    for (let index = 0; index < count; index += 1) {
      const radius = 2.45 + Math.random() * 1.55;
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(2 * Math.random() - 1);
      positions[index * 3] = radius * Math.sin(phi) * Math.cos(theta);
      positions[index * 3 + 1] = radius * Math.sin(phi) * Math.sin(theta) * 0.68;
      positions[index * 3 + 2] = radius * Math.cos(phi);
    }

    const buffer = new BufferGeometry();
    buffer.setAttribute("position", new BufferAttribute(positions, 3));
    return buffer;
  }, []);

  useFrame((_, delta) => {
    if (!reduceMotion && pointsRef.current) {
      pointsRef.current.rotation.y += delta * 0.012;
    }
  });

  return (
    <points ref={pointsRef} geometry={geometry}>
      <pointsMaterial
        size={0.013}
        color={HIGHLIGHT}
        transparent
        opacity={0.24}
        depthWrite={false}
        sizeAttenuation
      />
    </points>
  );
};

const CinematicLights = () => (
  <>
    <ambientLight intensity={0.28} color="#1A1230" />
    <hemisphereLight args={["#E9D5FF", "#05070B", 0.6]} />
    <directionalLight position={[3.6, 3.2, 2.6]} intensity={1.35} color="#FFFFFF" />
    <directionalLight position={[-3.6, 0.3, -2.1]} intensity={1.55} color={ACCENT} />
    <directionalLight position={[0.6, -2.2, 2.0]} intensity={0.45} color={GOLD} />
    <pointLight position={[0, 0.25, 2.9]} intensity={0.75} color={HIGHLIGHT} distance={7} />
  </>
);

export const PlanetScene = ({ reduceMotion }: SceneProps) => (
  <>
    <CinematicLights />
    <PlanetCore reduceMotion={reduceMotion} />
    <Atmosphere />
    <ShootingStar
      reduceMotion={reduceMotion}
      delay={0.4}
      tilt={[0.22, 0.1, 0.08]}
      radiusScale={1.12}
      speed={1.05}
    />
    <ShootingStar
      reduceMotion={reduceMotion}
      delay={2.8}
      tilt={[-0.48, 0.4, -0.18]}
      radiusScale={1.18}
      speed={1.2}
    />
    <ShootingStar
      reduceMotion={reduceMotion}
      delay={5.2}
      tilt={[0.78, -0.2, 0.4]}
      radiusScale={1.15}
      speed={0.95}
    />
    <AmbientParticles reduceMotion={reduceMotion} />
  </>
);
