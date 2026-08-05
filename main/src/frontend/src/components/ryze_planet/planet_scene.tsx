import { Text } from "@react-three/drei";
import { useFrame, useThree } from "@react-three/fiber";
import { Bloom, EffectComposer } from "@react-three/postprocessing";
import { useMemo, useRef } from "react";
import {
  AdditiveBlending,
  BufferAttribute,
  BufferGeometry,
  Color,
  DoubleSide,
  Group,
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

const PLANET_RADIUS = 1.72;
const ROTATION_SECONDS = 110;
const GLOW = "#BF00FF";
const ACCENT = "#D946EF";
const HIGHLIGHT = "#E879F9";
const GOLD = "#FFD56A";
const GOLD_HOT = "#FFE9A8";
const BASE = "#05070B";
const SURFACE = "#0B1220";

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
  { active: false, cooldown: 0.4, progress: 0, duration: 1, offset: 0.2 },
  { active: false, cooldown: 1.1, progress: 0, duration: 1, offset: 0.55 },
  { active: false, cooldown: 2.0, progress: 0, duration: 1, offset: 0.8 },
  { active: false, cooldown: 2.8, progress: 0, duration: 1, offset: 0.35 }
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
      uMeteorOffset: { value: new Vector4(0.2, 0.55, 0.8, 0.35) },
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
            slot.cooldown = 1.2 + Math.random() * 2.4;
            slot.progress = 0;
          }
        } else {
          slot.cooldown -= delta;
          if (slot.cooldown <= 0) {
            slot.active = true;
            slot.progress = 0;
            slot.duration = 0.75 + Math.random() * 0.45;
            slot.offset = Math.random();
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
    <group ref={groupRef}>
      <mesh castShadow>
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
      uGlowColor: { value: new Color(GLOW) },
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
    <mesh scale={1.055}>
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

const ContactShadow = () => (
  <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -PLANET_RADIUS * 1.08, 0]} receiveShadow>
    <circleGeometry args={[PLANET_RADIUS * 0.92, 64]} />
    <meshBasicMaterial color="#000000" transparent opacity={0.55} depthWrite={false} />
  </mesh>
);

interface OrbitConfig {
  radius: number;
  tube: number;
  tiltX: number;
  tiltZ: number;
  speed: number;
  phase: number;
  arc: number;
  opacity: number;
}

const ORBITS: OrbitConfig[] = [
  { radius: 1.03, tube: 0.007, tiltX: 0.18, tiltZ: 0.08, speed: 1.55, phase: 0.0, arc: 1.55, opacity: 0.85 },
  { radius: 1.06, tube: 0.005, tiltX: -0.32, tiltZ: 0.22, speed: -1.9, phase: 1.2, arc: 1.2, opacity: 0.7 },
  { radius: 1.09, tube: 0.009, tiltX: 0.55, tiltZ: -0.15, speed: 2.25, phase: 2.4, arc: 1.8, opacity: 0.95 },
  { radius: 1.12, tube: 0.004, tiltX: -0.7, tiltZ: 0.4, speed: -2.6, phase: 0.7, arc: 1.0, opacity: 0.55 },
  { radius: 1.04, tube: 0.006, tiltX: 1.05, tiltZ: 0.1, speed: 1.8, phase: 3.1, arc: 1.35, opacity: 0.75 },
  { radius: 1.15, tube: 0.0055, tiltX: 0.25, tiltZ: -0.55, speed: -2.1, phase: 4.2, arc: 1.45, opacity: 0.65 },
  { radius: 1.08, tube: 0.008, tiltX: -0.15, tiltZ: 0.85, speed: 2.4, phase: 1.8, arc: 1.65, opacity: 0.9 }
];

const OrbitalTrails = ({ reduceMotion }: SceneProps) => {
  const groupRef = useRef<Group>(null);

  useFrame((_, delta) => {
    if (reduceMotion || !groupRef.current) {
      return;
    }

    groupRef.current.children.forEach((child, index) => {
      const config = ORBITS[index];
      if (!config) {
        return;
      }
      child.rotation.y += delta * config.speed;
    });
  });

  return (
    <group ref={groupRef}>
      {ORBITS.map((orbit) => (
        <group
          key={`${orbit.radius}-${orbit.phase}`}
          rotation={[orbit.tiltX, orbit.phase, orbit.tiltZ]}
        >
          <mesh>
            <torusGeometry
              args={[
                PLANET_RADIUS * orbit.radius,
                orbit.tube,
                10,
                160,
                Math.PI * orbit.arc
              ]}
            />
            <meshBasicMaterial
              color={GOLD_HOT}
              transparent
              opacity={orbit.opacity}
              depthWrite={false}
              blending={AdditiveBlending}
              toneMapped={false}
            />
          </mesh>
          <mesh rotation={[0, 0, 0.02]}>
            <torusGeometry
              args={[
                PLANET_RADIUS * orbit.radius,
                orbit.tube * 2.4,
                8,
                96,
                Math.PI * Math.min(0.35, orbit.arc * 0.28)
              ]}
            />
            <meshBasicMaterial
              color="#FFF3C4"
              transparent
              opacity={orbit.opacity * 0.55}
              depthWrite={false}
              blending={AdditiveBlending}
              toneMapped={false}
            />
          </mesh>
        </group>
      ))}
    </group>
  );
};

const GoldWordmark = ({ reduceMotion }: SceneProps) => {
  const textRef = useRef<{
    fillOpacity: number;
    outlineOpacity: number;
    sync?: () => void;
  } | null>(null);
  const groupRef = useRef<Group>(null);

  useFrame((state) => {
    const pulse = reduceMotion ? 1 : 0.82 + Math.sin(state.clock.elapsedTime * 1.15) * 0.18;
    const text = textRef.current;

    if (text) {
      text.fillOpacity = pulse;
      text.outlineOpacity = pulse * 0.95;
      text.sync?.();
    }

    if (groupRef.current && !reduceMotion) {
      groupRef.current.position.y = Math.sin(state.clock.elapsedTime * 0.5) * 0.02;
    }
  });

  return (
    <group ref={groupRef} position={[0, 0.02, PLANET_RADIUS * 0.2]}>
      <Text
        ref={textRef as never}
        position={[0, 0, 0.05]}
        fontSize={0.48}
        letterSpacing={0.18}
        color={GOLD_HOT}
        anchorX="center"
        anchorY="middle"
        outlineWidth={0.02}
        outlineColor={GOLD}
        fillOpacity={1}
        outlineOpacity={0.95}
        depthOffset={-4}
      >
        RYZE
      </Text>
      <pointLight position={[0, 0, 0.3]} intensity={1.8} color={GOLD} distance={3.2} />
    </group>
  );
};

const AmbientParticles = ({ reduceMotion }: SceneProps) => {
  const pointsRef = useRef<Points>(null);

  const geometry = useMemo(() => {
    const count = 110;
    const positions = new Float32Array(count * 3);

    for (let index = 0; index < count; index += 1) {
      const radius = 2.35 + Math.random() * 1.8;
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(2 * Math.random() - 1);
      positions[index * 3] = radius * Math.sin(phi) * Math.cos(theta);
      positions[index * 3 + 1] = radius * Math.sin(phi) * Math.sin(theta) * 0.7;
      positions[index * 3 + 2] = radius * Math.cos(phi);
    }

    const buffer = new BufferGeometry();
    buffer.setAttribute("position", new BufferAttribute(positions, 3));
    return buffer;
  }, []);

  useFrame((_, delta) => {
    if (!reduceMotion && pointsRef.current) {
      pointsRef.current.rotation.y += delta * 0.01;
    }
  });

  return (
    <points ref={pointsRef} geometry={geometry}>
      <pointsMaterial
        size={0.015}
        color={ACCENT}
        transparent
        opacity={0.28}
        depthWrite={false}
        sizeAttenuation
      />
    </points>
  );
};

const CinematicLights = () => (
  <>
    <ambientLight intensity={0.04} color="#07040F" />
    <directionalLight
      castShadow
      position={[4.2, 3.4, 2.6]}
      intensity={0.7}
      color="#F8FAFC"
      shadow-mapSize={[1024, 1024]}
      shadow-bias={-0.0002}
    />
    <directionalLight position={[-3.8, -0.4, -2.4]} intensity={1.35} color={GLOW} />
    <directionalLight position={[1.2, -2.4, 2.0]} intensity={0.35} color="#1A0A2E" />
    <pointLight position={[0, 0.2, 2.8]} intensity={0.55} color={HIGHLIGHT} distance={7} />
    <spotLight
      position={[-3.2, 2.2, 3.4]}
      angle={0.5}
      penumbra={0.85}
      intensity={1.4}
      color={ACCENT}
      distance={14}
    />
  </>
);

export const PlanetScene = ({ reduceMotion }: SceneProps) => (
  <>
    <CinematicLights />
    <ContactShadow />
    <PlanetCore reduceMotion={reduceMotion} />
    <Atmosphere />
    <OrbitalTrails reduceMotion={reduceMotion} />
    <GoldWordmark reduceMotion={reduceMotion} />
    <AmbientParticles reduceMotion={reduceMotion} />
    <EffectComposer multisampling={0} enableNormalPass={false}>
      <Bloom
        intensity={1.05}
        luminanceThreshold={0.22}
        luminanceSmoothing={0.28}
        mipmapBlur
      />
    </EffectComposer>
  </>
);
