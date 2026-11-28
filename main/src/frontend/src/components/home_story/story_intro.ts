import * as THREE from "three";
import { RoundedBoxGeometry } from "three/addons/geometries/RoundedBoxGeometry.js";

import { isAppReady } from "@/services/app_ready";

/*
 * The Home hero's composition (owner's reference picture, 2026-11-28): on the
 * page's blue, with no floor, a RYZE round-plate dumbbell lies diagonally in
 * front on the left, the RYZE shaker (navy body, teal lid) stands tall in the
 * middle, the phone with the real RYZE training screen stands on its right on
 * a clear stand, and a large frosted glass pane behind them shows a glowing
 * cyan progress chart. A soft shade
 * grounds each object. Procedural, no model files.
 *
 * The drone's projection (objects built as holograms that turn solid) is kept
 * in this module but no longer plays: the scene starts at INTRO_END.
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
  group: THREE.Group; // surface, drone, beam, dumbbell, shaker; placed by the scene
  seat: THREE.Matrix4; // where the phone stands, in the group's frame
  // time: seconds into the opening (INTRO_END or more: the final composition).
  update: (time: number, elapsed: number) => void;
  // True while the entrance (objects falling into place, the chart rising)
  // plays, so the scene keeps its full frame rate.
  animating: (elapsed: number) => boolean;
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

// --- Models -------------------------------------------------------------------------------

// A RYZE shaker bottle after the owner's reference (2026-11-28): a deep navy,
// satin body tapering slightly to the base, a teal lid with a flip spout and a
// carry loop, and the RYZE mark on the front. Bottom at y = 0.
const buildShaker = (track: Track, logo: THREE.Texture, logoAspect: number) => {
  const group = new THREE.Group();
  const body = [
    [0, 0],
    [0.27, 0],
    [0.292, 0.012],
    [0.3, 0.05],
    [0.335, 1.24],
    [0.338, 1.27],
    [0, 1.27]
  ].map(([r, y]) => new THREE.Vector2(r, y));
  // After the owner's second reference (2026-11-28): a deep teal-navy
  // metallic body with a soft sheen, and a teal lid with a flip cap.
  const navy = track(
    new THREE.MeshPhysicalMaterial({
      color: 0x0c2335,
      roughness: 0.32,
      metalness: 0.45,
      envMapIntensity: 0.55,
      clearcoat: 1,
      clearcoatRoughness: 0.16,
      sheen: 0.4,
      sheenRoughness: 0.5,
      sheenColor: new THREE.Color(0x5fc4e0)
    })
  );
  group.add(new THREE.Mesh(track(new THREE.LatheGeometry(body, 72)), navy));
  // A darker foot ring.
  const foot = new THREE.Mesh(track(new THREE.TorusGeometry(0.288, 0.014, 10, 72)), track(new THREE.MeshStandardMaterial({ color: 0x08090b, roughness: 0.6 })));
  foot.rotation.x = Math.PI / 2;
  foot.position.y = 0.014;
  group.add(foot);
  // The teal lid: a band, a softly domed top and a flip cap with its spout.
  const teal = track(new THREE.MeshPhysicalMaterial({ color: 0x2b93b3, roughness: 0.28, metalness: 0.35, clearcoat: 1, clearcoatRoughness: 0.12 }));
  const tealDark = track(new THREE.MeshPhysicalMaterial({ color: 0x1f6f8a, roughness: 0.32, metalness: 0.3, clearcoat: 0.8 }));
  const lid = new THREE.Mesh(track(new THREE.CylinderGeometry(0.35, 0.345, 0.24, 72)), teal);
  lid.position.y = 1.39;
  const lip = new THREE.Mesh(track(new THREE.TorusGeometry(0.343, 0.012, 10, 72)), tealDark);
  lip.rotation.x = Math.PI / 2;
  lip.position.y = 1.275;
  const top = new THREE.Mesh(track(new THREE.SphereGeometry(0.33, 56, 16, 0, Math.PI * 2, 0, Math.PI / 2)), teal);
  top.scale.y = 0.24;
  top.position.y = 1.51;
  // The flip cap: a raised round cap, its spout angled up to the right.
  const cap = new THREE.Mesh(track(new THREE.CylinderGeometry(0.2, 0.22, 0.12, 56)), teal);
  cap.position.set(-0.04, 1.6, 0);
  const capTop = new THREE.Mesh(track(new THREE.SphereGeometry(0.2, 48, 12, 0, Math.PI * 2, 0, Math.PI / 2)), teal);
  capTop.scale.y = 0.22;
  capTop.position.set(-0.04, 1.66, 0);
  const spout = new THREE.Mesh(track(new THREE.CylinderGeometry(0.055, 0.075, 0.13, 32)), tealDark);
  spout.position.set(0.15, 1.67, 0.02);
  spout.rotation.z = -0.95;
  group.add(lid, lip, top, cap, capTop, spout);
  // The mark on a curved label facing the visitor.
  const labelHeight = 0.34;
  const labelArc = Math.min(1.25, (labelHeight * logoAspect) / 0.336);
  const label = new THREE.Mesh(
    track(new THREE.CylinderGeometry(0.336, 0.336, labelHeight, 32, 1, true, -labelArc / 2, labelArc)),
    track(new THREE.MeshBasicMaterial({ map: logo, transparent: true, depthWrite: false }))
  );
  label.position.y = 0.66;
  group.add(label);
  return group;
};

// A RYZE round-plate dumbbell after the owner's reference (2026-11-28): on each
// side two big plates and a smaller one in dark metallic navy (raised rim,
// recessed face), a chrome hub, a spin-lock collar and the threaded bar end;
// a knurled steel grip in the middle. Along x, centred.
const buildPlateDumbbell = (track: Track) => {
  const group = new THREE.Group();
  const plateMaterial = track(
    // Black rubber, after the owner's reference (2026-11-28).
    new THREE.MeshPhysicalMaterial({ color: 0x111215, roughness: 0.5, metalness: 0.15, clearcoat: 0.35, clearcoatRoughness: 0.4 })
  );
  const chrome = track(new THREE.MeshStandardMaterial({ color: 0xe3eaf3, metalness: 1, roughness: 0.16 }));
  const knurl = track(knurlTexture());
  const steel = track(new THREE.MeshStandardMaterial({ color: 0xa9b5c7, metalness: 1, roughness: 0.34, bumpMap: knurl, bumpScale: 1.2 }));
  // A plate's half profile (radius, along the axis), revolved around y, then
  // laid along x.
  const plateGeometry = (radius: number, half: number) => {
    const rim = radius * 0.86;
    const profile = [
      [0.05, -half * 0.7],
      [rim - 0.03, -half * 0.7],
      [rim, -half],
      [radius - 0.02, -half],
      [radius, -half * 0.6],
      [radius, half * 0.6],
      [radius - 0.02, half],
      [rim, half],
      [rim - 0.03, half * 0.7],
      [0.05, half * 0.7],
      [0.05, -half * 0.7]
    ].map(([r, y]) => new THREE.Vector2(r, y));
    const geometry = new THREE.LatheGeometry(profile, 72);
    geometry.rotateZ(Math.PI / 2);
    return track(geometry);
  };
  const bigPlate = plateGeometry(0.4, 0.055);
  const smallPlate = plateGeometry(0.3, 0.05);
  const hub = track(new THREE.CylinderGeometry(0.075, 0.075, 0.05, 40).rotateZ(Math.PI / 2));
  const grip = new THREE.Mesh(track(new THREE.CylinderGeometry(0.05, 0.05, 0.62, 40).rotateZ(Math.PI / 2)), steel);
  group.add(grip);
  const collarGeometry = track(new THREE.CylinderGeometry(0.085, 0.085, 0.09, 40).rotateZ(Math.PI / 2));
  const grooveGeometry = track(new THREE.TorusGeometry(0.085, 0.006, 8, 40).rotateY(Math.PI / 2));
  const threadGeometry = track(new THREE.CylinderGeometry(0.045, 0.045, 0.16, 32).rotateZ(Math.PI / 2));
  for (const side of [-1, 1]) {
    let x = side * 0.31;
    // An inner chrome flange against the grip.
    const flange = new THREE.Mesh(track(new THREE.CylinderGeometry(0.09, 0.09, 0.03, 40).rotateZ(Math.PI / 2)), chrome);
    flange.position.x = x + side * 0.015;
    group.add(flange);
    x += side * 0.03;
    for (const [geometry, half] of [
      [bigPlate, 0.055],
      [bigPlate, 0.055],
      [smallPlate, 0.05]
    ] as const) {
      x += side * half;
      const plate = new THREE.Mesh(geometry, plateMaterial);
      plate.position.x = x;
      const cap = new THREE.Mesh(hub, chrome);
      cap.position.x = x;
      group.add(plate, cap);
      x += side * (half + 0.006);
    }
    const collar = new THREE.Mesh(collarGeometry, chrome);
    collar.position.x = x + side * 0.045;
    group.add(collar);
    for (const offset of [-0.025, 0, 0.025]) {
      const groove = new THREE.Mesh(grooveGeometry, steel);
      groove.position.x = collar.position.x + offset;
      group.add(groove);
    }
    const thread = new THREE.Mesh(threadGeometry, steel);
    thread.position.x = collar.position.x + side * 0.12;
    group.add(thread);
  }
  return group;
};

// The progress chart behind the composition (owner's reference, 2026-11-28):
// a large pane of frosted glass with a soft bright edge, faint grid lines and
// a glowing line climbing to an arrow, with a light fill beneath it, in RYZE
// cyan. Centred on its own origin, facing +z.
const buildChartPanel = (track: Track, glow: THREE.Texture) => {
  const group = new THREE.Group();
  const width = 4.6;
  const height = 2.6;
  const radius = 0.26;
  const shape = new THREE.Shape();
  const w = width / 2;
  const h = height / 2;
  shape.moveTo(-w + radius, -h);
  shape.lineTo(w - radius, -h);
  shape.quadraticCurveTo(w, -h, w, -h + radius);
  shape.lineTo(w, h - radius);
  shape.quadraticCurveTo(w, h, w - radius, h);
  shape.lineTo(-w + radius, h);
  shape.quadraticCurveTo(-w, h, -w, h - radius);
  shape.lineTo(-w, -h + radius);
  shape.quadraticCurveTo(-w, -h, -w + radius, -h);
  const additive = { transparent: true, depthWrite: false, blending: THREE.AdditiveBlending, toneMapped: false } as const;

  // After the owner's second reference (2026-11-28): a light frosted pane with
  // a bright edge and a glowing cyan line.
  const pane = new THREE.Mesh(
    track(new THREE.ExtrudeGeometry(shape, { depth: 0.05, bevelEnabled: true, bevelThickness: 0.02, bevelSize: 0.02, bevelSegments: 3, curveSegments: 12 })),
    track(
      new THREE.MeshPhysicalMaterial({
        color: 0xc4e2ff,
        roughness: 0.2,
        metalness: 0,
        clearcoat: 1,
        clearcoatRoughness: 0.08,
        transparent: true,
        opacity: 0.22,
        depthWrite: false,
        side: THREE.DoubleSide
      })
    )
  );
  pane.position.z = -0.035;
  pane.renderOrder = 1;
  group.add(pane);
  const edgePoints = shape.getSpacedPoints(160).map((p) => new THREE.Vector3(p.x, p.y, 0.03));
  const edge = new THREE.Mesh(
    track(new THREE.TubeGeometry(new THREE.CatmullRomCurve3(edgePoints, true), 320, 0.01, 6, true)),
    track(new THREE.MeshBasicMaterial({ color: 0xc8f6ff, transparent: true, opacity: 0.7, toneMapped: false }))
  );
  edge.renderOrder = 2;
  group.add(edge);

  // Faint grid lines.
  const gridPoints: number[] = [];
  for (let row = 1; row <= 3; row += 1) {
    const y = -h + (height * row) / 4;
    gridPoints.push(-w + 0.3, y, 0.02, w - 0.3, y, 0.02);
  }
  const gridGeometry = track(new THREE.BufferGeometry());
  gridGeometry.setAttribute("position", new THREE.Float32BufferAttribute(gridPoints, 3));
  const grid = new THREE.LineSegments(gridGeometry, track(new THREE.LineBasicMaterial({ color: 0xbff3ff, transparent: true, opacity: 0.14, depthWrite: false })));
  grid.renderOrder = 2;
  group.add(grid);

  // The line: rising with dips.
  const values = [0.1, 0.34, 0.24, 0.52, 0.4, 0.78, 0.6, 0.98];
  const left = -w + 0.45;
  const right = w - 0.75;
  const bottom = -h + 0.35;
  const top = h - 0.45;
  const linePoints = values.map((value, index) => new THREE.Vector3(left + ((right - left) * index) / (values.length - 1), bottom + (top - bottom) * value, 0.06));
  const path = new THREE.CurvePath<THREE.Vector3>();
  for (let index = 0; index < linePoints.length - 1; index += 1) path.add(new THREE.LineCurve3(linePoints[index], linePoints[index + 1]));
  const curve = path as unknown as THREE.Curve<THREE.Vector3>;
  const TUBE_SEGMENTS = 240;
  const RADIAL = 10;
  const lineMaterial = track(new THREE.MeshBasicMaterial({ color: 0x9ff7ff, transparent: true, toneMapped: false }));
  const line = new THREE.Mesh(track(new THREE.TubeGeometry(curve, TUBE_SEGMENTS, 0.03, RADIAL, false)), lineMaterial);
  line.renderOrder = 4;
  const haloMaterial = track(new THREE.MeshBasicMaterial({ color: 0x00cbff, opacity: 0.3, ...additive }));
  const halo = new THREE.Mesh(track(new THREE.TubeGeometry(curve, TUBE_SEGMENTS, 0.1, RADIAL, false)), haloMaterial);
  halo.renderOrder = 3;
  group.add(halo, line);
  // Round joints, so the bends read as one line; each appears as the line reaches it.
  const lengths = path.getCurveLengths();
  const total = path.getLength();
  const jointGeometry = track(new THREE.SphereGeometry(0.03, 16, 12));
  const joints = linePoints.slice(1, -1).map((point, index) => {
    const joint = new THREE.Mesh(jointGeometry, lineMaterial);
    joint.position.copy(point);
    joint.renderOrder = 4;
    group.add(joint);
    return { joint, at: lengths[index] / total };
  });
  // The arrow rides the tip of the line as it rises.
  const arrow = new THREE.Mesh(track(new THREE.ConeGeometry(0.1, 0.22, 24)), lineMaterial);
  arrow.renderOrder = 4;
  group.add(arrow);
  const tipGlowMaterial = track(new THREE.SpriteMaterial({ map: glow, color: 0x7cebff, opacity: 0.8, ...additive }));
  const tipGlow = new THREE.Sprite(tipGlowMaterial);
  tipGlow.scale.setScalar(0.8);
  tipGlow.renderOrder = 5;
  group.add(tipGlow);

  // The light fill under the line, fading towards the bottom.
  const fillPositions: number[] = [];
  const fillColors: number[] = [];
  const steps = 96;
  const sample = new THREE.Vector3();
  for (let index = 0; index <= steps; index += 1) {
    path.getPointAt(index / steps, sample);
    fillPositions.push(sample.x, sample.y, 0.04, sample.x, bottom - 0.15, 0.04);
    fillColors.push(0.31, 0.89, 1, 0.3, 0.31, 0.89, 1, 0);
  }
  const fillIndex: number[] = [];
  for (let index = 0; index < steps; index += 1) {
    const a = index * 2;
    fillIndex.push(a, a + 1, a + 2, a + 1, a + 3, a + 2);
  }
  const fillGeometry = track(new THREE.BufferGeometry());
  fillGeometry.setAttribute("position", new THREE.Float32BufferAttribute(fillPositions, 3));
  fillGeometry.setAttribute("color", new THREE.Float32BufferAttribute(fillColors, 4));
  fillGeometry.setIndex(fillIndex);
  const fillMaterial = track(new THREE.MeshBasicMaterial({ vertexColors: true, transparent: true, depthWrite: false, side: THREE.DoubleSide, toneMapped: false }));
  const fill = new THREE.Mesh(fillGeometry, fillMaterial);
  fill.renderOrder = 2;
  group.add(fill);

  // The rise (owner requests, 2026-11-28): the pane fades in, then the line
  // draws itself once from the left with the arrow on its tip and stays.
  const paneMaterials = [pane.material, edge.material, grid.material] as THREE.Material[];
  const paneBase = paneMaterials.map((material) => material.opacity);
  const tip = new THREE.Vector3();
  const tangent = new THREE.Vector3();
  const up = new THREE.Vector3(0, 1, 0);
  // appear: 0..1, the pane's fade; rise: 0..1, how far the line has drawn.
  const update = (appear: number, rise: number) => {
    paneMaterials.forEach((material, index) => {
      material.opacity = paneBase[index] * appear;
    });
    const progress = Math.max(0.002, 1 - Math.pow(1 - rise, 3));
    const shown = rise > 0 ? appear : 0;
    line.geometry.setDrawRange(0, Math.floor(progress * TUBE_SEGMENTS) * RADIAL * 6);
    halo.geometry.setDrawRange(0, Math.floor(progress * TUBE_SEGMENTS) * RADIAL * 6);
    fill.geometry.setDrawRange(0, Math.floor(progress * steps) * 6);
    for (const { joint, at } of joints) joint.visible = shown > 0 && progress >= at;
    path.getPointAt(progress, tip);
    path.getTangentAt(progress, tangent);
    arrow.position.copy(tip).addScaledVector(tangent, 0.09);
    arrow.quaternion.setFromUnitVectors(up, tangent);
    arrow.visible = shown > 0;
    tipGlow.position.copy(tip);
    lineMaterial.opacity = shown;
    haloMaterial.opacity = 0.3 * shown;
    fillMaterial.opacity = shown;
    tipGlowMaterial.opacity = 0.8 * shown;
  };
  update(1, 1);
  return { group, update };
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

// The composition, in the group's frame (the floor is y = 0), after the
// owner's reference picture (2026-11-21): the phone, big, at the centre on its
// clear stand; the RYZE shaker tall behind on the left; the bowl in front of
// the shaker, left of the phone; the dumbbells lying on the right of the phone,
// one behind the other. The napkin and the cutlery sit under the bowl (the
// reference shows the bowl alone). Nothing touches the phone (checked in
// screen space at the hero's camera).
// v2.133.2 (owner's reference picture, 2026-11-28): the dumbbell lies in front
// on the left, diagonally; the RYZE shaker stands tall in the middle; the
// phone stands on its right where the reference has a bottle; the progress
// chart behind them all (the food plate was removed on the owner's request). No floor: the objects
// rest at the bottom of the hero, with a soft shade under each.
const LAYOUT = {
  phone: { x: 1.32, z: 0.85, scale: 1.04, lean: 0.12, yaw: -0.2 },
  dumbbell: { x: -0.95, z: 0.0, yaw: 0.2, scale: 1.32 },
  shaker: { x: 0.18, z: 0.92, yaw: 0.14, scale: 1.36 },
  chart: { x: -0.05, y: 2.25, z: -1.7, tilt: -0.04, yaw: -0.1, scale: 1.08 }
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

  // --- No floor (owner, 2026-11-28): a soft shade under each object grounds
  // it on the blue -------------------------------------------------------------------------
  const shadeMap = track(shadowTexture());
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
  const stand = new THREE.Group();
  stand.add(lip, standBase, strut);
  group.add(stand);

  // --- Dumbbell, shaker, chart -------------------------------------------------------------
  const dumbbells = new THREE.Group();
  const dumbbell = buildPlateDumbbell(track);
  dumbbell.position.set(LAYOUT.dumbbell.x, 0, LAYOUT.dumbbell.z);
  dumbbell.rotation.y = LAYOUT.dumbbell.yaw;
  dumbbell.scale.setScalar(LAYOUT.dumbbell.scale);
  dumbbells.add(dumbbell);
  rest(dumbbell, box);
  group.add(dumbbells);
  const { group: chart, update: updateChart } = buildChartPanel(track, glow);
  chart.position.set(LAYOUT.chart.x, LAYOUT.chart.y, LAYOUT.chart.z);
  chart.rotation.set(LAYOUT.chart.tilt, LAYOUT.chart.yaw, 0);
  chart.scale.setScalar(LAYOUT.chart.scale);
  group.add(chart);
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
  for (const object of [phone.pivot, dumbbells, shaker]) castsShadow(object);

  // Hologram twins (the stand appears with the phone).
  const ghosts = {
    phone: makeGhost(track, phone.pivot, { skip: phone.skip, screen: phone.screen, screenMap: phone.screenMap, lineOpacity: 0.35 }),
    dumbbells: makeGhost(track, dumbbells),
    shaker: makeGhost(track, shaker, { lineOpacity: 0.4 })
  };
  phone.pivot.add(ghosts.phone.root);
  dumbbells.add(ghosts.dumbbells.root);
  shaker.add(ghosts.shaker.root);
  acrylic.clippingPlanes = [ghosts.phone.plane];

  // Contact shade under each object.
  const shades = {
    phone: contact(lipAt.x, lipAt.z - 0.12, phone.size.width * phoneScale * 1.25, 0.55, yaw),
    dumbbells: [contact(LAYOUT.dumbbell.x, LAYOUT.dumbbell.z, 2.3 * LAYOUT.dumbbell.scale, 1.0 * LAYOUT.dumbbell.scale, LAYOUT.dumbbell.yaw)],
    shaker: contact(shaker.position.x, shaker.position.z, 1.2, 1.2, 0)
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
  const BOTTOM_RADIUS = 2.55;
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

  const play = (time: number, elapsed: number) => {
    const done = time >= INTRO_END;
    // Life once the composition is there: drifting motes.
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

  play(0, 0);

  // --- The entrance (owner request, 2026-11-28): nothing simply appears. The
  // pane fades up, then the dumbbell, the shaker and the phone on its stand
  // fall into place one after the other with a small bounce, each shade
  // darkening as it lands; then the chart's line rises once and stays.
  const ENTRANCE = {
    chart: [0.1, 0.9],
    dumbbell: [0.35, 1.2],
    shaker: [0.6, 1.45],
    phone: [0.85, 1.7],
    rise: [1.8, 4]
  } as const;
  const ENTRANCE_END = ENTRANCE.rise[1];
  const FALL = 5;
  // Height above the resting place, 1 at the start: an accelerating fall, then
  // one small bounce.
  const fall = (p: number) => {
    if (p <= 0) return 1;
    if (p < 0.72) return 1 - (p / 0.72) ** 2;
    if (p < 1) {
      const b = (p - 0.72) / 0.28;
      return 0.32 * b * (1 - b);
    }
    return 0;
  };
  const landed = (range: readonly [number, number], t: number) => smooth(span(t, [range[0] + (range[1] - range[0]) * 0.55, range[1]]));
  const restY = { dumbbells: dumbbells.position.y, shaker: shaker.position.y, chart: chart.position.y };
  const seatRest = seat.clone();
  let start: number | null = null;

  const entrance = (elapsed: number) => {
    // It starts once the loading screen has gone (and has faded), so it plays
    // in view; until then everything waits above, out of sight.
    if (start === null && isAppReady()) start = elapsed + 0.35;
    const t = start === null ? 0 : Math.max(0, elapsed - start);
    dumbbells.position.y = restY.dumbbells + FALL * fall(span(t, ENTRANCE.dumbbell));
    shaker.position.y = restY.shaker + FALL * fall(span(t, ENTRANCE.shaker));
    const phoneDrop = FALL * fall(span(t, ENTRANCE.phone));
    stand.position.y = phoneDrop;
    // Each object shows only from the moment it starts to fall.
    dumbbells.visible = t > ENTRANCE.dumbbell[0];
    shaker.visible = t > ENTRANCE.shaker[0];
    stand.visible = t > ENTRANCE.phone[0];
    phone.pivot.visible = t > ENTRANCE.phone[0];
    seat.copy(seatRest);
    seat.elements[13] += phoneDrop;
    shades.dumbbells.forEach((shade) => {
      shade.opacity = 0.9 * landed(ENTRANCE.dumbbell, t);
    });
    shades.shaker.opacity = 0.9 * landed(ENTRANCE.shaker, t);
    shades.phone.opacity = 0.9 * landed(ENTRANCE.phone, t);
    const appear = smooth(span(t, ENTRANCE.chart));
    chart.position.y = restY.chart - 0.35 * (1 - appear);
    updateChart(appear, span(t, ENTRANCE.rise));
  };

  const update = (time: number, elapsed: number) => {
    play(time, elapsed);
    entrance(elapsed);
  };
  const animating = (elapsed: number) => start === null || elapsed - start < ENTRANCE_END;

  return { group, seat, update, animating };
};
