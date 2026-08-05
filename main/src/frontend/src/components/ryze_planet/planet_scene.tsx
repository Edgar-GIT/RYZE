import { useTexture } from "@react-three/drei";
import { useFrame, useThree } from "@react-three/fiber";
import { Bloom, EffectComposer } from "@react-three/postprocessing";
import { useMemo, useRef } from "react";
import {
  AdditiveBlending,
  BufferAttribute,
  BufferGeometry,
  CanvasTexture,
  Color,
  DoubleSide,
  Group,
  type MeshBasicMaterial,
  Points,
  ShaderMaterial
} from "three";

import { BRAND_ASSETS } from "@/constants/brand_assets";

import {
  atmosphereFragmentShader,
  atmosphereVertexShader,
  planetFragmentShader,
  planetVertexShader
} from "./planet_shaders";

const PLANET_RADIUS = 1.2;
const ROTATION_SECONDS = 95;
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

const PlanetCore = ({ reduceMotion }: SceneProps) => {
  const groupRef = useRef<Group>(null);
  const materialRef = useRef<ShaderMaterial>(null);
  const streakState = useRef({
    active: false,
    cooldown: 2.5,
    duration: 0,
    maxDuration: 0.55
  });
  const { camera } = useThree();

  const uniforms = useMemo(
    () => ({
      uTime: { value: 0 },
      uGoldPulse: { value: 0 },
      uStreakPhase: { value: 0 },
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

    if (reduceMotion) {
      material.uniforms.uGoldPulse.value = 0;
      return;
    }

    const streak = streakState.current;

    if (streak.active) {
      streak.duration += delta;
      const progress = Math.min(streak.duration / streak.maxDuration, 1);
      const envelope = Math.sin(progress * Math.PI);
      material.uniforms.uGoldPulse.value = envelope;
      material.uniforms.uStreakPhase.value += delta * 4.8;

      if (progress >= 1) {
        streak.active = false;
        streak.cooldown = 3.5 + Math.random() * 5;
        material.uniforms.uGoldPulse.value = 0;
      }
    } else {
      streak.cooldown -= delta;
      if (streak.cooldown <= 0) {
        streak.active = true;
        streak.duration = 0;
        streak.maxDuration = 0.35 + Math.random() * 0.4;
        material.uniforms.uStreakPhase.value = Math.random();
      }
    }

    if (groupRef.current) {
      groupRef.current.rotation.y += (Math.PI * 2 * delta) / ROTATION_SECONDS;
    }
  });

  return (
    <group ref={groupRef}>
      <mesh>
        <sphereGeometry args={[PLANET_RADIUS, 96, 96]} />
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
    <mesh scale={1.045}>
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

const createGoldWordmarkTexture = () => {
  const canvas = document.createElement("canvas");
  canvas.width = 512;
  canvas.height = 160;
  const context = canvas.getContext("2d");

  if (!context) {
    return null;
  }

  context.clearRect(0, 0, canvas.width, canvas.height);
  context.fillStyle = GOLD_HOT;
  context.font = '800 88px "Google Sans", Inter, Roboto, Arial, sans-serif';
  context.textAlign = "center";
  context.textBaseline = "middle";
  context.shadowColor = "rgba(255, 213, 106, 0.95)";
  context.shadowBlur = 28;
  context.fillText("RYZE", canvas.width / 2, canvas.height / 2 + 4);
  context.shadowBlur = 10;
  context.fillStyle = GOLD;
  context.fillText("RYZE", canvas.width / 2, canvas.height / 2 + 4);

  const texture = new CanvasTexture(canvas);
  texture.needsUpdate = true;
  texture.anisotropy = 4;
  return texture;
};

const Hologram = ({ reduceMotion }: SceneProps) => {
  const groupRef = useRef<Group>(null);
  const logoMaterialRef = useRef<MeshBasicMaterial>(null);
  const goldMaterialRef = useRef<MeshBasicMaterial>(null);
  const logoTexture = useTexture(BRAND_ASSETS.icon);
  const goldTexture = useMemo(() => createGoldWordmarkTexture(), []);
  const goldCycle = useRef({
    visible: false,
    timer: 4,
    hold: 0
  });

  useFrame((state, delta) => {
    const pulse = reduceMotion ? 1 : 0.88 + Math.sin(state.clock.elapsedTime * 1.1) * 0.12;

    if (logoMaterialRef.current) {
      logoMaterialRef.current.opacity = 0.72 * pulse;
    }

    if (groupRef.current && !reduceMotion) {
      groupRef.current.position.y = Math.sin(state.clock.elapsedTime * 0.45) * 0.014;
    }

    if (!goldMaterialRef.current) {
      return;
    }

    if (reduceMotion) {
      goldMaterialRef.current.opacity = 0;
      return;
    }

    const cycle = goldCycle.current;

    if (cycle.visible) {
      cycle.hold += delta;
      const fadeIn = Math.min(cycle.hold / 0.45, 1);
      const fadeOut = cycle.hold > 1.8 ? Math.max(0, 1 - (cycle.hold - 1.8) / 0.55) : 1;
      goldMaterialRef.current.opacity = fadeIn * fadeOut * 0.95;

      if (cycle.hold >= 2.4) {
        cycle.visible = false;
        cycle.timer = 5 + Math.random() * 6;
        goldMaterialRef.current.opacity = 0;
      }
    } else {
      cycle.timer -= delta;
      if (cycle.timer <= 0) {
        cycle.visible = true;
        cycle.hold = 0;
      }
    }
  });

  return (
    <group ref={groupRef} position={[0, 0, PLANET_RADIUS * 0.22]}>
      <mesh position={[0, 0.12, 0]}>
        <planeGeometry args={[0.72, 0.72]} />
        <meshBasicMaterial
          ref={logoMaterialRef}
          map={logoTexture}
          color={HIGHLIGHT}
          transparent
          opacity={0.72}
          depthWrite={false}
          blending={AdditiveBlending}
          toneMapped={false}
        />
      </mesh>

      <mesh position={[0, 0.12, -0.02]}>
        <circleGeometry args={[0.32, 48]} />
        <meshBasicMaterial
          color={GLOW}
          transparent
          opacity={0.1}
          depthWrite={false}
          blending={AdditiveBlending}
          toneMapped={false}
        />
      </mesh>

      {goldTexture ? (
        <mesh position={[0, -0.08, 0.03]}>
          <planeGeometry args={[1.15, 0.36]} />
          <meshBasicMaterial
            ref={goldMaterialRef}
            map={goldTexture}
            transparent
            opacity={0}
            depthWrite={false}
            blending={AdditiveBlending}
            toneMapped={false}
          />
        </mesh>
      ) : null}
    </group>
  );
};

const AmbientParticles = ({ reduceMotion }: SceneProps) => {
  const pointsRef = useRef<Points>(null);

  const geometry = useMemo(() => {
    const count = 70;
    const positions = new Float32Array(count * 3);

    for (let index = 0; index < count; index += 1) {
      const radius = 1.9 + Math.random() * 1.4;
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(2 * Math.random() - 1);
      positions[index * 3] = radius * Math.sin(phi) * Math.cos(theta);
      positions[index * 3 + 1] = radius * Math.sin(phi) * Math.sin(theta) * 0.72;
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
        size={0.016}
        color={ACCENT}
        transparent
        opacity={0.32}
        depthWrite={false}
        sizeAttenuation
      />
    </points>
  );
};

const CinematicLights = () => (
  <>
    <ambientLight intensity={0.05} color="#0B1220" />
    <directionalLight position={[3.2, 2.4, 2.8]} intensity={0.45} color="#F8FAFC" />
    <directionalLight position={[-3.4, -0.6, -2.2]} intensity={0.95} color={GLOW} />
    <pointLight position={[0, 0, 2.4]} intensity={0.4} color={HIGHLIGHT} distance={6} />
    <spotLight
      position={[-2.8, 1.8, 3.2]}
      angle={0.55}
      penumbra={0.8}
      intensity={1.15}
      color={ACCENT}
      distance={12}
    />
  </>
);

export const PlanetScene = ({ reduceMotion }: SceneProps) => (
  <>
    <CinematicLights />
    <PlanetCore reduceMotion={reduceMotion} />
    <Atmosphere />
    <Hologram reduceMotion={reduceMotion} />
    <AmbientParticles reduceMotion={reduceMotion} />
    <EffectComposer multisampling={0} enableNormalPass={false}>
      <Bloom
        intensity={0.55}
        luminanceThreshold={0.48}
        luminanceSmoothing={0.32}
        mipmapBlur
      />
    </EffectComposer>
  </>
);
