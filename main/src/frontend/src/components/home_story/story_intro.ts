import * as THREE from "three";
import { RoundedBoxGeometry } from "three/addons/geometries/RoundedBoxGeometry.js";

/*
 * The Home hero's opening and its final composition (owner requests,
 * 2026-11-15): a scanning drone flies in from the right, settles over a light
 * stone table and opens a beam; inside it the scene materialises as holograms
 * — the phone with the real RYZE training screen, a pair of dumbbells, a
 * served plate with a napkin, fork and knife, and a RYZE shaker. The beam
 * closes with a flash, the drone leaves to the right and, as it goes, each
 * hologram turns into its solid object in place (a bright line climbs it,
 * solid below, hologram above, with a flash and a shock ring); shadows and
 * contact shade come with it.
 *
 * The final composition is one scene on one table, like a food and fitness
 * photograph: the phone, the protagonist, stands in front on a clear stand,
 * its screen to the visitor; the dumbbells rest behind it on the left; the
 * plate (grilled chicken, broccoli, quinoa, peppers) on the right with steam;
 * the shaker at the back. Light motes drift in the air. Procedural, no model
 * files; repeated small parts are instanced.
 *
 * The phone is the story's own phone (it carries on into the Train chapter):
 * this module gives its parts a hologram twin, clips its real materials during
 * the change and says where it stands (`seat`), nothing else.
 */

type Track = <T extends { dispose: () => void }>(item: T) => T;

export interface IntroPhone {
  pivot: THREE.Object3D; // the phone's turning group
  body: THREE.Mesh; // its outline, for the materialisation's extent
  screen: THREE.Mesh; // its twin shows the RYZE screen as a hologram
  skip: ReadonlySet<THREE.Object3D>; // parts with no twin (glass, sheen, glow)
  screenMap: THREE.Texture;
  size: { width: number; height: number; depth: number }; // in the phone's own units
}

export interface HeroIntro {
  group: THREE.Group; // surface, drone, beam, dumbbell, plate; placed by the scene
  seat: THREE.Matrix4; // where the phone stands, in the group's frame
  // time: seconds into the opening (INTRO_END or more: the final composition).
  update: (time: number, elapsed: number) => void;
}

// When the opening is over, the drone gone (the scene jumps here when the
// visitor scrolls).
export const INTRO_END = 7.3;

const ICE = new THREE.Color(0x9af3ff);
const HOLO = new THREE.Color(0x36d6ff);
const HOT = new THREE.Color(0xf2fdff);

// --- The opening's timeline (seconds) ------------------------------------------------

const T = {
  droneIn: [0, 1.9],
  beamOpen: [2.15, 2.6],
  phone: [2.5, 3.9],
  dumbbell: [2.8, 4.1],
  meal: [3.1, 4.4],
  beamClose: [4.9, 5.4],
  droneOut: [5.35, 7.2],
  solidPhone: [5.5, 6.0],
  solidDumbbell: [5.68, 6.18],
  solidMeal: [5.86, 6.4]
} as const;

const clamp01 = (t: number) => Math.min(Math.max(t, 0), 1);
const smooth = (t: number) => t * t * (3 - 2 * t);
const span = (time: number, [from, to]: readonly [number, number]) => clamp01((time - from) / (to - from));
const easeOutBack = (t: number) => {
  const c = 1.25;
  return 1 + (c + 1) * Math.pow(t - 1, 3) + c * Math.pow(t - 1, 2);
};
const easeInCubic = (t: number) => t * t * t;

// --- Shaders ---------------------------------------------------------------------------

const ghostVertex = /* glsl */ `
  varying vec3 vNormal;
  varying vec3 vViewDir;
  varying float vWorldY;
  varying vec2 vUv;

  void main() {
    vUv = uv;
    vec4 local = vec4(position, 1.0);
    vec3 objectNormal = normal;
    #ifdef USE_INSTANCING
      local = instanceMatrix * local;
      objectNormal = mat3(instanceMatrix) * objectNormal;
    #endif
    vec4 world = modelMatrix * local;
    vWorldY = world.y;
    vec4 mvPosition = viewMatrix * world;
    vViewDir = -mvPosition.xyz;
    vNormal = normalize(normalMatrix * objectNormal);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

// The hologram: drawn up to uReveal (it builds from the bottom) and down to
// uWipe (below it the solid object has taken over), with a bright line at
// both edges, a fresnel rim, scan lines and a faint flicker.
const ghostChunk = /* glsl */ `
  uniform vec3 uColor;
  uniform vec3 uHot;
  uniform float uTime;
  uniform float uReveal;
  uniform float uWipe;
  uniform float uOpacity;
  uniform float uFlash;
  varying float vWorldY;

  float edgeAt(float y, float width) {
    return exp(-pow((vWorldY - y) * width, 2.0));
  }

  bool hidden() {
    return vWorldY > uReveal || vWorldY < uWipe;
  }
`;

const ghostFragment = /* glsl */ `
  ${ghostChunk}
  varying vec3 vNormal;
  varying vec3 vViewDir;
  varying vec2 vUv;

  void main() {
    if (hidden()) discard;
    float facing = abs(dot(normalize(vNormal), normalize(vViewDir)));
    float rim = pow(1.0 - facing, 2.0);
    float slices = smoothstep(0.88, 1.0, abs(fract(vWorldY * 34.0 - uTime * 0.5) * 2.0 - 1.0));
    float build = edgeAt(uReveal, 28.0);
    float turn = edgeAt(uWipe, 34.0);
    float flicker = 0.93 + 0.07 * sin(uTime * 31.0 + vWorldY * 9.0);
    vec3 color = mix(uColor, uHot, clamp(build + turn + uFlash * 0.6, 0.0, 1.0));
    float light = 0.16 + rim * 1.3 + slices * 0.3 + build * 2.0 + turn * 3.0 + uFlash;
    float alpha = clamp((0.12 + rim * 0.78 + slices * 0.16 + build + turn) * flicker, 0.0, 1.0) * uOpacity;
    gl_FragColor = vec4(color * light, alpha);
    #include <colorspace_fragment>
  }
`;

// The phone's screen as a hologram: the real RYZE screen in light and cyan.
const ghostScreenFragment = /* glsl */ `
  ${ghostChunk}
  uniform sampler2D uMap;
  varying vec2 vUv;

  void main() {
    if (hidden()) discard;
    vec3 screen = texture2D(uMap, vUv).rgb;
    float lum = dot(screen, vec3(0.299, 0.587, 0.114));
    float lines = 0.85 + 0.15 * sin(vWorldY * 420.0 - uTime * 6.0);
    float build = edgeAt(uReveal, 28.0);
    float turn = edgeAt(uWipe, 34.0);
    vec3 color = mix(uColor * (0.35 + lum * 2.4), uHot, clamp(build + turn, 0.0, 1.0));
    float alpha = clamp((0.22 + lum * 1.4) * lines + build + turn, 0.0, 1.0) * uOpacity;
    gl_FragColor = vec4(color, alpha);
    #include <colorspace_fragment>
  }
`;

const lineVertex = /* glsl */ `
  varying float vWorldY;

  void main() {
    vec4 world = modelMatrix * vec4(position, 1.0);
    vWorldY = world.y;
    gl_Position = projectionMatrix * viewMatrix * world;
  }
`;

const lineFragment = /* glsl */ `
  ${ghostChunk}
  uniform float uLineOpacity;

  void main() {
    if (hidden()) discard;
    float build = edgeAt(uReveal, 28.0);
    vec3 color = mix(uColor, uHot, clamp(build + uFlash * 0.5, 0.0, 1.0));
    gl_FragColor = vec4(color * 1.4, (uLineOpacity + build) * uOpacity);
    #include <colorspace_fragment>
  }
`;

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

// The beam (uv.y: 1 at the lens, 0 at the floor) opens downwards (uExtend)
// and closes upwards (uRetract); its leading edge is bright.
const beamFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform float uTime;
  uniform float uOpacity;
  uniform float uExtend;
  uniform float uRetract;
  uniform float uPulse;
  varying vec2 vUv;
  varying vec3 vNormal;
  varying vec3 vViewDir;

  void main() {
    float lower = 1.0 - uExtend;
    if (vUv.y < lower || vUv.y < uRetract) discard;
    float edge = pow(1.0 - abs(dot(normalize(vNormal), normalize(vViewDir))), 1.5);
    float strength = mix(0.15, 1.0, pow(vUv.y, 0.8));
    float angle = vUv.x * 6.28318;
    float streaks = 0.5 + 0.5 * sin(angle * 16.0 + sin(angle * 3.0 + uTime * 0.6) * 1.8);
    float flow = 0.75 + 0.25 * sin(vUv.y * 12.0 + uTime * 3.0);
    float front = exp(-pow((vUv.y - max(lower, uRetract)) * 18.0, 2.0));
    float alpha = (0.03 + edge * 0.3 + streaks * 0.035 + front * 0.5 + uPulse * 0.12) * strength * flow * uOpacity;
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
    // Falling from the lens: the projection pours down.
    float t = 1.0 - fract(aSeed * 7.13 + uTime * (0.12 + aSeed * 0.12));
    float y = uFloor + t * uHeight;
    float coneRadius = mix(uRadius, uTop, t);
    vec3 p = vec3(position.x * coneRadius, y, position.z * coneRadius);
    vec4 mvPosition = modelViewMatrix * vec4(p, 1.0);
    vAlpha = sin(t * 3.14159) * (0.4 + 0.6 * fract(aSeed * 13.7));
    gl_PointSize = (1.4 + fract(aSeed * 31.0) * 2.4) * uPixelRatio * (8.0 / -mvPosition.z);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

const pointFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform float uOpacity;
  varying float vAlpha;

  void main() {
    float d = length(gl_PointCoord - 0.5);
    gl_FragColor = vec4(uColor, smoothstep(0.5, 0.0, d) * vAlpha * uOpacity);
    #include <colorspace_fragment>
  }
`;


// Steam: a soft wisp on a camera-facing quad, rising, swaying and thinning.
const steamVertex = /* glsl */ `
  varying vec2 vUv;

  void main() {
    vUv = uv;
    vec4 mvPosition = modelViewMatrix * vec4(0.0, 0.0, 0.0, 1.0);
    float size = length(modelMatrix[0].xyz);
    mvPosition.xy += position.xy * size;
    gl_Position = projectionMatrix * mvPosition;
  }
`;

const steamFragment = /* glsl */ `
  uniform float uTime;
  uniform float uSeed;
  uniform float uOpacity;
  varying vec2 vUv;

  float hash(vec2 p) {
    return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
  }

  float noise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    return mix(mix(hash(i), hash(i + vec2(1.0, 0.0)), f.x), mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), f.x), f.y);
  }

  float fbm(vec2 p) {
    float value = 0.0;
    float amplitude = 0.5;
    for (int k = 0; k < 4; k++) {
      value += amplitude * noise(p);
      p *= 2.03;
      amplitude *= 0.5;
    }
    return value;
  }

  void main() {
    float t = uTime * 0.32 + uSeed * 10.0;
    float sway = sin(vUv.y * 4.2 - t * 2.0 + uSeed * 6.0) * 0.13 * vUv.y;
    float x = (vUv.x - 0.5 - sway) * 2.0;
    float body = exp(-x * x * (7.0 - 4.5 * vUv.y));
    float wisps = smoothstep(0.38, 0.82, fbm(vec2(vUv.x * 3.2 + uSeed, vUv.y * 2.6 - t)));
    float rise = smoothstep(0.0, 0.22, vUv.y) * (1.0 - smoothstep(0.5, 1.0, vUv.y));
    gl_FragColor = vec4(vec3(0.93, 0.97, 1.0), body * wisps * rise * uOpacity);
    #include <colorspace_fragment>
  }
`;

// Motes: tiny lights drifting slowly in the air around the composition.
const moteVertex = /* glsl */ `
  attribute float aSeed;
  uniform float uTime;
  uniform float uPixelRatio;
  varying float vAlpha;

  void main() {
    vec3 p = position;
    float t = uTime * (0.05 + aSeed * 0.05) + aSeed * 20.0;
    p.x += sin(t * 1.3) * 0.25;
    p.y += sin(t * 0.9 + aSeed * 3.0) * 0.18;
    p.z += cos(t * 1.1) * 0.2;
    vec4 mvPosition = modelViewMatrix * vec4(p, 1.0);
    vAlpha = 0.35 + 0.65 * (0.5 + 0.5 * sin(uTime * (0.6 + aSeed) + aSeed * 30.0));
    gl_PointSize = (1.6 + aSeed * 2.4) * uPixelRatio * (9.0 / -mvPosition.z);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

const moteFragment = /* glsl */ `
  uniform float uOpacity;
  varying float vAlpha;

  void main() {
    float d = length(gl_PointCoord - 0.5);
    gl_FragColor = vec4(vec3(0.86, 0.95, 1.0), smoothstep(0.5, 0.0, d) * vAlpha * uOpacity);
    #include <colorspace_fragment>
  }
`;

// --- Textures ---------------------------------------------------------------------------

const canvasTexture = (width: number, height: number, draw: (context: CanvasRenderingContext2D) => void, srgb = true) => {
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const context = canvas.getContext("2d");
  if (context) draw(context);
  const texture = new THREE.CanvasTexture(canvas);
  if (srgb) texture.colorSpace = THREE.SRGBColorSpace;
  return texture;
};

const glowTexture = () =>
  canvasTexture(128, 128, (context) => {
    const gradient = context.createRadialGradient(64, 64, 0, 64, 64, 64);
    gradient.addColorStop(0, "rgba(255,255,255,1)");
    gradient.addColorStop(0.25, "rgba(160,240,255,0.55)");
    gradient.addColorStop(1, "rgba(0,160,255,0)");
    context.fillStyle = gradient;
    context.fillRect(0, 0, 128, 128);
  });

// A thin bright ring, for the shock wave when a hologram turns solid.
const ringTexture = () =>
  canvasTexture(256, 256, (context) => {
    const gradient = context.createRadialGradient(128, 128, 84, 128, 128, 126);
    gradient.addColorStop(0, "rgba(160,240,255,0)");
    gradient.addColorStop(0.55, "rgba(232,253,255,1)");
    gradient.addColorStop(1, "rgba(160,240,255,0)");
    context.fillStyle = gradient;
    context.fillRect(0, 0, 256, 256);
  });

// A soft contact shadow (normal blending), so the solid objects sit in the scene.
const shadowTexture = () =>
  canvasTexture(128, 128, (context) => {
    const gradient = context.createRadialGradient(64, 64, 0, 64, 64, 64);
    gradient.addColorStop(0, "rgba(2,8,40,0.55)");
    gradient.addColorStop(0.6, "rgba(2,8,40,0.18)");
    gradient.addColorStop(1, "rgba(2,8,40,0)");
    context.fillStyle = gradient;
    context.fillRect(0, 0, 128, 128);
  });

// Diamond knurling for the dumbbell's handle (a bump map).
const knurlTexture = () => {
  const texture = canvasTexture(
    128,
    128,
    (context) => {
      context.fillStyle = "#808080";
      context.fillRect(0, 0, 128, 128);
      context.strokeStyle = "#202020";
      context.lineWidth = 3;
      for (let k = -128; k <= 256; k += 16) {
        context.beginPath();
        context.moveTo(k, 0);
        context.lineTo(k + 128, 128);
        context.moveTo(k, 128);
        context.lineTo(k + 128, 0);
        context.stroke();
      }
    },
    false
  );
  texture.wrapS = THREE.RepeatWrapping;
  texture.wrapT = THREE.RepeatWrapping;
  texture.repeat.set(10, 2);
  return texture;
};

// Rice and grains: a speckled base for the bowl.
const grainTexture = () =>
  canvasTexture(256, 256, (context) => {
    context.fillStyle = "#efe7d6";
    context.fillRect(0, 0, 256, 256);
    let seed = 11;
    const rand = () => {
      seed = (seed * 16807) % 2147483647;
      return seed / 2147483647;
    };
    for (let index = 0; index < 900; index += 1) {
      const x = rand() * 256;
      const y = rand() * 256;
      context.fillStyle = rand() > 0.82 ? "#c9a96b" : rand() > 0.5 ? "#fbf8f1" : "#ddd2bc";
      context.beginPath();
      context.ellipse(x, y, 3.2, 1.4, rand() * Math.PI, 0, Math.PI * 2);
      context.fill();
    }
  });


// Fine noise, as a bump map for the rubber.
const noiseTexture = (size: number, repeat: number) => {
  const texture = canvasTexture(
    size,
    size,
    (context) => {
      const image = context.createImageData(size, size);
      for (let index = 0; index < image.data.length; index += 4) {
        const value = 110 + Math.random() * 60;
        image.data[index] = value;
        image.data[index + 1] = value;
        image.data[index + 2] = value;
        image.data[index + 3] = 255;
      }
      context.putImageData(image, 0, 0);
    },
    false
  );
  texture.wrapS = THREE.RepeatWrapping;
  texture.wrapT = THREE.RepeatWrapping;
  texture.repeat.set(repeat, repeat);
  return texture;
};

// A grilled chicken breast: golden roast, darker edges, crisp grill bars.
const chickenTexture = () =>
  canvasTexture(1024, 512, (context) => {
    const roast = context.createRadialGradient(512, 256, 40, 512, 256, 560);
    roast.addColorStop(0, "#e2a35a");
    roast.addColorStop(0.55, "#c47a32");
    roast.addColorStop(1, "#7e4416");
    context.fillStyle = roast;
    context.fillRect(0, 0, 1024, 512);
    // Fibres.
    context.strokeStyle = "rgba(255,226,170,0.28)";
    context.lineWidth = 3;
    for (let x = -100; x < 1100; x += 22) {
      context.beginPath();
      context.moveTo(x, 0);
      context.bezierCurveTo(x + 60, 170, x - 40, 340, x + 40, 512);
      context.stroke();
    }
    // Grill bars, slightly charred at their edges.
    context.lineCap = "round";
    for (let x = 110; x < 1000; x += 132) {
      context.strokeStyle = "rgba(92,44,12,0.55)";
      context.lineWidth = 46;
      context.beginPath();
      context.moveTo(x + 90, 20);
      context.lineTo(x - 90, 492);
      context.stroke();
      context.strokeStyle = "rgba(48,20,6,0.85)";
      context.lineWidth = 24;
      context.beginPath();
      context.moveTo(x + 90, 20);
      context.lineTo(x - 90, 492);
      context.stroke();
    }
    for (let index = 0; index < 900; index += 1) {
      context.fillStyle = `rgba(70,32,10,${Math.random() * 0.35})`;
      context.fillRect(Math.random() * 1024, Math.random() * 512, 2, 2);
    }
  });

// Linen for the napkin under the plate.
const linenTexture = () => {
  const texture = canvasTexture(256, 256, (context) => {
    context.fillStyle = "#c3cedd";
    context.fillRect(0, 0, 256, 256);
    for (let k = 0; k < 256; k += 3) {
      context.fillStyle = `rgba(255,255,255,${0.08 + Math.random() * 0.1})`;
      context.fillRect(0, k, 256, 1);
      context.fillStyle = `rgba(90,104,130,${0.06 + Math.random() * 0.08})`;
      context.fillRect(k, 0, 1, 256);
    }
  });
  texture.wrapS = THREE.RepeatWrapping;
  texture.wrapT = THREE.RepeatWrapping;
  texture.repeat.set(3, 3);
  return texture;
};

// --- Models -------------------------------------------------------------------------------

// A hexagon with flat top and bottom faces, so a dumbbell head rests on one.
const hexagon = (radius: number) => {
  const shape = new THREE.Shape();
  for (let index = 0; index < 6; index += 1) {
    const angle = (index / 6) * Math.PI * 2;
    const x = Math.cos(angle) * radius;
    const y = Math.sin(angle) * radius;
    if (index === 0) shape.moveTo(x, y);
    else shape.lineTo(x, y);
  }
  shape.closePath();
  return shape;
};

// A hex dumbbell along x, as found in a gym: matte black rubber heads with
// chamfered edges, chrome collars, a knurled chrome handle, the RYZE mark
// printed on each end face.
const buildDumbbell = (track: Track, materials: DumbbellMaterials) => {
  const group = new THREE.Group();
  const handle = new THREE.Mesh(materials.handleGeometry, materials.steel);
  handle.rotation.z = Math.PI / 2;
  group.add(handle);
  const headDepth = 0.32;
  for (const side of [-1, 1]) {
    const inner = new THREE.Mesh(materials.collarGeometry, materials.chrome);
    inner.rotation.z = Math.PI / 2;
    inner.position.x = side * 0.43;
    const head = new THREE.Group();
    head.position.x = side * (0.47 + headDepth / 2 + 0.035);
    head.rotation.y = Math.PI / 2;
    head.add(new THREE.Mesh(materials.headGeometry, materials.rubber));
    const face = side * (headDepth / 2 + 0.031);
    const ring = new THREE.Mesh(materials.capGeometry, materials.chrome);
    ring.position.z = face + side * 0.003;
    const mark = new THREE.Mesh(materials.logoGeometry, materials.logo);
    mark.position.z = face + side * 0.008;
    if (side < 0) mark.rotation.y = Math.PI;
    head.add(ring, mark);
    group.add(inner, head);
  }
  return group;
};

interface DumbbellMaterials {
  steel: THREE.Material;
  chrome: THREE.Material;
  rubber: THREE.Material;
  logo: THREE.Material;
  handleGeometry: THREE.BufferGeometry;
  collarGeometry: THREE.BufferGeometry;
  headGeometry: THREE.BufferGeometry;
  capGeometry: THREE.BufferGeometry;
  logoGeometry: THREE.BufferGeometry;
}

const dumbbellMaterials = (track: Track, logo: THREE.Texture, logoAspect: number): DumbbellMaterials => {
  const knurl = track(knurlTexture());
  const headGeometry = track(
    new THREE.ExtrudeGeometry(hexagon(0.27), { depth: 0.32, bevelEnabled: true, bevelSize: 0.035, bevelThickness: 0.035, bevelSegments: 5 })
  );
  headGeometry.translate(0, 0, -0.16);
  const capGeometry = track(new THREE.RingGeometry(0.105, 0.125, 48));
  return {
    steel: track(new THREE.MeshPhysicalMaterial({ color: 0xd8dee8, metalness: 1, roughness: 0.28, bumpMap: knurl, bumpScale: 1.4 })),
    chrome: track(new THREE.MeshPhysicalMaterial({ color: 0xf4f7fb, metalness: 1, roughness: 0.07, side: THREE.DoubleSide })),
    rubber: track(
      new THREE.MeshPhysicalMaterial({
        color: 0x1b1e25,
        roughness: 0.72,
        metalness: 0,
        bumpMap: track(noiseTexture(128, 4)),
        bumpScale: 0.6,
        clearcoat: 0.15,
        clearcoatRoughness: 0.6
      })
    ),
    logo: track(new THREE.MeshBasicMaterial({ map: logo, transparent: true, depthWrite: false })),
    handleGeometry: track(new THREE.CylinderGeometry(0.068, 0.068, 0.84, 40)),
    collarGeometry: track(new THREE.CylinderGeometry(0.098, 0.098, 0.06, 40)),
    headGeometry,
    capGeometry,
    logoGeometry: track(new THREE.PlaneGeometry(0.17 * logoAspect, 0.17))
  };
};

// Scatters copies of a geometry with a colour each (one draw call).
const scatter = (
  geometry: THREE.BufferGeometry,
  material: THREE.Material,
  items: Array<{ position: THREE.Vector3; rotation?: THREE.Euler; scale?: number; color: THREE.Color }>
) => {
  const mesh = new THREE.InstancedMesh(geometry, material, items.length);
  const matrix = new THREE.Matrix4();
  const quaternion = new THREE.Quaternion();
  items.forEach((item, index) => {
    quaternion.setFromEuler(item.rotation ?? new THREE.Euler());
    matrix.compose(item.position, quaternion, new THREE.Vector3().setScalar(item.scale ?? 1));
    mesh.setMatrixAt(index, matrix);
    mesh.setColorAt(index, item.color);
  });
  mesh.instanceMatrix.needsUpdate = true;
  if (mesh.instanceColor) mesh.instanceColor.needsUpdate = true;
  return mesh;
};

// A served plate, three-quarter view, like a food photograph: a white ceramic
// plate with a grilled chicken breast, broccoli, a mound of quinoa (grain by
// grain) and diced peppers. Bottom at y = 0.
const buildPlate = (track: Track) => {
  const group = new THREE.Group();
  let seed = 5;
  const rand = () => {
    seed = (seed * 16807) % 2147483647;
    return seed / 2147483647;
  };
  const ceramic = track(new THREE.MeshPhysicalMaterial({ color: 0xf6f7fa, roughness: 0.28, clearcoat: 1, clearcoatRoughness: 0.12 }));
  const profile = [
    [0, 0.012],
    [0.36, 0.012],
    [0.38, 0],
    [0.41, 0],
    [0.43, 0.014],
    [0.52, 0.04],
    [0.6, 0.068],
    [0.648, 0.084],
    [0.646, 0.095],
    [0.6, 0.087],
    [0.5, 0.056],
    [0.42, 0.04],
    [0.3, 0.034],
    [0, 0.034]
  ].map(([r, y]) => new THREE.Vector2(r, y));
  group.add(new THREE.Mesh(track(new THREE.LatheGeometry(profile, 128)), ceramic));

  // Grilled chicken breast: a plump teardrop with a roasted, grilled top.
  const breast = new THREE.Shape();
  breast.moveTo(-0.24, 0);
  breast.bezierCurveTo(-0.24, 0.13, 0.02, 0.15, 0.2, 0.08);
  breast.bezierCurveTo(0.3, 0.04, 0.3, -0.06, 0.2, -0.09);
  breast.bezierCurveTo(0.02, -0.14, -0.24, -0.12, -0.24, 0);
  const breastGeometry = track(
    new THREE.ExtrudeGeometry(breast, { depth: 0.05, bevelEnabled: true, bevelSize: 0.03, bevelThickness: 0.045, bevelSegments: 6, curveSegments: 28 })
  );
  breastGeometry.rotateX(-Math.PI / 2);
  const chickenMap = track(chickenTexture());
  chickenMap.repeat.set(1 / 0.56, 1 / 0.3);
  chickenMap.offset.set(0.43, 0.5);
  const chicken = new THREE.Mesh(
    breastGeometry,
    track(new THREE.MeshPhysicalMaterial({ map: chickenMap, roughness: 0.42, clearcoat: 0.55, clearcoatRoughness: 0.32 }))
  );
  chicken.position.set(-0.17, 0.08, 0.06);
  chicken.rotation.y = 0.5;
  group.add(chicken);

  // Broccoli: stalks and florets of many little buds (one draw call for the buds).
  const stalk = track(new THREE.MeshStandardMaterial({ color: 0x9cc25a, roughness: 0.55 }));
  const stalkGeometry = track(new THREE.CylinderGeometry(0.018, 0.03, 0.1, 10));
  const buds: Array<{ position: THREE.Vector3; scale: number; color: THREE.Color }> = [];
  const florets = [
    [0.1, -0.24],
    [0.2, -0.18],
    [0.02, -0.3],
    [0.24, -0.06]
  ];
  for (const [fx, fz] of florets) {
    const base = new THREE.Mesh(stalkGeometry, stalk);
    base.position.set(fx, 0.085, fz);
    base.rotation.set(0.25, 0, -0.2);
    group.add(base);
    for (let index = 0; index < 34; index += 1) {
      const angle = rand() * Math.PI * 2;
      const radius = Math.sqrt(rand()) * 0.07;
      const height = 0.13 + Math.cos((radius / 0.07) * 1.4) * 0.04;
      buds.push({
        position: new THREE.Vector3(fx + Math.cos(angle) * radius, height, fz + Math.sin(angle) * radius),
        scale: 0.7 + rand() * 0.6,
        color: new THREE.Color().setHSL(0.31 + rand() * 0.03, 0.5, 0.2 + rand() * 0.1)
      });
    }
  }
  group.add(scatter(track(new THREE.SphereGeometry(0.022, 8, 6)), track(new THREE.MeshStandardMaterial({ roughness: 0.85 })), buds));

  // Quinoa, grain by grain, in a low mound.
  const grains: Array<{ position: THREE.Vector3; rotation: THREE.Euler; color: THREE.Color }> = [];
  for (let index = 0; index < 620; index += 1) {
    const angle = rand() * Math.PI * 2;
    const radius = Math.sqrt(rand()) * 0.2;
    const height = 0.038 + (1 - (radius / 0.2) ** 2) * 0.06 + rand() * 0.008;
    grains.push({
      position: new THREE.Vector3(-0.05 + Math.cos(angle) * radius, height, -0.2 + Math.sin(angle) * radius * 0.8),
      rotation: new THREE.Euler(rand() * 3, rand() * 3, rand() * 3),
      color: new THREE.Color().setHSL(0.11 + rand() * 0.03, 0.35 + rand() * 0.2, 0.72 + rand() * 0.14)
    });
  }
  const grainGeometry = track(new THREE.SphereGeometry(0.0085, 6, 4));
  grainGeometry.scale(1, 0.55, 1);
  group.add(scatter(grainGeometry, track(new THREE.MeshStandardMaterial({ roughness: 0.6 })), grains));

  // Diced peppers: red, yellow and orange, glossy.
  const pepperColors = [0xd8261f, 0xf3b51c, 0xef6b1c];
  const peppers: Array<{ position: THREE.Vector3; rotation: THREE.Euler; color: THREE.Color }> = [];
  for (let index = 0; index < 22; index += 1) {
    const angle = 0.4 + rand() * 1.8;
    const radius = 0.24 + rand() * 0.14;
    peppers.push({
      position: new THREE.Vector3(Math.cos(angle) * radius + 0.05, 0.06 + rand() * 0.02, Math.sin(angle) * radius - 0.02),
      rotation: new THREE.Euler(rand() * 0.6, rand() * 3, rand() * 0.6),
      color: new THREE.Color(pepperColors[index % 3])
    });
  }
  group.add(
    scatter(
      track(new RoundedBoxGeometry(0.05, 0.036, 0.05, 2, 0.01)),
      track(new THREE.MeshPhysicalMaterial({ roughness: 0.25, clearcoat: 1, clearcoatRoughness: 0.1 })),
      peppers
    )
  );

  // A few herb flakes on the chicken.
  const flakes: Array<{ position: THREE.Vector3; rotation: THREE.Euler; color: THREE.Color }> = [];
  for (let index = 0; index < 26; index += 1) {
    flakes.push({
      position: new THREE.Vector3(-0.17 + (rand() - 0.5) * 0.36, 0.205 + rand() * 0.012, 0.06 + (rand() - 0.5) * 0.18),
      rotation: new THREE.Euler(-Math.PI / 2 + (rand() - 0.5) * 0.4, 0, rand() * 3),
      color: new THREE.Color().setHSL(0.3, 0.5, 0.18 + rand() * 0.1)
    });
  }
  const flakeGeometry = track(new THREE.CircleGeometry(0.009, 5));
  group.add(scatter(flakeGeometry, track(new THREE.MeshStandardMaterial({ roughness: 0.7, side: THREE.DoubleSide })), flakes));
  return group;
};

// A RYZE shaker bottle: smoky body, black lid with a flip cap, the mark and a
// cyan level line.
const buildShaker = (track: Track, logo: THREE.Texture, logoAspect: number) => {
  const group = new THREE.Group();
  const body = [
    [0, 0],
    [0.27, 0],
    [0.3, 0.02],
    [0.31, 0.08],
    [0.33, 1.15],
    [0.335, 1.2],
    [0, 1.2]
  ].map(([r, y]) => new THREE.Vector2(r, y));
  const plastic = track(new THREE.MeshPhysicalMaterial({ color: 0x1a2436, roughness: 0.32, clearcoat: 1, clearcoatRoughness: 0.15, sheen: 0.4, sheenColor: new THREE.Color(0x3a6ab0) }));
  group.add(new THREE.Mesh(track(new THREE.LatheGeometry(body, 64)), plastic));
  const black = track(new THREE.MeshPhysicalMaterial({ color: 0x0c0f15, roughness: 0.45, clearcoat: 0.6 }));
  const lid = new THREE.Mesh(track(new THREE.CylinderGeometry(0.345, 0.34, 0.22, 64)), black);
  lid.position.y = 1.31;
  const dome = new THREE.Mesh(track(new THREE.SphereGeometry(0.3, 48, 16, 0, Math.PI * 2, 0, Math.PI / 2)), black);
  dome.scale.y = 0.45;
  dome.position.y = 1.42;
  const cap = new THREE.Mesh(track(new RoundedBoxGeometry(0.16, 0.08, 0.22, 3, 0.03)), black);
  cap.position.set(0, 1.52, 0.04);
  group.add(lid, dome, cap);
  const band = new THREE.Mesh(track(new THREE.CylinderGeometry(0.333, 0.333, 0.012, 64, 1, true)), track(new THREE.MeshBasicMaterial({ color: 0x50e2fe })));
  band.position.y = 0.95;
  group.add(band);
  // The mark on a curved label facing the visitor.
  const label = new THREE.Mesh(
    track(new THREE.CylinderGeometry(0.334, 0.334, 0.34, 32, 1, true, -0.55, 1.1)),
    track(new THREE.MeshBasicMaterial({ map: logo, transparent: true, depthWrite: false }))
  );
  label.position.y = 0.58;
  group.add(label);
  return group;
};

// A linen napkin, folded flat, under the cutlery.
const buildNapkin = (track: Track) =>
  new THREE.Mesh(
    track(new RoundedBoxGeometry(1.05, 0.014, 0.62, 2, 0.006)),
    track(new THREE.MeshStandardMaterial({ map: track(linenTexture()), roughness: 0.9 }))
  );

// A fork and a knife in brushed steel, lying flat, side by side.
const buildCutlery = (track: Track) => {
  const group = new THREE.Group();
  const steel = track(new THREE.MeshPhysicalMaterial({ color: 0xe6ebf2, metalness: 1, roughness: 0.2, clearcoat: 0.5 }));
  const handleGeometry = track(new RoundedBoxGeometry(0.42, 0.022, 0.05, 3, 0.01));

  const knife = new THREE.Group();
  const knifeHandle = new THREE.Mesh(handleGeometry, steel);
  knifeHandle.position.x = -0.21;
  const blade = new THREE.Shape();
  blade.moveTo(0, -0.018);
  blade.lineTo(0.36, -0.02);
  blade.quadraticCurveTo(0.48, -0.016, 0.48, 0.012);
  blade.quadraticCurveTo(0.4, 0.025, 0.3, 0.025);
  blade.lineTo(0, 0.022);
  blade.closePath();
  const bladeGeometry = track(new THREE.ExtrudeGeometry(blade, { depth: 0.004, bevelEnabled: true, bevelSize: 0.002, bevelThickness: 0.002, bevelSegments: 1 }));
  bladeGeometry.rotateX(-Math.PI / 2);
  const knifeBlade = new THREE.Mesh(bladeGeometry, steel);
  knifeBlade.position.y = -0.006;
  knife.add(knifeHandle, knifeBlade);
  knife.position.z = 0.06;

  const fork = new THREE.Group();
  const forkHandle = new THREE.Mesh(handleGeometry, steel);
  forkHandle.position.x = -0.21;
  const neck = new THREE.Mesh(track(new RoundedBoxGeometry(0.1, 0.014, 0.034, 2, 0.006)), steel);
  neck.position.x = 0.04;
  const base = new THREE.Mesh(track(new RoundedBoxGeometry(0.05, 0.012, 0.072, 2, 0.005)), steel);
  base.position.x = 0.11;
  fork.add(forkHandle, neck, base);
  const tineGeometry = track(new RoundedBoxGeometry(0.15, 0.008, 0.012, 2, 0.003));
  for (let index = 0; index < 4; index += 1) {
    const tine = new THREE.Mesh(tineGeometry, steel);
    tine.position.set(0.2, 0, -0.028 + index * 0.0187);
    fork.add(tine);
  }
  fork.position.z = -0.06;
  group.add(knife, fork);
  return group;
};

// --- Holograms -------------------------------------------------------------------------

interface Ghost {
  uniforms: {
    uColor: { value: THREE.Color };
    uHot: { value: THREE.Color };
    uTime: { value: number };
    uReveal: { value: number };
    uWipe: { value: number };
    uOpacity: { value: number };
    uFlash: { value: number };
    uLineOpacity: { value: number };
  };
  root: THREE.Object3D; // the hologram twin
  plane: THREE.Plane; // clips the solid object to below the change line
}

const FAR = 1e4;

// A hologram twin of a model: the same geometries with hologram materials and
// glowing edges; the model's own materials are clipped to below the change line.
const makeGhost = (
  track: Track,
  model: THREE.Object3D,
  options: { skip?: ReadonlySet<THREE.Object3D>; screen?: THREE.Mesh; screenMap?: THREE.Texture; lineOpacity?: number } = {}
): Ghost => {
  const uniforms = {
    uColor: { value: HOLO.clone() },
    uHot: { value: HOT.clone() },
    uTime: { value: 0 },
    uReveal: { value: -FAR },
    uWipe: { value: -FAR },
    uOpacity: { value: 1 },
    uFlash: { value: 0 },
    uLineOpacity: { value: options.lineOpacity ?? 0.42 }
  };
  const additive = { transparent: true, depthWrite: false, blending: THREE.AdditiveBlending } as const;
  const fill = track(new THREE.ShaderMaterial({ uniforms, vertexShader: ghostVertex, fragmentShader: ghostFragment, ...additive }));
  const lines = track(new THREE.ShaderMaterial({ uniforms, vertexShader: lineVertex, fragmentShader: lineFragment, ...additive }));
  const screenGhost = options.screenMap
    ? track(
        new THREE.ShaderMaterial({
          uniforms: { ...uniforms, uMap: { value: options.screenMap } },
          vertexShader: ghostVertex,
          fragmentShader: ghostScreenFragment,
          ...additive
        })
      )
    : null;
  const plane = new THREE.Plane(new THREE.Vector3(0, -1, 0), -FAR);

  // Clip the model's own materials (once each).
  const clipped = new Set<THREE.Material>();
  model.traverse((object) => {
    const mesh = object as THREE.Mesh;
    if (!mesh.isMesh) return;
    for (const material of Array.isArray(mesh.material) ? mesh.material : [mesh.material]) {
      if (clipped.has(material)) continue;
      clipped.add(material);
      material.clippingPlanes = [plane];
      // Its shadow grows with it.
      material.clipShadows = true;
    }
  });

  // The twin mirrors the model's hierarchy and transforms.
  const twin = (source: THREE.Object3D): THREE.Object3D | null => {
    if (options.skip?.has(source)) return null;
    const mesh = source as THREE.Mesh;
    let copy: THREE.Object3D;
    const instanced = mesh as THREE.InstancedMesh;
    if (instanced.isInstancedMesh) {
      // Repeated small parts: the same instances, no edges.
      const twinInstances = new THREE.InstancedMesh(instanced.geometry, fill, instanced.count);
      twinInstances.instanceMatrix = instanced.instanceMatrix;
      copy = twinInstances;
    } else if (mesh.isMesh) {
      const material = mesh === options.screen && screenGhost ? screenGhost : fill;
      copy = new THREE.Mesh(mesh.geometry, material);
      if (mesh !== options.screen) {
        const edges = new THREE.LineSegments(track(new THREE.EdgesGeometry(mesh.geometry, 28)), lines);
        copy.add(edges);
      }
    } else {
      copy = new THREE.Object3D();
    }
    copy.position.copy(source.position);
    copy.quaternion.copy(source.quaternion);
    copy.scale.copy(source.scale);
    for (const child of source.children) {
      const childCopy = twin(child);
      if (childCopy) copy.add(childCopy);
    }
    return copy;
  };
  const root = new THREE.Object3D();
  for (const child of model.children) {
    const copy = twin(child);
    if (copy) root.add(copy);
  }
  return { uniforms, root, plane };
};

// --- Builder --------------------------------------------------------------------------------

// The composition, in the group's frame (the floor is y = 0), like a food and
// fitness photograph, held close together: the phone at the centre; the plate
// in front on the left on a napkin with a fork and a knife, overlapping the
// phone's corner; a pair of dumbbells on the right; the RYZE shaker behind them.
const LAYOUT = {
  phone: { x: 0.12, z: 0, scale: 1.36, lean: 0.44, yaw: -0.06 },
  // Axes towards the visitor, so the near end face (the RYZE mark) shows.
  dumbbells: [
    { x: 1.1, z: 0.25, yaw: -1.25, scale: 1.08 },
    { x: 1.62, z: -0.05, yaw: -1.25, scale: 1.08 }
  ],
  plate: { x: -1.08, z: 0.8, yaw: 0.1, scale: 1.22 },
  napkin: { x: -1.32, z: 0.86, yaw: 0.35, scale: 1.45 },
  cutlery: { x: -1.95, z: 0.66, yaw: 1.3, scale: 1.05 },
  shaker: { x: 0.95, z: -1.55, yaw: 0.35, scale: 0.95 },
  floor: { width: 7, depth: 4.6 }
} as const;

// Sets an object on the table: its lowest point at y = base.
const rest = (object: THREE.Object3D, box: THREE.Box3, base = 0) => {
  object.updateMatrixWorld(true);
  box.setFromObject(object, true);
  object.position.y += base - box.min.y;
};

export const createIntro = (track: Track, pixelRatio: number, phone: IntroPhone, logo: THREE.Texture, logoAspect: number): HeroIntro => {
  const group = new THREE.Group();
  const glow = track(glowTexture());
  const additive = { transparent: true, depthWrite: false, blending: THREE.AdditiveBlending } as const;
  const box = new THREE.Box3();

  // --- The floor: invisible, it only receives the composition's shadows; a
  // soft shade under each object grounds it on the blue --------------------------------
  const shadeMap = track(shadowTexture());
  const floor = new THREE.Mesh(
    track(new THREE.PlaneGeometry(LAYOUT.floor.width, LAYOUT.floor.depth)),
    track(new THREE.ShadowMaterial({ color: 0x00051a, opacity: 0.36, transparent: true, depthWrite: false }))
  );
  floor.rotation.x = -Math.PI / 2;
  floor.position.y = 0.001;
  floor.receiveShadow = true;
  group.add(floor);
  // Soft contact shade where each object meets the floor (it comes with the solid object).
  const contact = (x: number, z: number, width: number, depth: number, yaw: number) => {
    const material = track(new THREE.MeshBasicMaterial({ map: shadeMap, transparent: true, depthWrite: false, opacity: 0, toneMapped: false }));
    const shade = new THREE.Mesh(track(new THREE.PlaneGeometry(1, 1)), material);
    shade.rotation.set(-Math.PI / 2, 0, yaw);
    shade.position.set(x, 0.003, z);
    shade.scale.set(width, depth, 1);
    group.add(shade);
    return material;
  };

  // --- Phone: stands on its bottom edge, leaning back on a clear stand -------------------
  const { scale: phoneScale, lean, yaw } = LAYOUT.phone;
  const halfHeight = phone.size.height / 2;
  const halfDepth = phone.size.depth / 2;
  const seatY = phoneScale * (halfHeight * Math.cos(lean) + halfDepth * Math.sin(lean));
  const seatRotation = new THREE.Quaternion().setFromEuler(new THREE.Euler(-lean, yaw, 0, "YXZ"));
  const seat = new THREE.Matrix4().compose(
    new THREE.Vector3(LAYOUT.phone.x, seatY, LAYOUT.phone.z),
    seatRotation,
    new THREE.Vector3(phoneScale, phoneScale, phoneScale)
  );
  const onPhone = (x: number, y: number, z: number) => new THREE.Vector3(x, y, z).applyMatrix4(seat);
  const acrylic = track(
    new THREE.MeshPhysicalMaterial({ color: 0xd2efff, roughness: 0.05, clearcoat: 1, transparent: true, opacity: 0.22, depthWrite: false, envMapIntensity: 1.4 })
  );
  const lipAt = onPhone(0, -halfHeight, halfDepth + 0.018);
  const lip = new THREE.Mesh(track(new RoundedBoxGeometry(phone.size.width * phoneScale * 0.62, 0.05, 0.03, 2, 0.008)), acrylic);
  lip.position.set(lipAt.x, 0.025, lipAt.z);
  lip.rotation.y = yaw;
  const plateAt = onPhone(0, -halfHeight, -halfDepth - 0.2);
  const standBase = new THREE.Mesh(track(new RoundedBoxGeometry(phone.size.width * phoneScale * 0.62, 0.016, 0.62, 2, 0.006)), acrylic);
  standBase.position.set((lipAt.x + plateAt.x) / 2, 0.008, (lipAt.z + plateAt.z) / 2 - 0.08);
  standBase.rotation.y = yaw;
  const strutTop = onPhone(0, -halfHeight + phone.size.height * 0.36, -halfDepth - 0.006);
  const strutFoot = new THREE.Vector3(standBase.position.x, 0.016, standBase.position.z - 0.24);
  const strut = new THREE.Mesh(track(new RoundedBoxGeometry(phone.size.width * phoneScale * 0.4, strutTop.distanceTo(strutFoot), 0.014, 2, 0.006)), acrylic);
  strut.position.copy(strutTop).add(strutFoot).multiplyScalar(0.5);
  strut.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), strutTop.clone().sub(strutFoot).normalize());
  group.add(lip, standBase, strut);

  // --- Dumbbells, plate, napkin, cutlery, shaker ----------------------------------------------
  const sharedDumbbell = dumbbellMaterials(track, logo, logoAspect);
  const dumbbells = new THREE.Group();
  for (const spec of LAYOUT.dumbbells) {
    const dumbbell = buildDumbbell(track, sharedDumbbell);
    dumbbell.position.set(spec.x, 0, spec.z);
    dumbbell.rotation.y = spec.yaw;
    dumbbell.scale.setScalar(spec.scale);
    dumbbells.add(dumbbell);
    rest(dumbbell, box);
  }
  group.add(dumbbells);
  const napkin = buildNapkin(track);
  napkin.position.set(LAYOUT.napkin.x, 0, LAYOUT.napkin.z);
  napkin.rotation.y = LAYOUT.napkin.yaw;
  napkin.scale.set(LAYOUT.napkin.scale, 1, LAYOUT.napkin.scale);
  group.add(napkin);
  rest(napkin, box);
  // The plate sits on the napkin.
  const plate = buildPlate(track);
  plate.position.set(LAYOUT.plate.x, 0, LAYOUT.plate.z);
  plate.rotation.y = LAYOUT.plate.yaw;
  plate.scale.setScalar(LAYOUT.plate.scale);
  group.add(plate);
  rest(plate, box, 0.014);
  const cutlery = buildCutlery(track);
  cutlery.position.set(LAYOUT.cutlery.x, 0, LAYOUT.cutlery.z);
  cutlery.rotation.y = LAYOUT.cutlery.yaw;
  cutlery.scale.setScalar(LAYOUT.cutlery.scale);
  group.add(cutlery);
  rest(cutlery, box, 0.014);
  const shaker = buildShaker(track, logo, logoAspect);
  shaker.position.set(LAYOUT.shaker.x, 0, LAYOUT.shaker.z);
  shaker.rotation.y = LAYOUT.shaker.yaw;
  shaker.scale.setScalar(LAYOUT.shaker.scale);
  group.add(shaker);
  rest(shaker, box);

  // Shadows: every solid, not-tiny part casts one.
  const scaleProbe = new THREE.Vector3();
  const castsShadow = (object: THREE.Object3D) => {
    object.traverse((child) => {
      const mesh = child as THREE.Mesh;
      if (!mesh.isMesh || (mesh as THREE.InstancedMesh).isInstancedMesh) return;
      const materials = Array.isArray(mesh.material) ? mesh.material : [mesh.material];
      if (!mesh.geometry.boundingSphere) mesh.geometry.computeBoundingSphere();
      const radius = (mesh.geometry.boundingSphere?.radius ?? 0) * mesh.getWorldScale(scaleProbe).x;
      mesh.castShadow = radius > 0.05 && materials.every((material) => !material.transparent);
    });
  };
  for (const object of [phone.pivot, dumbbells, plate, napkin, cutlery, shaker]) castsShadow(object);

  // Hologram twins (the stand appears with the phone).
  const ghosts = {
    phone: makeGhost(track, phone.pivot, { skip: phone.skip, screen: phone.screen, screenMap: phone.screenMap, lineOpacity: 0.35 }),
    dumbbells: makeGhost(track, dumbbells),
    plate: makeGhost(track, plate, { lineOpacity: 0.28 }),
    napkin: makeGhost(track, napkin, { lineOpacity: 0.3 }),
    cutlery: makeGhost(track, cutlery, { lineOpacity: 0.5 }),
    shaker: makeGhost(track, shaker, { lineOpacity: 0.4 })
  };
  phone.pivot.add(ghosts.phone.root);
  dumbbells.add(ghosts.dumbbells.root);
  plate.add(ghosts.plate.root);
  napkin.add(ghosts.napkin.root);
  cutlery.add(ghosts.cutlery.root);
  shaker.add(ghosts.shaker.root);
  acrylic.clippingPlanes = [ghosts.phone.plane];

  // Steam rising from the warm plate (beside its hierarchy: no hologram twin).
  const steamGroup = new THREE.Group();
  steamGroup.position.copy(plate.position);
  steamGroup.rotation.copy(plate.rotation);
  steamGroup.scale.copy(plate.scale);
  group.add(steamGroup);
  const steamGeometry = track(new THREE.PlaneGeometry(1, 1));
  steamGeometry.translate(0, 0.5, 0);
  const steam = [0, 1, 2, 3].map((index) => {
    const material = track(
      new THREE.ShaderMaterial({
        uniforms: { uTime: { value: 0 }, uSeed: { value: index * 0.37 + 0.1 }, uOpacity: { value: 0 } },
        vertexShader: steamVertex,
        fragmentShader: steamFragment,
        transparent: true,
        depthWrite: false
      })
    );
    const wisp = new THREE.Mesh(steamGeometry, material);
    wisp.position.set(-0.22 + index * 0.1, 0.16, -0.02 + (index % 2) * 0.06);
    wisp.scale.setScalar(0.7 + (index % 3) * 0.1);
    wisp.frustumCulled = false;
    wisp.renderOrder = 3;
    steamGroup.add(wisp);
    return material;
  });

  // Contact shade under each object.
  const shades = {
    phone: contact(lipAt.x, lipAt.z - 0.12, phone.size.width * phoneScale * 1.25, 0.55, yaw),
    dumbbells: [
      contact(LAYOUT.dumbbells[0].x, LAYOUT.dumbbells[0].z, 2.1, 0.8, LAYOUT.dumbbells[0].yaw),
      contact(LAYOUT.dumbbells[1].x, LAYOUT.dumbbells[1].z, 2.1, 0.8, LAYOUT.dumbbells[1].yaw)
    ],
    plate: contact(plate.position.x, plate.position.z, 1.9, 1.85, 0),
    napkin: contact(napkin.position.x, napkin.position.z, 1.25 * LAYOUT.napkin.scale, 0.8 * LAYOUT.napkin.scale, LAYOUT.napkin.yaw),
    shaker: contact(shaker.position.x, shaker.position.z, 1.0, 1.0, 0)
  };

  // Discreet motes drifting in the light around the composition.
  const MOTES = 46;
  const motePositions = new Float32Array(MOTES * 3);
  const moteSeeds = new Float32Array(MOTES);
  for (let index = 0; index < MOTES; index += 1) {
    motePositions[index * 3] = (Math.random() - 0.5) * 5.2;
    motePositions[index * 3 + 1] = 0.3 + Math.random() * 3.2;
    motePositions[index * 3 + 2] = (Math.random() - 0.5) * 2.6;
    moteSeeds[index] = Math.random();
  }
  const moteGeometry = track(new THREE.BufferGeometry());
  moteGeometry.setAttribute("position", new THREE.BufferAttribute(motePositions, 3));
  moteGeometry.setAttribute("aSeed", new THREE.BufferAttribute(moteSeeds, 1));
  const moteMaterial = track(
    new THREE.ShaderMaterial({
      uniforms: { uTime: { value: 0 }, uPixelRatio: { value: pixelRatio }, uOpacity: { value: 0 } },
      vertexShader: moteVertex,
      fragmentShader: moteFragment,
      ...additive
    })
  );
  const motes = new THREE.Points(moteGeometry, moteMaterial);
  motes.frustumCulled = false;
  group.add(motes);

  // --- Drone (the scanner) ---------------------------------------------------------------
  const drone = new THREE.Group();
  drone.scale.setScalar(1.15);
  group.add(drone);
  const graphite = track(new THREE.MeshPhysicalMaterial({ color: 0x151c2e, metalness: 0.85, roughness: 0.34, clearcoat: 1, clearcoatRoughness: 0.22 }));
  const trim = track(new THREE.MeshStandardMaterial({ color: 0x3a4a6e, metalness: 0.92, roughness: 0.24 }));
  const glass = track(new THREE.MeshPhysicalMaterial({ color: 0x040914, metalness: 0.3, roughness: 0.06, clearcoat: 1 }));
  const lightMaterial = track(new THREE.MeshBasicMaterial({ color: 0x7cebff, transparent: true, toneMapped: false }));
  const dimLight = track(new THREE.MeshBasicMaterial({ color: 0x7cebff, transparent: true, opacity: 0.45, toneMapped: false }));
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
  drone.add(waist);
  const crown = new THREE.Mesh(track(new THREE.TorusGeometry(0.15, 0.0035, 6, 72)), dimLight);
  crown.rotation.x = Math.PI / 2;
  crown.position.y = 0.1;
  drone.add(crown);
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
  const lensGlow = new THREE.Sprite(track(new THREE.SpriteMaterial({ map: glow, color: 0x9af3ff, ...additive, toneMapped: false })));
  lensGlow.scale.setScalar(0.5);
  lensGlow.position.y = -0.07;
  drone.add(lensGlow);
  const armGeometry = track(new THREE.BoxGeometry(0.34, 0.02, 0.046));
  const ductGeometry = track(new THREE.TorusGeometry(0.12, 0.02, 14, 64));
  const ductLightGeometry = track(new THREE.TorusGeometry(0.1, 0.0032, 6, 64));
  const hubGeometry = track(new THREE.CylinderGeometry(0.022, 0.026, 0.03, 16));
  const bladeGeometry = track(new THREE.BoxGeometry(0.19, 0.003, 0.018));
  const blurMaterial = track(new THREE.MeshBasicMaterial({ map: glow, color: 0xbff6ff, opacity: 0.12, ...additive, toneMapped: false }));
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

  // --- Beam, pulses and particles ------------------------------------------------------------
  // Hovering over the phone, clear of the navigation bar.
  const HOVER = new THREE.Vector3(-0.05, 3.42, 0.05);
  const ENTRY = new THREE.Vector3(7.4, 3.9, -0.8);
  const EXIT = new THREE.Vector3(8.4, 4.5, -1.3);
  const LENS_DROP = 0.08;
  const FLOOR = 0;
  const beamTop = HOVER.y - LENS_DROP;
  const beamHeight = beamTop - FLOOR;
  const TOP_RADIUS = 0.3;
  const BOTTOM_RADIUS = 2.3;
  const beamMaterial = track(
    new THREE.ShaderMaterial({
      uniforms: {
        uColor: { value: ICE.clone() },
        uTime: { value: 0 },
        uOpacity: { value: 1 },
        uExtend: { value: 0 },
        uRetract: { value: 0 },
        uPulse: { value: 0 }
      },
      vertexShader: beamVertex,
      fragmentShader: beamFragment,
      side: THREE.DoubleSide,
      ...additive
    })
  );
  const beam = new THREE.Mesh(track(new THREE.CylinderGeometry(TOP_RADIUS, BOTTOM_RADIUS, beamHeight, 96, 1, true)), beamMaterial);
  beam.position.set(HOVER.x, (beamTop + FLOOR) / 2, HOVER.z);
  group.add(beam);
  const pool = new THREE.Mesh(
    track(new THREE.CircleGeometry(BOTTOM_RADIUS * 0.95, 64)),
    track(new THREE.MeshBasicMaterial({ map: glow, color: 0x7cebff, opacity: 0, ...additive, toneMapped: false }))
  );
  pool.rotation.x = -Math.PI / 2;
  pool.position.set(HOVER.x, FLOOR + 0.006, HOVER.z);
  group.add(pool);
  const ringGeometry = track(new THREE.RingGeometry(0.985, 1, 128));
  const pulses = [0, 1, 2].map(() => {
    const material = track(new THREE.MeshBasicMaterial({ color: ICE, side: THREE.DoubleSide, opacity: 0, ...additive, toneMapped: false }));
    const ring = new THREE.Mesh(ringGeometry, material);
    ring.rotation.x = -Math.PI / 2;
    group.add(ring);
    return { ring, material };
  });
  const dustCount = 240;
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
  const dustMaterial = track(
    new THREE.ShaderMaterial({
      uniforms: {
        uColor: { value: ICE.clone() },
        uTime: { value: 0 },
        uOpacity: { value: 0 },
        uPixelRatio: { value: pixelRatio },
        uFloor: { value: FLOOR },
        uHeight: { value: beamHeight },
        uRadius: { value: BOTTOM_RADIUS },
        uTop: { value: TOP_RADIUS }
      },
      vertexShader: dustVertex,
      fragmentShader: pointFragment,
      ...additive
    })
  );
  const dust = new THREE.Points(dustGeometry, dustMaterial);
  dust.position.set(HOVER.x, 0, HOVER.z);
  dust.frustumCulled = false;
  group.add(dust);

  const flashSprite = (color: number) => {
    const sprite = new THREE.Sprite(track(new THREE.SpriteMaterial({ map: glow, color, opacity: 0, ...additive, toneMapped: false })));
    group.add(sprite);
    return sprite;
  };
  const ringMap = track(ringTexture());
  const shockSprite = () => {
    const sprite = new THREE.Sprite(track(new THREE.SpriteMaterial({ map: ringMap, color: 0xe8fdff, opacity: 0, ...additive, toneMapped: false })));
    group.add(sprite);
    return sprite;
  };
  const lensFlash = flashSprite(0xe8fdff);


  // --- Animation ----------------------------------------------------------------------------
  const point = new THREE.Vector3();
  const centre = new THREE.Vector3();
  // Everything but the phone stays put in the group: measure once.
  const localBounds = (object: THREE.Object3D) => {
    group.updateMatrixWorld(true);
    return new THREE.Box3().setFromObject(object);
  };
  const objectFlash = () => flashSprite(0xe8fdff);
  const targets = [
    { ghosts: [ghosts.phone], bounds: phone.body as THREE.Object3D, local: null, build: T.phone, solid: T.solidPhone, shade: [shades.phone] },
    { ghosts: [ghosts.dumbbells], bounds: dumbbells as THREE.Object3D, local: localBounds(dumbbells), build: T.dumbbell, solid: T.solidDumbbell, shade: shades.dumbbells },
    {
      ghosts: [ghosts.plate, ghosts.napkin, ghosts.cutlery],
      bounds: plate as THREE.Object3D,
      local: localBounds(plate),
      build: T.meal,
      solid: T.solidMeal,
      shade: [shades.plate, shades.napkin]
    },
    {
      ghosts: [ghosts.shaker],
      bounds: shaker as THREE.Object3D,
      local: localBounds(shaker),
      build: [T.meal[0] + 0.15, T.meal[1] + 0.1] as const,
      solid: [T.solidMeal[0] + 0.12, T.solidMeal[1] + 0.12] as const,
      shade: [shades.shaker]
    }
  ].map((target) => ({ ...target, flash: objectFlash(), shock: shockSprite() }));
  let settled = false;

  const flash = (sprite: THREE.Sprite, at: THREE.Vector3, amount: number, size: number) => {
    sprite.visible = amount > 0.002;
    sprite.position.copy(at);
    sprite.scale.setScalar(size);
    sprite.material.opacity = amount;
  };

  const update = (time: number, elapsed: number) => {
    const done = time >= INTRO_END;
    // Life once the composition is there: steam and motes.
    const warm = smooth(span(time, [T.solidMeal[1], T.solidMeal[1] + 1.2]));
    steam.forEach((material) => {
      material.uniforms.uTime.value = elapsed;
      material.uniforms.uOpacity.value = 0.3 * warm;
    });
    moteMaterial.uniforms.uTime.value = elapsed;
    moteMaterial.uniforms.uOpacity.value = 0.55 * smooth(span(time, [T.droneIn[1], T.droneIn[1] + 1.5]));
    if (done && settled) return;

    // The drone: in from the right with a small overshoot and settle, a slow
    // hover, then away to the right, banking into its motion.
    const inside = time < T.droneOut[1];
    drone.visible = inside;
    if (inside) {
      const arrive = span(time, T.droneIn);
      const leave = easeInCubic(span(time, T.droneOut));
      const approach = easeOutBack(smooth(arrive) * 0.35 + arrive * 0.65);
      point.copy(ENTRY).lerp(HOVER, approach);
      point.lerp(EXIT, leave);
      const settle = Math.exp(-Math.max(0, time - T.droneIn[1]) * 2.5);
      point.y += Math.sin(elapsed * 1.6) * 0.03 + Math.sin(time * 9) * 0.02 * settle * (arrive >= 1 ? 1 : 0);
      drone.position.copy(point);
      const speedIn = arrive < 1 ? (1 - arrive) * 1.2 : 0;
      drone.rotation.set(0.12 * speedIn + 0.25 * leave, Math.sin(elapsed * 0.3) * 0.06, 0.28 * speedIn - 0.35 * leave);
      rotors.forEach((rotor, index) => {
        rotor.rotation.y = elapsed * (index % 2 === 0 ? 42 : -42);
      });
    }

    // The beam opens down, projects with pulses, then closes up into the lens.
    const open = smooth(span(time, T.beamOpen));
    const close = smooth(span(time, T.beamClose));
    const projecting = open > 0 && close < 1;
    beam.visible = projecting;
    dust.visible = projecting;
    pool.visible = projecting;
    beamMaterial.uniforms.uExtend.value = open;
    beamMaterial.uniforms.uRetract.value = close;
    beamMaterial.uniforms.uTime.value = elapsed;
    beamMaterial.uniforms.uPulse.value = projecting ? 0.5 + 0.5 * Math.sin(elapsed * 7) : 0;
    dustMaterial.uniforms.uTime.value = elapsed;
    dustMaterial.uniforms.uOpacity.value = open * (1 - close);
    pool.material.opacity = 0.3 * open * (1 - close);
    pulses.forEach(({ ring, material }, index) => {
      const t = (elapsed * 0.55 + index / 3) % 1;
      const y = beamTop - t * beamHeight;
      const visible = projecting && 1 - t <= open + 0.02 && 1 - t >= close;
      ring.visible = visible;
      ring.position.set(HOVER.x, y, HOVER.z);
      ring.scale.setScalar(TOP_RADIUS + (BOTTOM_RADIUS - TOP_RADIUS) * t);
      material.opacity = visible ? Math.sin(t * Math.PI) * 0.6 : 0;
    });
    lensGlow.material.opacity = inside ? 0.45 + open * (1 - close) * 0.5 : 0;
    aperture.scale.setScalar(1 + open * (1 - close) * 0.05);
    const lensBurst = Math.max(Math.exp(-Math.pow((time - T.beamOpen[0] - 0.05) * 9, 2)), Math.exp(-Math.pow((time - T.beamClose[1]) * 8, 2)));
    drone.updateMatrixWorld(true);
    drone.localToWorld(point.set(0, -0.07, 0));
    group.worldToLocal(point);
    flash(lensFlash, point, inside ? lensBurst * 0.95 : 0, 1.4 + lensBurst * 0.8);

    // Each object: built as a hologram from the bottom up, then turned solid in place.
    group.updateMatrixWorld(true);
    for (const target of targets) {
      const build = span(time, target.build);
      const solid = span(time, target.solid);
      if (target.local) box.copy(target.local).applyMatrix4(group.matrixWorld);
      else box.setFromObject(target.bounds);
      const bottom = box.min.y - 0.03;
      const height = box.max.y - box.min.y + 0.06;
      for (const { uniforms, root, plane } of target.ghosts) {
        uniforms.uTime.value = elapsed;
        uniforms.uReveal.value = done ? FAR : build <= 0 ? -FAR : bottom + smooth(build) * height * 1.02;
        uniforms.uWipe.value = done ? FAR : solid <= 0 ? -FAR : bottom + smooth(solid) * height;
        uniforms.uFlash.value = Math.exp(-Math.pow((time - target.solid[0]) * 6, 2)) * 0.8;
        root.visible = !done && build > 0 && solid < 1;
        plane.constant = done ? FAR : solid <= 0 ? -FAR : bottom + smooth(solid) * height;
      }
      box.getCenter(centre);
      group.worldToLocal(centre);
      const buildFlash = Math.exp(-Math.pow((time - target.build[0] - 0.05) * 8, 2)) * 0.6;
      const solidFlash = Math.exp(-Math.pow((time - target.solid[0] - 0.12) * 5, 2));
      flash(target.flash, centre, done ? 0 : Math.max(buildFlash, solidFlash * 0.9), 1.6 + solidFlash * 1.2);
      const shock = clamp01((time - target.solid[0]) / 0.75);
      flash(target.shock, centre, done || shock <= 0 ? 0 : (1 - shock) * 0.85, 0.6 + shock * 3.4);
      for (const shade of target.shade) shade.opacity = 0.9 * smooth(solid);
    }
    settled = done;
  };

  update(0, 0);
  return { group, seat, update };
};
