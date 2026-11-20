import * as THREE from "three";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";

import { BUILD } from "@/components/home_story/story_athlete";
import { createHologram } from "@/components/home_story/story_hologram";

/*
 * The Feed's construction hologram (owner request, 2026-11-20): the Home's
 * scanning drone flies in, opens its beam over a platform and projects a
 * hologram of a person building the Feed. The person (the same continuous,
 * skinned hologram body as the Home's former athlete, story_athlete.ts) takes
 * a post card from a floating stack, turns, and pushes it into a Feed column,
 * card after card; when the column is full it completes with a sweep and
 * clears, and the building starts again. Everything is procedural (no model
 * files). Desktop only; phones, reduced motion and no-WebGL get the light
 * fallback in feed_construction_hologram.tsx.
 */

export interface FeedConstructionScene {
  setPointer: (x: number, y: number) => void;
  start: () => void;
  stop: () => void;
  resize: (width: number, height: number) => void;
  dispose: () => void;
}

type Disposable = { dispose: () => void };

// The person faces the stack (front left) to take a card and the wall (right) to place it.
const STACK_YAW = -0.95;
const WALL_YAW = 1.3;
const SLOTS = 4;
const CARD = { width: 0.64, height: 0.42 };
const WALL = { width: 0.84, height: 2.2, distance: 1.08, header: 0.2, gap: 0.06, lift: 0.3 };
const FLIGHT = 0.55; // seconds from the hands to the slot
const STACK_CARDS = 3;
const STACK_GAP = 0.04;

const smooth = (t: number) => t * t * (3 - 2 * t);
const clamp01 = (t: number) => Math.min(Math.max(t, 0), 1);

// --- Textures -------------------------------------------------------------------

const canvasTexture = (width: number, height: number, draw: (context: CanvasRenderingContext2D) => void) => {
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const context = canvas.getContext("2d");
  if (context) draw(context);
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.anisotropy = 4;
  return texture;
};

const roundRect = (context: CanvasRenderingContext2D, x: number, y: number, width: number, height: number, radius: number) => {
  context.beginPath();
  context.roundRect(x, y, width, height, radius);
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

// A Feed post, drawn in light on transparency: avatar, name and time, a
// picture with a glyph (a dumbbell, a meal, a medal or a progress chart),
// two lines of text and the reactions.
const cardTexture = (glyph: number) =>
  canvasTexture(512, 336, (context) => {
    context.strokeStyle = "rgba(255,255,255,0.95)";
    context.fillStyle = "rgba(255,255,255,0.08)";
    context.lineWidth = 5;
    roundRect(context, 6, 6, 500, 324, 30);
    context.fill();
    context.stroke();
    // Avatar, name and time.
    context.fillStyle = "rgba(255,255,255,0.9)";
    context.beginPath();
    context.arc(52, 50, 22, 0, Math.PI * 2);
    context.fill();
    context.fillStyle = "rgba(255,255,255,0.75)";
    roundRect(context, 88, 34, 150, 14, 7);
    context.fill();
    context.fillStyle = "rgba(255,255,255,0.4)";
    roundRect(context, 88, 58, 90, 10, 5);
    context.fill();
    // The picture.
    context.fillStyle = "rgba(255,255,255,0.14)";
    context.strokeStyle = "rgba(255,255,255,0.55)";
    context.lineWidth = 3;
    roundRect(context, 30, 90, 452, 150, 18);
    context.fill();
    context.stroke();
    context.strokeStyle = "rgba(255,255,255,0.95)";
    context.fillStyle = "rgba(255,255,255,0.95)";
    context.lineWidth = 9;
    context.lineCap = "round";
    context.lineJoin = "round";
    const cx = 256;
    const cy = 165;
    if (glyph === 0) {
      // Dumbbell.
      context.beginPath();
      context.moveTo(cx - 60, cy);
      context.lineTo(cx + 60, cy);
      context.stroke();
      for (const side of [-1, 1]) {
        roundRect(context, cx + side * 62 - 14, cy - 40, 28, 80, 8);
        context.fill();
        roundRect(context, cx + side * 92 - 10, cy - 28, 20, 56, 6);
        context.fill();
      }
    } else if (glyph === 1) {
      // A bowl, steaming.
      context.beginPath();
      context.arc(cx, cy - 6, 70, 0, Math.PI);
      context.closePath();
      context.fill();
      context.lineWidth = 6;
      for (const x of [-26, 0, 26]) {
        context.beginPath();
        context.moveTo(cx + x, cy - 22);
        context.bezierCurveTo(cx + x - 14, cy - 42, cx + x + 14, cy - 52, cx + x, cy - 70);
        context.stroke();
      }
    } else if (glyph === 2) {
      // A medal.
      context.lineWidth = 8;
      context.beginPath();
      context.moveTo(cx - 34, cy - 66);
      context.lineTo(cx - 12, cy - 22);
      context.moveTo(cx + 34, cy - 66);
      context.lineTo(cx + 12, cy - 22);
      context.stroke();
      context.beginPath();
      context.arc(cx, cy + 14, 40, 0, Math.PI * 2);
      context.stroke();
      context.beginPath();
      context.arc(cx, cy + 14, 20, 0, Math.PI * 2);
      context.fill();
    } else {
      // Progress: rising bars and a line.
      [30, 52, 44, 76, 96].forEach((h, index) => {
        roundRect(context, cx - 100 + index * 42, cy + 52 - h, 26, h, 5);
        context.fill();
      });
      context.lineWidth = 6;
      context.beginPath();
      context.moveTo(cx - 110, cy + 2);
      context.lineTo(cx - 30, cy - 18);
      context.lineTo(cx + 30, cy - 4);
      context.lineTo(cx + 110, cy - 58);
      context.stroke();
    }
    // Text and reactions.
    context.fillStyle = "rgba(255,255,255,0.6)";
    roundRect(context, 30, 258, 330, 12, 6);
    context.fill();
    context.fillStyle = "rgba(255,255,255,0.4)";
    roundRect(context, 30, 280, 230, 12, 6);
    context.fill();
    context.strokeStyle = "rgba(255,255,255,0.85)";
    context.lineWidth = 5;
    // Heart.
    context.beginPath();
    context.moveTo(430, 300);
    context.bezierCurveTo(400, 280, 410, 256, 430, 268);
    context.bezierCurveTo(450, 256, 460, 280, 430, 300);
    context.stroke();
    // Comment.
    roundRect(context, 380, 262, 30, 24, 8);
    context.stroke();
  });

// The Feed column: a rounded frame with a header and dashed slots.
const wallTexture = () =>
  canvasTexture(320, 892, (context) => {
    const scale = 892 / WALL.height;
    context.strokeStyle = "rgba(255,255,255,0.9)";
    context.fillStyle = "rgba(255,255,255,0.05)";
    context.lineWidth = 5;
    roundRect(context, 5, 5, 310, 882, 34);
    context.fill();
    context.stroke();
    // Header: the title bar and a small dot.
    context.fillStyle = "rgba(255,255,255,0.85)";
    context.font = "800 40px Archivo, 'Google Sans', Inter, Arial, sans-serif";
    context.textBaseline = "middle";
    context.fillText("FEED", 34, WALL.header * scale * 0.55);
    context.beginPath();
    context.arc(276, WALL.header * scale * 0.55, 10, 0, Math.PI * 2);
    context.fill();
    context.fillStyle = "rgba(255,255,255,0.35)";
    context.fillRect(24, WALL.header * scale - 4, 272, 2);
    // Dashed slots.
    context.setLineDash([14, 10]);
    context.strokeStyle = "rgba(255,255,255,0.45)";
    context.lineWidth = 3;
    for (let index = 0; index < SLOTS; index += 1) {
      const top = (WALL.header + WALL.gap + index * (CARD.height + WALL.gap)) * scale;
      const left = ((WALL.width - CARD.width) / 2) * scale;
      roundRect(context, left, top, CARD.width * scale, CARD.height * scale, 22);
      context.stroke();
    }
  });

// --- Hologram light material ------------------------------------------------------

const lightVertex = /* glsl */ `
  varying vec2 vUv;
  varying float vWorldY;

  void main() {
    vUv = uv;
    vec4 world = modelMatrix * vec4(position, 1.0);
    vWorldY = world.y;
    gl_Position = projectionMatrix * viewMatrix * world;
  }
`;

// Light drawn by the projector: the drawing, a faint glass fill, fine scan
// lines, a little flicker; it builds up from below (uReveal) and can flash.
const lightFragment = /* glsl */ `
  uniform sampler2D uMap;
  uniform vec3 uColor;
  uniform float uOpacity;
  uniform float uTime;
  uniform float uFlash;
  uniform float uReveal;
  varying vec2 vUv;
  varying float vWorldY;

  void main() {
    if (vWorldY > uReveal) discard;
    vec4 drawing = texture2D(uMap, vUv);
    float edge = exp(-pow((vWorldY - uReveal) * 22.0, 2.0));
    float scan = 0.84 + 0.16 * sin(vWorldY * 120.0 - uTime * 5.0);
    float flicker = 0.95 + 0.05 * sin(uTime * 23.0 + vWorldY * 6.0);
    vec3 color = uColor * (0.25 + drawing.rgb) + vec3(uFlash * 0.6 + edge * 0.8);
    float alpha = (drawing.a * 0.95 + 0.05 + uFlash * 0.35 + edge * 0.6) * scan * flicker * uOpacity;
    gl_FragColor = vec4(color, alpha);
    #include <colorspace_fragment>
  }
`;

export const createFeedConstructionScene = async (canvas: HTMLCanvasElement): Promise<FeedConstructionScene> => {
  const disposables = new Set<Disposable>();
  const track = <T extends Disposable>(item: T): T => {
    disposables.add(item);
    return item;
  };

  const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true, stencil: true, powerPreference: "high-performance" });
  const pixelRatio = Math.min(window.devicePixelRatio || 1, 1.75);
  renderer.setPixelRatio(pixelRatio);
  renderer.setClearColor(0x000000, 0);
  renderer.outputColorSpace = THREE.SRGBColorSpace;
  renderer.toneMapping = THREE.ACESFilmicToneMapping;
  renderer.toneMappingExposure = 1.05;

  const scene = new THREE.Scene();
  const pmrem = new THREE.PMREMGenerator(renderer);
  const room = new RoomEnvironment();
  const environment = pmrem.fromScene(room, 0.04).texture;
  room.dispose();
  pmrem.dispose();
  scene.environment = environment;
  scene.environmentIntensity = 0.8;
  scene.add(new THREE.HemisphereLight(0xcfe3ff, 0x02041f, 0.55));
  const key = new THREE.DirectionalLight(0xffffff, 2.1);
  key.position.set(-4, 5, 7);
  const rim = new THREE.DirectionalLight(0x36d6ff, 3);
  rim.position.set(5, 2.5, -4);
  scene.add(key, rim);

  const camera = new THREE.PerspectiveCamera(32, 1, 0.1, 60);
  const cameraHome = new THREE.Vector3(0.3, 0.75, 8.9);
  const lookAt = new THREE.Vector3(0.22, -0.2, 0);
  camera.position.copy(cameraHome);
  camera.lookAt(lookAt);

  // --- The installation: drone, beam, platform and the person ----------------------
  const hologram = await createHologram(track, pixelRatio, { build: { stackYaw: STACK_YAW, wallYaw: WALL_YAW }, intro: true });
  hologram.group.rotation.y = -0.18;
  scene.add(hologram.group);
  const { rig, floorY } = hologram;
  const glow = track(glowTexture());

  const lightMaterial = (map: THREE.Texture, opacity: number) =>
    track(
      new THREE.ShaderMaterial({
        uniforms: {
          uMap: { value: map },
          uColor: { value: new THREE.Color(0x36d6ff) },
          uOpacity: { value: opacity },
          uTime: { value: 0 },
          uFlash: { value: 0 },
          uReveal: { value: -100 }
        },
        vertexShader: lightVertex,
        fragmentShader: lightFragment,
        transparent: true,
        depthWrite: false,
        side: THREE.DoubleSide,
        blending: THREE.AdditiveBlending
      })
    );
  const materials: THREE.ShaderMaterial[] = [];
  const cardGeometry = track(new THREE.PlaneGeometry(CARD.width, CARD.height));
  const cardMaps = [0, 1, 2, 3].map((glyph) => track(cardTexture(glyph)));
  const makeCard = (map: THREE.Texture) => {
    const material = lightMaterial(map, 0);
    materials.push(material);
    const mesh = new THREE.Mesh(cardGeometry, material);
    mesh.renderOrder = 4;
    rig.add(mesh);
    return mesh;
  };

  // --- The Feed column, on the right, turned between the person and the visitor ----
  const wallDirection = new THREE.Vector3(Math.sin(WALL_YAW), 0, Math.cos(WALL_YAW));
  const wall = new THREE.Group();
  wall.position.copy(wallDirection).multiplyScalar(WALL.distance).setY(floorY + WALL.lift + WALL.height / 2);
  // Turned between the person (who pushes the cards in) and the visitor.
  wall.rotation.y = Math.atan2(-wallDirection.x * 0.35, -wallDirection.z * 0.35 + 0.65);
  rig.add(wall);
  const wallMaterial = lightMaterial(track(wallTexture()), 0);
  wallMaterial.uniforms.uColor.value.set(0x7cebff);
  materials.push(wallMaterial);
  const wallPlane = new THREE.Mesh(track(new THREE.PlaneGeometry(WALL.width, WALL.height)), wallMaterial);
  wallPlane.renderOrder = 3;
  wall.add(wallPlane);
  // A light line along the column's foot, joining it to the platform.
  const foot = new THREE.Sprite(
    track(new THREE.SpriteMaterial({ map: glow, color: 0x7cebff, transparent: true, opacity: 0, depthWrite: false, blending: THREE.AdditiveBlending, toneMapped: false }))
  );
  foot.scale.set(1.1, 0.35, 1);
  foot.position.set(wall.position.x, floorY + 0.05, wall.position.z);
  rig.add(foot);
  // Slot centres in the wall's frame, top to bottom.
  const slotLocal = Array.from({ length: SLOTS }, (_, index) => {
    const top = WALL.height / 2 - WALL.header - WALL.gap - index * (CARD.height + WALL.gap);
    return new THREE.Vector3(0, top - CARD.height / 2, 0.012);
  });
  wall.updateMatrixWorld(true);
  const slotPosition = slotLocal.map((local) => {
    const point = local.clone().applyMatrix4(wall.matrix);
    return point;
  });
  const wallQuaternion = wall.quaternion.clone();
  const placed = slotLocal.map((local, index) => {
    const card = makeCard(cardMaps[index % cardMaps.length]);
    card.position.copy(slotPosition[index]);
    card.quaternion.copy(wallQuaternion);
    return card;
  });
  const slotFlash = slotLocal.map(() => {
    const sprite = new THREE.Sprite(
      track(new THREE.SpriteMaterial({ map: glow, color: 0xe8fdff, transparent: true, opacity: 0, depthWrite: false, blending: THREE.AdditiveBlending, toneMapped: false }))
    );
    sprite.scale.set(1.1, 0.8, 1);
    rig.add(sprite);
    return sprite;
  });

  // --- The stack: cards floating over an emitter, front left ----------------------
  const stackYaw = new THREE.Quaternion().setFromAxisAngle(new THREE.Vector3(0, 1, 0), STACK_YAW);
  const stackTop = hologram.pickPoint.clone().applyQuaternion(stackYaw);
  // Tilted back, so the visitor sees the cards waiting on the stack.
  const flat = new THREE.Quaternion().setFromEuler(new THREE.Euler(-1.05, STACK_YAW, 0, "YXZ"));
  const stack = Array.from({ length: STACK_CARDS }, (_, index) => {
    const card = makeCard(cardMaps[(index + 1) % cardMaps.length]);
    card.position.copy(stackTop).setY(stackTop.y - index * STACK_GAP);
    card.quaternion.copy(flat);
    return card;
  });
  const emitter = new THREE.Mesh(
    track(new THREE.RingGeometry(0.2, 0.34, 64)),
    track(new THREE.MeshBasicMaterial({ map: glow, color: 0x7cebff, transparent: true, opacity: 0, depthWrite: false, blending: THREE.AdditiveBlending, toneMapped: false, side: THREE.DoubleSide }))
  );
  emitter.rotation.x = -Math.PI / 2;
  emitter.position.set(stackTop.x, floorY + 0.01, stackTop.z);
  rig.add(emitter);
  const column = new THREE.Sprite(
    track(new THREE.SpriteMaterial({ map: glow, color: 0x36d6ff, transparent: true, opacity: 0, depthWrite: false, blending: THREE.AdditiveBlending, toneMapped: false }))
  );
  column.scale.set(0.5, stackTop.y - floorY + 0.2, 1);
  column.position.set(stackTop.x, (stackTop.y + floorY) / 2, stackTop.z);
  rig.add(column);

  // The card in the person's hands, then in flight to its slot.
  const carried = makeCard(cardMaps[0]);
  const upright = new THREE.Quaternion();
  const hands = new THREE.Vector3();
  const flightFrom = new THREE.Vector3();
  const flightTurn = new THREE.Quaternion();
  const facing = new THREE.Vector3();
  const UP = new THREE.Vector3(0, 1, 0);

  // --- Animation --------------------------------------------------------------------
  const pointer = new THREE.Vector2();
  const pointerTarget = new THREE.Vector2();
  const revealPoint = new THREE.Vector3();
  let running = false;
  let frameId = 0;
  let lastFrame = 0;
  let elapsed = 0;

  const update = (time: number, delta: number) => {
    const state = hologram.update(time, 1);
    const presence = state.presence;
    rig.localToWorld(revealPoint.set(0, floorY - 0.05 + presence * (WALL.lift + WALL.height + 0.2), 0));
    const reveal = presence >= 1 ? 1e3 : revealPoint.y;
    for (const material of materials) {
      material.uniforms.uTime.value = time;
      material.uniforms.uReveal.value = reveal;
      material.uniforms.uFlash.value = 0;
    }

    // Which card this is, and the wall's round (it holds SLOTS cards, then clears).
    const working = state.time >= 0;
    const t = working ? state.time : 0;
    const index = working ? state.exercise : -1;
    const slot = index >= 0 ? index % SLOTS : 0;
    const round = index >= 0 ? Math.floor(index / SLOTS) : 0;
    const landing = BUILD.release + FLIGHT;
    const landed = working && t >= landing;

    // The wall: the cards of this round placed so far, and the one just landed.
    wallMaterial.uniforms.uOpacity.value = 0.8 * presence;
    const completing = working && slot === 0 && round > 0 && t < 2.2;
    placed.forEach((card, cardSlot) => {
      const material = card.material as THREE.ShaderMaterial;
      let opacity = 0;
      if (completing) {
        // The previous round's full column glows with a sweep, then clears upwards.
        const clear = smooth(clamp01((t - 1.2) / 1));
        opacity = 1 - clear;
        material.uniforms.uFlash.value = Math.exp(-Math.pow((t - 0.35 - cardSlot * 0.12) * 6, 2)) * 0.9;
        card.position.copy(slotPosition[cardSlot]).setY(slotPosition[cardSlot].y + clear * 0.25);
      } else {
        card.position.copy(slotPosition[cardSlot]);
        if (cardSlot < slot || (cardSlot === slot && landed)) opacity = 1;
        if (cardSlot === slot && landed) material.uniforms.uFlash.value = Math.exp(-(t - landing) * 5) * 0.9;
      }
      material.uniforms.uOpacity.value = opacity * presence;
      card.visible = opacity > 0.002;
    });
    slotFlash.forEach((sprite, cardSlot) => {
      const flash = cardSlot === slot && landed ? Math.exp(-(t - landing) * 4) : 0;
      sprite.visible = flash > 0.002;
      sprite.material.opacity = flash * 0.9;
      sprite.position.copy(slotPosition[cardSlot]);
    });
    foot.material.opacity = 0.35 * presence;

    // The stack: the top card goes at the grip, the others move up and a new
    // one builds up underneath (the first card's mesh, reused).
    const taken = working && t >= BUILD.grip;
    const shift = taken ? smooth(clamp01((t - BUILD.grip) / 0.5)) : 0;
    const refill = taken ? smooth(clamp01((t - BUILD.grip - 0.3) / 0.6)) : 1;
    stack.forEach((card, cardIndex) => {
      const material = card.material as THREE.ShaderMaterial;
      const renewed = taken && cardIndex === 0;
      const level = renewed ? STACK_CARDS - 1 : cardIndex - shift;
      card.position.copy(stackTop).setY(stackTop.y - level * STACK_GAP - (renewed ? (1 - refill) * 0.12 : 0));
      material.uniforms.uOpacity.value = (renewed ? refill : 1) * presence;
      card.visible = material.uniforms.uOpacity.value > 0.002;
    });
    (emitter.material as THREE.MeshBasicMaterial).opacity = 0.55 * presence;
    column.material.opacity = 0.16 * presence;

    // The carried card: in the hands from the grip, then off to its slot.
    rig.worldToLocal(hands.copy(state.hands));
    facing.set(Math.sin(state.yaw), 0, Math.cos(state.yaw));
    const carriedMaterial = carried.material as THREE.ShaderMaterial;
    if (working && t >= BUILD.grip && t < landing) {
      carried.visible = true;
      carriedMaterial.uniforms.uMap.value = cardMaps[slot % cardMaps.length];
      carriedMaterial.uniforms.uOpacity.value = presence;
      // Upright and facing out from the chest once lifted.
      upright.setFromAxisAngle(UP, state.yaw);
      if (t < BUILD.release) {
        const lift = smooth(clamp01((t - BUILD.grip) / (BUILD.lift[1] - BUILD.grip)));
        carried.quaternion.copy(flat).slerp(upright, lift);
        carried.position.copy(hands).addScaledVector(facing, 0.05 * lift);
        flightFrom.copy(carried.position);
        flightTurn.copy(carried.quaternion);
      } else {
        // In flight: an arc from the hands into the slot, turning to the wall.
        const fly = smooth(clamp01((t - BUILD.release) / FLIGHT));
        carried.position.lerpVectors(flightFrom, slotPosition[slot], fly);
        carried.position.y += Math.sin(fly * Math.PI) * 0.18;
        carried.quaternion.copy(flightTurn).slerp(wallQuaternion, fly);
        carriedMaterial.uniforms.uFlash.value = Math.sin(fly * Math.PI) * 0.4;
      }
    } else {
      carried.visible = false;
      carriedMaterial.uniforms.uOpacity.value = 0;
    }
    // The pointer turns the view a little.
    pointer.lerp(pointerTarget, 1 - Math.exp(-delta * 3));
    camera.position.set(cameraHome.x + pointer.x * 0.5, cameraHome.y - pointer.y * 0.3, cameraHome.z);
    camera.lookAt(lookAt);
  };

  const frame = (now: number) => {
    if (!running) return;
    const delta = Math.min((now - lastFrame) / 1000, 0.1);
    lastFrame = now;
    elapsed += delta;
    update(elapsed, delta);
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
      disposables.forEach((item) => item.dispose());
      environment.dispose();
      renderer.dispose();
      renderer.forceContextLoss();
    }
  };
};
