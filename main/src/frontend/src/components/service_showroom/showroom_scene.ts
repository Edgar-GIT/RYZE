import * as THREE from "three";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";

/*
 * The Services showroom stage: real geometry under the HTML service cards.
 * Each card stands on a slim two-tier pedestal (glossy navy, a cyan light ring
 * on its edge, a pool of light on its top, a soft glow on the ground below it)
 * with a soft light behind the card. There is no floor: the pedestals stand on
 * the page's own navy. The cards themselves stay HTML
 * (crisp, protected); the scene is placed from their on-screen positions.
 *
 * Rendering is on demand: a frame is drawn after a layout change and during
 * the short arrival and focus transitions, never in a permanent loop.
 */

export interface ShowroomSlot {
  // Card box in canvas pixels.
  left: number;
  top: number;
  width: number;
  height: number;
  featured: boolean;
}

export interface ShowroomScene {
  // Places the stage; returns, per slot, how far below the card's bottom edge
  // the pedestal ends (in canvas pixels), so a label can sit just under it.
  layout: (width: number, height: number, slots: ShowroomSlot[]) => number[];
  setActive: (index: number | null) => void;
  // Plays the arrival (pedestals rise, lights come on), staggered per slot.
  arrive: () => void;
  dispose: () => void;
}

const NAVY = 0x0a1748;
const CYAN = 0x00cbff;
const ICE = 0x7cebff;

const radialTexture = (stops: Array<[number, string]>, size = 256): THREE.CanvasTexture => {
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const context = canvas.getContext("2d");
  if (!context) throw new Error("2D canvas unavailable");
  const gradient = context.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
  for (const [offset, color] of stops) gradient.addColorStop(offset, color);
  context.fillStyle = gradient;
  context.fillRect(0, 0, size, size);
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  return texture;
};

// Two-tier pedestal profile (radius 1, top surface at y = 0), revolved.
const PEDESTAL_DEPTH = 0.3;
const pedestalGeometry = (): THREE.LatheGeometry => {
  const points = [
    [0, -PEDESTAL_DEPTH],
    [1.07, -PEDESTAL_DEPTH],
    [1.09, -PEDESTAL_DEPTH + 0.015],
    [1.09, -0.17],
    [1.07, -0.155],
    [1.0, -0.155],
    [1.0, -0.025],
    [0.988, -0.005],
    [0.965, 0],
    [0, 0]
  ].map(([x, y]) => new THREE.Vector2(x, y));
  return new THREE.LatheGeometry(points, 128);
};

interface SlotObjects {
  group: THREE.Group;
  ring: THREE.Mesh<THREE.TorusGeometry, THREE.MeshBasicMaterial>;
  baseRing: THREE.Mesh<THREE.TorusGeometry, THREE.MeshBasicMaterial>;
  pool: THREE.Mesh<THREE.CircleGeometry, THREE.MeshBasicMaterial>;
  ground: THREE.Mesh<THREE.CircleGeometry, THREE.MeshBasicMaterial>;
  halo: THREE.Mesh<THREE.PlaneGeometry, THREE.MeshBasicMaterial>;
  light: THREE.PointLight;
  featured: boolean;
  // Animated values: 0..1 arrival, focus -1 (behind) .. 0 (rest) .. 1 (active).
  arrival: number;
  arrivalTarget: number;
  arrivalDelay: number;
  focus: number;
  focusTarget: number;
}

export const createShowroomScene = (canvas: HTMLCanvasElement): ShowroomScene => {
  const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true, powerPreference: "low-power" });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 1.75));
  renderer.setClearColor(0x000000, 0);
  renderer.outputColorSpace = THREE.SRGBColorSpace;
  renderer.toneMapping = THREE.ACESFilmicToneMapping;
  renderer.toneMappingExposure = 1.05;

  const scene = new THREE.Scene();
  const pmrem = new THREE.PMREMGenerator(renderer);
  const room = new RoomEnvironment();
  const environment = pmrem.fromScene(room, 0.04).texture;
  scene.environment = environment;
  scene.environmentIntensity = 0.55;
  room.dispose();
  pmrem.dispose();

  // Seen slightly from above, like a product photograph.
  const camera = new THREE.PerspectiveCamera(26, 1, 0.1, 200);
  camera.position.set(0, 4.4, 26);
  camera.lookAt(0, 2.4, 0);

  scene.add(new THREE.HemisphereLight(0x9fc7ff, 0x02041f, 0.28));
  const key = new THREE.DirectionalLight(0xdfeaff, 0.9);
  key.position.set(-6, 12, 10);
  const rim = new THREE.DirectionalLight(CYAN, 1.4);
  rim.position.set(8, 6, -10);
  scene.add(key, rim);

  const disposables: Array<{ dispose: () => void }> = [environment];
  const track = <T extends { dispose: () => void }>(item: T): T => {
    disposables.push(item);
    return item;
  };

  const pedestalShape = track(pedestalGeometry());
  const pedestalMaterial = track(
    new THREE.MeshPhysicalMaterial({
      color: NAVY,
      metalness: 0.4,
      roughness: 0.34,
      clearcoat: 0.45,
      clearcoatRoughness: 0.18,
      envMapIntensity: 0.16
    })
  );
  const ringGeometry = track(new THREE.TorusGeometry(0.985, 0.012, 10, 160));
  const baseRingGeometry = track(new THREE.TorusGeometry(1.092, 0.006, 8, 160));
  const groundTexture = track(radialTexture([[0, "rgba(0,203,255,0.45)"], [0.5, "rgba(4,129,220,0.16)"], [1, "rgba(2,4,31,0)"]]));
  const groundGeometry = track(new THREE.CircleGeometry(1.9, 96));
  const poolTexture = track(radialTexture([[0, "rgba(0,203,255,0.6)"], [0.5, "rgba(4,129,220,0.22)"], [1, "rgba(4,129,220,0)"]]));
  const haloTexture = track(
    radialTexture([[0, "rgba(0,203,255,0.55)"], [0.35, "rgba(4,129,220,0.32)"], [0.7, "rgba(7,42,119,0.12)"], [1, "rgba(2,4,31,0)"]])
  );
  const poolGeometry = track(new THREE.CircleGeometry(0.96, 96));
  // The top is a matte navy stage surface: at this low viewing angle a glossy
  // top would mirror the studio environment and read grey.
  const capMaterial = track(
    new THREE.MeshStandardMaterial({ color: 0x07113a, roughness: 0.72, metalness: 0.15, envMapIntensity: 0.05 })
  );
  const haloGeometry = track(new THREE.PlaneGeometry(1, 1));

  let slots: SlotObjects[] = [];
  let frame = 0;
  let running = false;
  let last = 0;
  let started = 0;

  const raycaster = new THREE.Raycaster();
  const ndc = new THREE.Vector2();
  const floorPlane = new THREE.Plane(new THREE.Vector3(0, 1, 0), 0);
  const hit = new THREE.Vector3();
  let viewWidth = 1;
  let viewHeight = 1;

  // Where a canvas pixel meets a plane.
  const project = (x: number, y: number, plane: THREE.Plane): THREE.Vector3 | null => {
    ndc.set((x / viewWidth) * 2 - 1, -(y / viewHeight) * 2 + 1);
    raycaster.setFromCamera(ndc, camera);
    return raycaster.ray.intersectPlane(plane, hit) ? hit.clone() : null;
  };

  const disposeSlots = () => {
    for (const slot of slots) {
      scene.remove(slot.group);
      slot.ring.material.dispose();
      slot.baseRing.material.dispose();
      slot.pool.material.dispose();
      slot.ground.material.dispose();
      slot.halo.material.dispose();
    }
    slots = [];
  };

  const buildSlot = (featured: boolean, index: number): SlotObjects => {
    const group = new THREE.Group();
    const pedestal = new THREE.Mesh(pedestalShape, pedestalMaterial);
    const ring = new THREE.Mesh(
      ringGeometry,
      new THREE.MeshBasicMaterial({ color: CYAN, transparent: true, toneMapped: false })
    );
    ring.rotation.x = Math.PI / 2;
    ring.position.y = -0.004;
    const baseRing = new THREE.Mesh(
      baseRingGeometry,
      new THREE.MeshBasicMaterial({ color: ICE, transparent: true, toneMapped: false })
    );
    baseRing.rotation.x = Math.PI / 2;
    baseRing.position.y = -PEDESTAL_DEPTH + 0.012;
    const pool = new THREE.Mesh(
      poolGeometry,
      new THREE.MeshBasicMaterial({
        map: poolTexture,
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
        toneMapped: false
      })
    );
    pool.rotation.x = -Math.PI / 2;
    pool.position.y = 0.003;
    const cap = new THREE.Mesh(poolGeometry, capMaterial);
    cap.rotation.x = -Math.PI / 2;
    cap.position.y = 0.0015;
    const halo = new THREE.Mesh(
      haloGeometry,
      new THREE.MeshBasicMaterial({
        map: haloTexture,
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
        toneMapped: false
      })
    );
    const ground = new THREE.Mesh(
      groundGeometry,
      new THREE.MeshBasicMaterial({
        map: groundTexture,
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
        toneMapped: false
      })
    );
    ground.rotation.x = -Math.PI / 2;
    ground.position.y = -PEDESTAL_DEPTH - 0.004;
    const light = new THREE.PointLight(CYAN, 0, 9, 1.6);
    group.add(ground, pedestal, cap, ring, baseRing, pool, light);
    scene.add(group, halo);
    return {
      group,
      ring,
      baseRing,
      pool,
      ground,
      halo,
      light,
      featured,
      arrival: 0,
      arrivalTarget: 0,
      arrivalDelay: index * 0.16,
      focus: 0,
      focusTarget: 0
    };
  };

  const ease = (t: number) => 1 - Math.pow(1 - THREE.MathUtils.clamp(t, 0, 1), 3);

  // Applies the animated values to the objects.
  const apply = () => {
    for (const slot of slots) {
      const rise = ease(slot.arrival);
      const glow = slot.featured ? 1.15 : 1;
      const focus = slot.focus;
      const lit = THREE.MathUtils.clamp(1 + focus * 0.9, 0.35, 1.9) * glow;
      slot.group.scale.setScalar(slot.group.userData.radius * (0.7 + 0.3 * rise));
      slot.group.position.y = (1 - rise) * -0.9;
      slot.ring.material.opacity = rise * Math.min(1, 0.75 * lit);
      slot.ring.material.color.setHex(CYAN).multiplyScalar(Math.max(1, lit));
      slot.baseRing.material.opacity = rise * 0.35 * Math.min(1.4, lit);
      slot.pool.material.opacity = rise * 0.42 * Math.min(1.6, lit);
      slot.ground.material.opacity = rise * 0.8 * Math.min(1.5, lit);
      slot.halo.material.opacity = rise * 0.55 * lit;
      slot.light.intensity = rise * 14 * lit;
    }
  };

  const draw = () => renderer.render(scene, camera);

  const step = (now: number) => {
    const delta = Math.max(0, Math.min((now - last) / 1000, 0.05));
    last = now;
    const elapsed = (now - started) / 1000;
    let moving = false;
    for (const slot of slots) {
      // A slot still to arrive keeps the loop going even while its stagger
      // delay runs (the first frame's timestamp can precede arrive()).
      if (slot.arrival < slot.arrivalTarget) {
        if (elapsed > slot.arrivalDelay) {
          slot.arrival = Math.min(slot.arrivalTarget, slot.arrival + delta / 0.9);
        }
        moving = true;
      }
      const gap = slot.focusTarget - slot.focus;
      if (Math.abs(gap) > 0.002) {
        slot.focus += gap * Math.min(1, delta * 9);
        moving = true;
      } else {
        slot.focus = slot.focusTarget;
      }
    }
    apply();
    draw();
    if (moving) {
      frame = window.requestAnimationFrame(step);
    } else {
      running = false;
    }
  };

  const animate = () => {
    if (running) return;
    running = true;
    last = performance.now();
    frame = window.requestAnimationFrame(step);
  };

  return {
    layout: (width, height, cards) => {
      if (width <= 0 || height <= 0) return [];
      viewWidth = width;
      viewHeight = height;
      renderer.setSize(width, height, false);
      camera.aspect = width / height;
      camera.updateProjectionMatrix();
      camera.updateMatrixWorld();

      if (slots.length !== cards.length || slots.some((slot, index) => slot.featured !== cards[index].featured)) {
        const previous = slots.map((slot) => ({ arrival: slot.arrival, target: slot.arrivalTarget, focus: slot.focus }));
        disposeSlots();
        slots = cards.map((card, index) => {
          const slot = buildSlot(card.featured, index);
          if (previous[index]) {
            slot.arrival = previous[index].arrival;
            slot.arrivalTarget = previous[index].target;
            slot.focus = previous[index].focus;
          }
          return slot;
        });
      }

      // Pedestals stand where the cards' bottom edges meet the ground plane;
      // their radius covers the card width with a small margin.
      const plinths: number[] = [];
      const front = new THREE.Vector3();
      cards.forEach((card, index) => {
        const slot = slots[index];
        const bottom = card.top + card.height;
        const centre = project(card.left + card.width / 2, bottom, floorPlane);
        const left = project(card.left - card.width * 0.03, bottom, floorPlane);
        const right = project(card.left + card.width * 1.03, bottom, floorPlane);
        if (!centre || !left || !right) return;
        const radius = left.distanceTo(right) / 2;
        slot.group.userData.radius = radius;
        slot.group.position.set(centre.x, 0, centre.z);
        // Lowest visible point of the pedestal: the front of its base.
        front.set(centre.x, -PEDESTAL_DEPTH * radius, centre.z + radius * 1.09).project(camera);
        plinths[index] = Math.max(0, ((1 - front.y) / 2) * viewHeight - bottom);

        // The light behind the card: on a vertical plane just behind it.
        const backPlane = new THREE.Plane(new THREE.Vector3(0, 0, 1), -(centre.z - radius * 0.55));
        const top = project(card.left + card.width / 2, card.top, backPlane);
        const base = project(card.left + card.width / 2, bottom, backPlane);
        if (top && base) {
          const cardHeight = top.distanceTo(base);
          slot.halo.position.set(centre.x, (top.y + base.y) / 2, backPlane.constant * -1);
          slot.halo.scale.set(radius * 3.4, cardHeight * 1.5, 1);
        }
        slot.light.position.set(0, 1.4 / Math.max(radius, 0.001), 0.6 / Math.max(radius, 0.001));
      });
      apply();
      draw();
      return plinths;
    },
    arrive: () => {
      started = performance.now();
      for (const slot of slots) slot.arrivalTarget = 1;
      animate();
    },
    setActive: (index) => {
      slots.forEach((slot, slotIndex) => {
        slot.focusTarget = index === null ? 0 : slotIndex === index ? 1 : -1;
      });
      animate();
    },
    dispose: () => {
      running = false;
      window.cancelAnimationFrame(frame);
      disposeSlots();
      for (const item of disposables) item.dispose();
      renderer.dispose();
    }
  };
};
