import * as THREE from "three";
import { RoundedBoxGeometry } from "three/addons/geometries/RoundedBoxGeometry.js";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";

export interface FeedConstructionScene {
  setPointer: (x: number, y: number) => void;
  start: () => void;
  stop: () => void;
  resize: (width: number, height: number) => void;
  dispose: () => void;
}

const COLORS = {
  cyan: new THREE.Color(0x36d6ff),
  ice: new THREE.Color(0xe8fdff),
  blue: new THREE.Color(0x0481dc),
  navy: 0x02041f
} as const;

const INTRO = {
  droneIn: [0, 1.9],
  beamOpen: [2.05, 2.6],
  workerBuild: [2.45, 4.75],
  beamClose: [5.05, 5.55],
  droneOut: [5.35, 7.2]
} as const;

const clamp01 = (value: number) => Math.min(Math.max(value, 0), 1);
const smooth = (value: number) => value * value * (3 - 2 * value);
const span = (time: number, [from, to]: readonly [number, number]) => clamp01((time - from) / (to - from));
const easeOutBack = (value: number) => {
  const c = 1.25;
  return 1 + (c + 1) * Math.pow(value - 1, 3) + c * Math.pow(value - 1, 2);
};
const easeInCubic = (value: number) => value * value * value;

const additive = {
  transparent: true,
  depthWrite: false,
  blending: THREE.AdditiveBlending,
  toneMapped: false
} as const;

const hologramVertex = /* glsl */ `
  varying vec3 vNormal;
  varying vec3 vViewDir;
  varying float vWorldY;
  varying vec2 vUv;

  void main() {
    vUv = uv;
    vec4 world = modelMatrix * vec4(position, 1.0);
    vWorldY = world.y;
    vNormal = normalize(normalMatrix * normal);
    vec4 viewPosition = viewMatrix * world;
    vViewDir = -viewPosition.xyz;
    gl_Position = projectionMatrix * viewPosition;
  }
`;

const hologramFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform vec3 uHot;
  uniform float uTime;
  uniform float uReveal;
  uniform float uOpacity;
  varying vec3 vNormal;
  varying vec3 vViewDir;
  varying float vWorldY;
  varying vec2 vUv;

  void main() {
    if (vWorldY > uReveal) discard;
    float facing = abs(dot(normalize(vNormal), normalize(vViewDir)));
    float rim = pow(1.0 - facing, 2.0);
    float scan = smoothstep(0.72, 1.0, abs(fract(vWorldY * 30.0 - uTime * 0.55) * 2.0 - 1.0));
    float grid = smoothstep(0.92, 1.0, abs(fract(vUv.x * 18.0 + vUv.y * 9.0) * 2.0 - 1.0));
    float revealEdge = exp(-pow((vWorldY - uReveal) * 26.0, 2.0));
    float flicker = 0.92 + 0.08 * sin(uTime * 29.0 + vWorldY * 8.0);
    vec3 color = mix(uColor, uHot, clamp(rim * 0.9 + scan * 0.22 + revealEdge * 0.8, 0.0, 1.0));
    float light = 0.22 + rim * 1.7 + scan * 0.3 + grid * 0.12 + revealEdge * 2.8;
    float alpha = (0.15 + rim * 0.68 + scan * 0.18 + grid * 0.08 + revealEdge) * flicker * uOpacity;
    gl_FragColor = vec4(color * light, alpha);
    #include <colorspace_fragment>
  }
`;

const createHologramMaterial = () =>
  new THREE.ShaderMaterial({
    uniforms: {
      uColor: { value: COLORS.cyan.clone() },
      uHot: { value: COLORS.ice.clone() },
      uTime: { value: 0 },
      uReveal: { value: -10000 },
      uOpacity: { value: 0 }
    },
    vertexShader: hologramVertex,
    fragmentShader: hologramFragment,
    side: THREE.DoubleSide,
    ...additive
  });

const segment = (material: THREE.Material, from: THREE.Vector3, to: THREE.Vector3, radius: number) => {
  const direction = to.clone().sub(from);
  const mesh = new THREE.Mesh(new THREE.CylinderGeometry(radius, radius, direction.length(), 16, 1), material);
  mesh.position.copy(from).add(to).multiplyScalar(0.5);
  mesh.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), direction.normalize());
  return mesh;
};

const steelBeam = (
  material: THREE.Material,
  from: THREE.Vector3,
  to: THREE.Vector3,
  width: number,
  depth: number
) => {
  const direction = to.clone().sub(from);
  const length = direction.length();
  const beam = new THREE.Group();
  beam.position.copy(from).add(to).multiplyScalar(0.5);
  beam.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), direction.normalize());

  const web = new THREE.Mesh(new RoundedBoxGeometry(width * 0.32, length, depth * 0.5, 3, 0.018), material);
  const flangeGeometry = new RoundedBoxGeometry(width, 0.065, depth, 3, 0.018);
  const top = new THREE.Mesh(flangeGeometry, material);
  const bottom = new THREE.Mesh(flangeGeometry, material);
  top.position.y = length * 0.5 - 0.035;
  bottom.position.y = -length * 0.5 + 0.035;
  beam.add(web, top, bottom);
  return beam;
};

const bolt = (material: THREE.Material, position: THREE.Vector3, rotation = 0) => {
  const mesh = new THREE.Mesh(new THREE.CylinderGeometry(0.032, 0.032, 0.022, 16), material);
  mesh.position.copy(position);
  mesh.rotation.set(Math.PI / 2, rotation, 0);
  return mesh;
};

const createConstruction = (material: THREE.Material, accentMaterial: THREE.Material) => {
  const root = new THREE.Group();
  const worker = new THREE.Group();
  const beams = new THREE.Group();

  // A proper steel portal frame: I-beams, gusset plates and visible fasteners
  // give the hologram a readable manufactured silhouette instead of a set of
  // disconnected glowing cylinders.
  beams.add(
    steelBeam(material, new THREE.Vector3(-1.26, 0.08, -0.24), new THREE.Vector3(-1.26, 2.02, -0.24), 0.22, 0.2),
    steelBeam(material, new THREE.Vector3(1.26, 0.08, -0.24), new THREE.Vector3(1.26, 2.02, -0.24), 0.22, 0.2),
    steelBeam(material, new THREE.Vector3(-1.45, 2.02, -0.24), new THREE.Vector3(1.45, 2.02, -0.24), 0.28, 0.22),
    steelBeam(material, new THREE.Vector3(-1.12, 0.72, -0.18), new THREE.Vector3(1.12, 1.86, -0.18), 0.13, 0.13)
  );

  const gusset = new THREE.Mesh(new RoundedBoxGeometry(0.38, 0.06, 0.22, 3, 0.018), material);
  gusset.rotation.z = -0.65;
  gusset.position.set(0.98, 1.74, -0.06);
  beams.add(gusset);

  for (const x of [-1.26, 1.26]) {
    for (const y of [0.48, 1.14, 1.85]) {
      beams.add(bolt(accentMaterial, new THREE.Vector3(x, y, -0.1)), bolt(accentMaterial, new THREE.Vector3(x + (x < 0 ? 0.04 : -0.04), y + 0.1, -0.1)));
    }
  }
  for (const x of [-0.94, -0.48, 0, 0.48, 0.94]) {
    beams.add(bolt(accentMaterial, new THREE.Vector3(x, 2.02, -0.1), Math.PI / 2));
  }

  const torso = new THREE.Mesh(new RoundedBoxGeometry(0.66, 0.82, 0.42, 5, 0.08), material);
  torso.position.set(0, 1.26, 0.08);
  worker.add(torso);

  // Safety vest, harness and utility belt. These small layered surfaces create
  // the same intentional detail density as the site's existing holograms.
  const vest = new THREE.Mesh(new RoundedBoxGeometry(0.54, 0.63, 0.045, 3, 0.018), accentMaterial);
  vest.position.set(0, 1.29, 0.305);
  worker.add(vest);
  for (const x of [-0.2, 0.2]) {
    const strap = new THREE.Mesh(new RoundedBoxGeometry(0.075, 0.7, 0.035, 2, 0.012), accentMaterial);
    strap.position.set(x, 1.3, 0.34);
    strap.rotation.z = x < 0 ? -0.12 : 0.12;
    worker.add(strap);
  }
  const belt = new THREE.Mesh(new RoundedBoxGeometry(0.7, 0.095, 0.46, 3, 0.02), accentMaterial);
  belt.position.set(0, 0.9, 0.08);
  worker.add(belt);
  const toolPouch = new THREE.Mesh(new RoundedBoxGeometry(0.2, 0.24, 0.16, 3, 0.025), accentMaterial);
  toolPouch.position.set(-0.38, 0.78, 0.1);
  worker.add(toolPouch);

  const neck = new THREE.Mesh(new THREE.CylinderGeometry(0.13, 0.15, 0.18, 16), material);
  neck.position.set(0, 1.78, 0.08);
  worker.add(neck);
  const head = new THREE.Mesh(new THREE.SphereGeometry(0.255, 24, 16), material);
  head.scale.set(0.92, 1.08, 0.9);
  head.position.set(0, 2.02, 0.08);
  worker.add(head);

  const helmet = new THREE.Mesh(new THREE.SphereGeometry(0.33, 24, 12, 0, Math.PI * 2, 0, Math.PI / 2), accentMaterial);
  helmet.position.set(0, 2.18, 0.08);
  worker.add(helmet);
  const brim = new THREE.Mesh(new RoundedBoxGeometry(0.66, 0.055, 0.3, 3, 0.02), accentMaterial);
  brim.position.set(0, 2.08, 0.17);
  const helmetBand = new THREE.Mesh(new THREE.TorusGeometry(0.285, 0.018, 8, 32), material);
  helmetBand.rotation.x = Math.PI / 2;
  helmetBand.position.set(0, 2.17, 0.08);
  const visor = new THREE.Mesh(new RoundedBoxGeometry(0.38, 0.09, 0.025, 3, 0.015), accentMaterial);
  visor.position.set(0, 2.02, 0.315);
  worker.add(brim, helmetBand, visor);
  for (const x of [-0.31, 0.31]) {
    const ear = new THREE.Mesh(new RoundedBoxGeometry(0.06, 0.2, 0.1, 3, 0.018), accentMaterial);
    ear.position.set(x, 1.99, 0.08);
    worker.add(ear);
  }

  worker.add(
    segment(material, new THREE.Vector3(-0.22, 0.88, 0.08), new THREE.Vector3(-0.28, 0.2, 0.1), 0.11),
    segment(material, new THREE.Vector3(0.22, 0.88, 0.08), new THREE.Vector3(0.3, 0.2, 0.1), 0.11),
    segment(accentMaterial, new THREE.Vector3(-0.28, 0.18, 0.1), new THREE.Vector3(-0.48, 0.12, 0.24), 0.12),
    segment(accentMaterial, new THREE.Vector3(0.3, 0.18, 0.1), new THREE.Vector3(0.5, 0.12, 0.24), 0.12),
    segment(material, new THREE.Vector3(-0.34, 1.58, 0.08), new THREE.Vector3(-0.66, 1.3, 0.1), 0.085),
    segment(material, new THREE.Vector3(-0.66, 1.3, 0.1), new THREE.Vector3(-0.52, 1.02, 0.14), 0.07)
  );
  worker.add(
    new THREE.Mesh(new THREE.SphereGeometry(0.1, 16, 10), accentMaterial),
    new THREE.Mesh(new THREE.SphereGeometry(0.1, 16, 10), accentMaterial)
  );
  (worker.children[worker.children.length - 2] as THREE.Mesh).position.set(-0.34, 1.58, 0.08);
  (worker.children[worker.children.length - 1] as THREE.Mesh).position.set(-0.52, 1.02, 0.14);

  // The right arm is a separate pivot so the hammer can strike the beam.
  const hammerArm = new THREE.Group();
  hammerArm.position.set(0.34, 1.58, 0.08);
  hammerArm.rotation.z = -0.38;
  hammerArm.add(
    segment(material, new THREE.Vector3(0, 0, 0), new THREE.Vector3(0.16, -0.28, 0.02), 0.085),
    segment(material, new THREE.Vector3(0.16, -0.28, 0.02), new THREE.Vector3(0.4, -0.18, 0.04), 0.07),
    new THREE.Mesh(new THREE.SphereGeometry(0.09, 16, 10), accentMaterial)
  );
  (hammerArm.children[2] as THREE.Mesh).position.set(0.4, -0.18, 0.04);
  const hammerHandle = new THREE.Mesh(new RoundedBoxGeometry(0.065, 0.48, 0.065, 3, 0.018), accentMaterial);
  hammerHandle.position.set(0.4, 0.04, 0.04);
  hammerHandle.rotation.z = -0.12;
  hammerArm.add(hammerHandle);
  const hammerHead = new THREE.Mesh(new RoundedBoxGeometry(0.34, 0.15, 0.19, 4, 0.03), accentMaterial);
  hammerHead.position.set(0.4, 0.29, 0.04);
  hammerHead.rotation.z = -0.12;
  const hammerFace = new THREE.Mesh(new THREE.CylinderGeometry(0.095, 0.095, 0.08, 16), material);
  hammerFace.rotation.z = Math.PI / 2;
  hammerFace.position.set(0.56, 0.29, 0.04);
  hammerArm.add(hammerHead, hammerFace);
  worker.add(hammerArm);

  root.add(beams, worker);
  return { root, worker, beams, hammerArm };
};

const createDrone = (
  bodyMaterial: THREE.Material,
  trimMaterial: THREE.Material,
  glassMaterial: THREE.Material,
  lightMaterial: THREE.Material,
  glowTexture: THREE.Texture
) => {
  const drone = new THREE.Group();
  const profile = [
    [0, 0.118], [0.06, 0.115], [0.12, 0.104], [0.18, 0.084], [0.24, 0.058],
    [0.29, 0.032], [0.32, 0.008], [0.325, -0.008], [0.31, -0.028], [0.26, -0.044],
    [0.18, -0.053], [0, -0.056]
  ].map(([radius, height]) => new THREE.Vector2(radius, height));
  drone.add(new THREE.Mesh(new THREE.LatheGeometry(profile, 72), bodyMaterial));

  const waist = new THREE.Mesh(new THREE.TorusGeometry(0.327, 0.006, 8, 128), lightMaterial);
  waist.rotation.x = Math.PI / 2;
  drone.add(waist);
  const crown = new THREE.Mesh(new THREE.TorusGeometry(0.15, 0.0035, 6, 72), lightMaterial);
  crown.rotation.x = Math.PI / 2;
  crown.position.y = 0.1;
  drone.add(crown);

  const lens = new THREE.Mesh(new THREE.CircleGeometry(0.255, 72), glassMaterial);
  lens.rotation.x = Math.PI / 2;
  lens.position.y = -0.056;
  const aperture = new THREE.Mesh(new THREE.RingGeometry(0.258, 0.27, 96), lightMaterial);
  aperture.rotation.x = Math.PI / 2;
  aperture.position.y = -0.057;
  const innerAperture = new THREE.Mesh(new THREE.RingGeometry(0.15, 0.156, 72), lightMaterial);
  innerAperture.rotation.x = Math.PI / 2;
  innerAperture.position.y = -0.058;
  const core = new THREE.Mesh(new THREE.CircleGeometry(0.045, 40), lightMaterial);
  core.rotation.x = Math.PI / 2;
  core.position.y = -0.059;
  const lensGlow = new THREE.Sprite(new THREE.SpriteMaterial({ map: glowTexture, color: COLORS.cyan, transparent: true, depthWrite: false, blending: THREE.AdditiveBlending, toneMapped: false }));
  lensGlow.scale.setScalar(0.5);
  lensGlow.position.y = -0.07;
  drone.add(lens, aperture, innerAperture, core, lensGlow);

  const rotors: THREE.Group[] = [];
  const armGeometry = new THREE.BoxGeometry(0.34, 0.02, 0.046);
  const ductGeometry = new THREE.TorusGeometry(0.12, 0.02, 14, 64);
  const ductLightGeometry = new THREE.TorusGeometry(0.1, 0.0032, 6, 64);
  const hubGeometry = new THREE.CylinderGeometry(0.022, 0.026, 0.03, 16);
  const bladeGeometry = new THREE.BoxGeometry(0.19, 0.003, 0.018);
  const blurGeometry = new THREE.CircleGeometry(0.1, 40);
  const blurMaterial = new THREE.MeshBasicMaterial({ map: glowTexture, color: COLORS.ice, transparent: true, opacity: 0.12, depthWrite: false, blending: THREE.AdditiveBlending, toneMapped: false });
  for (let index = 0; index < 4; index += 1) {
    const angle = Math.PI / 4 + (index * Math.PI) / 2;
    const direction = new THREE.Vector3(Math.cos(angle), 0, Math.sin(angle));
    const arm = new THREE.Mesh(armGeometry, bodyMaterial);
    arm.position.copy(direction).multiplyScalar(0.44).setY(-0.004);
    arm.rotation.y = -angle;
    drone.add(arm);
    const centre = direction.clone().multiplyScalar(0.64);
    const duct = new THREE.Mesh(ductGeometry, trimMaterial);
    duct.rotation.x = Math.PI / 2;
    duct.position.copy(centre);
    const ductLight = new THREE.Mesh(ductLightGeometry, lightMaterial);
    ductLight.rotation.x = Math.PI / 2;
    ductLight.position.copy(centre).setY(-0.012);
    drone.add(duct, ductLight);
    const rotor = new THREE.Group();
    rotor.position.copy(centre);
    const blur = new THREE.Mesh(blurGeometry, blurMaterial);
    blur.rotation.x = -Math.PI / 2;
    rotor.add(new THREE.Mesh(hubGeometry, bodyMaterial), new THREE.Mesh(bladeGeometry, trimMaterial), blur);
    drone.add(rotor);
    rotors.push(rotor);
  }
  return { drone, rotors };
};

const createGlowTexture = () => {
  const canvas = document.createElement("canvas");
  canvas.width = 128;
  canvas.height = 128;
  const context = canvas.getContext("2d");
  if (context) {
    const gradient = context.createRadialGradient(64, 64, 0, 64, 64, 64);
    gradient.addColorStop(0, "rgba(255,255,255,1)");
    gradient.addColorStop(0.22, "rgba(160,240,255,0.58)");
    gradient.addColorStop(1, "rgba(0,160,255,0)");
    context.fillStyle = gradient;
    context.fillRect(0, 0, 128, 128);
  }
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  return texture;
};

const createSparkField = (count: number) => {
  const positions = new Float32Array(count * 3);
  const seeds = new Float32Array(count);
  for (let index = 0; index < count; index += 1) {
    seeds[index] = Math.random();
  }
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute("position", new THREE.BufferAttribute(positions, 3));
  const material = new THREE.PointsMaterial({
    color: COLORS.ice,
    size: 0.045,
    transparent: true,
    opacity: 0,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
    sizeAttenuation: true
  });
  return { geometry, material, positions, seeds };
};

export const createFeedConstructionScene = (canvas: HTMLCanvasElement): FeedConstructionScene => {
  const renderer = new THREE.WebGLRenderer({
    canvas,
    antialias: window.devicePixelRatio < 1.5,
    alpha: true,
    powerPreference: "low-power"
  });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 1.5));
  renderer.setClearColor(COLORS.navy, 0);
  renderer.outputColorSpace = THREE.SRGBColorSpace;
  renderer.toneMapping = THREE.ACESFilmicToneMapping;
  renderer.toneMappingExposure = 1.05;

  const scene = new THREE.Scene();
  const pmrem = new THREE.PMREMGenerator(renderer);
  const room = new RoomEnvironment();
  const environment = pmrem.fromScene(room, 0.04).texture;
  scene.environment = environment;
  scene.environmentIntensity = 0.8;
  room.dispose();
  pmrem.dispose();

  const camera = new THREE.PerspectiveCamera(34, 1, 0.1, 40);
  camera.position.set(0, 1.42, 7.1);
  camera.lookAt(0, 1.25, 0);

  scene.add(new THREE.HemisphereLight(0xcfe3ff, COLORS.navy, 0.55));
  const key = new THREE.DirectionalLight(0xffffff, 2.1);
  key.position.set(-4, 5, 7);
  const rim = new THREE.DirectionalLight(COLORS.cyan, 3.2);
  rim.position.set(5, 2.5, -4);
  const fill = new THREE.PointLight(COLORS.blue, 24, 20);
  fill.position.set(2.5, -1, 4);
  scene.add(key, rim, fill);

  const root = new THREE.Group();
  scene.add(root);
  const material = createHologramMaterial();
  const accentMaterial = createHologramMaterial();
  accentMaterial.uniforms.uColor.value.copy(COLORS.ice);
  accentMaterial.uniforms.uHot.value.copy(new THREE.Color(0xffffff));
  const hologramMaterials = [material, accentMaterial];
  const construction = createConstruction(material, accentMaterial);
  root.add(construction.root);

  const platformMaterial = new THREE.MeshPhysicalMaterial({
    color: 0x071631,
    metalness: 0.82,
    roughness: 0.28,
    clearcoat: 1,
    clearcoatRoughness: 0.16,
    transparent: true,
    opacity: 0.92
  });
  const platform = new THREE.Mesh(new THREE.CylinderGeometry(1.65, 1.85, 0.05, 64), platformMaterial);
  platform.position.y = 0.04;
  root.add(platform);

  const ringMaterial = new THREE.MeshBasicMaterial({ color: COLORS.ice, ...additive, opacity: 0.8 });
  const floorRing = new THREE.Mesh(new THREE.RingGeometry(1.8, 1.83, 96), ringMaterial);
  floorRing.rotation.x = -Math.PI / 2;
  floorRing.position.y = 0.08;
  root.add(floorRing);
  const innerRing = new THREE.Mesh(new THREE.RingGeometry(1.05, 1.058, 96), ringMaterial);
  innerRing.rotation.x = -Math.PI / 2;
  innerRing.position.y = 0.085;
  root.add(innerRing);

  const beamMaterial = new THREE.MeshBasicMaterial({ color: COLORS.cyan, ...additive, opacity: 0 });
  const beam = new THREE.Mesh(new THREE.CylinderGeometry(0.25, 1.95, 3.25, 64, 1, true), beamMaterial);
  root.add(beam);

  const droneBodyMaterial = new THREE.MeshPhysicalMaterial({
    color: 0x151c2e,
    metalness: 0.85,
    roughness: 0.34,
    clearcoat: 1,
    clearcoatRoughness: 0.22
  });
  const droneTrimMaterial = new THREE.MeshStandardMaterial({ color: 0x3a4a6e, metalness: 0.92, roughness: 0.24 });
  const droneGlassMaterial = new THREE.MeshPhysicalMaterial({ color: 0x040914, metalness: 0.3, roughness: 0.06, clearcoat: 1 });
  const droneLightMaterial = new THREE.MeshBasicMaterial({ color: 0x7cebff, transparent: true, toneMapped: false });
  const glowTexture = createGlowTexture();
  const { drone, rotors } = createDrone(
    droneBodyMaterial,
    droneTrimMaterial,
    droneGlassMaterial,
    droneLightMaterial,
    glowTexture
  );
  drone.scale.setScalar(1.15);
  root.add(drone);

  const dust = createSparkField(130);
  const dustPoints = new THREE.Points(dust.geometry, dust.material);
  root.add(dustPoints);

  const sparks = createSparkField(34);
  const sparkPoints = new THREE.Points(sparks.geometry, sparks.material);
  root.add(sparkPoints);

  const bounds = new THREE.Box3().setFromObject(construction.root);
  const minY = bounds.min.y;
  const height = Math.max(bounds.max.y - minY, 1);
  const pointerTarget = new THREE.Vector2();
  const pointer = new THREE.Vector2();
  const dronePoint = new THREE.Vector3();
  const ENTRY = new THREE.Vector3(5.5, 3.5, -0.8);
  const HOVER = new THREE.Vector3(0, 3.05, 0);
  const EXIT = new THREE.Vector3(6.2, 3.8, -1.2);

  let elapsed = 0;
  let lastFrame = 0;
  let frameId = 0;
  let running = false;

  const update = (time: number, delta: number) => {
    elapsed = time;
    pointer.lerp(pointerTarget, 1 - Math.exp(-delta * 4));
    root.rotation.y = pointer.x * 0.1;
    root.rotation.x = pointer.y * -0.04;
    hologramMaterials.forEach((hologram) => {
      hologram.uniforms.uTime.value = time;
    });

    const arrive = span(time, INTRO.droneIn);
    const leave = easeInCubic(span(time, INTRO.droneOut));
    const inside = time < INTRO.droneOut[1];
    drone.visible = inside;
    if (inside) {
      dronePoint.copy(ENTRY).lerp(HOVER, easeOutBack(smooth(arrive) * 0.35 + arrive * 0.65));
      dronePoint.lerp(EXIT, leave);
      dronePoint.y += Math.sin(time * 1.7) * 0.035;
      drone.position.copy(dronePoint);
      drone.rotation.set(0.12 * (1 - arrive) + 0.22 * leave, Math.sin(time * 0.3) * 0.05, 0.28 * (1 - arrive) - 0.35 * leave);
      rotors.forEach((rotor, index) => {
        rotor.rotation.y = time * (index % 2 === 0 ? 42 : -42);
      });
    }

    const open = smooth(span(time, INTRO.beamOpen));
    const close = smooth(span(time, INTRO.beamClose));
    const projecting = open > 0 && close < 1;
    beam.visible = projecting;
    beam.scale.y = open * (1 - close);
    beam.position.set(HOVER.x, 1.63 * open * (1 - close), HOVER.z);
    beamMaterial.opacity = 0.1 + open * (1 - close) * 0.2;
    floorRing.scale.setScalar(0.9 + open * 0.18 + Math.sin(time * 2.2) * 0.025);
    innerRing.scale.setScalar(0.96 + open * 0.08 + Math.sin(time * 2.2 + 1.2) * 0.018);
    ringMaterial.opacity = projecting ? 0.5 + Math.sin(time * 6) * 0.2 : 0.35;

    const build = smooth(span(time, INTRO.workerBuild));
    construction.root.visible = time > INTRO.workerBuild[0] - 0.2;
    const reveal = time < INTRO.workerBuild[0] ? -10000 : minY + build * height * 1.04;
    const opacity = time < INTRO.workerBuild[0] ? 0 : 0.74 + build * 0.26;
    hologramMaterials.forEach((hologram) => {
      hologram.uniforms.uReveal.value = reveal;
      hologram.uniforms.uOpacity.value = opacity;
    });
    construction.worker.rotation.y = Math.sin(time * 0.9) * 0.025;
    construction.worker.position.y = Math.sin(time * 1.8) * 0.008;

    const hammerCycle = Math.max(0, time - 4.35) * 1.65;
    const hammerPhase = hammerCycle % 1;
    const impact = hammerPhase < 0.26 ? smooth(hammerPhase / 0.26) : smooth(1 - (hammerPhase - 0.26) / 0.74);
    construction.hammerArm.rotation.z = -0.38 - impact * 0.62;
    sparks.material.opacity = hammerPhase < 0.22 ? (1 - hammerPhase / 0.22) * 0.9 : 0;
    for (let index = 0; index < sparks.seeds.length; index += 1) {
      const seed = sparks.seeds[index];
      const life = (hammerPhase + seed * 0.17) % 1;
      const active = life < 0.28;
      const angle = seed * Math.PI * 6.28;
      const distance = life * 1.1;
      sparks.positions[index * 3] = 0.52 + Math.cos(angle) * distance;
      sparks.positions[index * 3 + 1] = 1.16 + life * 0.85 - life * life * 0.65;
      sparks.positions[index * 3 + 2] = 0.08 + Math.sin(angle) * distance * 0.35;
      if (!active) sparks.positions[index * 3 + 1] = -100;
    }
    sparks.geometry.attributes.position.needsUpdate = true;

    for (let index = 0; index < dust.seeds.length; index += 1) {
      const seed = dust.seeds[index];
      const travel = (seed + time * (0.08 + seed * 0.08)) % 1;
      const angle = seed * 19.7;
      const radius = 0.3 + travel * 1.45;
      dust.positions[index * 3] = Math.cos(angle) * radius;
      dust.positions[index * 3 + 1] = 0.1 + travel * 3.05;
      dust.positions[index * 3 + 2] = Math.sin(angle) * radius * 0.55;
    }
    dust.geometry.attributes.position.needsUpdate = true;
    dust.material.opacity = projecting ? 0.65 : 0.2;
  };

  const frame = (now: number) => {
    if (!running) return;
    const delta = Math.min((now - lastFrame) / 1000, 0.1);
    lastFrame = now;
    update(elapsed + delta, delta);
    renderer.render(scene, camera);
    frameId = requestAnimationFrame(frame);
  };

  update(0, 0);

  return {
    setPointer: (x, y) => pointerTarget.set(x, y),
    start: () => {
      if (running) return;
      running = true;
      lastFrame = performance.now();
      frameId = requestAnimationFrame(frame);
    },
    stop: () => {
      running = false;
      cancelAnimationFrame(frameId);
    },
    resize: (width, height) => {
      if (!width || !height) return;
      renderer.setSize(width, height, false);
      camera.aspect = width / height;
      camera.updateProjectionMatrix();
    },
    dispose: () => {
      running = false;
      cancelAnimationFrame(frameId);
      const geometries = new Set<THREE.BufferGeometry>();
      const materials = new Set<THREE.Material>();
      scene.traverse((object) => {
        const mesh = object as THREE.Mesh;
        if (mesh.geometry) geometries.add(mesh.geometry);
        const objectMaterial = mesh.material;
        if (Array.isArray(objectMaterial)) objectMaterial.forEach((item) => materials.add(item));
        else if (objectMaterial) materials.add(objectMaterial);
      });
      geometries.forEach((geometry) => geometry.dispose());
      materials.forEach((item) => item.dispose());
      glowTexture.dispose();
      environment.dispose();
      renderer.dispose();
      renderer.forceContextLoss();
    }
  };
};
