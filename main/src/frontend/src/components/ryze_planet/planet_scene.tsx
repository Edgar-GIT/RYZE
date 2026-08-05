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

const PLANET_RADIUS = 1.35;
const ROTATION_SECONDS = 50;
const GLOW = "#25D9FF";
const ACCENT = "#1FA2FF";
const HIGHLIGHT = "#63E6FF";
const BASE = "#05070B";
const SURFACE = "#0B1220";

interface SceneProps {
  reduceMotion: boolean;
}

const PlanetCore = ({ reduceMotion }: SceneProps) => {
  const groupRef = useRef<Group>(null);
  const materialRef = useRef<ShaderMaterial>(null);
  const { camera } = useThree();

  const uniforms = useMemo(
    () => ({
      uTime: { value: 0 },
      uBaseColor: { value: new Color(BASE) },
      uSurfaceColor: { value: new Color(SURFACE) },
      uGlowColor: { value: new Color(GLOW) },
      uHighlight: { value: new Color(HIGHLIGHT) },
      uCameraPosition: { value: camera.position.clone() }
    }),
    [camera]
  );

  useFrame((_, delta) => {
    if (materialRef.current) {
      materialRef.current.uniforms.uTime.value += delta;
      materialRef.current.uniforms.uCameraPosition.value.copy(camera.position);
    }

    if (!reduceMotion && groupRef.current) {
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

const createWordmarkTexture = () => {
  const canvas = document.createElement("canvas");
  canvas.width = 512;
  canvas.height = 128;
  const context = canvas.getContext("2d");

  if (!context) {
    return null;
  }

  context.clearRect(0, 0, canvas.width, canvas.height);
  context.fillStyle = "#25D9FF";
  context.font = '700 72px "Google Sans", Inter, Roboto, Arial, sans-serif';
  context.textAlign = "center";
  context.textBaseline = "middle";
  context.shadowColor = "rgba(37, 217, 255, 0.9)";
  context.shadowBlur = 20;
  context.fillText("RYZE", canvas.width / 2, canvas.height / 2 + 4);

  const texture = new CanvasTexture(canvas);
  texture.needsUpdate = true;
  texture.anisotropy = 4;
  return texture;
};

const Hologram = ({ reduceMotion }: SceneProps) => {
  const groupRef = useRef<Group>(null);
  const logoMaterialRef = useRef<MeshBasicMaterial>(null);
  const wordMaterialRef = useRef<MeshBasicMaterial>(null);
  const logoTexture = useTexture(BRAND_ASSETS.icon);
  const wordTexture = useMemo(() => createWordmarkTexture(), []);

  useFrame((state) => {
    const pulse = reduceMotion ? 1 : 0.88 + Math.sin(state.clock.elapsedTime * 1.1) * 0.12;

    if (logoMaterialRef.current) {
      logoMaterialRef.current.opacity = 0.78 * pulse;
    }

    if (wordMaterialRef.current) {
      wordMaterialRef.current.opacity = 0.84 * pulse;
    }

    if (groupRef.current && !reduceMotion) {
      groupRef.current.position.y = 0.08 + Math.sin(state.clock.elapsedTime * 0.55) * 0.018;
    }
  });

  return (
    <group ref={groupRef} position={[0, 0.08, PLANET_RADIUS * 0.2]}>
      <mesh position={[0, 0.22, 0]}>
        <planeGeometry args={[0.92, 0.92]} />
        <meshBasicMaterial
          ref={logoMaterialRef}
          map={logoTexture}
          color={GLOW}
          transparent
          opacity={0.78}
          depthWrite={false}
          blending={AdditiveBlending}
          toneMapped={false}
        />
      </mesh>

      {wordTexture ? (
        <mesh position={[0, -0.42, 0.01]}>
          <planeGeometry args={[1.2, 0.3]} />
          <meshBasicMaterial
            ref={wordMaterialRef}
            map={wordTexture}
            transparent
            opacity={0.84}
            depthWrite={false}
            blending={AdditiveBlending}
            toneMapped={false}
          />
        </mesh>
      ) : null}

      <mesh position={[0, 0.22, -0.02]}>
        <circleGeometry args={[0.4, 48]} />
        <meshBasicMaterial
          color={GLOW}
          transparent
          opacity={0.12}
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
    const count = 90;
    const positions = new Float32Array(count * 3);

    for (let index = 0; index < count; index += 1) {
      const radius = 2.1 + Math.random() * 1.7;
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
      pointsRef.current.rotation.y += delta * 0.015;
    }
  });

  return (
    <points ref={pointsRef} geometry={geometry}>
      <pointsMaterial
        size={0.018}
        color={ACCENT}
        transparent
        opacity={0.35}
        depthWrite={false}
        sizeAttenuation
      />
    </points>
  );
};

const CinematicLights = () => (
  <>
    <ambientLight intensity={0.06} color="#0B1220" />
    <directionalLight position={[3.2, 2.4, 2.8]} intensity={0.55} color="#F8FAFC" />
    <directionalLight position={[-3.4, -0.6, -2.2]} intensity={0.85} color={GLOW} />
    <pointLight position={[0, 0, 2.4]} intensity={0.35} color={HIGHLIGHT} distance={6} />
    <spotLight
      position={[-2.8, 1.8, 3.2]}
      angle={0.55}
      penumbra={0.8}
      intensity={1.1}
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
        intensity={0.42}
        luminanceThreshold={0.55}
        luminanceSmoothing={0.35}
        mipmapBlur
      />
    </EffectComposer>
  </>
);
