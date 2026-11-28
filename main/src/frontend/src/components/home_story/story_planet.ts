import * as THREE from "three";
import { mergeGeometries } from "three/addons/utils/BufferGeometryUtils.js";

import { GLOBE_CITIES, GLOBE_HOME, GLOBE_ROUTES } from "@/components/brand_globe/globe_data";

/*
 * Planet RYZE: the Earth with RYZE routes leaving Lisbon, restored from the
 * Home story (it led the hero until 2026-11-15) for the "Get started" chapter
 * (owner request, 2026-11-15: "join planet RYZE"). The surface, clouds,
 * routes, cities and the Lisbon badge are the original's, unchanged; the land
 * comes from a Natural Earth land mask.
 */

export interface PlanetOptions {
  maskImage: HTMLImageElement;
  logoImage: HTMLImageElement;
  anisotropy: number;
  pixelRatio: number;
}

export interface Planet {
  group: THREE.Group;
  // spin: extra turn from the scroll state; scale: the planet's on-screen scale.
  update: (elapsed: number, spin: number, scale: number, pointer: THREE.Vector2) => void;
}

type Track = <T extends { dispose: () => void }>(item: T) => T;

const TAU = Math.PI * 2;
const COLORS = {
  ring: 0x0481dc,
  cyanBright: 0x50e2fe,
  ice: 0x7cebff
} as const;

const canvas2d = (width: number, height: number) => {
  const element = document.createElement("canvas");
  element.width = width;
  element.height = height;
  return { element, context: element.getContext("2d") as CanvasRenderingContext2D };
};

const colorTexture = (source: HTMLCanvasElement, anisotropy: number) => {
  const texture = new THREE.CanvasTexture(source);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.anisotropy = anisotropy;
  return texture;
};

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
  uniform vec3 uDot;
  uniform vec3 uCity;
  uniform vec3 uGlint;
  uniform float uTime;
  varying vec2 vUv;
  varying vec3 vNormalW;
  varying vec3 vPositionW;

  float hash(vec2 p) {
    return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
  }

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

    // Dot matrix over the land. Each row holds fewer dots towards the poles,
    // so the dots stay round on the sphere; a cell is lit when its centre is
    // land (sampled at the base level: the cell coordinate is discontinuous).
    const float PI = 3.14159265;
    float rows = 150.0;
    float row = floor(vUv.y * rows);
    float rowLat = ((row + 0.5) / rows - 0.5) * PI;
    float cols = max(floor(rows * 2.0 * cos(rowLat)), 1.0);
    vec2 cell = vec2(vUv.x * cols, vUv.y * rows);
    vec2 cellId = floor(cell);
    float d = length(fract(cell) - 0.5);
    float aa = fwidth(d) * 1.2;
    float dotMask = 1.0 - smoothstep(0.24 - aa, 0.24 + aa, d);
    float cellLand = smoothstep(0.45, 0.6, textureLod(uMask, (cellId + 0.5) / vec2(cols, rows), 0.0).r);
    color += dotMask * cellLand * uDot * mix(0.3, 1.0, day);

    // City lights on the night side: some land cells glow and twinkle.
    float seed = hash(cellId + vec2(row * 0.37, 0.0));
    float night = 1.0 - smoothstep(-0.2, 0.22, light);
    float twinkle = 0.6 + 0.4 * sin(uTime * (0.8 + seed * 1.8) + seed * 40.0);
    color += step(0.82, seed) * cellLand * night * dotMask * uCity * twinkle * 1.8;

    // Sun glint on the oceans.
    vec3 halfway = normalize(uSun + view);
    color += pow(max(dot(normal, halfway), 0.0), 260.0) * (1.0 - land) * day * uGlint;

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

const cloudVertex = /* glsl */ `
  varying vec3 vObject;
  varying vec3 vNormalW;
  varying vec3 vPositionW;
  void main() {
    vObject = normalize(position);
    vec4 world = modelMatrix * vec4(position, 1.0);
    vPositionW = world.xyz;
    vNormalW = normalize(mat3(modelMatrix) * normal);
    gl_Position = projectionMatrix * viewMatrix * world;
  }
`;

// Thin procedural clouds (value-noise fbm), lit by the same sun, drifting
// slowly over the surface.
const cloudFragment = /* glsl */ `
  uniform vec3 uSun;
  uniform vec3 uColor;
  uniform float uTime;
  uniform float uOpacity;
  varying vec3 vObject;
  varying vec3 vNormalW;
  varying vec3 vPositionW;

  float hash3(vec3 p) {
    p = fract(p * 0.3183099 + 0.1);
    p *= 17.0;
    return fract(p.x * p.y * p.z * (p.x + p.y + p.z));
  }

  float noise(vec3 x) {
    vec3 i = floor(x);
    vec3 f = fract(x);
    f = f * f * (3.0 - 2.0 * f);
    return mix(
      mix(mix(hash3(i), hash3(i + vec3(1.0, 0.0, 0.0)), f.x), mix(hash3(i + vec3(0.0, 1.0, 0.0)), hash3(i + vec3(1.0, 1.0, 0.0)), f.x), f.y),
      mix(mix(hash3(i + vec3(0.0, 0.0, 1.0)), hash3(i + vec3(1.0, 0.0, 1.0)), f.x), mix(hash3(i + vec3(0.0, 1.0, 1.0)), hash3(i + vec3(1.0)), f.x), f.y),
      f.z
    );
  }

  float fbm(vec3 p) {
    float value = 0.0;
    float amplitude = 0.5;
    for (int octave = 0; octave < 5; octave++) {
      value += amplitude * noise(p);
      p *= 2.03;
      amplitude *= 0.5;
    }
    return value;
  }

  void main() {
    vec3 p = vObject * 3.2 + vec3(uTime * 0.012, 0.0, uTime * 0.006);
    // Stretched along the latitudes, like weather bands.
    float n = fbm(vec3(p.x * 1.6, p.y * 3.4, p.z * 1.6));
    float cover = smoothstep(0.52, 0.76, n);
    vec3 normal = normalize(vNormalW);
    float day = smoothstep(-0.25, 0.55, dot(normal, uSun));
    vec3 view = normalize(cameraPosition - vPositionW);
    float edge = smoothstep(0.0, 0.35, dot(normal, view));
    // Additive: clouds only ever brighten the surface below them.
    float alpha = cover * uOpacity * mix(0.2, 1.0, day) * edge;
    gl_FragColor = vec4(uColor, alpha);
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


// --- Builders --------------------------------------------------------------------

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

export const createPlanet = (track: Track, options: PlanetOptions): Planet => {
  const globeRadius = 1.55;
  const globe = new THREE.Group();
  const earth = new THREE.Group(); // spins around its own axis
  const maskTexture = track(new THREE.Texture(options.maskImage));
  maskTexture.wrapS = THREE.RepeatWrapping;
  maskTexture.anisotropy = options.anisotropy;
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
          uRim: { value: new THREE.Color(0x3fb8ff) },
          uDot: { value: new THREE.Color(0x4fd2ff).multiplyScalar(0.42) },
          uCity: { value: new THREE.Color(0xc4f6ff) },
          uGlint: { value: new THREE.Color(0x8fdcff).multiplyScalar(0.55) },
          uTime: { value: 0 }
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
        uniforms: { uColor: { value: new THREE.Color(0x22b4ff) }, uStrength: { value: 1.25 } },
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
        uPixelRatio: { value: options.pixelRatio },
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
    track(new THREE.SpriteMaterial({ map: track(badgeTexture(options.logoImage, options.anisotropy)), transparent: true, depthWrite: false, toneMapped: false }))
  );
  badge.scale.setScalar(0.42);
  badge.position.y = stemLength + 0.1;
  marker.add(stem, pulse, badge);

  const cloudMaterial = track(
    new THREE.ShaderMaterial({
      uniforms: {
        uSun: { value: sun },
        uColor: { value: new THREE.Color(0xe6f6ff) },
        uTime: { value: 0 },
        uOpacity: { value: 0.3 }
      },
      vertexShader: cloudVertex,
      fragmentShader: cloudFragment,
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending
    })
  );
  const clouds = new THREE.Mesh(track(new THREE.SphereGeometry(globeRadius * 1.012, 96, 64)), cloudMaterial);

  earth.add(surface, clouds, routes, cities, marker);
  globe.add(atmosphere, earth);
  // Centred just west of Lisbon, so the RYZE marker faces the viewer with its
  // routes over the Atlantic beside it (owner request, 2026-11-28: it sat on
  // the edge of the globe).
  const view = latLonToVector(32, -11, 1);
  const baseYaw = Math.atan2(-view.x, view.z);
  const baseTilt = THREE.MathUtils.degToRad(24);
  globe.visible = false;

  const update = (elapsed: number, spin: number, scale: number, pointer: THREE.Vector2) => {
    globe.rotation.set(baseTilt + pointer.y * 0.08, pointer.x * 0.12, 0);
    // A slow sway keeps the Atlantic routes in view instead of a full spin.
    earth.rotation.set(0, baseYaw + spin + Math.sin(elapsed * 0.05) * 0.12, 0);
    routeMaterial.uniforms.uTime.value = elapsed;
    (surface.material as THREE.ShaderMaterial).uniforms.uTime.value = elapsed;
    cloudMaterial.uniforms.uTime.value = elapsed;
    cityMaterial.uniforms.uScale.value = scale * 10;
    const beat = (elapsed * 0.45) % 1;
    pulse.scale.setScalar(1 + beat * 3.2);
    pulseMaterial.opacity = (1 - beat) * 0.9;
  };

  return { group: globe, update };
};
