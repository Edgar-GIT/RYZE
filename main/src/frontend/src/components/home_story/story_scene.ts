import * as THREE from "three";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";
import { RoundedBoxGeometry } from "three/addons/geometries/RoundedBoxGeometry.js";
import { mergeGeometries } from "three/addons/utils/BufferGeometryUtils.js";

import { GLOBE_CITIES, GLOBE_HOME, GLOBE_ROUTES } from "@/components/brand_globe/globe_data";

/*
 * The Home story scene. One persistent WebGL scene whose state is a function of
 * scroll progress (0..1): the Earth with RYZE routes leaving Lisbon, a phone
 * that presents the training and nutrition screens, and a laptop that opens on
 * the programme page. Loaded with a dynamic import on desktop only.
 *
 * Everything is procedural (no model files). The continents come from a small
 * Natural Earth land mask; the device screens are illustrative RYZE screens.
 */

export interface StorySceneAssets {
  logo: string;
  landMask: string;
  screens: { training: string; nutrition: string; synced: string; platform: string };
  callouts: { train: readonly string[]; fuel: readonly string[]; sync: string };
}

export interface StoryScene {
  setProgress: (progress: number) => void;
  setPointer: (x: number, y: number) => void;
  start: () => void;
  stop: () => void;
  resize: (width: number, height: number) => void;
  dispose: () => void;
}

const COLORS = {
  navy: 0x02041f,
  royal: 0x072a77,
  ring: 0x0481dc,
  cyan: 0x00cbff,
  cyanBright: 0x50e2fe,
  ice: 0x7cebff,
  titanium: 0x3a4a6a,
  shell: 0x2e3852
} as const;

// --- Scroll keyframes ----------------------------------------------------------

interface SceneState {
  globeX: number; // fractions of the visible half width
  globeY: number;
  globeZ: number;
  globeScale: number;
  globeSpin: number;
  stars: number;
  phoneX: number;
  phoneY: number;
  phoneZ: number;
  phoneRotX: number;
  phoneRotY: number;
  phoneRotZ: number;
  phoneScale: number;
  laptopX: number;
  laptopY: number;
  laptopZ: number;
  laptopRotY: number;
  laptopScale: number;
  lid: number;
  cameraZ: number;
}

const TAU = Math.PI * 2;
const LID_CLOSED = Math.PI / 2;
const LID_OPEN = -0.3;

const base: SceneState = {
  globeX: 0.37,
  globeY: -0.02,
  globeZ: 0,
  globeScale: 0.88,
  globeSpin: 0,
  stars: 1,
  phoneX: 0.62,
  phoneY: -6.5,
  phoneZ: 1,
  phoneRotX: 0.4,
  phoneRotY: -1.8,
  phoneRotZ: 0.35,
  phoneScale: 1.2,
  laptopX: 0.3,
  laptopY: -7,
  laptopZ: 0,
  laptopRotY: -0.26,
  laptopScale: 0.92,
  lid: LID_CLOSED,
  cameraZ: 10
};

// Right of centre, leaving room on its left for the cards lifted out of its screen.
const phoneHero = { phoneX: 0.56, phoneY: -0.1, phoneZ: 0.5, phoneRotX: 0.05, phoneRotZ: 0.03, phoneScale: 1.5 };
// The Earth drifts far back to the upper right while the devices are on stage.
const globeAway = { globeX: 1.2, globeY: 2.3, globeZ: -7, globeScale: 0.62, globeSpin: 1.4, stars: 0.35 };
// Laptop and phone side by side, both turned slightly towards the copy; the
// Earth leaves the frame so the two screens carry the chapter.
const phoneBesideLaptop = {
  globeY: 6.5,
  phoneX: 0.71,
  phoneY: -0.38,
  phoneZ: 0.6,
  phoneRotX: 0.02,
  phoneRotZ: 0,
  phoneScale: 0.82,
  laptopX: 0.27
};

const KEYS: Array<{ at: number; state: SceneState }> = [
  { at: 0, state: base },
  { at: 0.12, state: { ...base, cameraZ: 9.6, globeSpin: 0.25 } },
  { at: 0.27, state: { ...base, ...globeAway, ...phoneHero, phoneRotY: -0.42 } },
  { at: 0.42, state: { ...base, ...globeAway, ...phoneHero, phoneRotY: -0.16 } },
  { at: 0.46, state: { ...base, ...globeAway, ...phoneHero, phoneRotY: -0.16 } },
  { at: 0.56, state: { ...base, ...globeAway, ...phoneHero, phoneRotY: TAU - 0.16 } },
  { at: 0.64, state: { ...base, ...globeAway, ...phoneHero, phoneRotY: TAU - 0.3 } },
  { at: 0.72, state: { ...base, ...globeAway, ...phoneBesideLaptop, phoneRotY: 2 * TAU - 0.42, laptopY: -0.9 } },
  { at: 0.79, state: { ...base, ...globeAway, ...phoneBesideLaptop, phoneRotY: 2 * TAU - 0.38, laptopY: -0.9, lid: LID_OPEN } },
  { at: 0.88, state: { ...base, ...globeAway, ...phoneBesideLaptop, phoneRotY: 2 * TAU - 0.34, laptopY: -0.9, laptopRotY: -0.16, lid: LID_OPEN } },
  { at: 0.95, state: { ...base, globeSpin: 0.5, phoneY: -7, phoneRotY: 2 * TAU, laptopY: -8, lid: LID_OPEN } },
  { at: 1, state: { ...base, globeSpin: 0.5, phoneY: -7, phoneRotY: 2 * TAU, laptopY: -8, lid: LID_OPEN } }
];

// The phone shows its back half-way through each spin; the screen is swapped
// there so the change is never visible.
// In the Evolve chapter the phone shows a large-type screen that stays legible
// at its smaller size there.
type PhoneScreen = "training" | "nutrition" | "synced";
const screenFor = (progress: number): PhoneScreen =>
  progress >= 0.51 && progress < 0.68 ? "nutrition" : progress >= 0.68 ? "synced" : "training";

const smooth = (t: number) => t * t * (3 - 2 * t);

// 0 → 1 between a and b, 1 → 0 between c and d (chapter visibility).
const windowed = (p: number, a: number, b: number, c: number, d: number) =>
  Math.min(smooth(THREE.MathUtils.clamp((p - a) / (b - a), 0, 1)), 1 - smooth(THREE.MathUtils.clamp((p - c) / (d - c), 0, 1)));

const STATE_KEYS = Object.keys(base) as Array<keyof SceneState>;

const sampleState = (progress: number, out: SceneState): SceneState => {
  const p = THREE.MathUtils.clamp(progress, 0, 1);
  let index = 0;
  while (index < KEYS.length - 2 && p > KEYS[index + 1].at) {
    index += 1;
  }
  const from = KEYS[index];
  const to = KEYS[index + 1];
  const t = smooth(THREE.MathUtils.clamp((p - from.at) / (to.at - from.at), 0, 1));
  for (const key of STATE_KEYS) {
    out[key] = from.state[key] + (to.state[key] - from.state[key]) * t;
  }
  return out;
};

// --- Helpers ---------------------------------------------------------------------

const loadImage = (src: string): Promise<HTMLImageElement> =>
  new Promise((resolve, reject) => {
    const image = new Image();
    image.decoding = "async";
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error(`Unable to load ${src}`));
    image.src = src;
  });

const colorTexture = (source: HTMLImageElement | HTMLCanvasElement, anisotropy: number): THREE.Texture => {
  const texture = source instanceof HTMLCanvasElement ? new THREE.CanvasTexture(source) : new THREE.Texture(source);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.anisotropy = anisotropy;
  texture.needsUpdate = true;
  return texture;
};

// Screens and cards are read, not just seen: sample their textures one mip level
// sharper than the GPU would choose, which removes the soft look of small text.
const sharpen = <T extends THREE.MeshBasicMaterial | THREE.SpriteMaterial>(material: T): T => {
  material.onBeforeCompile = (shader) => {
    shader.fragmentShader = shader.fragmentShader.replace(
      "#include <map_fragment>",
      THREE.ShaderChunk.map_fragment.replace("texture2D( map, vMapUv )", "texture2D( map, vMapUv, -0.8 )")
    );
  };
  material.customProgramCacheKey = () => "sharpened-map";
  return material;
};

const roundedRectShape = (width: number, height: number, radius: number): THREE.Shape => {
  const x = -width / 2;
  const y = -height / 2;
  const shape = new THREE.Shape();
  shape.moveTo(x + radius, y);
  shape.lineTo(x + width - radius, y);
  shape.quadraticCurveTo(x + width, y, x + width, y + radius);
  shape.lineTo(x + width, y + height - radius);
  shape.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
  shape.lineTo(x + radius, y + height);
  shape.quadraticCurveTo(x, y + height, x, y + height - radius);
  shape.lineTo(x, y + radius);
  shape.quadraticCurveTo(x, y, x + radius, y);
  return shape;
};

// A flat rounded panel whose UVs span 0..1 over its bounds (for screens).
const panelGeometry = (width: number, height: number, radius: number, segments = 12): THREE.BufferGeometry => {
  const geometry = new THREE.ShapeGeometry(roundedRectShape(width, height, radius), segments);
  const position = geometry.attributes.position;
  const uv = geometry.attributes.uv;
  for (let index = 0; index < position.count; index += 1) {
    uv.setXY(index, position.getX(index) / width + 0.5, position.getY(index) / height + 0.5);
  }
  return geometry;
};

// A rounded slab with bevelled edges, centred on z = 0 (device bodies).
const slabGeometry = (width: number, height: number, radius: number, depth: number, bevel: number): THREE.BufferGeometry => {
  const geometry = new THREE.ExtrudeGeometry(roundedRectShape(width - bevel * 2, height - bevel * 2, Math.max(radius - bevel, 0.001)), {
    depth: depth - bevel * 2,
    bevelEnabled: true,
    bevelThickness: bevel,
    bevelSize: bevel,
    bevelSegments: 6,
    curveSegments: 18
  });
  geometry.translate(0, 0, -(depth - bevel * 2) / 2);
  return geometry;
};

const canvas2d = (width: number, height: number) => {
  const element = document.createElement("canvas");
  element.width = width;
  element.height = height;
  return { element, context: element.getContext("2d") as CanvasRenderingContext2D };
};

// Soft light around a device silhouette with the centre cut out, so the glow
// only shows outside the body whichever side faces the camera. Returns the
// plane size that makes the cut-out match the body exactly.
const edgeGlow = (bodyWidth: number, bodyHeight: number, bodyRadius: number, color: string) => {
  const inner = 300;
  const innerHeight = Math.round((inner * bodyHeight) / bodyWidth);
  const inset = 72;
  const radius = (bodyRadius / bodyWidth) * inner;
  const { element, context } = canvas2d(inner + inset * 2, innerHeight + inset * 2);
  context.shadowColor = color;
  context.shadowBlur = inset * 0.8;
  context.fillStyle = color;
  context.beginPath();
  context.roundRect(inset, inset, inner, innerHeight, radius);
  context.fill();
  context.shadowBlur = 0;
  context.globalCompositeOperation = "destination-out";
  context.beginPath();
  context.roundRect(inset + 2, inset + 2, inner - 4, innerHeight - 4, radius);
  context.fill();
  const texture = new THREE.CanvasTexture(element);
  texture.colorSpace = THREE.SRGBColorSpace;
  const scale = (inner + inset * 2) / inner;
  return { texture, width: bodyWidth * scale, height: (bodyHeight * (innerHeight + inset * 2)) / innerHeight };
};

const radialTexture = (inner: string, outer: string) => {
  const size = 256;
  const { element, context } = canvas2d(size, size);
  const gradient = context.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
  gradient.addColorStop(0, inner);
  gradient.addColorStop(1, outer);
  context.fillStyle = gradient;
  context.fillRect(0, 0, size, size);
  const texture = new THREE.CanvasTexture(element);
  texture.colorSpace = THREE.SRGBColorSpace;
  return texture;
};

// Diagonal reflection on the screen glass.
const sheenTexture = () => {
  const { element, context } = canvas2d(256, 512);
  const gradient = context.createLinearGradient(0, 0, 256, 512);
  gradient.addColorStop(0, "rgba(255,255,255,0.0)");
  gradient.addColorStop(0.28, "rgba(255,255,255,0.0)");
  gradient.addColorStop(0.36, "rgba(255,255,255,0.55)");
  gradient.addColorStop(0.5, "rgba(255,255,255,0.08)");
  gradient.addColorStop(1, "rgba(255,255,255,0.0)");
  context.fillStyle = gradient;
  context.fillRect(0, 0, 256, 512);
  const texture = new THREE.CanvasTexture(element);
  texture.colorSpace = THREE.SRGBColorSpace;
  return texture;
};

// The home-city marker: the official icon on a navy badge with a ring-blue edge.
const badgeTexture = (logo: HTMLImageElement, anisotropy: number) => {
  const size = 256;
  const { element, context } = canvas2d(size, size);
  const center = size / 2;
  const glow = context.createRadialGradient(center, center, size * 0.3, center, center, size * 0.5);
  glow.addColorStop(0, "rgba(0,203,255,0.55)");
  glow.addColorStop(1, "rgba(0,203,255,0)");
  context.fillStyle = glow;
  context.fillRect(0, 0, size, size);
  context.beginPath();
  context.arc(center, center, size * 0.33, 0, TAU);
  context.fillStyle = "#02041f";
  context.fill();
  context.lineWidth = 6;
  context.strokeStyle = "#0481dc";
  context.stroke();
  const logoHeight = size * 0.4;
  const logoWidth = logoHeight * (logo.width / logo.height);
  context.drawImage(logo, center - logoWidth / 2, center - logoHeight / 2, logoWidth, logoHeight);
  return colorTexture(element, anisotropy);
};

// Matches SphereGeometry's UV layout and an equirectangular texture.
const latLonToVector = (lat: number, lon: number, radius: number, target = new THREE.Vector3()) => {
  const phi = ((lon + 180) / 360) * TAU;
  const theta = ((90 - lat) / 180) * Math.PI;
  return target.set(-Math.cos(phi) * Math.sin(theta), Math.cos(theta), Math.sin(phi) * Math.sin(theta)).multiplyScalar(radius);
};

// --- Shaders ---------------------------------------------------------------------

const earthVertex = /* glsl */ `
  varying vec2 vUv;
  varying vec3 vNormalW;
  varying vec3 vPositionW;
  void main() {
    vUv = uv;
    vec4 world = modelMatrix * vec4(position, 1.0);
    vPositionW = world.xyz;
    vNormalW = normalize(mat3(modelMatrix) * normal);
    gl_Position = projectionMatrix * viewMatrix * world;
  }
`;

const earthFragment = /* glsl */ `
  uniform sampler2D uMask;
  uniform vec3 uSun;
  uniform vec3 uOceanDeep;
  uniform vec3 uOceanShallow;
  uniform vec3 uLandLow;
  uniform vec3 uLandHigh;
  uniform vec3 uCoast;
  uniform vec3 uRim;
  varying vec2 vUv;
  varying vec3 vNormalW;
  varying vec3 vPositionW;
  void main() {
    float raw = texture2D(uMask, vUv).r;
    float land = smoothstep(0.38, 0.62, raw);
    // A mip-biased sample is a cheap blur: shallow water hugging the coasts.
    float shelf = texture2D(uMask, vUv, 3.5).r;
    float coast = clamp(texture2D(uMask, vUv, 1.5).r - land, 0.0, 1.0);

    vec3 normal = normalize(vNormalW);
    vec3 view = normalize(cameraPosition - vPositionW);
    float light = dot(normal, uSun);
    float day = smoothstep(-0.3, 0.5, light);

    vec3 ocean = mix(uOceanDeep, uOceanShallow, clamp(shelf * 1.4, 0.0, 1.0));
    vec3 ground = mix(uLandLow, uLandHigh, smoothstep(-0.1, 0.9, light));
    vec3 color = mix(ocean, ground, land) * mix(0.14, 1.0, day);

    color += coast * uCoast * mix(0.2, 0.75, day);

    float fresnel = pow(1.0 - max(dot(normal, view), 0.0), 3.2);
    color += fresnel * uRim * mix(0.3, 0.8, day);
    gl_FragColor = vec4(color, 1.0);
    #include <colorspace_fragment>
  }
`;

const haloVertex = /* glsl */ `
  varying vec3 vNormal;
  varying vec3 vView;
  void main() {
    vec4 viewPosition = modelViewMatrix * vec4(position, 1.0);
    vNormal = normalize(normalMatrix * normal);
    vView = normalize(-viewPosition.xyz);
    gl_Position = projectionMatrix * viewPosition;
  }
`;

// Atmosphere drawn from its far side: brightest at the planet's limb (where the
// shell's normals turn away from the viewer), fading to nothing at its edge.
const haloFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform float uStrength;
  varying vec3 vNormal;
  varying vec3 vView;
  void main() {
    float facing = dot(normalize(vNormal), normalize(vView));
    float glow = pow(clamp(-facing / 0.5, 0.0, 1.0), 2.4) * uStrength;
    gl_FragColor = vec4(uColor * glow, glow);
    #include <colorspace_fragment>
  }
`;

const routeVertex = /* glsl */ `
  attribute float aPhase;
  varying float vAlong;
  varying float vPhase;
  void main() {
    vAlong = uv.x;
    vPhase = aPhase;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

// A light pulse travels each route from origin to destination, then rests.
const routeFragment = /* glsl */ `
  uniform float uTime;
  uniform vec3 uBase;
  uniform vec3 uHead;
  varying float vAlong;
  varying float vPhase;
  void main() {
    float head = fract(uTime * 0.12 + vPhase) * 1.5;
    float tail = 0.34;
    float pulse = clamp((vAlong - (head - tail)) / tail, 0.0, 1.0) * step(vAlong, head);
    pulse *= pulse;
    float alpha = 0.2 + pulse;
    gl_FragColor = vec4(mix(uBase, uHead, pulse) * alpha, alpha);
    #include <colorspace_fragment>
  }
`;

const cityVertex = /* glsl */ `
  attribute float aSize;
  uniform float uPixelRatio;
  uniform float uScale;
  void main() {
    vec4 viewPosition = modelViewMatrix * vec4(position, 1.0);
    gl_PointSize = aSize * uPixelRatio * uScale / -viewPosition.z;
    gl_Position = projectionMatrix * viewPosition;
  }
`;

const cityFragment = /* glsl */ `
  uniform vec3 uColor;
  void main() {
    float distanceToCenter = length(gl_PointCoord - 0.5);
    float core = smoothstep(0.5, 0.0, distanceToCenter);
    float alpha = pow(core, 2.4) + smoothstep(0.16, 0.08, distanceToCenter);
    gl_FragColor = vec4(uColor * alpha, alpha);
    #include <colorspace_fragment>
  }
`;

// --- Builders ---------------------------------------------------------------------

const starField = (count: number) => {
  const positions = new Float32Array(count * 3);
  const colors = new Float32Array(count * 3);
  const palette = [new THREE.Color(0xffffff), new THREE.Color(COLORS.cyanBright), new THREE.Color(0x9fc7ff)];
  for (let index = 0; index < count; index += 1) {
    positions.set(
      [THREE.MathUtils.randFloatSpread(56), THREE.MathUtils.randFloatSpread(32), THREE.MathUtils.randFloat(-36, -6)],
      index * 3
    );
    const color = palette[index % palette.length];
    colors.set([color.r, color.g, color.b], index * 3);
  }
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute("position", new THREE.BufferAttribute(positions, 3));
  geometry.setAttribute("color", new THREE.BufferAttribute(colors, 3));
  return geometry;
};

// All routes merged into one geometry (one draw call); each carries its own
// pulse phase.
const routesGeometry = (radius: number) => {
  const byId = new Map(GLOBE_CITIES.map((city) => [city.id, city]));
  const parts: THREE.BufferGeometry[] = [];
  const from = new THREE.Vector3();
  const to = new THREE.Vector3();
  GLOBE_ROUTES.forEach(([fromId, toId], index) => {
    const a = byId.get(fromId);
    const b = byId.get(toId);
    if (!a || !b) {
      return;
    }
    latLonToVector(a.lat, a.lon, radius, from);
    latLonToVector(b.lat, b.lon, radius, to);
    const angle = from.angleTo(to);
    const lift = 1 + 0.05 + 0.28 * (angle / Math.PI);
    const control = (t: number) => {
      const direction = from.clone().normalize().lerp(to.clone().normalize(), t).normalize();
      return direction.multiplyScalar(radius * lift);
    };
    const curve = new THREE.CubicBezierCurve3(from.clone().multiplyScalar(1.002), control(0.3), control(0.7), to.clone().multiplyScalar(1.002));
    const tube = new THREE.TubeGeometry(curve, 72, 0.0065, 6, false);
    tube.deleteAttribute("normal");
    const phases = new Float32Array(tube.attributes.position.count).fill((index * 0.37) % 1);
    tube.setAttribute("aPhase", new THREE.BufferAttribute(phases, 1));
    parts.push(tube);
  });
  const merged = mergeGeometries(parts);
  for (const part of parts) {
    part.dispose();
  }
  if (!merged) {
    throw new Error("Unable to build the globe routes");
  }
  return merged;
};

const citiesGeometry = (radius: number) => {
  const positions: number[] = [];
  const sizes: number[] = [];
  const point = new THREE.Vector3();
  for (const city of GLOBE_CITIES) {
    latLonToVector(city.lat, city.lon, radius * 1.004, point);
    positions.push(point.x, point.y, point.z);
    sizes.push(city.id === GLOBE_HOME ? 15 : 9);
  }
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute("position", new THREE.Float32BufferAttribute(positions, 3));
  geometry.setAttribute("aSize", new THREE.Float32BufferAttribute(sizes, 1));
  return geometry;
};

// Keyboard rows: [legend, width in key units]; every row spans the same width.
type KeyDef = [string, number];
const one = (legends: string): KeyDef[] => [...legends].map((legend) => [legend, 1]);
const KEY_ROWS: Array<{ depth: number; keys: KeyDef[] }> = [
  { depth: 0.6, keys: [["esc", 1], ...Array.from({ length: 12 }, (_, i): KeyDef => [`F${i + 1}`, 1]), ["touch-id", 1]] },
  { depth: 1, keys: [...one("`1234567890-="), ["delete", 1.5]] },
  { depth: 1, keys: [["tab", 1.5], ...one("QWERTYUIOP[]\\")] },
  { depth: 1, keys: [["caps lock", 1.8], ...one("ASDFGHJKL;'"), ["return", 1.7]] },
  { depth: 1, keys: [["shift", 2.3], ...one("ZXCVBNM,./"), ["shift", 2.2]] },
  {
    depth: 1,
    keys: [["fn", 1], ["control", 1], ["option", 1], ["command", 1.25], ["", 5], ["command", 1.25], ["option", 1], ["left", 1], ["updown", 1], ["right", 1]]
  }
];

interface KeyCap {
  x: number;
  z: number;
  width: number;
  depth: number;
  legend: string;
}

// Key caps in deck coordinates (x across, z towards the user), centred on the
// keyboard well. Shared by the key geometry and the legend texture.
const KEYBOARD = { width: 2.92, depth: 1.12, centerZ: -0.38, usable: 2.82, gap: 0.026, unitDepth: 0.152 };
const layoutKeys = (): KeyCap[] => {
  const caps: KeyCap[] = [];
  let z = KEYBOARD.centerZ - KEYBOARD.depth / 2 + 0.05;
  for (const row of KEY_ROWS) {
    const units = row.keys.reduce((sum, [, width]) => sum + width, 0);
    const unit = (KEYBOARD.usable - KEYBOARD.gap * (row.keys.length - 1)) / units;
    const depth = KEYBOARD.unitDepth * row.depth;
    let x = -KEYBOARD.usable / 2;
    for (const [legend, width] of row.keys) {
      const keyWidth = unit * width;
      caps.push({ x: x + keyWidth / 2, z: z + depth / 2, width: keyWidth, depth, legend });
      x += keyWidth + KEYBOARD.gap;
    }
    z += depth + KEYBOARD.gap;
  }
  return caps;
};

// Backlit legends drawn at each key's position on a transparent texture that
// lies on top of the key caps.
const keyLegendTexture = (caps: KeyCap[], anisotropy: number) => {
  const scale = 2048 / KEYBOARD.width;
  const { element, context } = canvas2d(2048, Math.round(KEYBOARD.depth * scale));
  const top = KEYBOARD.centerZ - KEYBOARD.depth / 2;
  context.fillStyle = "rgba(226, 234, 248, 0.94)";
  context.strokeStyle = "rgba(226, 234, 248, 0.94)";
  context.shadowColor = "rgba(124, 235, 255, 0.55)";
  context.shadowBlur = 6;
  context.textAlign = "center";
  context.textBaseline = "middle";
  const font = (size: number, weight = 600) => `${weight} ${Math.round(size)}px "RYZE Archivo", Arial, sans-serif`;
  const triangle = (cx: number, cy: number, size: number, rotation: number) => {
    context.save();
    context.translate(cx, cy);
    context.rotate(rotation);
    context.beginPath();
    context.moveTo(0, -size);
    context.lineTo(size * 0.9, size * 0.6);
    context.lineTo(-size * 0.9, size * 0.6);
    context.closePath();
    context.fill();
    context.restore();
  };
  for (const cap of caps) {
    const cx = (cap.x + KEYBOARD.width / 2) * scale;
    const cy = (cap.z - top) * scale;
    const w = cap.width * scale;
    const h = cap.depth * scale;
    const unit = KEYBOARD.unitDepth * scale;
    if (cap.legend === "") {
      continue;
    } else if (cap.legend === "touch-id") {
      context.lineWidth = 3;
      context.beginPath();
      context.arc(cx, cy, h * 0.28, 0, TAU);
      context.stroke();
    } else if (cap.legend === "left" || cap.legend === "right") {
      triangle(cx, cy, unit * 0.1, cap.legend === "left" ? -Math.PI / 2 : Math.PI / 2);
    } else if (cap.legend === "updown") {
      triangle(cx, cy - h * 0.2, unit * 0.08, 0);
      triangle(cx, cy + h * 0.2, unit * 0.08, Math.PI);
    } else if (cap.legend.length === 1) {
      context.font = font(unit * 0.42, 650);
      context.fillText(cap.legend, cx, cy);
    } else if (cap.legend.startsWith("F") || cap.legend === "esc") {
      context.font = font(unit * 0.2, 560);
      context.fillText(cap.legend, cx, cy);
    } else {
      // Modifier words sit at the lower edge, aligned to the key's outer side.
      const leftSide = cap.x < 0;
      context.font = font(unit * 0.19, 560);
      context.textAlign = leftSide ? "left" : "right";
      context.fillText(cap.legend, leftSide ? cx - w / 2 + unit * 0.12 : cx + w / 2 - unit * 0.12, cy + h * 0.28);
      context.textAlign = "center";
    }
  }
  return colorTexture(element, anisotropy);
};

export const createStoryScene = async (canvas: HTMLCanvasElement, assets: StorySceneAssets): Promise<StoryScene> => {
  // The key legends are drawn with the site's typeface once it is available.
  await document.fonts.load('600 48px "RYZE Archivo"').catch(() => undefined);
  // Images first: a failed download must not leave a WebGL context behind.
  const [logoImage, maskImage, trainingImage, nutritionImage, syncedImage, platformImage, trainCallouts, fuelCallouts, syncImage] =
    await Promise.all([
      loadImage(assets.logo),
      loadImage(assets.landMask),
      loadImage(assets.screens.training),
      loadImage(assets.screens.nutrition),
      loadImage(assets.screens.synced),
      loadImage(assets.screens.platform),
      Promise.all(assets.callouts.train.map(loadImage)),
      Promise.all(assets.callouts.fuel.map(loadImage)),
      loadImage(assets.callouts.sync)
    ]);

  // Default GPU preference (no forced discrete GPU); high-DPI screens skip MSAA.
  const pixelRatio = Math.min(window.devicePixelRatio, 1.5);
  const renderer = new THREE.WebGLRenderer({ canvas, antialias: window.devicePixelRatio < 1.5, alpha: true });
  try {
    renderer.setPixelRatio(pixelRatio);
    renderer.setClearColor(0x000000, 0);
    renderer.toneMapping = THREE.ACESFilmicToneMapping;
    renderer.toneMappingExposure = 1.05;
    const anisotropy = Math.min(8, renderer.capabilities.getMaxAnisotropy());

    const scene = new THREE.Scene();
    const pmrem = new THREE.PMREMGenerator(renderer);
    const room = new RoomEnvironment();
    const environment = pmrem.fromScene(room, 0.04).texture;
    scene.environment = environment;
    scene.environmentIntensity = 0.8;
    room.dispose();
    pmrem.dispose();

    const camera = new THREE.PerspectiveCamera(30, 1, 0.1, 100);
    camera.position.set(0, 0, base.cameraZ);

    // Key from the upper left (the copy side), cyan rim from behind on the right.
    scene.add(new THREE.HemisphereLight(0xcfe3ff, COLORS.navy, 0.55));
    const key = new THREE.DirectionalLight(0xffffff, 2.2);
    key.position.set(-4, 5, 6);
    const rim = new THREE.DirectionalLight(COLORS.cyan, 3.4);
    rim.position.set(6, 2, -4);
    const fill = new THREE.PointLight(COLORS.ring, 30, 26);
    fill.position.set(3, -3, 5);
    scene.add(key, rim, fill);

    const disposables: Array<{ dispose: () => void }> = [environment];
    const track = <T extends { dispose: () => void }>(item: T): T => {
      disposables.push(item);
      return item;
    };
    const logoAspect = logoImage.width / logoImage.height;
    const logoTexture = track(colorTexture(logoImage, anisotropy));

    // Stars --------------------------------------------------------------------------
    const dot = track(radialTexture("rgba(255,255,255,1)", "rgba(255,255,255,0)"));
    const starMaterial = track(
      new THREE.PointsMaterial({ size: 0.12, map: dot, vertexColors: true, transparent: true, depthWrite: false, opacity: 1 })
    );
    const stars = new THREE.Points(track(starField(1100)), starMaterial);
    scene.add(stars);

    // Earth --------------------------------------------------------------------------
    const globeRadius = 1.55;
    const globe = new THREE.Group();
    const earth = new THREE.Group(); // spins around its own axis
    const maskTexture = track(new THREE.Texture(maskImage));
    maskTexture.wrapS = THREE.RepeatWrapping;
    maskTexture.anisotropy = anisotropy;
    maskTexture.needsUpdate = true;
    const sun = new THREE.Vector3(-0.62, 0.42, 0.66).normalize();
    const surface = new THREE.Mesh(
      track(new THREE.SphereGeometry(globeRadius, 128, 96)),
      track(
        new THREE.ShaderMaterial({
          uniforms: {
            uMask: { value: maskTexture },
            uSun: { value: sun },
            uOceanDeep: { value: new THREE.Color(0x03113f) },
            uOceanShallow: { value: new THREE.Color(0x0a3c9a) },
            uLandLow: { value: new THREE.Color(0x103f9e) },
            uLandHigh: { value: new THREE.Color(0x4aa6f2) },
            uCoast: { value: new THREE.Color(COLORS.ice) },
            uRim: { value: new THREE.Color(0x3fb8ff) }
          },
          vertexShader: earthVertex,
          fragmentShader: earthFragment
        })
      )
    );
    const atmosphere = new THREE.Mesh(
      track(new THREE.SphereGeometry(globeRadius * 1.16, 96, 64)),
      track(
        new THREE.ShaderMaterial({
          uniforms: { uColor: { value: new THREE.Color(0x1aa8ff) }, uStrength: { value: 1.0 } },
          vertexShader: haloVertex,
          fragmentShader: haloFragment,
          side: THREE.BackSide,
          transparent: true,
          depthWrite: false,
          blending: THREE.AdditiveBlending
        })
      )
    );

    const routeMaterial = track(
      new THREE.ShaderMaterial({
        uniforms: {
          uTime: { value: 0 },
          uBase: { value: new THREE.Color(COLORS.ring) },
          uHead: { value: new THREE.Color(COLORS.ice) }
        },
        vertexShader: routeVertex,
        fragmentShader: routeFragment,
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending
      })
    );
    const routes = new THREE.Mesh(track(routesGeometry(globeRadius)), routeMaterial);

    const cityMaterial = track(
      new THREE.ShaderMaterial({
        uniforms: {
          uColor: { value: new THREE.Color(COLORS.cyanBright) },
          uPixelRatio: { value: pixelRatio },
          uScale: { value: 1 }
        },
        vertexShader: cityVertex,
        fragmentShader: cityFragment,
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending
      })
    );
    const cities = new THREE.Points(track(citiesGeometry(globeRadius)), cityMaterial);

    // Home marker: a short stem from Lisbon to the badge, and a pulse ring.
    const home = GLOBE_CITIES.find((city) => city.id === GLOBE_HOME) ?? GLOBE_CITIES[0];
    const homeNormal = latLonToVector(home.lat, home.lon, 1);
    const marker = new THREE.Group();
    marker.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), homeNormal);
    marker.position.copy(homeNormal).multiplyScalar(globeRadius);
    const stemLength = 0.36;
    const stem = new THREE.Mesh(
      track(new THREE.CylinderGeometry(0.005, 0.005, stemLength, 8)),
      track(new THREE.MeshBasicMaterial({ color: COLORS.cyanBright, transparent: true, opacity: 0.85, toneMapped: false }))
    );
    stem.position.y = stemLength / 2;
    const pulseMaterial = track(
      new THREE.MeshBasicMaterial({
        color: COLORS.cyanBright,
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
        side: THREE.DoubleSide,
        toneMapped: false
      })
    );
    const pulse = new THREE.Mesh(track(new THREE.RingGeometry(0.035, 0.05, 48)), pulseMaterial);
    pulse.rotation.x = -Math.PI / 2;
    pulse.position.y = 0.004;
    const badge = new THREE.Sprite(
      track(new THREE.SpriteMaterial({ map: track(badgeTexture(logoImage, anisotropy)), transparent: true, depthWrite: false, toneMapped: false }))
    );
    badge.scale.setScalar(0.42);
    badge.position.y = stemLength + 0.1;
    marker.add(stem, pulse, badge);

    earth.add(surface, routes, cities, marker);
    globe.add(atmosphere, earth);
    // Start over the Atlantic so Lisbon and the routes face the viewer.
    const view = latLonToVector(24, -16, 1);
    const baseYaw = Math.atan2(-view.x, view.z);
    const baseTilt = THREE.MathUtils.degToRad(24);
    scene.add(globe);

    // Phone ----------------------------------------------------------------------------
    const phone = new THREE.Group();
    const phonePivot = new THREE.Group();
    const screens = {
      training: track(colorTexture(trainingImage, anisotropy)),
      nutrition: track(colorTexture(nutritionImage, anisotropy)),
      synced: track(colorTexture(syncedImage, anisotropy))
    };
    const screenHeight = 1.9;
    const screenWidth = screenHeight * (trainingImage.width / trainingImage.height);
    const bezel = 0.035;
    const phoneWidth = screenWidth + bezel * 2 + 0.03;
    const phoneHeight = screenHeight + bezel * 2 + 0.03;
    const phoneDepth = 0.085;
    const titanium = track(
      new THREE.MeshPhysicalMaterial({ color: COLORS.titanium, metalness: 1, roughness: 0.24, clearcoat: 0.35, clearcoatRoughness: 0.2 })
    );
    const phoneBody = new THREE.Mesh(track(slabGeometry(phoneWidth, phoneHeight, 0.16, phoneDepth, 0.022)), titanium);
    const frontGlass = new THREE.Mesh(
      track(panelGeometry(phoneWidth - 0.028, phoneHeight - 0.028, 0.145)),
      track(new THREE.MeshPhysicalMaterial({ color: 0x010206, roughness: 0.06, metalness: 0, clearcoat: 1, clearcoatRoughness: 0.03 }))
    );
    frontGlass.position.z = phoneDepth / 2 + 0.0006;
    const screenMaterial = track(sharpen(new THREE.MeshBasicMaterial({ map: screens.training, toneMapped: false })));
    const screen = new THREE.Mesh(track(panelGeometry(screenWidth, screenHeight, 0.12, 16)), screenMaterial);
    screen.position.z = phoneDepth / 2 + 0.0012;
    const island = new THREE.Mesh(
      track(panelGeometry(0.28, 0.082, 0.041, 10)),
      track(new THREE.MeshBasicMaterial({ color: 0x000000 }))
    );
    island.position.set(0, screenHeight / 2 - 0.066, phoneDepth / 2 + 0.0018);
    const sheen = new THREE.Mesh(
      track(panelGeometry(screenWidth, screenHeight, 0.12, 16)),
      track(
        new THREE.MeshBasicMaterial({
          map: track(sheenTexture()),
          transparent: true,
          opacity: 0.16,
          depthWrite: false,
          blending: THREE.AdditiveBlending,
          toneMapped: false
        })
      )
    );
    sheen.position.z = phoneDepth / 2 + 0.0024;

    const backGlass = new THREE.Mesh(
      track(panelGeometry(phoneWidth - 0.028, phoneHeight - 0.028, 0.145)),
      track(
        new THREE.MeshPhysicalMaterial({ color: 0x0d1f52, roughness: 0.48, metalness: 0.15, clearcoat: 1, clearcoatRoughness: 0.45 })
      )
    );
    backGlass.position.z = -phoneDepth / 2 - 0.0006;
    backGlass.rotation.y = Math.PI;
    const backLogo = new THREE.Mesh(
      track(new THREE.PlaneGeometry(0.36 * logoAspect, 0.36)),
      track(new THREE.MeshBasicMaterial({ map: logoTexture, transparent: true, toneMapped: false, depthWrite: false }))
    );
    backLogo.position.set(0, -0.05, -phoneDepth / 2 - 0.0016);
    backLogo.rotation.y = Math.PI;

    // Camera module, top left when the back faces the viewer.
    const cameraModule = new THREE.Group();
    cameraModule.position.set(phoneWidth / 2 - 0.25, phoneHeight / 2 - 0.25, -phoneDepth / 2);
    const plateau = new THREE.Mesh(
      track(slabGeometry(0.4, 0.4, 0.1, 0.024, 0.008)),
      track(new THREE.MeshPhysicalMaterial({ color: 0x0b1b48, roughness: 0.2, metalness: 0.3, clearcoat: 1, clearcoatRoughness: 0.08 }))
    );
    plateau.position.z = -0.006;
    const lensGlass = track(new THREE.MeshPhysicalMaterial({ color: 0x03050c, roughness: 0.04, metalness: 0.4, clearcoat: 1, clearcoatRoughness: 0.02 }));
    const lensGeometry = track(new THREE.CylinderGeometry(0.068, 0.068, 0.03, 40));
    const lensRingGeometry = track(new THREE.TorusGeometry(0.068, 0.009, 12, 48));
    for (const [x, y] of [
      [0.085, 0.085],
      [0.085, -0.085],
      [-0.085, 0]
    ] as const) {
      const lens = new THREE.Mesh(lensGeometry, lensGlass);
      lens.rotation.x = Math.PI / 2;
      lens.position.set(x, y, -0.03);
      const ringMesh = new THREE.Mesh(lensRingGeometry, titanium);
      ringMesh.position.set(x, y, -0.045);
      cameraModule.add(lens, ringMesh);
    }
    const flash = new THREE.Mesh(
      track(new THREE.CircleGeometry(0.022, 24)),
      track(new THREE.MeshStandardMaterial({ color: 0xf2efe6, roughness: 0.3, emissive: 0x332f22 }))
    );
    flash.position.set(-0.085, 0.13, -0.0185);
    flash.rotation.y = Math.PI;
    cameraModule.add(plateau, flash);

    const buttonGeometry = track(new RoundedBoxGeometry(0.02, 1, 0.034, 2, 0.008));
    const addButton = (side: -1 | 1, y: number, length: number) => {
      const button = new THREE.Mesh(buttonGeometry, titanium);
      button.scale.y = length;
      button.position.set(side * (phoneWidth / 2 + 0.004), y, 0);
      phonePivot.add(button);
    };
    addButton(-1, 0.62, 0.08);
    addButton(-1, 0.42, 0.16);
    addButton(-1, 0.22, 0.16);
    addButton(1, 0.36, 0.26);

    const glow = edgeGlow(phoneWidth, phoneHeight, 0.16, "rgba(4,150,255,1)");
    const phoneGlow = new THREE.Mesh(
      track(new THREE.PlaneGeometry(glow.width, glow.height)),
      track(
        new THREE.MeshBasicMaterial({
          map: track(glow.texture),
          transparent: true,
          opacity: 0.95,
          depthWrite: false,
          blending: THREE.AdditiveBlending,
          side: THREE.DoubleSide,
          toneMapped: false
        })
      )
    );
    phoneGlow.position.z = -phoneDepth / 2 - 0.03;
    phoneGlow.renderOrder = 2;
    phonePivot.add(phoneBody, frontGlass, screen, island, sheen, backGlass, backLogo, cameraModule, phoneGlow);
    phone.add(phonePivot);
    scene.add(phone);

    // Cards lifted out of the phone screen, set out on its left and joined to it
    // by a right-angled tree: a stub from the phone's edge, a vertical trunk and
    // one branch per card, with nodes at the joints. Everything sits in one
    // plane in the phone group (not the spinning pivot), so the lines stay exact.
    const TREE = { plane: 0.3, phoneEdge: -0.5, trunk: -0.7, cardEdge: -0.9, rows: [0.64, 0, -0.64], line: 0.011 };
    const cardWidth = 1.06;
    const lineMaterialFor = () =>
      track(
        new THREE.MeshBasicMaterial({
          color: COLORS.ice,
          transparent: true,
          opacity: 0,
          depthWrite: false,
          blending: THREE.AdditiveBlending,
          toneMapped: false
        })
      );
    const segmentGeometry = track(new THREE.PlaneGeometry(1, 1));
    const nodeGeometry = track(new THREE.CircleGeometry(1, 32));
    const makeTree = (images: HTMLImageElement[]) => {
      const group = new THREE.Group();
      group.visible = false;
      const lines = lineMaterialFor();
      const nodes = lineMaterialFor();
      const segment = (x1: number, y1: number, x2: number, y2: number) => {
        const mesh = new THREE.Mesh(segmentGeometry, lines);
        mesh.position.set((x1 + x2) / 2, (y1 + y2) / 2, 0);
        mesh.scale.set(Math.max(Math.abs(x2 - x1), TREE.line), Math.max(Math.abs(y2 - y1), TREE.line), 1);
        group.add(mesh);
      };
      const node = (x: number, y: number, radius: number) => {
        const mesh = new THREE.Mesh(nodeGeometry, nodes);
        mesh.position.set(x, y, 0.001);
        mesh.scale.setScalar(radius);
        group.add(mesh);
      };
      const [top, , bottom] = TREE.rows;
      segment(TREE.phoneEdge, 0, TREE.trunk, 0);
      segment(TREE.trunk, top, TREE.trunk, bottom);
      node(TREE.phoneEdge, 0, 0.028);
      node(TREE.trunk, 0, 0.02);
      const cards = images.map((image, index) => {
        const y = TREE.rows[index % TREE.rows.length];
        segment(TREE.trunk, y, TREE.cardEdge, y);
        node(TREE.cardEdge, y, 0.02);
        const material = track(
          sharpen(
            new THREE.MeshBasicMaterial({
              map: track(colorTexture(image, anisotropy)),
              transparent: true,
              opacity: 0,
              depthWrite: false,
              toneMapped: false
            })
          )
        );
        const card = new THREE.Mesh(track(new THREE.PlaneGeometry(cardWidth, (cardWidth * image.height) / image.width)), material);
        // The image has a transparent margin around the card; its visible right
        // edge (85% of the plane) meets the branch.
        const x = TREE.cardEdge - (cardWidth * 0.85) / 2;
        card.position.set(x, y, 0.002);
        card.renderOrder = 3;
        group.add(card);
        return { card, material, x, y };
      });
      group.position.z = TREE.plane;
      phone.add(group);
      return { group, lines, nodes, cards };
    };
    const trainTree = makeTree(trainCallouts);
    const fuelTree = makeTree(fuelCallouts);

    // Evolve: one badge between the two screens.
    const syncBadge = new THREE.Sprite(
      track(
        sharpen(
          new THREE.SpriteMaterial({
            map: track(colorTexture(syncImage, anisotropy)),
            transparent: true,
            opacity: 0,
            depthWrite: false,
            toneMapped: false
          })
        )
      )
    );
    syncBadge.scale.set(1.25, (1.25 * syncImage.height) / syncImage.width, 1);
    syncBadge.visible = false;
    scene.add(syncBadge);

    // Laptop -----------------------------------------------------------------------------
    const laptop = new THREE.Group();
    const shell = track(
      new THREE.MeshPhysicalMaterial({
        color: COLORS.shell,
        metalness: 1,
        roughness: 0.32,
        clearcoat: 0.2,
        clearcoatRoughness: 0.3,
        envMapIntensity: 0.75
      })
    );
    // The screen is what people read: sample it with the GPU's best filtering.
    const sharpest = renderer.capabilities.getMaxAnisotropy();
    const deckWidth = 3.3;
    const deckDepth = 2.25;
    const deckThickness = 0.09;
    const deckMaterial = track(shell.clone());
    deckMaterial.envMapIntensity = 0.42;
    deckMaterial.roughness = 0.38;
    const deck = new THREE.Mesh(track(slabGeometry(deckWidth, deckDepth, 0.13, deckThickness, 0.028)), deckMaterial);
    deck.rotation.x = -Math.PI / 2;
    const top = deckThickness / 2;
    const well = new THREE.Mesh(
      track(panelGeometry(KEYBOARD.width, KEYBOARD.depth, 0.05)),
      track(new THREE.MeshStandardMaterial({ color: 0x070a14, roughness: 0.85, metalness: 0.3 }))
    );
    well.rotation.x = -Math.PI / 2;
    well.position.set(0, top + 0.0008, KEYBOARD.centerZ);

    const caps = layoutKeys();
    const keyHeight = 0.022;
    const keys = new THREE.InstancedMesh(
      track(new RoundedBoxGeometry(1, 1, 1, 3, 0.22)),
      // Matte black caps: horizontal parts are seen at a grazing angle, where the
      // studio environment would mirror white, so they barely reflect it.
      track(new THREE.MeshStandardMaterial({ color: 0x05070d, roughness: 0.7, metalness: 0, envMapIntensity: 0.06 })),
      caps.length
    );
    {
      const placement = new THREE.Object3D();
      caps.forEach((cap, index) => {
        placement.position.set(cap.x, top + 0.012, cap.z);
        placement.scale.set(cap.width, keyHeight, cap.depth);
        placement.updateMatrix();
        keys.setMatrixAt(index, placement.matrix);
      });
      keys.instanceMatrix.needsUpdate = true;
      keys.computeBoundingSphere();
    }
    const legends = new THREE.Mesh(
      track(new THREE.PlaneGeometry(KEYBOARD.width, KEYBOARD.depth)),
      track(
        new THREE.MeshBasicMaterial({
          map: track(keyLegendTexture(caps, sharpest)),
          transparent: true,
          depthWrite: false,
          toneMapped: false
        })
      )
    );
    legends.rotation.x = -Math.PI / 2;
    legends.position.set(0, top + 0.012 + keyHeight / 2 + 0.0008, KEYBOARD.centerZ);

    // Glass trackpad with a chamfered edge, and the opening scoop on the front.
    const trackpadEdge = new THREE.Mesh(
      track(panelGeometry(1.33, 0.83, 0.075)),
      track(new THREE.MeshStandardMaterial({ color: 0x121829, roughness: 0.5, metalness: 0.6, envMapIntensity: 0.5 }))
    );
    trackpadEdge.rotation.x = -Math.PI / 2;
    trackpadEdge.position.set(0, top + 0.0005, 0.6);
    const trackpad = new THREE.Mesh(
      track(panelGeometry(1.3, 0.8, 0.065)),
      track(
        new THREE.MeshPhysicalMaterial({
          color: 0x2b3550,
          roughness: 0.3,
          metalness: 0.6,
          clearcoat: 0.3,
          clearcoatRoughness: 0.2,
          envMapIntensity: 0.22
        })
      )
    );
    trackpad.rotation.x = -Math.PI / 2;
    trackpad.position.set(0, top + 0.001, 0.6);
    const scoop = new THREE.Mesh(
      track(panelGeometry(0.52, 0.034, 0.017)),
      track(new THREE.MeshStandardMaterial({ color: 0x141a2a, roughness: 0.6, metalness: 0.5 }))
    );
    scoop.position.set(0, top - 0.004, deckDepth / 2 + 0.0015);
    const hinge = new THREE.Mesh(track(new THREE.CylinderGeometry(0.045, 0.045, 2.7, 24)), shell);
    hinge.rotation.z = Math.PI / 2;
    hinge.position.set(0, top, -deckDepth / 2 + 0.05);

    const lidPivot = new THREE.Group();
    lidPivot.position.set(0, top, -deckDepth / 2 + 0.05);
    const lidHeight = 2.14;
    const lidThickness = 0.04;
    const lidShell = new THREE.Mesh(track(slabGeometry(deckWidth, lidHeight, 0.13, lidThickness, 0.014)), shell);
    lidShell.position.set(0, lidHeight / 2 + 0.02, 0);
    const lidGlass = new THREE.Mesh(
      track(panelGeometry(deckWidth - 0.04, lidHeight - 0.04, 0.12)),
      track(new THREE.MeshPhysicalMaterial({ color: 0x010206, roughness: 0.08, clearcoat: 1, clearcoatRoughness: 0.04 }))
    );
    lidGlass.position.set(0, lidHeight / 2 + 0.02, lidThickness / 2 + 0.0006);
    const displayWidth = deckWidth - 0.2;
    const displayHeight = displayWidth / (platformImage.width / platformImage.height);
    const display = new THREE.Mesh(
      track(panelGeometry(displayWidth, displayHeight, 0.03)),
      track(sharpen(new THREE.MeshBasicMaterial({ map: track(colorTexture(platformImage, sharpest)), toneMapped: false })))
    );
    display.position.set(0, lidHeight / 2 + 0.06, lidThickness / 2 + 0.0012);
    // Camera notch at the top centre of the display.
    const notch = new THREE.Mesh(track(panelGeometry(0.3, 0.075, 0.025)), track(new THREE.MeshBasicMaterial({ color: 0x010206 })));
    notch.position.set(0, lidHeight / 2 + 0.06 + displayHeight / 2 - 0.03, lidThickness / 2 + 0.0018);
    const lidLogo = new THREE.Mesh(
      track(new THREE.PlaneGeometry(0.5 * logoAspect, 0.5)),
      track(new THREE.MeshBasicMaterial({ map: logoTexture, transparent: true, toneMapped: false, depthWrite: false }))
    );
    lidLogo.position.set(0, lidHeight / 2 + 0.02, -lidThickness / 2 - 0.0012);
    lidLogo.rotation.y = Math.PI;
    lidPivot.add(lidShell, lidGlass, display, notch, lidLogo);

    const laptopGlow = new THREE.Mesh(
      track(new THREE.PlaneGeometry(6.4, 3.4)),
      track(
        new THREE.MeshBasicMaterial({
          map: track(radialTexture("rgba(4,129,220,0.5)", "rgba(4,129,220,0)")),
          transparent: true,
          depthWrite: false,
          blending: THREE.AdditiveBlending,
          toneMapped: false
        })
      )
    );
    laptopGlow.rotation.x = -Math.PI / 2;
    laptopGlow.position.y = -0.08;
    laptop.add(laptopGlow, deck, well, keys, legends, trackpadEdge, trackpad, scoop, hinge, lidPivot);
    scene.add(laptop);

    // Frame loop ----------------------------------------------------------------------------
    const timer = new THREE.Timer();
    timer.connect(document);
    const state = { ...base };
    let targetProgress = 0;
    let currentProgress = 0;
    const pointerTarget = new THREE.Vector2();
    const pointer = new THREE.Vector2();
    let halfWidth = 1;
    // Narrower (squarer) stages shrink the objects and push the devices right so
    // they stay clear of the copy column.
    let fit = 1;
    let running = false;
    let activeScreen: PhoneScreen = "training";
    const badgeFrom = new THREE.Vector3();
    const badgeTo = new THREE.Vector3();

    // The tree draws in first, then each card arrives along its branch.
    const showTree = (tree: typeof trainTree, amount: number, elapsed: number) => {
      tree.group.visible = amount > 0.01;
      if (!tree.group.visible) {
        return;
      }
      const lineIn = smooth(THREE.MathUtils.clamp(amount * 2, 0, 1));
      tree.lines.opacity = lineIn * 0.75;
      tree.nodes.opacity = lineIn;
      tree.group.position.y = Math.sin(elapsed * 0.8) * 0.02;
      tree.cards.forEach(({ card, material, x, y }, index) => {
        const eased = smooth(THREE.MathUtils.clamp(amount * 1.8 - 0.3 - index * 0.25, 0, 1));
        material.opacity = eased;
        card.visible = eased > 0.01;
        card.position.set(x + (1 - eased) * 0.25, y, 0.002);
      });
    };

    const apply = (elapsed: number) => {
      sampleState(currentProgress, state);
      globe.position.set(state.globeX * halfWidth, state.globeY, state.globeZ);
      globe.scale.setScalar(state.globeScale * fit);
      globe.rotation.set(baseTilt + pointer.y * 0.08, pointer.x * 0.12, 0);
      // A slow sway keeps the Atlantic routes in view instead of a full spin.
      earth.rotation.set(0, baseYaw + state.globeSpin + Math.sin(elapsed * 0.05) * 0.35, 0);
      routeMaterial.uniforms.uTime.value = elapsed;
      cityMaterial.uniforms.uScale.value = state.globeScale * fit * 10;
      const beat = (elapsed * 0.45) % 1;
      pulse.scale.setScalar(1 + beat * 3.2);
      pulseMaterial.opacity = (1 - beat) * 0.9;
      starMaterial.opacity = state.stars;
      stars.visible = state.stars > 0.02;

      phone.position.set(state.phoneX * halfWidth, state.phoneY + Math.sin(elapsed * 1.1) * 0.03, state.phoneZ);
      phone.scale.setScalar(state.phoneScale * fit);
      phonePivot.rotation.set(state.phoneRotX + pointer.y * 0.06, state.phoneRotY + pointer.x * 0.12, state.phoneRotZ);
      phone.visible = state.phoneY > -6;

      const wanted = screenFor(currentProgress);
      if (wanted !== activeScreen) {
        activeScreen = wanted;
        screenMaterial.map = screens[wanted];
      }

      laptop.position.set((state.laptopX + (1 - fit) * 0.35) * halfWidth, state.laptopY, state.laptopZ);
      laptop.scale.setScalar(state.laptopScale * fit);
      // Tilted so the open display faces the viewer almost square-on.
      laptop.rotation.set(0.28, state.laptopRotY + pointer.x * 0.05, 0);
      lidPivot.rotation.x = state.lid;
      laptop.visible = state.laptopY > -6.5;

      showTree(trainTree, windowed(currentProgress, 0.25, 0.31, 0.41, 0.45), elapsed);
      showTree(fuelTree, windowed(currentProgress, 0.555, 0.6, 0.63, 0.67), elapsed);

      // The sync badge floats above the gap between laptop and phone.
      const sync = windowed(currentProgress, 0.79, 0.83, 0.87, 0.9);
      syncBadge.visible = sync > 0.01;
      if (syncBadge.visible) {
        phone.updateMatrixWorld(true);
        laptop.updateMatrixWorld(true);
        screen.localToWorld(badgeFrom.set(0, 0.95, 0.01));
        display.localToWorld(badgeTo.set(displayWidth * 0.5, displayHeight * 0.5, 0.01));
        syncBadge.position.set((badgeFrom.x + badgeTo.x) / 2, Math.max(badgeFrom.y, badgeTo.y) + 0.38, (badgeFrom.z + badgeTo.z) / 2 + 0.3);
        syncBadge.material.opacity = sync;
      }

      camera.position.set(pointer.x * 0.15, pointer.y * 0.1, state.cameraZ);
      camera.lookAt(0, 0, 0);
    };

    // Full frame rate while scroll or pointer is moving; once settled only the
    // slow idle motion remains, so it is drawn at about 30 fps.
    const IDLE_FRAME = 1 / 30;
    let sinceRender = IDLE_FRAME;
    const frame = () => {
      timer.update();
      const delta = Math.min(timer.getDelta(), 0.1);
      const follow = 1 - Math.exp(-delta * 7);
      currentProgress += (targetProgress - currentProgress) * follow;
      pointer.lerp(pointerTarget, 1 - Math.exp(-delta * 3));
      const settled = Math.abs(targetProgress - currentProgress) < 1e-4 && pointer.distanceToSquared(pointerTarget) < 1e-6;
      sinceRender += delta;
      if (settled && sinceRender < IDLE_FRAME) {
        return;
      }
      sinceRender = 0;
      apply(timer.getElapsed());
      renderer.render(scene, camera);
    };

    const resize = (width: number, height: number) => {
      if (width === 0 || height === 0) {
        return;
      }
      renderer.setSize(width, height, false);
      camera.aspect = width / height;
      camera.updateProjectionMatrix();
      const halfHeight = Math.tan(THREE.MathUtils.degToRad(camera.fov / 2)) * base.cameraZ;
      halfWidth = halfHeight * camera.aspect;
      fit = THREE.MathUtils.clamp(camera.aspect / 1.6, 0.72, 1);
      cityMaterial.uniforms.uPixelRatio.value = renderer.getPixelRatio() * (height / 900);
    };

    const rect = canvas.getBoundingClientRect();
    resize(rect.width, rect.height);
    apply(0);
    await renderer.compileAsync(scene, camera);
    renderer.render(scene, camera);

    return {
      setProgress: (progress) => {
        targetProgress = progress;
      },
      setPointer: (x, y) => {
        pointerTarget.set(x, y);
      },
      start: () => {
        if (!running) {
          running = true;
          timer.reset();
          renderer.setAnimationLoop(frame);
        }
      },
      stop: () => {
        running = false;
        renderer.setAnimationLoop(null);
      },
      resize,
      dispose: () => {
        running = false;
        renderer.setAnimationLoop(null);
        timer.dispose();
        for (const item of disposables) {
          item.dispose();
        }
        renderer.dispose();
        renderer.forceContextLoss();
      }
    };
  } catch (error) {
    renderer.setAnimationLoop(null);
    renderer.dispose();
    renderer.forceContextLoss();
    throw error;
  }
};
