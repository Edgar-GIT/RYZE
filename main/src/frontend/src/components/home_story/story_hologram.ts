import * as THREE from "three";

import { type AthleteState, ROUTINE_REP_SECONDS, ROUTINE_SETS, createAthlete } from "./story_athlete";

/*
 * The Home story's hero object (owner requests, 2026-11-15): a scanning drone
 * reads an athlete's training session — train anywhere, progress everywhere.
 * The scan is a picture of the session being recorded (sets, loads, the
 * exercise's muscles, EXP), the data RYZE keeps; it is not a body or
 * biometric measurement. Everything is procedural (no model files):
 *
 *   scanner  a satin graphite drone with ducted rotors, a light ring and a
 *            sensor array underneath, lit by the scene;
 *   beam     light from the sensor array to the platform, with a scan plane
 *            that runs up and down the body and particles rising inside;
 *   athlete  a faceless, genderless mannequin (story_athlete.ts) cycling
 *            through curls, presses, squats and Romanian deadlifts, the worked
 *            muscles pulsing gold to red; every rep stands for a logged set,
 *            and a burst of data rises from the worked muscle to the sensor;
 *   platform a glossy disc with a metal rim, a light line and rings; it
 *            reflects the athlete and pulses when a set is logged.
 *
 * The installation spans about -1.7..1.85 in y around its origin.
 */

export interface HologramState extends AthleteState {
  presence: number; // 0..1 while the hologram materialises
  lens: THREE.Vector3; // world position of the sensor array
  base: THREE.Vector3; // world position of the platform's centre
  arrived: number; // sets of this exercise whose data has reached the sensor
  logged: number; // seconds since the last of them arrived (-1: none yet)
}

export interface Hologram {
  group: THREE.Group;
  // elapsed: seconds since the scene started; emphasis: 1 on the hero, lower
  // while it waits in the corner.
  update: (elapsed: number, emphasis: number) => HologramState;
}

type Track = <T extends { dispose: () => void }>(item: T) => T;

const ICE = new THREE.Color(0x9af3ff);
const CYAN_LIGHT = 0x7cebff;

// Heights in the installation's frame (before the centring offset). The beam
// leaves a wide sensor array (a frustum, not a point) so raised arms and a hip
// hinge stay inside the light.
const FLOOR_Y = -1.5;
const LENS_Y = 1.86;
const LENS_RADIUS = 0.35;
const BEAM_RADIUS = 1.4;
const PLATFORM_RADIUS = 1.45;
const PLATFORM_DEPTH = 0.16;
const CENTRE_OFFSET = -0.22;
const ATHLETE_SCALE = 0.78;
const ATHLETE_TOP = 3.9 * ATHLETE_SCALE; // dumbbells overhead
const STANDING_TOP = 3.4 * ATHLETE_SCALE;
const SCAN_SECONDS = 4.6; // up and back down
const DRONE_SCALE = 1.3;

// --- Shaders --------------------------------------------------------------------

const beamVertex = /* glsl */ `
  varying vec2 vUv;
  varying vec3 vNormal;
  varying vec3 vViewDir;

  void main() {
    vUv = uv;
    vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
    vViewDir = -mvPosition.xyz;
    vNormal = normalize(normalMatrix * normal);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

// Soft light: brightest at the edges and near the sensor, a few slow streaks.
const beamFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform float uTime;
  uniform float uOpacity;
  varying vec2 vUv;
  varying vec3 vNormal;
  varying vec3 vViewDir;

  void main() {
    float edge = pow(1.0 - abs(dot(normalize(vNormal), normalize(vViewDir))), 1.5);
    // uv.y is 1 at the sensor and 0 on the platform.
    float strength = mix(0.12, 1.0, pow(vUv.y, 0.8));
    float angle = vUv.x * 6.28318;
    float streaks = 0.5 + 0.5 * sin(angle * 14.0 + sin(angle * 3.0 + uTime * 0.4) * 1.6);
    float flow = 0.8 + 0.2 * sin(vUv.y * 10.0 + uTime * 2.2);
    float alpha = (0.025 + edge * 0.24 + streaks * 0.025) * strength * flow * uOpacity;
    gl_FragColor = vec4(uColor, alpha);
    #include <colorspace_fragment>
  }
`;

const flatVertex = /* glsl */ `
  varying vec2 vUv;

  void main() {
    vUv = uv;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

// The rings drawn on the platform, and the pulse that runs out from the
// athlete's feet when a set is logged.
const floorFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform float uTime;
  uniform float uOpacity;
  uniform float uPulse;
  varying vec2 vUv;

  float ring(float r, float radius, float width) {
    return 1.0 - smoothstep(0.0, width, abs(r - radius));
  }

  void main() {
    vec2 p = vUv * 2.0 - 1.0;
    float r = length(p);
    float angle = atan(p.y, p.x) / 6.28318 + 0.5;
    float rings = ring(r, 0.9, 0.008) * 0.8 + ring(r, 0.62, 0.006) * 0.5 + ring(r, 0.3, 0.005) * 0.35;
    float dashes = step(0.6, fract(angle * 64.0 + uTime * 0.03)) * ring(r, 0.78, 0.01) * 0.8;
    float ticks = step(0.86, fract(angle * 144.0)) * ring(r, 0.95, 0.018) * 0.5;
    float pulseRadius = 0.12 + uPulse * 0.85;
    float pulse = ring(r, pulseRadius, 0.035) * (1.0 - smoothstep(0.0, 1.1, uPulse));
    float glow = exp(-r * r * 9.0) * 0.38;
    float fade = 1.0 - smoothstep(0.93, 1.0, r);
    float alpha = (rings + dashes + ticks + pulse * 0.9 + glow) * fade * uOpacity;
    gl_FragColor = vec4(uColor, alpha);
    #include <colorspace_fragment>
  }
`;

// A light strip around the platform's side: short segments that drift, and
// brighten with the pulse.
const stripFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform float uTime;
  uniform float uPulse;
  varying vec2 vUv;

  void main() {
    float segment = step(0.42, fract(vUv.x * 96.0 - uTime * 0.05));
    float boost = 1.0 - smoothstep(0.0, 1.2, uPulse);
    float alpha = segment * (0.28 + boost * 0.5);
    gl_FragColor = vec4(uColor, alpha);
    #include <colorspace_fragment>
  }
`;

// The platform's top: deep navy metal (unlit, so the grazing view
// never turns it into a mirror of the studio light), lighter towards the
// centre, with a faint sheen behind the athlete.
const topFragment = /* glsl */ `
  uniform vec3 uDeep;
  uniform vec3 uLift;
  varying vec2 vUv;

  void main() {
    vec2 p = vUv * 2.0 - 1.0;
    float r = length(p);
    vec3 color = mix(uLift, uDeep, smoothstep(0.05, 1.0, r));
    color += vec3(0.3, 0.65, 1.0) * exp(-pow((p.y - 0.45) * 2.4, 2.0)) * exp(-p.x * p.x * 2.0) * 0.05;
    gl_FragColor = vec4(color, 1.0);
    #include <colorspace_fragment>
  }
`;

// The scan plane: a faint disc, brighter towards its rim.
const planeFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform float uOpacity;
  varying vec2 vUv;

  void main() {
    float r = length(vUv * 2.0 - 1.0);
    float alpha = (smoothstep(0.2, 1.0, r) * 0.6 + 0.08) * (1.0 - smoothstep(0.97, 1.0, r)) * uOpacity;
    gl_FragColor = vec4(uColor, alpha);
    #include <colorspace_fragment>
  }
`;

const dustVertex = /* glsl */ `
  attribute float aSeed;
  uniform float uTime;
  uniform float uPixelRatio;
  uniform float uFloor;
  uniform float uHeight;
  uniform float uRadius;
  uniform float uTop;
  varying float vAlpha;

  void main() {
    float t = fract(aSeed * 7.13 + uTime * (0.04 + aSeed * 0.05));
    float y = uFloor + t * uHeight;
    // The beam narrows towards the sensor.
    float coneRadius = mix(uRadius, uTop, t);
    vec3 p = vec3(position.x * coneRadius, y, position.z * coneRadius);
    vec4 mvPosition = modelViewMatrix * vec4(p, 1.0);
    vAlpha = sin(t * 3.14159) * (0.35 + 0.45 * fract(aSeed * 13.7));
    gl_PointSize = (1.2 + fract(aSeed * 31.0) * 2.0) * uPixelRatio * (8.0 / -mvPosition.z);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

const pointFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform float uOpacity;
  varying float vAlpha;

  void main() {
    float d = length(gl_PointCoord - 0.5);
    float alpha = smoothstep(0.5, 0.0, d) * vAlpha * uOpacity;
    gl_FragColor = vec4(uColor, alpha);
    #include <colorspace_fragment>
  }
`;

// Data rising from the worked muscle to the sensor (positions set per frame).
const dataVertex = /* glsl */ `
  attribute float aAlpha;
  attribute float aSize;
  uniform float uPixelRatio;
  varying float vAlpha;

  void main() {
    vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
    vAlpha = aAlpha;
    gl_PointSize = aSize * uPixelRatio * (8.0 / -mvPosition.z);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

// --- Helpers ---------------------------------------------------------------------

const glowTexture = () => {
  const canvas = document.createElement("canvas");
  canvas.width = 128;
  canvas.height = 128;
  const context = canvas.getContext("2d");
  if (context) {
    const gradient = context.createRadialGradient(64, 64, 0, 64, 64, 64);
    gradient.addColorStop(0, "rgba(255,255,255,1)");
    gradient.addColorStop(0.25, "rgba(160,240,255,0.55)");
    gradient.addColorStop(1, "rgba(0,160,255,0)");
    context.fillStyle = gradient;
    context.fillRect(0, 0, 128, 128);
  }
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  return texture;
};

const smooth = (t: number) => t * t * (3 - 2 * t);
const clamp01 = (t: number) => Math.min(Math.max(t, 0), 1);
const beamRadiusAt = (y: number) => LENS_RADIUS + ((LENS_Y - y) / (LENS_Y - FLOOR_Y)) * (BEAM_RADIUS - LENS_RADIUS);

export const createHologram = async (track: Track, pixelRatio: number): Promise<Hologram> => {
  const group = new THREE.Group();
  const rig = new THREE.Group();
  rig.position.y = CENTRE_OFFSET;
  group.add(rig);
  const glow = track(glowTexture());
  const additive = { transparent: true, depthWrite: false, blending: THREE.AdditiveBlending } as const;

  // --- Athlete ---------------------------------------------------------------------
  const athlete = await createAthlete(track, pixelRatio, glow, PLATFORM_RADIUS);
  athlete.group.position.y = FLOOR_Y;
  athlete.group.scale.setScalar(ATHLETE_SCALE);
  rig.add(athlete.group);

  // --- Platform ----------------------------------------------------------------------
  // A dark polished-metal top. It is drawn first (before the athlete's depth
  // pre-pass, so the translucent body shows the platform behind it) and marks
  // the stencil, so the athlete's reflection shows only on it.
  const top = new THREE.Mesh(
    track(new THREE.CircleGeometry(PLATFORM_RADIUS, 128)),
    track(
      new THREE.ShaderMaterial({
        uniforms: { uDeep: { value: new THREE.Color(0x030a22) }, uLift: { value: new THREE.Color(0x0c2a66) } },
        vertexShader: flatVertex,
        fragmentShader: topFragment,
        stencilWrite: true,
        stencilRef: 1,
        stencilFunc: THREE.AlwaysStencilFunc,
        stencilZPass: THREE.ReplaceStencilOp
      })
    )
  );
  top.rotation.x = -Math.PI / 2;
  top.position.y = FLOOR_Y;
  top.renderOrder = -2;
  rig.add(top);

  const metal = track(new THREE.MeshStandardMaterial({ color: 0x0a1534, metalness: 0.8, roughness: 0.38, envMapIntensity: 0.35 }));
  const side = new THREE.Mesh(track(new THREE.CylinderGeometry(PLATFORM_RADIUS, PLATFORM_RADIUS + 0.05, PLATFORM_DEPTH, 128, 1, true)), metal);
  side.position.y = FLOOR_Y - PLATFORM_DEPTH / 2;
  rig.add(side);
  const bevel = new THREE.Mesh(
    track(new THREE.TorusGeometry(PLATFORM_RADIUS, 0.016, 10, 160)),
    track(new THREE.MeshStandardMaterial({ color: 0x3a4f7c, metalness: 0.95, roughness: 0.22 }))
  );
  bevel.rotation.x = Math.PI / 2;
  bevel.position.y = FLOOR_Y;
  rig.add(bevel);
  const rimLight = new THREE.Mesh(
    track(new THREE.TorusGeometry(PLATFORM_RADIUS + 0.018, 0.005, 6, 160)),
    track(new THREE.MeshBasicMaterial({ color: CYAN_LIGHT, transparent: true, opacity: 0.85, toneMapped: false }))
  );
  rimLight.rotation.x = Math.PI / 2;
  rimLight.position.y = FLOOR_Y - 0.012;
  rig.add(rimLight);

  const pulseUniform = { value: 10 };
  const timeUniform = { value: 0 };
  const strip = new THREE.Mesh(
    track(new THREE.CylinderGeometry(PLATFORM_RADIUS + 0.033, PLATFORM_RADIUS + 0.035, 0.016, 160, 1, true)),
    track(
      new THREE.ShaderMaterial({
        uniforms: { uColor: { value: ICE.clone() }, uTime: timeUniform, uPulse: pulseUniform },
        vertexShader: flatVertex,
        fragmentShader: stripFragment,
        ...additive
      })
    )
  );
  strip.position.y = FLOOR_Y - PLATFORM_DEPTH * 0.55;
  rig.add(strip);

  const floorMaterial = track(
    new THREE.ShaderMaterial({
      uniforms: { uColor: { value: ICE.clone() }, uTime: timeUniform, uOpacity: { value: 1 }, uPulse: pulseUniform },
      vertexShader: flatVertex,
      fragmentShader: floorFragment,
      side: THREE.DoubleSide,
      ...additive
    })
  );
  const floor = new THREE.Mesh(track(new THREE.CircleGeometry(PLATFORM_RADIUS, 128)), floorMaterial);
  floor.rotation.x = -Math.PI / 2;
  floor.position.y = FLOOR_Y + 0.004;
  rig.add(floor);

  // The platform's frame: the athlete's reflection mirrors across it.
  const floorFrame = new THREE.Object3D();
  floorFrame.position.y = FLOOR_Y;
  rig.add(floorFrame);
  const flipY = new THREE.Matrix4().makeScale(1, -1, 1);
  const mirror = new THREE.Matrix4();
  const floorInverse = new THREE.Matrix4();

  // --- Beam, scan plane and particles ----------------------------------------------
  const beamHeight = LENS_Y - FLOOR_Y;
  const beamMaterial = track(
    new THREE.ShaderMaterial({
      uniforms: { uColor: { value: ICE.clone() }, uTime: timeUniform, uOpacity: { value: 1 } },
      vertexShader: beamVertex,
      fragmentShader: beamFragment,
      side: THREE.DoubleSide,
      ...additive
    })
  );
  const beam = new THREE.Mesh(track(new THREE.CylinderGeometry(LENS_RADIUS, BEAM_RADIUS, beamHeight, 96, 1, true)), beamMaterial);
  beam.position.y = (LENS_Y + FLOOR_Y) / 2;
  rig.add(beam);

  // A soft light behind the athlete, so the projection glows from within.
  const halo = new THREE.Sprite(
    track(
      new THREE.SpriteMaterial({ map: glow, color: 0x36d6ff, transparent: true, opacity: 0.14, depthWrite: false, blending: THREE.AdditiveBlending, toneMapped: false })
    )
  );
  halo.scale.set(2.4, 3.4, 1);
  halo.position.set(0, FLOOR_Y + 1.4, -0.45);
  rig.add(halo);

  // The scan plane follows the scanner's line on the body: a bright rim at the
  // beam's edge over a faint disc.
  const scanPlane = new THREE.Group();
  const scanRingMaterial = track(new THREE.MeshBasicMaterial({ color: ICE, side: THREE.DoubleSide, toneMapped: false, opacity: 0.6, ...additive }));
  const scanRing = new THREE.Mesh(track(new THREE.RingGeometry(0.985, 1, 128)), scanRingMaterial);
  const planeMaterial = track(
    new THREE.ShaderMaterial({
      uniforms: { uColor: { value: ICE.clone() }, uOpacity: { value: 0.14 } },
      vertexShader: flatVertex,
      fragmentShader: planeFragment,
      side: THREE.DoubleSide,
      ...additive
    })
  );
  const scanDisc = new THREE.Mesh(track(new THREE.CircleGeometry(1, 96)), planeMaterial);
  for (const mesh of [scanRing, scanDisc]) {
    mesh.rotation.x = -Math.PI / 2;
    scanPlane.add(mesh);
  }
  rig.add(scanPlane);

  const dustCount = 170;
  const dustPositions = new Float32Array(dustCount * 3);
  const dustSeeds = new Float32Array(dustCount);
  for (let index = 0; index < dustCount; index += 1) {
    const angle = Math.random() * Math.PI * 2;
    const radius = Math.sqrt(Math.random()) * 0.92;
    dustPositions[index * 3] = Math.cos(angle) * radius;
    dustPositions[index * 3 + 2] = Math.sin(angle) * radius;
    dustSeeds[index] = Math.random();
  }
  const dustGeometry = track(new THREE.BufferGeometry());
  dustGeometry.setAttribute("position", new THREE.BufferAttribute(dustPositions, 3));
  dustGeometry.setAttribute("aSeed", new THREE.BufferAttribute(dustSeeds, 1));
  const dust = new THREE.Points(
    dustGeometry,
    track(
      new THREE.ShaderMaterial({
        uniforms: {
          uColor: { value: ICE.clone() },
          uTime: timeUniform,
          uOpacity: { value: 0.8 },
          uPixelRatio: { value: pixelRatio },
          uFloor: { value: FLOOR_Y },
          uHeight: { value: beamHeight },
          uRadius: { value: BEAM_RADIUS },
          uTop: { value: LENS_RADIUS }
        },
        vertexShader: dustVertex,
        fragmentShader: pointFragment,
        ...additive
      })
    )
  );
  dust.frustumCulled = false;
  rig.add(dust);

  // Data bursts: when a set is logged, points rise from the worked muscle to
  // the sensor along a gentle curve.
  const DATA_POINTS = 22;
  const DATA_TRAVEL = 0.85;
  const LOG_DELAY = DATA_TRAVEL + 0.3;
  const dataPositions = new Float32Array(DATA_POINTS * 3);
  const dataAlpha = new Float32Array(DATA_POINTS);
  const dataSize = Float32Array.from({ length: DATA_POINTS }, (_, index) => 2.2 + (index % 4) * 0.7);
  const dataDelay = Float32Array.from({ length: DATA_POINTS }, (_, index) => (index / DATA_POINTS) * 0.45);
  const dataSpread = Float32Array.from({ length: DATA_POINTS * 2 }, () => Math.random() * 2 - 1);
  const dataGeometry = track(new THREE.BufferGeometry());
  const dataPosition = new THREE.BufferAttribute(dataPositions, 3).setUsage(THREE.DynamicDrawUsage);
  const dataOpacity = new THREE.BufferAttribute(dataAlpha, 1).setUsage(THREE.DynamicDrawUsage);
  dataGeometry.setAttribute("position", dataPosition);
  dataGeometry.setAttribute("aAlpha", dataOpacity);
  dataGeometry.setAttribute("aSize", new THREE.BufferAttribute(dataSize, 1));
  const data = new THREE.Points(
    dataGeometry,
    track(
      new THREE.ShaderMaterial({
        uniforms: { uColor: { value: new THREE.Color(0xe8fdff) }, uOpacity: { value: 1 }, uPixelRatio: { value: pixelRatio } },
        vertexShader: dataVertex,
        fragmentShader: pointFragment,
        ...additive
      })
    )
  );
  data.frustumCulled = false;
  rig.add(data);

  // --- Scanner -----------------------------------------------------------------------
  const drone = new THREE.Group();
  drone.scale.setScalar(DRONE_SCALE);
  rig.add(drone);
  const graphite = track(new THREE.MeshPhysicalMaterial({ color: 0x151c2e, metalness: 0.85, roughness: 0.34, clearcoat: 1, clearcoatRoughness: 0.22 }));
  const trim = track(new THREE.MeshStandardMaterial({ color: 0x3a4a6e, metalness: 0.92, roughness: 0.24 }));
  const glass = track(new THREE.MeshPhysicalMaterial({ color: 0x040914, metalness: 0.3, roughness: 0.06, clearcoat: 1 }));
  const lightMaterial = track(new THREE.MeshBasicMaterial({ color: CYAN_LIGHT, transparent: true, toneMapped: false }));
  const dimLight = track(new THREE.MeshBasicMaterial({ color: CYAN_LIGHT, transparent: true, opacity: 0.45, toneMapped: false }));

  // The pod: a low, lens-shaped body.
  const profile = [
    [0, 0.118],
    [0.06, 0.115],
    [0.12, 0.104],
    [0.18, 0.084],
    [0.24, 0.058],
    [0.29, 0.032],
    [0.32, 0.008],
    [0.325, -0.008],
    [0.31, -0.028],
    [0.26, -0.044],
    [0.18, -0.053],
    [0, -0.056]
  ].map(([r, y]) => new THREE.Vector2(r, y));
  drone.add(new THREE.Mesh(track(new THREE.LatheGeometry(profile, 72)), graphite));
  const waist = new THREE.Mesh(track(new THREE.TorusGeometry(0.327, 0.006, 8, 128)), lightMaterial);
  waist.rotation.x = Math.PI / 2;
  waist.position.y = -0.001;
  drone.add(waist);
  const crown = new THREE.Mesh(track(new THREE.TorusGeometry(0.15, 0.0035, 6, 72)), dimLight);
  crown.rotation.x = Math.PI / 2;
  crown.position.y = 0.1;
  drone.add(crown);

  // The sensor array underneath: dark glass, two light rings and a bright core.
  const facingDown = (mesh: THREE.Mesh, y: number) => {
    mesh.rotation.x = Math.PI / 2;
    mesh.position.y = y;
    drone.add(mesh);
    return mesh;
  };
  facingDown(new THREE.Mesh(track(new THREE.CircleGeometry(0.255, 72)), glass), -0.0565);
  const aperture = facingDown(new THREE.Mesh(track(new THREE.RingGeometry(0.258, 0.27, 96)), lightMaterial), -0.0575);
  facingDown(new THREE.Mesh(track(new THREE.RingGeometry(0.15, 0.156, 72)), dimLight), -0.0575);
  facingDown(new THREE.Mesh(track(new THREE.CircleGeometry(0.045, 40)), track(new THREE.MeshBasicMaterial({ color: 0xe8fdff, toneMapped: false }))), -0.058);
  const lensGlow = new THREE.Sprite(
    track(new THREE.SpriteMaterial({ map: glow, color: 0x9af3ff, transparent: true, depthWrite: false, blending: THREE.AdditiveBlending, toneMapped: false }))
  );
  lensGlow.scale.setScalar(0.5);
  lensGlow.position.y = -0.07;
  drone.add(lensGlow);

  // Four slim arms to ducted rotors.
  const armGeometry = track(new THREE.BoxGeometry(0.34, 0.02, 0.046));
  const ductGeometry = track(new THREE.TorusGeometry(0.12, 0.02, 14, 64));
  const ductLightGeometry = track(new THREE.TorusGeometry(0.1, 0.0032, 6, 64));
  const hubGeometry = track(new THREE.CylinderGeometry(0.022, 0.026, 0.03, 16));
  const bladeGeometry = track(new THREE.BoxGeometry(0.19, 0.003, 0.018));
  const blurMaterial = track(
    new THREE.MeshBasicMaterial({ map: glow, color: 0xbff6ff, transparent: true, opacity: 0.12, depthWrite: false, blending: THREE.AdditiveBlending, toneMapped: false })
  );
  const blurGeometry = track(new THREE.CircleGeometry(0.1, 40));
  const rotors: THREE.Group[] = [];
  for (let index = 0; index < 4; index += 1) {
    const angle = Math.PI / 4 + (index * Math.PI) / 2;
    const dir = new THREE.Vector3(Math.cos(angle), 0, Math.sin(angle));
    const arm = new THREE.Mesh(armGeometry, graphite);
    arm.position.copy(dir).multiplyScalar(0.44).setY(-0.004);
    arm.rotation.y = -angle;
    drone.add(arm);
    const centre = dir.clone().multiplyScalar(0.64);
    const duct = new THREE.Mesh(ductGeometry, trim);
    duct.rotation.x = Math.PI / 2;
    duct.position.copy(centre);
    drone.add(duct);
    const ductLight = new THREE.Mesh(ductLightGeometry, dimLight);
    ductLight.rotation.x = Math.PI / 2;
    ductLight.position.copy(centre).setY(-0.012);
    drone.add(ductLight);
    const rotor = new THREE.Group();
    rotor.position.copy(centre);
    const blur = new THREE.Mesh(blurGeometry, blurMaterial);
    blur.rotation.x = -Math.PI / 2;
    rotor.add(new THREE.Mesh(hubGeometry, graphite), new THREE.Mesh(bladeGeometry, trim), blur);
    drone.add(rotor);
    rotors.push(rotor);
  }

  // --- Animation ----------------------------------------------------------------------
  const scanPoint = new THREE.Vector3();
  const revealPoint = new THREE.Vector3();
  const lens = new THREE.Vector3();
  const base = new THREE.Vector3();
  const from = new THREE.Vector3();
  const to = new THREE.Vector3(0, LENS_Y - 0.08, 0);
  const control = new THREE.Vector3();
  let nextGlitch = 3.5;
  let glitchUntil = 0;
  let glitchY = 0;

  const update = (elapsed: number, emphasis: number): HologramState => {
    group.updateMatrixWorld(true);

    // The scanner's line runs up the body and back down.
    const phase = (elapsed % SCAN_SECONDS) / SCAN_SECONDS;
    const scanLocal = FLOOR_Y + 0.03 + (STANDING_TOP + 0.05) * (0.5 - 0.5 * Math.cos(phase * Math.PI * 2));
    const scanDir = phase < 0.5 ? 1 : -1;
    rig.localToWorld(scanPoint.set(0, scanLocal, 0));
    scanPlane.position.y = scanLocal;
    scanPlane.scale.setScalar(beamRadiusAt(scanLocal));

    // Materialises from the floor in the first seconds.
    const reveal = smooth(clamp01((elapsed - 0.25) / 1.8));
    rig.localToWorld(revealPoint.set(0, FLOOR_Y + reveal * (ATHLETE_TOP + 0.2), 0));

    if (elapsed > nextGlitch) {
      glitchUntil = elapsed + 0.12;
      glitchY = scanPoint.y + (Math.random() - 0.5) * 1.5;
      nextGlitch = elapsed + 5 + Math.random() * 5;
    }

    floorInverse.copy(floorFrame.matrixWorld).invert();
    mirror.copy(floorFrame.matrixWorld).multiply(flipY).multiply(floorInverse);

    const state = athlete.update({
      elapsed,
      emphasis,
      presence: reveal,
      scanY: scanPoint.y,
      scanDir,
      reveal: reveal >= 1 ? 1e3 : revealPoint.y,
      glitch: elapsed < glitchUntil ? 1 : 0,
      glitchY,
      mirror,
      floorInverse
    });

    // Each completed set sends a burst of data from the worked muscle to the
    // sensor; it is logged when it arrives. All of it follows from the time
    // into the exercise, so the readout can never drift from the scene.
    const bursts = Math.min(ROUTINE_SETS, Math.floor(state.time / ROUTINE_REP_SECONDS));
    const sinceBurst = bursts > 0 ? state.time - bursts * ROUTINE_REP_SECONDS : Infinity;
    const landed = sinceBurst >= LOG_DELAY;
    const arrived = landed ? bursts : Math.max(0, bursts - 1);
    const logged = arrived === 0 ? -1 : landed ? sinceBurst - LOG_DELAY : sinceBurst + ROUTINE_REP_SECONDS - LOG_DELAY;
    rig.worldToLocal(from.copy(state.muscle));
    control.copy(from).lerp(to, 0.5);
    control.x += from.x * 0.6;
    control.z += from.z * 0.6;
    for (let index = 0; index < DATA_POINTS; index += 1) {
      const t = clamp01((sinceBurst - dataDelay[index]) / DATA_TRAVEL);
      const s = smooth(t);
      // Quadratic Bézier, with a little spread around the path.
      const a = (1 - s) * (1 - s);
      const b = 2 * (1 - s) * s;
      const c = s * s;
      const spread = Math.sin(s * Math.PI) * 0.09;
      dataPositions[index * 3] = a * from.x + b * control.x + c * to.x + dataSpread[index * 2] * spread;
      dataPositions[index * 3 + 1] = a * from.y + b * control.y + c * to.y;
      dataPositions[index * 3 + 2] = a * from.z + b * control.z + c * to.z + dataSpread[index * 2 + 1] * spread;
      dataAlpha[index] = t > 0 && t < 1 ? Math.sin(t * Math.PI) * reveal * emphasis : 0;
    }
    dataPosition.needsUpdate = true;
    dataOpacity.needsUpdate = true;
    // The data reaches the sensor; the platform pulses and the readout logs it.
    pulseUniform.value = logged >= 0 ? Math.min(logged, 10) : 10;

    timeUniform.value = elapsed;
    beamMaterial.uniforms.uOpacity.value = 0.55 + 0.45 * emphasis;
    floorMaterial.uniforms.uOpacity.value = 0.7 + 0.3 * emphasis;
    planeMaterial.uniforms.uOpacity.value = 0.14 * reveal;
    scanRingMaterial.opacity = 0.6 * reveal;

    // The scanner hovers with a slow, small sway; its lights brighten as data arrives.
    drone.position.y = LENS_Y + 0.14 + Math.sin(elapsed * 1.3) * 0.025;
    // Pitched a little towards the viewer, so its sensor array reads.
    drone.rotation.set(-0.24 + Math.sin(elapsed * 0.7) * 0.02, Math.sin(elapsed * 0.23) * 0.12, Math.cos(elapsed * 0.9) * 0.018);
    rotors.forEach((rotor, index) => {
      rotor.rotation.y = elapsed * (index % 2 === 0 ? 40 : -40);
    });
    const arrival = logged >= 0 ? 1 - smooth(clamp01(logged / 0.6)) : 0;
    lightMaterial.opacity = Math.min(1, 0.75 + 0.12 * Math.sin(elapsed * 2.4) + arrival * 0.25);
    aperture.scale.setScalar(1 + arrival * 0.04);
    lensGlow.material.opacity = 0.6 + Math.sin(elapsed * 3) * 0.08 + arrival * 0.4;
    halo.material.opacity = (0.12 + 0.1 * state.stimulus) * reveal;

    drone.updateMatrixWorld(true);
    drone.localToWorld(lens.set(0, -0.06, 0));
    floorFrame.getWorldPosition(base);
    return { ...state, presence: reveal, lens, base, arrived, logged };
  };

  update(0, 1);
  return { group, update };
};
