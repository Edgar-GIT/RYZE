import * as THREE from "three";

/*
 * The Our Vision hero scene: "Thousands of fitness apps. All the same."
 *
 * An endless tunnel of identical, greyed-out app cards streams past the camera
 * while the RYZE card (HTML, layered above the canvas) stays still and lit in
 * the middle, circled by two rings of cyan light. Everything is procedural
 * (canvas textures, no model or image files) and drawn in a handful of draw
 * calls: one instanced mesh for every card, one point cloud, two rings and a
 * halo. Loaded with a dynamic import on tablets and desktops only.
 */

export interface VisionScene {
  setPointer: (x: number, y: number) => void;
  start: () => void;
  stop: () => void;
  resize: (width: number, height: number) => void;
  dispose: () => void;
}

export interface VisionSceneOptions {
  // The caption printed on every identical card ("12-week plan").
  cardLabel: string;
  // Fewer cards and particles on smaller screens.
  density: "full" | "light";
}

const NAVY = 0x02041f;
const CYAN = 0x00cbff;
const ICE = 0x7cebff;
const RING_BLUE = 0x0481dc;

const TUNNEL_NEAR = 6;
const TUNNEL_FAR = -58;
const TUNNEL_LENGTH = TUNNEL_NEAR - TUNNEL_FAR;
const MIN_RADIUS = 3.4;
const MAX_RADIUS = 9.5;
const CARD_SPEED = 3.4;

// drawCardTexture paints the one template every "other app" shares: a dim
// card with a dumbbell, two placeholder lines and the same caption.
const drawCardTexture = (label: string): THREE.CanvasTexture => {
  const width = 256;
  const height = 320;
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("2D canvas unavailable");
  }

  const radius = 34;
  context.beginPath();
  context.roundRect(6, 6, width - 12, height - 12, radius);
  context.fillStyle = "rgba(22, 34, 78, 0.92)";
  context.fill();
  context.lineWidth = 4;
  context.strokeStyle = "rgba(147, 164, 200, 0.42)";
  context.stroke();

  // Dumbbell.
  context.strokeStyle = "rgba(147, 164, 200, 0.7)";
  context.lineWidth = 9;
  context.lineCap = "round";
  context.beginPath();
  context.moveTo(52, 70);
  context.lineTo(104, 70);
  context.stroke();
  context.lineWidth = 14;
  context.beginPath();
  context.moveTo(46, 52);
  context.lineTo(46, 88);
  context.moveTo(110, 52);
  context.lineTo(110, 88);
  context.stroke();

  // Placeholder lines.
  context.fillStyle = "rgba(147, 164, 200, 0.32)";
  context.beginPath();
  context.roundRect(36, 136, 184, 16, 8);
  context.fill();
  context.beginPath();
  context.roundRect(36, 166, 112, 16, 8);
  context.fill();

  // Caption.
  context.fillStyle = "rgba(147, 164, 200, 0.75)";
  context.font = "600 22px system-ui, sans-serif";
  context.fillText(label, 36, 262, width - 72);

  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.anisotropy = 4;
  return texture;
};

const drawHaloTexture = (): THREE.CanvasTexture => {
  const size = 256;
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("2D canvas unavailable");
  }
  const gradient = context.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
  gradient.addColorStop(0, "rgba(124, 235, 255, 0.9)");
  gradient.addColorStop(0.25, "rgba(0, 203, 255, 0.45)");
  gradient.addColorStop(0.6, "rgba(4, 129, 220, 0.12)");
  gradient.addColorStop(1, "rgba(2, 4, 31, 0)");
  context.fillStyle = gradient;
  context.fillRect(0, 0, size, size);
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  return texture;
};

interface CardState {
  angle: number;
  radius: number;
  z: number;
  spin: number;
  tilt: number;
}

const randomCard = (z: number): CardState => ({
  angle: Math.random() * Math.PI * 2,
  radius: MIN_RADIUS + Math.random() * (MAX_RADIUS - MIN_RADIUS),
  z,
  spin: (Math.random() - 0.5) * 0.5,
  tilt: (Math.random() - 0.5) * 0.6
});

export const createVisionScene = (canvas: HTMLCanvasElement, options: VisionSceneOptions): VisionScene => {
  const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true, powerPreference: "low-power" });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, options.density === "full" ? 1.75 : 1.5));
  renderer.setClearColor(0x000000, 0);
  renderer.outputColorSpace = THREE.SRGBColorSpace;

  const scene = new THREE.Scene();
  scene.fog = new THREE.Fog(NAVY, 9, 52);

  const camera = new THREE.PerspectiveCamera(52, 1, 0.1, 80);
  camera.position.set(0, 0, 8);

  // --- The identical apps -----------------------------------------------------
  const cardCount = options.density === "full" ? 340 : 200;
  const cardTexture = drawCardTexture(options.cardLabel);
  const cardGeometry = new THREE.PlaneGeometry(1.15, 1.44);
  const cardMaterial = new THREE.MeshBasicMaterial({
    map: cardTexture,
    transparent: true,
    depthWrite: false,
    side: THREE.DoubleSide,
    fog: true
  });
  const cards = new THREE.InstancedMesh(cardGeometry, cardMaterial, cardCount);
  cards.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
  const cardStates: CardState[] = Array.from({ length: cardCount }, (_, index) =>
    randomCard(TUNNEL_FAR + (index / cardCount) * TUNNEL_LENGTH)
  );
  scene.add(cards);

  // --- Speed dust ---------------------------------------------------------------
  const dustCount = options.density === "full" ? 900 : 450;
  const dustPositions = new Float32Array(dustCount * 3);
  for (let i = 0; i < dustCount; i += 1) {
    const angle = Math.random() * Math.PI * 2;
    const radius = 1.8 + Math.random() * 12;
    dustPositions[i * 3] = Math.cos(angle) * radius;
    dustPositions[i * 3 + 1] = Math.sin(angle) * radius;
    dustPositions[i * 3 + 2] = TUNNEL_FAR + Math.random() * TUNNEL_LENGTH;
  }
  const dustGeometry = new THREE.BufferGeometry();
  dustGeometry.setAttribute("position", new THREE.BufferAttribute(dustPositions, 3));
  const dustMaterial = new THREE.PointsMaterial({
    color: ICE,
    size: 0.05,
    transparent: true,
    opacity: 0.75,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
    fog: true
  });
  const dust = new THREE.Points(dustGeometry, dustMaterial);
  scene.add(dust);

  // --- RYZE: halo and rings of light (the HTML card sits in front) -------------
  const haloTexture = drawHaloTexture();
  const halo = new THREE.Sprite(
    new THREE.SpriteMaterial({
      map: haloTexture,
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
      fog: false
    })
  );
  halo.position.set(0, 0, -1.5);
  halo.scale.setScalar(10);
  scene.add(halo);

  const ringGroup = new THREE.Group();
  const ringMaterial = new THREE.MeshBasicMaterial({
    color: CYAN,
    transparent: true,
    opacity: 0.85,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
    fog: false
  });
  const ringOuter = new THREE.Mesh(new THREE.TorusGeometry(3.45, 0.022, 8, 220), ringMaterial);
  ringOuter.rotation.set(1.18, 0.22, 0);
  const ringInnerMaterial = ringMaterial.clone();
  ringInnerMaterial.color.setHex(RING_BLUE);
  ringInnerMaterial.opacity = 0.9;
  const ringInner = new THREE.Mesh(new THREE.TorusGeometry(3.05, 0.016, 8, 200), ringInnerMaterial);
  ringInner.rotation.set(-1.02, -0.38, 0.4);
  ringGroup.add(ringOuter, ringInner);

  // Sparks travelling along the outer ring.
  const sparkCount = 26;
  const sparkPositions = new Float32Array(sparkCount * 3);
  const sparkGeometry = new THREE.BufferGeometry();
  sparkGeometry.setAttribute("position", new THREE.BufferAttribute(sparkPositions, 3));
  const sparkMaterial = new THREE.PointsMaterial({
    color: ICE,
    size: 0.11,
    transparent: true,
    opacity: 0.95,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
    fog: false
  });
  const sparks = new THREE.Points(sparkGeometry, sparkMaterial);
  ringOuter.add(sparks);
  scene.add(ringGroup);

  // --- Animation ------------------------------------------------------------------
  const pointer = { x: 0, y: 0 };
  const dummy = new THREE.Object3D();
  const clock = new THREE.Clock(false);
  let frame = 0;
  let running = false;
  let elapsed = 0;

  const updateCards = (delta: number) => {
    for (let i = 0; i < cardCount; i += 1) {
      const card = cardStates[i];
      card.z += CARD_SPEED * delta;
      card.angle += 0.035 * delta;
      if (card.z > TUNNEL_NEAR) {
        cardStates[i] = randomCard(TUNNEL_FAR);
        continue;
      }
      const x = Math.cos(card.angle) * card.radius;
      const y = Math.sin(card.angle) * card.radius * 0.72;
      dummy.position.set(x, y, card.z);
      // Each card turns slightly towards the tunnel axis, like a screen
      // glancing at the viewer as it passes.
      dummy.rotation.set(card.tilt - y * 0.035, -x * 0.06, card.spin * elapsed * 0.4);
      dummy.updateMatrix();
      cards.setMatrixAt(i, dummy.matrix);
    }
    cards.instanceMatrix.needsUpdate = true;
  };

  const updateDust = (delta: number) => {
    const positions = dustGeometry.attributes.position.array as Float32Array;
    for (let i = 0; i < dustCount; i += 1) {
      const index = i * 3 + 2;
      positions[index] += CARD_SPEED * 2.4 * delta;
      if (positions[index] > TUNNEL_NEAR) {
        positions[index] = TUNNEL_FAR;
      }
    }
    dustGeometry.attributes.position.needsUpdate = true;
  };

  const updateSparks = () => {
    for (let i = 0; i < sparkCount; i += 1) {
      const angle = (i / sparkCount) * Math.PI * 2 + elapsed * 0.55;
      const wobble = 1 + Math.sin(elapsed * 2 + i) * 0.004;
      sparkPositions[i * 3] = Math.cos(angle) * 3.45 * wobble;
      sparkPositions[i * 3 + 1] = Math.sin(angle) * 3.45 * wobble;
      sparkPositions[i * 3 + 2] = 0;
    }
    sparkGeometry.attributes.position.needsUpdate = true;
  };

  const render = () => {
    const delta = Math.min(clock.getDelta(), 0.05);
    elapsed += delta;

    updateCards(delta);
    updateDust(delta);
    updateSparks();

    ringOuter.rotation.z += delta * 0.35;
    ringInner.rotation.z -= delta * 0.5;
    ringGroup.rotation.y = Math.sin(elapsed * 0.3) * 0.12;
    const pulse = 1 + Math.sin(elapsed * 1.4) * 0.05;
    halo.scale.setScalar(10 * pulse);
    halo.material.opacity = 0.85 + Math.sin(elapsed * 1.4) * 0.15;

    // Gentle parallax towards the pointer.
    camera.position.x += (pointer.x * 0.9 - camera.position.x) * 0.04;
    camera.position.y += (-pointer.y * 0.6 - camera.position.y) * 0.04;
    camera.lookAt(0, 0, -4);

    renderer.render(scene, camera);
  };

  const loop = () => {
    if (!running) return;
    render();
    frame = window.requestAnimationFrame(loop);
  };

  // Lay out one frame before the first start, so the canvas never appears
  // empty.
  updateCards(0);
  updateSparks();

  return {
    setPointer: (x, y) => {
      pointer.x = Math.max(-1, Math.min(1, x));
      pointer.y = Math.max(-1, Math.min(1, y));
    },
    start: () => {
      if (running) return;
      running = true;
      clock.start();
      frame = window.requestAnimationFrame(loop);
    },
    stop: () => {
      running = false;
      clock.stop();
      window.cancelAnimationFrame(frame);
    },
    resize: (width, height) => {
      if (width <= 0 || height <= 0) return;
      renderer.setSize(width, height, false);
      camera.aspect = width / height;
      // Keep the tunnel framing similar on tall (portrait) stages.
      camera.fov = width / height < 0.9 ? 64 : 52;
      camera.updateProjectionMatrix();
      if (!running) {
        renderer.render(scene, camera);
      }
    },
    dispose: () => {
      running = false;
      window.cancelAnimationFrame(frame);
      cardGeometry.dispose();
      cardMaterial.dispose();
      cardTexture.dispose();
      dustGeometry.dispose();
      dustMaterial.dispose();
      haloTexture.dispose();
      halo.material.dispose();
      ringOuter.geometry.dispose();
      ringInner.geometry.dispose();
      ringMaterial.dispose();
      ringInnerMaterial.dispose();
      sparkGeometry.dispose();
      sparkMaterial.dispose();
      cards.dispose();
      renderer.dispose();
    }
  };
};
