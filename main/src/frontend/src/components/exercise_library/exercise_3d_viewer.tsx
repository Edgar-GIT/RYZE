import { Pause, Play } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import * as THREE from "three";

import styles from "./exercise_3d_viewer.module.css";

// The viewer implements one procedural scene per stored three_d_key. Scenes are
// built from primitives (no external assets), so the demonstration loads no
// media and carries no licensing constraints. Every scene animates a rig of
// jointed groups: the pose function below returns joint angles for a point in
// the loop, and the render loop applies them.

interface JointPose {
  rootY: number;
  rootRotX: number;
  torsoRotX: number;
  headRotX: number;
  thighL: number;
  thighR: number;
  shinL: number;
  shinR: number;
  footL: number;
  footR: number;
  shoulderL: number;
  shoulderR: number;
  elbowL: number;
  elbowR: number;
}

const BASE_POSE: JointPose = {
  rootY: 0.95,
  rootRotX: 0,
  torsoRotX: 0,
  headRotX: 0,
  thighL: 0,
  thighR: 0,
  shinL: 0,
  shinR: 0,
  footL: 0,
  footR: 0,
  shoulderL: 0,
  shoulderR: 0,
  elbowL: 0,
  elbowR: 0
};

type EquipmentKind = "barbell" | "dumbbells" | "none";

interface SceneDefinition {
  label: string;
  durationMs: number;
  cameraTargetY: number;
  equipment: EquipmentKind;
  pose: (t: number) => JointPose;
}

// Squat: a barbell back squat. t runs 0 (standing) → 1 (bottom) → 0.
const squatPose = (t: number): JointPose => {
  const depth = 0.5 - 0.5 * Math.cos(2 * Math.PI * t);
  return {
    ...BASE_POSE,
    rootY: 0.95 - 0.34 * depth,
    torsoRotX: -0.3 * depth,
    headRotX: 0.12 * depth,
    thighL: -1.05 * depth,
    thighR: -1.05 * depth,
    shinL: 1.85 * depth,
    shinR: 1.85 * depth,
    footL: -0.7 * depth,
    footR: -0.7 * depth,
    shoulderL: -1.35,
    shoulderR: -1.35,
    elbowL: -0.55,
    elbowR: -0.55
  };
};

// Push-up: prone body whose elbows flex and extend, lowering and raising the
// chest while the hands stay planted.
const pushUpPose = (t: number): JointPose => {
  const bend = 0.5 - 0.5 * Math.cos(2 * Math.PI * t);
  return {
    ...BASE_POSE,
    rootRotX: -Math.PI / 2,
    rootY: 0.62 - 0.2 * bend,
    torsoRotX: 0.05,
    headRotX: 0.35,
    thighL: 0.04,
    thighR: 0.04,
    shinL: 0,
    shinR: 0,
    footL: 0.55,
    footR: 0.55,
    shoulderL: 1.35,
    shoulderR: 1.35,
    elbowL: -(0.25 + 1.35 * bend),
    elbowR: -(0.25 + 1.35 * bend)
  };
};

// Biceps curl: standing, both elbows flex to bring the dumbbells up.
const bicepsCurlPose = (t: number): JointPose => {
  const curl = 0.5 - 0.5 * Math.cos(2 * Math.PI * t);
  return {
    ...BASE_POSE,
    shoulderL: -0.12,
    shoulderR: -0.12,
    elbowL: -2.05 * curl,
    elbowR: -2.05 * curl
  };
};

// Lunge: a stationary split-squat lunge; the forward knee drives over the toe
// while the back leg lowers toward the floor.
const lungePose = (t: number): JointPose => {
  const drop = 0.5 - 0.5 * Math.cos(2 * Math.PI * t);
  return {
    ...BASE_POSE,
    rootY: 0.95 - 0.2 * drop,
    torsoRotX: -0.12 * drop,
    thighL: -0.7 - 0.25 * drop,
    thighR: 0.35 + 0.1 * drop,
    shinL: 0.6 + 0.55 * drop,
    shinR: 0.35 + 0.4 * drop,
    footL: -0.25 * drop,
    footR: 0.3 + 0.15 * drop
  };
};

const SCENES: Record<string, SceneDefinition> = {
  squat: {
    label: "Barbell squat",
    durationMs: 2600,
    cameraTargetY: 0.85,
    equipment: "barbell",
    pose: squatPose
  },
  push_up: {
    label: "Push-up",
    durationMs: 2300,
    cameraTargetY: 0.45,
    equipment: "none",
    pose: pushUpPose
  },
  biceps_curl: {
    label: "Biceps curl",
    durationMs: 1900,
    cameraTargetY: 0.9,
    equipment: "dumbbells",
    pose: bicepsCurlPose
  },
  lunge: {
    label: "Walking lunge",
    durationMs: 2700,
    cameraTargetY: 0.8,
    equipment: "none",
    pose: lungePose
  }
};

export const isThreeDKeySupported = (key: string): boolean => Object.prototype.hasOwnProperty.call(SCENES, key);

export const supportsWebGl = (): boolean => {
  try {
    const canvas = document.createElement("canvas");
    return Boolean(canvas.getContext("webgl2") ?? canvas.getContext("webgl"));
  } catch {
    return false;
  }
};

interface Rig {
  root: THREE.Group;
  spine: THREE.Group;
  head: THREE.Group;
  thighL: THREE.Group;
  thighR: THREE.Group;
  shinL: THREE.Group;
  shinR: THREE.Group;
  footL: THREE.Group;
  footR: THREE.Group;
  upperArmL: THREE.Group;
  upperArmR: THREE.Group;
  forearmL: THREE.Group;
  forearmR: THREE.Group;
}

interface RigMaterials {
  skin: THREE.MeshStandardMaterial;
  shirt: THREE.MeshStandardMaterial;
  shorts: THREE.MeshStandardMaterial;
  metal: THREE.MeshStandardMaterial;
}

// Materials are created per mounted viewer so disposal on unmount never
// invalidates another instance.
const createMaterials = (): RigMaterials => ({
  skin: new THREE.MeshStandardMaterial({ color: 0xe4b28c, roughness: 0.75 }),
  shirt: new THREE.MeshStandardMaterial({ color: 0x2a2f3a, roughness: 0.6 }),
  shorts: new THREE.MeshStandardMaterial({ color: 0x1f242e, roughness: 0.65 }),
  metal: new THREE.MeshStandardMaterial({ color: 0x9aa4b2, roughness: 0.35, metalness: 0.75 })
});

const addCapsule = (
  parent: THREE.Object3D,
  radius: number,
  length: number,
  material: THREE.Material,
  position: [number, number, number]
): THREE.Mesh => {
  const mesh = new THREE.Mesh(new THREE.CapsuleGeometry(radius, length, 6, 12), material);
  mesh.position.set(position[0], position[1], position[2]);
  mesh.castShadow = true;
  parent.add(mesh);
  return mesh;
};

const addJoint = (parent: THREE.Object3D, position: [number, number, number]): THREE.Group => {
  const group = new THREE.Group();
  group.position.set(position[0], position[1], position[2]);
  parent.add(group);
  return group;
};

const buildRig = (materials: RigMaterials): Rig => {
  const root = new THREE.Group();

  const hips = addJoint(root, [0, 0, 0]);
  const pelvis = new THREE.Mesh(new THREE.CapsuleGeometry(0.13, 0.08, 6, 12), materials.shorts);
  pelvis.rotation.z = Math.PI / 2;
  pelvis.position.y = 0.02;
  pelvis.castShadow = true;
  hips.add(pelvis);

  const spine = addJoint(hips, [0, 0.08, 0]);
  addCapsule(spine, 0.16, 0.3, materials.shirt, [0, 0.24, 0]);

  const head = addJoint(spine, [0, 0.58, 0]);
  const neck = new THREE.Mesh(new THREE.CylinderGeometry(0.055, 0.06, 0.1, 12), materials.skin);
  neck.position.y = -0.03;
  neck.castShadow = true;
  head.add(neck);
  const skull = new THREE.Mesh(new THREE.SphereGeometry(0.115, 20, 16), materials.skin);
  skull.position.y = 0.1;
  skull.castShadow = true;
  head.add(skull);

  const thighL = addJoint(hips, [0.1, 0, 0]);
  addCapsule(thighL, 0.085, 0.28, materials.shorts, [0, -0.23, 0]);
  const shinL = addJoint(thighL, [0, -0.45, 0]);
  addCapsule(shinL, 0.07, 0.3, materials.skin, [0, -0.22, 0]);
  const footL = addJoint(shinL, [0, -0.45, 0]);
  addCapsule(footL, 0.05, 0.1, materials.metal, [0, -0.05, 0.05]).rotation.x = Math.PI / 2;

  const thighR = addJoint(hips, [-0.1, 0, 0]);
  addCapsule(thighR, 0.085, 0.28, materials.shorts, [0, -0.23, 0]);
  const shinR = addJoint(thighR, [0, -0.45, 0]);
  addCapsule(shinR, 0.07, 0.3, materials.skin, [0, -0.22, 0]);
  const footR = addJoint(shinR, [0, -0.45, 0]);
  addCapsule(footR, 0.05, 0.1, materials.metal, [0, -0.05, 0.05]).rotation.x = Math.PI / 2;

  const upperArmL = addJoint(spine, [0.19, 0.42, 0]);
  addCapsule(upperArmL, 0.055, 0.18, materials.shirt, [0, -0.14, 0]);
  const forearmL = addJoint(upperArmL, [0, -0.28, 0]);
  addCapsule(forearmL, 0.048, 0.17, materials.skin, [0, -0.13, 0]);
  addJoint(forearmL, [0, -0.26, 0]);

  const upperArmR = addJoint(spine, [-0.19, 0.42, 0]);
  addCapsule(upperArmR, 0.055, 0.18, materials.shirt, [0, -0.14, 0]);
  const forearmR = addJoint(upperArmR, [0, -0.28, 0]);
  addCapsule(forearmR, 0.048, 0.17, materials.skin, [0, -0.13, 0]);
  addJoint(forearmR, [0, -0.26, 0]);

  return { root, spine, head, thighL, thighR, shinL, shinR, footL, footR, upperArmL, upperArmR, forearmL, forearmR };
};

const attachEquipment = (rig: Rig, kind: EquipmentKind, materials: RigMaterials): void => {
  if (kind === "barbell") {
    const bar = new THREE.Mesh(new THREE.CylinderGeometry(0.022, 0.022, 1.3, 10), materials.metal);
    bar.rotation.z = Math.PI / 2;
    bar.position.set(0, 0.5, -0.14);
    bar.castShadow = true;
    rig.spine.add(bar);

    for (const side of [1, -1]) {
      for (const offset of [0.5, 0.62]) {
        const plate = new THREE.Mesh(new THREE.CylinderGeometry(0.14, 0.14, 0.045, 18), materials.metal);
        plate.rotation.z = Math.PI / 2;
        plate.position.set(side * offset, 0.5, -0.14);
        plate.castShadow = true;
        rig.spine.add(plate);
      }
    }
  }

  if (kind === "dumbbells") {
    for (const forearm of [rig.forearmL, rig.forearmR]) {
      const handle = new THREE.Mesh(new THREE.CylinderGeometry(0.018, 0.018, 0.16, 8), materials.metal);
      handle.rotation.z = Math.PI / 2;
      handle.position.y = -0.3;
      handle.castShadow = true;
      forearm.add(handle);

      for (const side of [1, -1]) {
        const weight = new THREE.Mesh(new THREE.CylinderGeometry(0.055, 0.055, 0.05, 14), materials.metal);
        weight.rotation.z = Math.PI / 2;
        weight.position.set(side * 0.09, -0.3, 0);
        weight.castShadow = true;
        forearm.add(weight);
      }
    }
  }
};

const applyPose = (rig: Rig, pose: JointPose): void => {
  rig.root.position.y = pose.rootY;
  rig.root.rotation.x = pose.rootRotX;
  rig.spine.rotation.x = pose.torsoRotX;
  rig.head.rotation.x = pose.headRotX;
  rig.thighL.rotation.x = pose.thighL;
  rig.thighR.rotation.x = pose.thighR;
  rig.shinL.rotation.x = pose.shinL;
  rig.shinR.rotation.x = pose.shinR;
  rig.footL.rotation.x = pose.footL;
  rig.footR.rotation.x = pose.footR;
  rig.upperArmL.rotation.x = pose.shoulderL;
  rig.upperArmR.rotation.x = pose.shoulderR;
  rig.forearmL.rotation.x = pose.elbowL;
  rig.forearmR.rotation.x = pose.elbowR;
};

interface Exercise3DViewerProps {
  threeDKey: string;
  exerciseName: string;
}

const SPEEDS = [0.5, 1, 1.5] as const;

// Exercise3DViewer is the heavy scene component. It is loaded through a
// dynamic import by Exercise3DDemo so Three.js is only ever downloaded when a
// client actually opens a 3D demonstration.
const Exercise3DViewer = ({ threeDKey, exerciseName }: Exercise3DViewerProps) => {
  const sceneDefinition = SCENES[threeDKey] ?? null;
  const mountRef = useRef<HTMLDivElement>(null);
  const [playing, setPlaying] = useState(true);
  const [speed, setSpeed] = useState<(typeof SPEEDS)[number]>(1);
  const [unsupported, setUnsupported] = useState(() => !supportsWebGl());

  const playingRef = useRef(playing);
  const speedRef = useRef(speed);
  playingRef.current = playing;
  speedRef.current = speed;

  useEffect(() => {
    const mount = mountRef.current;
    if (!sceneDefinition || unsupported || !mount) {
      return;
    }

    let renderer: THREE.WebGLRenderer;
    try {
      renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    } catch {
      setUnsupported(true);
      return;
    }

    const width = mount.clientWidth || 640;
    const height = mount.clientHeight || 360;
    renderer.setSize(width, height);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    renderer.shadowMap.enabled = true;
    renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    mount.appendChild(renderer.domElement);

    const scene = new THREE.Scene();
    scene.background = null;

    const camera = new THREE.PerspectiveCamera(42, width / height, 0.1, 50);
    const target = new THREE.Vector3(0, sceneDefinition.cameraTargetY, 0);
    const cameraState = { yaw: 0.6, pitch: 0.18, radius: 3.1 };

    const updateCamera = (): void => {
      const cosPitch = Math.cos(cameraState.pitch);
      camera.position.set(
        target.x + cameraState.radius * cosPitch * Math.sin(cameraState.yaw),
        target.y + cameraState.radius * Math.sin(cameraState.pitch),
        target.z + cameraState.radius * cosPitch * Math.cos(cameraState.yaw)
      );
      camera.lookAt(target);
    };
    updateCamera();

    scene.add(new THREE.HemisphereLight(0xdfe8ff, 0x1a1d24, 1.15));
    const keyLight = new THREE.DirectionalLight(0xffffff, 2.1);
    keyLight.position.set(2.4, 4.2, 2.6);
    keyLight.castShadow = true;
    keyLight.shadow.mapSize.set(1024, 1024);
    scene.add(keyLight);
    const rimLight = new THREE.DirectionalLight(0x18d8ff, 0.55);
    rimLight.position.set(-2.6, 2.2, -2.4);
    scene.add(rimLight);

    const floor = new THREE.Mesh(
      new THREE.CircleGeometry(2.6, 48),
      new THREE.MeshStandardMaterial({ color: 0x14171d, roughness: 0.95 })
    );
    floor.rotation.x = -Math.PI / 2;
    floor.receiveShadow = true;
    scene.add(floor);

    const grid = new THREE.GridHelper(5.2, 20, 0x18d8ff, 0x2a2f3a);
    grid.position.y = 0.002;
    (grid.material as THREE.Material).opacity = 0.22;
    (grid.material as THREE.Material).transparent = true;
    scene.add(grid);

    const materials = createMaterials();
    const rig = buildRig(materials);
    attachEquipment(rig, sceneDefinition.equipment, materials);
    scene.add(rig.root);
    applyPose(rig, sceneDefinition.pose(0));

    let disposed = false;
    let frame = 0;
    let phase = 0;
    let lastTime = performance.now();

    const renderFrame = (time: number): void => {
      if (disposed) {
        return;
      }
      const delta = Math.min(64, time - lastTime);
      lastTime = time;
      if (playingRef.current) {
        phase = (phase + (delta * speedRef.current) / sceneDefinition.durationMs) % 1;
      }
      applyPose(rig, sceneDefinition.pose(phase));
      renderer.render(scene, camera);
      frame = window.requestAnimationFrame(renderFrame);
    };
    frame = window.requestAnimationFrame(renderFrame);

    const resize = (): void => {
      const nextWidth = mount.clientWidth || width;
      const nextHeight = mount.clientHeight || height;
      renderer.setSize(nextWidth, nextHeight);
      camera.aspect = nextWidth / nextHeight;
      camera.updateProjectionMatrix();
    };
    const resizeObserver = new ResizeObserver(resize);
    resizeObserver.observe(mount);

    let dragging = false;
    let lastX = 0;
    let lastY = 0;
    const canvas = renderer.domElement;
    canvas.classList.add(styles.canvas);

    const onPointerDown = (event: PointerEvent): void => {
      dragging = true;
      lastX = event.clientX;
      lastY = event.clientY;
      canvas.setPointerCapture(event.pointerId);
    };
    const onPointerMove = (event: PointerEvent): void => {
      if (!dragging) {
        return;
      }
      cameraState.yaw -= (event.clientX - lastX) * 0.008;
      cameraState.pitch = Math.max(-0.25, Math.min(1.1, cameraState.pitch + (event.clientY - lastY) * 0.006));
      lastX = event.clientX;
      lastY = event.clientY;
      updateCamera();
    };
    const onPointerUp = (): void => {
      dragging = false;
    };
    const onWheel = (event: WheelEvent): void => {
      event.preventDefault();
      cameraState.radius = Math.max(1.7, Math.min(6, cameraState.radius + event.deltaY * 0.003));
      updateCamera();
    };

    canvas.addEventListener("pointerdown", onPointerDown);
    canvas.addEventListener("pointermove", onPointerMove);
    canvas.addEventListener("pointerup", onPointerUp);
    canvas.addEventListener("pointercancel", onPointerUp);
    canvas.addEventListener("wheel", onWheel, { passive: false });

    return () => {
      disposed = true;
      window.cancelAnimationFrame(frame);
      resizeObserver.disconnect();
      canvas.removeEventListener("pointerdown", onPointerDown);
      canvas.removeEventListener("pointermove", onPointerMove);
      canvas.removeEventListener("pointerup", onPointerUp);
      canvas.removeEventListener("pointercancel", onPointerUp);
      canvas.removeEventListener("wheel", onWheel);
      scene.traverse((object) => {
        if (object instanceof THREE.Mesh || object instanceof THREE.LineSegments) {
          object.geometry.dispose();
          const materials = Array.isArray(object.material) ? object.material : [object.material];
          materials.forEach((material) => material.dispose());
        }
      });
      renderer.dispose();
      if (canvas.parentElement === mount) {
        mount.removeChild(canvas);
      }
    };
  }, [sceneDefinition, unsupported]);

  if (!sceneDefinition) {
    return <p className={styles.fallback}>This 3D demonstration is not available yet.</p>;
  }

  if (unsupported) {
    return (
      <p className={styles.fallback} role="status">
        3D previews are not supported on this device. The written instructions above describe the full movement.
      </p>
    );
  }

  return (
    <div className={styles.viewer}>
      <div
        ref={mountRef}
        className={styles.stage}
        role="img"
        aria-label={`3D demonstration of ${exerciseName}`}
      />
      <div className={styles.controls}>
        <button type="button" className={styles.controlButton} onClick={() => setPlaying((value) => !value)}>
          {playing ? <Pause size={14} aria-hidden="true" /> : <Play size={14} aria-hidden="true" />}
          {playing ? "Pause" : "Play"}
        </button>
        <label className={styles.speedControl}>
          <span>Speed</span>
          <select
            value={speed}
            onChange={(event) => setSpeed(Number(event.target.value) as (typeof SPEEDS)[number])}
            className={styles.speedSelect}
          >
            {SPEEDS.map((option) => (
              <option key={option} value={option}>
                {option}×
              </option>
            ))}
          </select>
        </label>
        <span className={styles.hint}>Drag to rotate · Scroll to zoom</span>
      </div>
    </div>
  );
};

export default Exercise3DViewer;
