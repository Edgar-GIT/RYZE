import * as THREE from "three";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";
import { RoundedBoxGeometry } from "three/addons/geometries/RoundedBoxGeometry.js";

import { INTRO_END, createIntro } from "./story_intro";
import { createPlanet } from "./story_planet";

/*
 * The Home story scene. One persistent WebGL scene whose state is a function of
 * scroll progress (0..1): the hero's opening, where a drone projects the phone,
 * a dumbbell and a meal as holograms that turn solid (story_intro.ts), the
 * phone that then presents the training and nutrition screens, a
 * laptop that opens on the programme page. Loaded with a dynamic import on
 * desktop only.
 *
 * Everything is procedural (no model files); the device screens are
 * illustrative RYZE screens.
 */

export interface StorySceneAssets {
  logo: string;
  screens: { training: string; nutrition: string; synced: string; platform: string };
  callouts: { train: readonly string[]; fuel: readonly string[]; sync: string };
  landMask: string; // Natural Earth land mask for planet RYZE
}

export interface StoryScene {
  // `immediate` jumps straight to the state instead of easing towards it (used
  // when the canvas moves to the other stage).
  setProgress: (progress: number, immediate?: boolean) => void;
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
  // Device metals since 2026-11-28 (owner request): black, silver and chrome
  // — a polished chrome phone frame on black glass, a space-black laptop with
  // chrome accents: mirror-like, absorbing, less contrasting on the blue.
  // Owner report, 2026-11-28: the first chrome still read white once the
  // phone turned (the clear coat mirrored the studio's white walls at grazing
  // angles), so the metals are now a dark graphite chrome with no clear coat.
  titanium: 0x3d4046,
  shell: 0x1e2024
} as const;

// --- Scroll keyframes ----------------------------------------------------------

interface SceneState {
  // The hero's composition around the phone (drone, beam, dumbbell, meal).
  heroX: number; // fractions of the visible half width
  heroY: number;
  heroZ: number;
  heroScale: number;
  heroSpin: number;
  // Planet RYZE, for the last chapter.
  planetX: number;
  planetY: number;
  planetZ: number;
  planetScale: number;
  planetSpin: number;
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

// The hero: the phone stands at the centre of its composition, turned towards
// the copy, with the dumbbell in front and the meal beside it.
const base: SceneState = {
  heroX: 0.34,
  heroY: -1.32,
  heroZ: 0,
  heroScale: 1,
  heroSpin: 0,
  planetX: 1.2,
  planetY: 6.5,
  planetZ: -7,
  planetScale: 0.62,
  // The globe turns as it comes down and stops with RYZE (Lisbon) facing the
  // viewer (owner request, 2026-11-28).
  planetSpin: -0.5,
  stars: 1,
  // Close to its seat on the hero's surface (the scene locks it there at the top).
  phoneX: 0.38,
  phoneY: -0.45,
  phoneZ: 0.9,
  phoneRotX: -0.17,
  phoneRotY: -0.08,
  phoneRotZ: 0,
  phoneScale: 1.36,
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
// The dumbbell and the meal leave up and away once the hero scrolls on.
const heroAway = { heroX: 1.0, heroY: 6, heroZ: -6, heroScale: 0.6, heroSpin: 1, stars: 0.35 };
// The phone rises out of the frame (up and right) while the programs pass,
// then comes back down for the Train chapter.
const phoneTucked = { phoneX: 1.4, phoneY: 4.2, phoneZ: -2, phoneRotX: 0.12, phoneRotY: -0.55, phoneRotZ: 0, phoneScale: 0.7 };
// Laptop and phone side by side, both turned slightly towards the copy.
const phoneBesideLaptop = {
  phoneX: 0.71,
  phoneY: -0.38,
  phoneZ: 0.6,
  phoneRotX: 0.02,
  phoneRotZ: 0,
  phoneScale: 0.82,
  laptopX: 0.27
};
// Planet RYZE comes down for the last chapter ("join planet RYZE").
const planetHere = { planetX: 0.35, planetY: -0.02, planetZ: 0, planetScale: 1.0, planetSpin: 0, stars: 1 };

// While the hero scrolls away (0 → 0.1) the phone rises out to the upper
// right, away while the programs pass (0.1 → 0.14); the chapters then bring it
// back (0.14 → 0.25) as the programs leave.
const KEYS: Array<{ at: number; state: SceneState }> = [
  { at: 0, state: base },
  { at: 0.1, state: { ...base, ...heroAway, ...phoneTucked } },
  { at: 0.14, state: { ...base, ...heroAway, ...phoneTucked } },
  { at: 0.25, state: { ...base, ...heroAway, ...phoneHero, phoneRotY: -0.42 } },
  { at: 0.36, state: { ...base, ...heroAway, ...phoneHero, phoneRotY: -0.16 } },
  { at: 0.42, state: { ...base, ...heroAway, ...phoneHero, phoneRotY: -0.16 } },
  { at: 0.51, state: { ...base, ...heroAway, ...phoneHero, phoneRotY: TAU - 0.16 } },
  { at: 0.64, state: { ...base, ...heroAway, ...phoneHero, phoneRotY: TAU - 0.3 } },
  { at: 0.72, state: { ...base, ...heroAway, ...phoneBesideLaptop, phoneRotY: 2 * TAU - 0.42, laptopY: -0.9 } },
  { at: 0.79, state: { ...base, ...heroAway, ...phoneBesideLaptop, phoneRotY: 2 * TAU - 0.38, laptopY: -0.9, lid: LID_OPEN } },
  { at: 0.88, state: { ...base, ...heroAway, ...phoneBesideLaptop, phoneRotY: 2 * TAU - 0.34, laptopY: -0.9, laptopRotY: -0.16, lid: LID_OPEN } },
  { at: 0.95, state: { ...base, ...heroAway, ...planetHere, phoneY: -7, phoneRotY: 2 * TAU, laptopY: -8, lid: LID_OPEN } },
  { at: 1, state: { ...base, ...heroAway, ...planetHere, phoneY: -7, phoneRotY: 2 * TAU, laptopY: -8, lid: LID_OPEN } }
];

// The phone shows its back half-way through each spin; the screen is swapped
// there so the change is never visible.
// In the Evolve chapter the phone shows a large-type screen that stays legible
// at its smaller size there.
type PhoneScreen = "training" | "nutrition" | "synced";
const screenFor = (progress: number): PhoneScreen =>
  progress >= 0.465 && progress < 0.68 ? "nutrition" : progress >= 0.68 ? "synced" : "training";

// Where each lifted card comes from on its phone screen, in screen fractions
// (left, top, right, bottom), the height of its card and where its leader turns.
// Cards are ordered top to bottom like their elements, so leaders never cross.
interface CalloutTarget {
  rect: readonly [number, number, number, number];
  row: number;
  elbow: number;
}
const CALLOUT_TARGETS: Record<"train" | "fuel", readonly CalloutTarget[]> = {
  train: [
    { rect: [0.035, 0.494, 0.965, 0.551], row: 0.68, elbow: -0.64 }, // week days → "Week 4 of 6"
    { rect: [0.07, 0.619, 0.93, 0.657], row: 0, elbow: -0.78 }, // bench press → logged set
    { rect: [0.35, 0.773, 0.65, 0.855], row: -0.66, elbow: -0.72 } // rest → rest timer
  ],
  fuel: [
    { rect: [0.04, 0.128, 0.96, 0.252], row: 0.68, elbow: -0.72 }, // daily target → target card
    { rect: [0.042, 0.4, 0.347, 0.495], row: 0, elbow: -0.72 }, // protein → protein card
    { rect: [0.04, 0.645, 0.96, 0.757], row: -0.66, elbow: -0.72 } // lunch → meal card
  ]
};

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

type Point2 = [number, number];

// Points along a path whose corners are rounded with quadratic curves (radius
// limited to half of each neighbouring segment). Open paths keep their ends.
const roundedPath = (corners: Point2[], radius: number, closed = false): Point2[] => {
  const points: Point2[] = [];
  const count = corners.length;
  for (let index = 0; index < count; index += 1) {
    const current = corners[index];
    if (!closed && (index === 0 || index === count - 1)) {
      points.push(current);
      continue;
    }
    const previous = corners[(index - 1 + count) % count];
    const next = corners[(index + 1) % count];
    const inLength = Math.hypot(current[0] - previous[0], current[1] - previous[1]);
    const outLength = Math.hypot(next[0] - current[0], next[1] - current[1]);
    const r = Math.min(radius, inLength / 2, outLength / 2);
    if (r < 1e-4) {
      points.push(current);
      continue;
    }
    const from: Point2 = [current[0] + ((previous[0] - current[0]) / inLength) * r, current[1] + ((previous[1] - current[1]) / inLength) * r];
    const to: Point2 = [current[0] + ((next[0] - current[0]) / outLength) * r, current[1] + ((next[1] - current[1]) / outLength) * r];
    for (let step = 0; step <= 10; step += 1) {
      const t = step / 10;
      const a = (1 - t) * (1 - t);
      const b = 2 * (1 - t) * t;
      const c = t * t;
      points.push([a * from[0] + b * current[0] + c * to[0], a * from[1] + b * current[1] + c * to[1]]);
    }
  }
  if (closed) {
    points.push(points[0]);
  }
  return points.filter((point, index) => index === 0 || Math.hypot(point[0] - points[index - 1][0], point[1] - points[index - 1][1]) > 1e-5);
};

// A flat strip along a 2D path. `along` runs 0..1 from the first point to the
// last (for drawing it in), `across` −1..1 over its width (for soft edges).
const ribbonGeometry = (points: Point2[], width: number): THREE.BufferGeometry => {
  const lengths = [0];
  for (let index = 1; index < points.length; index += 1) {
    lengths.push(lengths[index - 1] + Math.hypot(points[index][0] - points[index - 1][0], points[index][1] - points[index - 1][1]));
  }
  const total = lengths[lengths.length - 1] || 1;
  const positions: number[] = [];
  const along: number[] = [];
  const across: number[] = [];
  const indices: number[] = [];
  points.forEach((point, index) => {
    const previous = points[Math.max(index - 1, 0)];
    const next = points[Math.min(index + 1, points.length - 1)];
    const length = Math.hypot(next[0] - previous[0], next[1] - previous[1]) || 1;
    const nx = (-(next[1] - previous[1]) / length) * (width / 2);
    const ny = ((next[0] - previous[0]) / length) * (width / 2);
    positions.push(point[0] + nx, point[1] + ny, 0, point[0] - nx, point[1] - ny, 0);
    along.push(lengths[index] / total, lengths[index] / total);
    across.push(1, -1);
    if (index > 0) {
      const a = (index - 1) * 2;
      indices.push(a, a + 1, a + 2, a + 1, a + 3, a + 2);
    }
  });
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute("position", new THREE.Float32BufferAttribute(positions, 3));
  geometry.setAttribute("along", new THREE.Float32BufferAttribute(along, 1));
  geometry.setAttribute("across", new THREE.Float32BufferAttribute(across, 1));
  geometry.setIndex(indices);
  return geometry;
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
// Perforated speaker grille: a dense grid of small dark holes with a faint
// cyan rim, on a transparent background.
const speakerGrilleTexture = (anisotropy: number) => {
  const { element, context } = canvas2d(128, 1280);
  const step = 16;
  for (let y = step / 2; y < 1280; y += step) {
    for (let x = step / 2 + ((y / step) % 2 ? step / 2 : 0); x < 128; x += step) {
      context.beginPath();
      context.arc(x, y, 3.4, 0, TAU);
      context.fillStyle = "rgba(2, 4, 12, 0.92)";
      context.fill();
      context.lineWidth = 1;
      context.strokeStyle = "rgba(124, 200, 255, 0.18)";
      context.stroke();
    }
  }
  return colorTexture(element, anisotropy);
};

const LEGEND_TEXTURE_WIDTH = 3072;
const keyLegendTexture = (caps: KeyCap[], anisotropy: number) => {
  const scale = LEGEND_TEXTURE_WIDTH / KEYBOARD.width;
  const { element, context } = canvas2d(LEGEND_TEXTURE_WIDTH, Math.round(KEYBOARD.depth * scale));
  const top = KEYBOARD.centerZ - KEYBOARD.depth / 2;
  context.fillStyle = "rgba(226, 234, 248, 0.94)";
  context.strokeStyle = "rgba(226, 234, 248, 0.94)";
  context.shadowColor = "rgba(124, 235, 255, 0.6)";
  context.shadowBlur = 9;
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

// Owner test, 2026-11-28: the hero plays a generated video (story_immersive)
// instead of the 3D composition. The composition and the seated phone stay in
// the code but are not drawn; the later chapters keep their 3D.
const HERO_3D_HIDDEN = true;

export const createStoryScene = async (canvas: HTMLCanvasElement, assets: StorySceneAssets): Promise<StoryScene> => {
  // The key legends are drawn with the site's typeface once it is available.
  await document.fonts.load('600 48px "RYZE Archivo"').catch(() => undefined);
  // Images first: a failed download must not leave a WebGL context behind.
  const [logoImage, trainingImage, nutritionImage, syncedImage, platformImage, trainCallouts, fuelCallouts, syncImage, maskImage] =
    await Promise.all([
      loadImage(assets.logo),
      loadImage(assets.screens.training),
      loadImage(assets.screens.nutrition),
      loadImage(assets.screens.synced),
      loadImage(assets.screens.platform),
      Promise.all(assets.callouts.train.map(loadImage)),
      Promise.all(assets.callouts.fuel.map(loadImage)),
      loadImage(assets.callouts.sync),
      loadImage(assets.landMask)
    ]);

  // Default GPU preference (no forced discrete GPU); high-DPI screens skip MSAA.
  const pixelRatio = Math.min(window.devicePixelRatio, 1.5);
  const renderer = new THREE.WebGLRenderer({ canvas, antialias: window.devicePixelRatio < 1.5, alpha: true });
  try {
    renderer.setPixelRatio(pixelRatio);
    // The hero's holograms turn solid behind a moving clipping plane.
    renderer.localClippingEnabled = true;
    // Real shadows for the hero composition on its surface.
    renderer.shadowMap.enabled = true;
    renderer.shadowMap.type = THREE.PCFShadowMap;
    renderer.shadowMap.autoUpdate = false;
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
    // It also casts the hero composition's shadows: aimed at the hero, from
    // the same direction; its shadow camera frames the composition only.
    key.target.position.set(1.35, -1.6, 0);
    key.position.copy(key.target.position).add(new THREE.Vector3(-4, 5, 6).normalize().multiplyScalar(14));
    key.castShadow = true;
    key.shadow.mapSize.set(2048, 2048);
    key.shadow.camera.left = -3.4;
    key.shadow.camera.right = 3.4;
    key.shadow.camera.top = 3.4;
    key.shadow.camera.bottom = -3.4;
    key.shadow.camera.near = 4;
    key.shadow.camera.far = 26;
    key.shadow.bias = -0.0004;
    key.shadow.normalBias = 0.02;
    key.shadow.radius = 4;
    scene.add(key.target);
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

    // Planet RYZE, for the last chapter (it led the hero until 2026-11-15) -------------
    const planet = createPlanet(track, { maskImage, logoImage, anisotropy, pixelRatio });
    scene.add(planet.group);

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
      // Polished chrome (owner, 2026-11-28): a mirror of the studio light.
      new THREE.MeshPhysicalMaterial({
        color: COLORS.titanium,
        metalness: 1,
        roughness: 0.16,
        envMapIntensity: 0.9
      })
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
    // A streak of light that crosses the glass every few seconds.
    const glintMaterial = track(
      new THREE.ShaderMaterial({
        uniforms: { uTime: { value: 0 }, uOpacity: { value: 0 } },
        vertexShader: /* glsl */ `
          varying vec2 vUv;
          void main() {
            vUv = uv;
            gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
          }
        `,
        fragmentShader: /* glsl */ `
          uniform float uTime;
          uniform float uOpacity;
          varying vec2 vUv;
          void main() {
            float cycle = fract(uTime / 7.0);
            float position = (vUv.x + vUv.y * 0.6) - (cycle * 3.2 - 0.6);
            float streak = exp(-position * position * 90.0) * smoothstep(0.0, 0.08, cycle) * (1.0 - smoothstep(0.32, 0.4, cycle));
            gl_FragColor = vec4(vec3(1.0), streak * 0.22 * uOpacity);
          }
        `,
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending
      })
    );
    const glint = new THREE.Mesh(track(panelGeometry(screenWidth, screenHeight, 0.12, 16)), glintMaterial);
    glint.position.z = phoneDepth / 2 + 0.003;

    const backGlass = new THREE.Mesh(
      track(panelGeometry(phoneWidth - 0.028, phoneHeight - 0.028, 0.145)),
      track(
        // Black glass back, deep and glossy.
        new THREE.MeshPhysicalMaterial({ color: 0x0b0c0f, roughness: 0.26, metalness: 0.2, clearcoat: 0.4, clearcoatRoughness: 0.2, envMapIntensity: 0.55 })
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
      track(new THREE.MeshPhysicalMaterial({ color: 0x1c1e22, roughness: 0.22, metalness: 0.7, clearcoat: 0.4, clearcoatRoughness: 0.15, envMapIntensity: 0.6 }))
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

    // A soft light rim (light on the blue field, not a darker blue halo).
    const glow = edgeGlow(phoneWidth, phoneHeight, 0.16, "rgba(170,236,255,1)");
    const phoneGlow = new THREE.Mesh(
      track(new THREE.PlaneGeometry(glow.width, glow.height)),
      track(
        new THREE.MeshBasicMaterial({
          map: track(glow.texture),
          transparent: true,
          opacity: 0.42,
          depthWrite: false,
          blending: THREE.AdditiveBlending,
          side: THREE.DoubleSide,
          toneMapped: false
        })
      )
    );
    phoneGlow.position.z = -phoneDepth / 2 - 0.03;
    phoneGlow.renderOrder = 2;
    phonePivot.add(phoneBody, frontGlass, screen, island, sheen, glint, backGlass, backLogo, cameraModule, phoneGlow);
    // A soft shadow below and behind, so the phone floats over the field.
    const phoneShadow = new THREE.Mesh(
      track(new THREE.PlaneGeometry(phoneWidth * 1.9, phoneHeight * 1.35)),
      track(
        new THREE.MeshBasicMaterial({
          map: track(radialTexture("rgba(1,8,40,0.42)", "rgba(1,8,40,0)")),
          transparent: true,
          depthWrite: false,
          toneMapped: false
        })
      )
    );
    phoneShadow.position.set(0.16, -0.24, -0.55);
    phoneShadow.renderOrder = -3;
    phone.add(phonePivot, phoneShadow);
    scene.add(phone);

    // The hero's opening: the drone projects the phone, a dumbbell and a meal,
    // which then turn solid (story_intro.ts). Built before the cards are joined
    // to the screen, so the phone's hologram twin has only the phone's parts.
    const intro = createIntro(
      track,
      pixelRatio,
      {
        pivot: phonePivot,
        body: phoneBody,
        screen,
        skip: new Set([frontGlass, island, sheen, glint, backLogo, phoneGlow]),
        screenMap: screens.training,
        size: { width: phoneWidth, height: phoneHeight, depth: phoneDepth }
      },
      logoTexture,
      logoAspect
    );
    // The phone's shadow appears with the solid phone (same clipping plane).
    phoneShadow.material.clippingPlanes = titanium.clippingPlanes;
    // A near-frontal view, slightly from above, as in the owner's reference
    // (2026-11-28).
    const heroTilt = 0.16;
    // Turned a touch to the left, towards the copy.
    const heroYaw = -0.12;
    const seatMatrix = new THREE.Matrix4();
    const seatPosition = new THREE.Vector3();
    const seatQuaternion = new THREE.Quaternion();
    const seatScale = new THREE.Vector3();
    const phoneShadowMaterial = phoneShadow.material;
    scene.add(intro.group);
    // The composition is there from the start (owner's reference,
    // 2026-11-28): the drone's projection no longer plays.
    let introTime = INTRO_END;
    let shadowFrame = 0;
    let introSettled = false;

    // Cards lifted out of the phone screen. Each one is joined to the part of
    // the screen it shows: that element is outlined on the screen (so it turns
    // with the phone), and a rounded leader runs from the phone's edge at the
    // element's height to the card. Outline, leader and card arrive in that
    // order; once drawn, a light pulse travels along each leader.
    const DIAGRAM = { plane: 0.1, phoneEdge: -0.52, cardEdge: -0.95, line: 0.014, outline: 0.009, radius: 0.08 };
    const cardWidth = 1.06;
    const leaderColor = new THREE.Color().setHex(COLORS.ice, THREE.LinearSRGBColorSpace);
    const leaderMaterial = (pulse: number) =>
      track(
        new THREE.ShaderMaterial({
          uniforms: {
            uColor: { value: leaderColor },
            uDraw: { value: 0 },
            uOpacity: { value: 1 },
            uTime: { value: 0 },
            uPulse: { value: pulse }
          },
          vertexShader: /* glsl */ `
            attribute float along;
            attribute float across;
            varying float vAlong;
            varying float vAcross;
            void main() {
              vAlong = along;
              vAcross = across;
              gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
            }
          `,
          fragmentShader: /* glsl */ `
            uniform vec3 uColor;
            uniform float uDraw;
            uniform float uOpacity;
            uniform float uTime;
            uniform float uPulse;
            varying float vAlong;
            varying float vAcross;
            void main() {
              float drawn = 1.0 - smoothstep(uDraw - 0.012, uDraw, vAlong);
              float edge = 1.0 - smoothstep(0.3, 1.0, abs(vAcross));
              float tip = exp(-pow((vAlong - uDraw) * 24.0, 2.0)) * (1.0 - step(0.999, uDraw));
              float travel = fract(uTime * 0.3 + uPulse);
              float pulse = exp(-pow((vAlong - travel) * 14.0, 2.0)) * step(0.999, uDraw);
              float body = mix(0.95, 0.55, vAlong);
              float alpha = (body + tip * 1.4 + pulse) * drawn * edge * uOpacity;
              gl_FragColor = vec4(uColor * (1.0 + tip * 0.5 + pulse * 0.6), alpha);
            }
          `,
          transparent: true,
          depthWrite: false,
          blending: THREE.AdditiveBlending,
          side: THREE.DoubleSide
        })
      );
    const nodeGeometry = track(new THREE.CircleGeometry(1, 32));
    const ringGeometry = track(new THREE.RingGeometry(0.72, 1, 40));
    const nodeMaterial = () =>
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
    // Screen fractions (left, top, right, bottom) → coordinates on the screen.
    const screenX = (fraction: number) => (fraction - 0.5) * screenWidth;
    const screenY = (fraction: number) => (0.5 - fraction) * screenHeight;
    const makeDiagram = (images: HTMLImageElement[], targets: readonly CalloutTarget[]) => {
      const group = new THREE.Group();
      group.visible = false;
      group.position.z = DIAGRAM.plane;
      const outlines = new THREE.Group();
      outlines.position.z = 0.0016;
      outlines.visible = false;
      const callouts = targets.map((target, index) => {
        const [left, top, right, bottom] = target.rect;
        const x0 = screenX(left);
        const x1 = screenX(right);
        const y0 = screenY(top);
        const y1 = screenY(bottom);
        const middle = (y0 + y1) / 2;
        // The outline starts at the element's left middle, where the leader leaves.
        const outlineMaterial = leaderMaterial(index * 0.33);
        const outline = new THREE.Mesh(
          track(
            ribbonGeometry(
              roundedPath(
                [
                  [x0, middle],
                  [x0, y0],
                  [x1, y0],
                  [x1, y1],
                  [x0, y1]
                ],
                0.035,
                true
              ),
              DIAGRAM.outline
            )
          ),
          outlineMaterial
        );
        outlines.add(outline);

        const leaderMaterialInstance = leaderMaterial(index * 0.33);
        const leader = new THREE.Mesh(
          track(
            ribbonGeometry(
              roundedPath(
                [
                  [DIAGRAM.phoneEdge, middle],
                  [target.elbow, middle],
                  [target.elbow, target.row],
                  [DIAGRAM.cardEdge, target.row]
                ],
                DIAGRAM.radius
              ),
              DIAGRAM.line
            )
          ),
          leaderMaterialInstance
        );
        group.add(leader);

        const startMaterial = nodeMaterial();
        const start = new THREE.Mesh(nodeGeometry, startMaterial);
        start.position.set(DIAGRAM.phoneEdge, middle, 0.001);
        start.scale.setScalar(0.019);
        const halo = new THREE.Mesh(ringGeometry, startMaterial);
        halo.position.copy(start.position);
        halo.scale.setScalar(0.038);
        const endMaterial = nodeMaterial();
        const end = new THREE.Mesh(nodeGeometry, endMaterial);
        end.position.set(DIAGRAM.cardEdge, target.row, 0.001);
        end.scale.setScalar(0.018);
        group.add(start, halo, end);

        const image = images[index];
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
        // edge (85% of the plane) meets the leader.
        const x = DIAGRAM.cardEdge - (cardWidth * 0.85) / 2;
        card.position.set(x, target.row, 0.002);
        card.renderOrder = 3;
        group.add(card);
        return { outlineMaterial, leaderMaterial: leaderMaterialInstance, startMaterial, endMaterial, halo, card, material, x };
      });
      phone.add(group);
      screen.add(outlines);
      return { group, outlines, callouts };
    };
    const trainTree = makeDiagram(trainCallouts, CALLOUT_TARGETS.train);
    const fuelTree = makeDiagram(fuelCallouts, CALLOUT_TARGETS.fuel);

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
    syncBadge.renderOrder = 4;
    // A soft light behind the badge that breathes while both screens are in sync.
    const syncHalo = new THREE.Sprite(
      track(
        new THREE.SpriteMaterial({
          map: track(radialTexture("rgba(0,203,255,0.7)", "rgba(0,203,255,0)")),
          transparent: true,
          opacity: 0,
          depthWrite: false,
          blending: THREE.AdditiveBlending,
          toneMapped: false
        })
      )
    );
    syncHalo.scale.set(2.1, 0.9, 1);
    syncHalo.renderOrder = 3;
    syncHalo.visible = false;
    scene.add(syncBadge, syncHalo);

    // Laptop -----------------------------------------------------------------------------
    const laptop = new THREE.Group();
    const shell = track(
      new THREE.MeshPhysicalMaterial({
        color: COLORS.shell,
        metalness: 1,
        roughness: 0.34,
        envMapIntensity: 0.55
      })
    );
    // The screen is what people read: sample it with the GPU's best filtering.
    const sharpest = renderer.capabilities.getMaxAnisotropy();
    const deckWidth = 3.3;
    const deckDepth = 2.25;
    const deckThickness = 0.09;
    const deckMaterial = track(shell.clone());
    deckMaterial.envMapIntensity = 0.4;
    deckMaterial.roughness = 0.4;
    const deck = new THREE.Mesh(track(slabGeometry(deckWidth, deckDepth, 0.13, deckThickness, 0.028)), deckMaterial);
    deck.rotation.x = -Math.PI / 2;
    const top = deckThickness / 2;
    const well = new THREE.Mesh(
      track(panelGeometry(KEYBOARD.width, KEYBOARD.depth, 0.05)),
      track(new THREE.MeshStandardMaterial({ color: 0x07080a, roughness: 0.85, metalness: 0.3 }))
    );
    well.rotation.x = -Math.PI / 2;
    well.position.set(0, top + 0.0008, KEYBOARD.centerZ);

    const caps = layoutKeys();
    const keyHeight = 0.022;
    // Matte dark caps: horizontal parts are seen at a grazing angle, where the
    // studio environment would mirror white, so they barely reflect it. Each
    // key size gets its own rounded geometry, so the bevels stay round instead
    // of being stretched with the key.
    const keyMaterial = track(
      new THREE.MeshStandardMaterial({ color: 0x0b0c0e, roughness: 0.58, metalness: 0.08, envMapIntensity: 0.08 })
    );
    const keys = new THREE.Group();
    {
      const bySize = new Map<string, KeyCap[]>();
      for (const cap of caps) {
        const size = `${cap.width.toFixed(4)}x${cap.depth.toFixed(4)}`;
        bySize.set(size, [...(bySize.get(size) ?? []), cap]);
      }
      const placement = new THREE.Object3D();
      for (const group of bySize.values()) {
        const { width, depth } = group[0];
        const radius = Math.min(0.011, keyHeight * 0.48, width * 0.2, depth * 0.2);
        const mesh = new THREE.InstancedMesh(
          track(new RoundedBoxGeometry(width - 0.004, keyHeight, depth - 0.004, 3, radius)),
          keyMaterial,
          group.length
        );
        group.forEach((cap, index) => {
          placement.position.set(cap.x, top + 0.012, cap.z);
          placement.updateMatrix();
          mesh.setMatrixAt(index, placement.matrix);
        });
        mesh.instanceMatrix.needsUpdate = true;
        mesh.computeBoundingSphere();
        keys.add(mesh);
      }
    }
    // Backlight: cyan light under the caps, seen in the gaps between them.
    const backlight = new THREE.Mesh(
      track(new THREE.PlaneGeometry(KEYBOARD.width * 0.98, KEYBOARD.depth * 0.96)),
      track(
        new THREE.MeshBasicMaterial({
          map: track(radialTexture("rgba(0,203,255,0.6)", "rgba(0,120,220,0.14)")),
          transparent: true,
          depthWrite: false,
          blending: THREE.AdditiveBlending,
          toneMapped: false
        })
      )
    );
    backlight.rotation.x = -Math.PI / 2;
    backlight.position.set(0, top + 0.0016, KEYBOARD.centerZ);
    // Speaker grilles on both sides of the keyboard.
    const grilleTexture = track(speakerGrilleTexture(anisotropy));
    const grilleMaterial = track(
      new THREE.MeshBasicMaterial({ map: grilleTexture, transparent: true, depthWrite: false, toneMapped: false })
    );
    const grilleGeometry = track(new THREE.PlaneGeometry(0.11, KEYBOARD.depth * 0.94));
    const grilles = [-1, 1].map((side) => {
      const grille = new THREE.Mesh(grilleGeometry, grilleMaterial);
      grille.rotation.x = -Math.PI / 2;
      grille.position.set(side * (KEYBOARD.width / 2 + (deckWidth - KEYBOARD.width) / 4), top + 0.0009, KEYBOARD.centerZ);
      return grille;
    });
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
      track(new THREE.MeshStandardMaterial({ color: COLORS.titanium, roughness: 0.16, metalness: 1, envMapIntensity: 0.8 }))
    );
    trackpadEdge.rotation.x = -Math.PI / 2;
    trackpadEdge.position.set(0, top + 0.0005, 0.6);
    const trackpad = new THREE.Mesh(
      track(panelGeometry(1.3, 0.8, 0.065)),
      track(
        new THREE.MeshPhysicalMaterial({
          color: 0x1a1c20,
          roughness: 0.45,
          metalness: 0.4,
          clearcoat: 0,
          clearcoatRoughness: 0.2,
          envMapIntensity: 0.22
        })
      )
    );
    trackpad.rotation.x = -Math.PI / 2;
    trackpad.position.set(0, top + 0.001, 0.6);
    const scoop = new THREE.Mesh(
      track(panelGeometry(0.52, 0.034, 0.017)),
      track(new THREE.MeshStandardMaterial({ color: 0x141518, roughness: 0.5, metalness: 0.6 }))
    );
    scoop.position.set(0, top - 0.004, deckDepth / 2 + 0.0015);
    const chrome = track(new THREE.MeshStandardMaterial({ color: COLORS.titanium, roughness: 0.16, metalness: 1, envMapIntensity: 0.9 }));
    const hinge = new THREE.Mesh(track(new THREE.CylinderGeometry(0.045, 0.045, 2.7, 24)), chrome);
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
    // Closed (rotation pi/2) the lid's local -z points up: pushing the body
    // back by LID_REST lifts its inner face just above the key tops instead of
    // through them; open, it simply sits behind the hinge.
    const keyTop = 0.012 + keyHeight / 2;
    const LID_REST = keyTop + 0.006 + lidThickness / 2 + 0.0012;
    const lidBody = new THREE.Group();
    lidBody.position.z = -LID_REST;
    lidBody.add(lidShell, lidGlass, display, notch, lidLogo);
    lidPivot.add(lidBody);

    const laptopGlow = new THREE.Mesh(
      track(new THREE.PlaneGeometry(6.4, 3.4)),
      track(
        new THREE.MeshBasicMaterial({
          map: track(radialTexture("rgba(150,225,255,0.3)", "rgba(150,225,255,0)")),
          transparent: true,
          depthWrite: false,
          blending: THREE.AdditiveBlending,
          toneMapped: false
        })
      )
    );
    laptopGlow.rotation.x = -Math.PI / 2;
    laptopGlow.position.y = -0.08;
    laptop.add(laptopGlow, deck, well, backlight, keys, legends, ...grilles, trackpadEdge, trackpad, scoop, hinge, lidPivot);
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

    // Per callout: the screen element is outlined, then the leader draws out to
    // the card's place, then the card slides in. Scrolling back reverses it.
    const showTree = (tree: typeof trainTree, amount: number, elapsed: number) => {
      const visible = amount > 0.005;
      tree.group.visible = visible;
      tree.outlines.visible = visible;
      if (!visible) {
        return;
      }
      tree.callouts.forEach((callout, index) => {
        const t = THREE.MathUtils.clamp(amount * 1.7 - index * 0.22, 0, 1) * 3;
        const outlineDraw = smooth(THREE.MathUtils.clamp(t, 0, 1));
        const leaderDraw = smooth(THREE.MathUtils.clamp(t - 0.7, 0, 1));
        const cardIn = smooth(THREE.MathUtils.clamp(t - 1.6, 0, 1));
        callout.outlineMaterial.uniforms.uDraw.value = outlineDraw;
        callout.outlineMaterial.uniforms.uOpacity.value = 0.85;
        callout.outlineMaterial.uniforms.uTime.value = elapsed;
        callout.leaderMaterial.uniforms.uDraw.value = leaderDraw;
        callout.leaderMaterial.uniforms.uTime.value = elapsed;
        callout.startMaterial.opacity = smooth(THREE.MathUtils.clamp(leaderDraw * 6, 0, 1));
        callout.halo.scale.setScalar(0.038 + Math.sin(elapsed * 2.2 + index) * 0.005);
        callout.endMaterial.opacity = smooth(THREE.MathUtils.clamp((leaderDraw - 0.94) * 16, 0, 1));
        callout.material.opacity = cardIn;
        callout.card.visible = cardIn > 0.01;
        callout.card.position.x = callout.x + (1 - cardIn) * 0.2;
      });
    };

    const apply = (elapsed: number) => {
      sampleState(currentProgress, state);
      intro.group.position.set(state.heroX * halfWidth, state.heroY, state.heroZ);
      intro.group.scale.setScalar(state.heroScale * fit);
      intro.group.rotation.set(heroTilt, heroYaw + state.heroSpin * 0.35, 0);
      intro.group.visible = !HERO_3D_HIDDEN && state.heroY < 4;
      intro.group.updateMatrixWorld(true);
      // The composition's shadows: redrawn every other frame while the opening
      // plays or the story scrolls; the still scene keeps its shadow map.
      shadowFrame += 1;
      const heroMoving = Math.abs(targetProgress - currentProgress) > 1e-4 || introTime < INTRO_END || intro.animating(elapsed);
      // The first frame always draws it: with the hero picture the table is
      // never shown, and a shadow map never drawn darkens the phone's frame.
      renderer.shadowMap.needsUpdate = shadowFrame === 1 || (intro.group.visible && (heroMoving ? shadowFrame % 2 === 0 : shadowFrame === 1));
      // The opening drives the phone's hologram too, so it runs to its end even
      // if the hero has already scrolled away; after that only the idle motion.
      // It also runs until the entrance is over: the entrance shows the phone
      // (hidden until its fall), even when the hero composition is not drawn.
      if (!introSettled || intro.group.visible || intro.animating(elapsed)) {
        intro.update(introTime, elapsed);
        introSettled = introTime >= INTRO_END;
      }

      planet.group.position.set(state.planetX * halfWidth, state.planetY, state.planetZ);
      planet.group.scale.setScalar(state.planetScale * fit);
      planet.group.visible = state.planetY < 6;
      if (planet.group.visible) planet.update(elapsed, state.planetSpin, state.planetScale * fit, pointer);
      starMaterial.opacity = state.stars;
      stars.visible = state.stars > 0.02;

      // At the top of the page the phone stands on the hero's surface (its seat
      // in the composition); as the story scrolls on it eases into its own pose.
      const seated = 1 - smooth(THREE.MathUtils.clamp(currentProgress / 0.08, 0, 1));
      phone.position.set(state.phoneX * halfWidth, state.phoneY + Math.sin(elapsed * 1.1) * 0.03 * (1 - seated), state.phoneZ);
      phone.scale.setScalar(state.phoneScale * fit);
      phonePivot.rotation.set(state.phoneRotX + pointer.y * 0.06, state.phoneRotY + pointer.x * 0.12, state.phoneRotZ);
      if (seated > 0) {
        seatMatrix.multiplyMatrices(intro.group.matrixWorld, intro.seat);
        seatMatrix.decompose(seatPosition, seatQuaternion, seatScale);
        phone.position.lerp(seatPosition, seated);
        phone.scale.lerp(seatScale, seated);
        phonePivot.quaternion.slerp(seatQuaternion, seated);
      }
      phoneShadowMaterial.opacity = 1 - seated;
      // With the hero video, the phone shows only once it has left its seat.
      phone.visible = state.phoneY > -6 && !(HERO_3D_HIDDEN && seated > 0.02);

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

      showTree(trainTree, windowed(currentProgress, 0.25, 0.31, 0.38, 0.42), elapsed);
      showTree(fuelTree, windowed(currentProgress, 0.52, 0.56, 0.63, 0.67), elapsed);

      // The sync badge sits over the pair: between the laptop display and the phone.
      const sync = windowed(currentProgress, 0.79, 0.83, 0.87, 0.9);
      syncBadge.visible = sync > 0.01;
      if (syncBadge.visible) {
        phone.updateMatrixWorld(true);
        laptop.updateMatrixWorld(true);
        screen.localToWorld(badgeFrom.set(0, 0.95, 0.01));
        display.localToWorld(badgeTo.set(displayWidth * 0.2, displayHeight * 0.5, 0.01));
        syncBadge.position.set((badgeFrom.x + badgeTo.x) / 2, Math.max(badgeFrom.y, badgeTo.y) + 0.2, (badgeFrom.z + badgeTo.z) / 2 + 0.3);
        syncBadge.material.opacity = sync;
        syncHalo.position.copy(syncBadge.position).setZ(syncBadge.position.z - 0.05);
        syncHalo.material.opacity = sync * (0.7 + Math.sin(elapsed * 2) * 0.3);
      }
      syncHalo.visible = syncBadge.visible;

      // A slow drift of the camera keeps the scene alive; the pointer adds parallax.
      camera.position.set(pointer.x * 0.15 + Math.sin(elapsed * 0.13) * 0.12, pointer.y * 0.1 + Math.sin(elapsed * 0.17) * 0.06, state.cameraZ);
      camera.lookAt(0, 0, 0);
      // The cyan rim light breathes, and a streak of light crosses the phone's glass now and then.
      rim.intensity = 3.4 * (0.9 + 0.1 * Math.sin(elapsed * 0.5));
      glintMaterial.uniforms.uTime.value = elapsed;
      glintMaterial.uniforms.uOpacity.value = introTime >= INTRO_END ? 1 : 0;
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
      // The opening plays once, from when the story is first on screen; any
      // scroll jumps to its final composition, so it never holds anything up.
      introTime = currentProgress > 0.012 ? Math.max(introTime, INTRO_END) : introTime + delta;
      const settled =
        introTime >= INTRO_END &&
        !intro.animating(timer.getElapsed()) &&
        Math.abs(targetProgress - currentProgress) < 1e-4 &&
        pointer.distanceToSquared(pointerTarget) < 1e-6;
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
    };

    const rect = canvas.getBoundingClientRect();
    resize(rect.width, rect.height);
    // Compile every material up front (compileAsync skips hidden objects), so
    // the diagrams and badge never stall the first frame they appear in.
    for (const object of [phone, laptop, trainTree.group, trainTree.outlines, fuelTree.group, fuelTree.outlines, syncBadge, syncHalo]) {
      object.visible = true;
    }
    // The opening's drone, beam, flashes and holograms, and the planet.
    for (const root of [intro.group, phonePivot, planet.group]) {
      root.traverse((object) => {
        object.visible = true;
      });
    }
    for (const { card } of [...trainTree.callouts, ...fuelTree.callouts]) {
      card.visible = true;
    }
    await renderer.compileAsync(scene, camera);
    apply(0);
    renderer.render(scene, camera);

    return {
      setProgress: (progress, immediate = false) => {
        targetProgress = progress;
        if (immediate) {
          currentProgress = progress;
          // Draw the new state now, even when the loop is idle.
          sinceRender = IDLE_FRAME;
          if (!running) {
            apply(timer.getElapsed());
            renderer.render(scene, camera);
          }
        }
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
