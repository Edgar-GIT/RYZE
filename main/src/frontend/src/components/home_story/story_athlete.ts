import * as THREE from "three";

/*
 * The hologram athlete (owner request, 2026-11-15): a faceless, genderless
 * training mannequin with a real anatomical form that cycles through dumbbell
 * exercises — biceps curl, shoulder press, squat, Romanian deadlift — while the
 * muscles being worked pulse gold to red.
 *
 * The body is one continuous surface, not parts glued together: head, torso,
 * pelvis, muscles and limbs are signed distance fields joined with smooth
 * unions, turned into a mesh once (surface nets on an even grid, vertices
 * projected onto the exact surface; no model files, no WebAssembly). A
 * 17-bone skeleton skins it, with weights from the same fields, so shoulders,
 * elbows, hips and knees bend. A light network — evenly spaced nodes on the
 * skin joined to their neighbours — is drawn over the body, like a scan.
 *
 * Body space: the floor is y = 0, the athlete is about 3.4 units tall and
 * faces +z.
 */

export interface AthleteFrame {
  elapsed: number;
  emphasis: number; // 1 on the hero, lower while the hologram waits
  presence: number; // 0..1 while the hologram materialises
  scanY: number; // world y of the scanner's line
  scanDir: number; // +1 while the line rises, -1 while it falls
  reveal: number; // world y up to which the hologram is drawn
  glitch: number;
  glitchY: number;
  mirror: THREE.Matrix4; // reflects world points across the platform's top
  floorInverse: THREE.Matrix4; // world → platform space (y up from its top)
}

// What the athlete is doing, for the scanner's readout. Each rep stands for a
// logged set (the readout is a sped-up picture of a session, not a measurement).
export interface AthleteState {
  stimulus: number; // 0..1
  exercise: number; // index in the routine
  sets: number; // reps (sets) completed in this exercise so far (0..ROUTINE_SETS)
  rest: number; // 0 while working; 0..1 through the change to the next exercise
  time: number; // seconds into this exercise
  round: number; // how many times the whole routine has already run
  muscle: THREE.Vector3; // world position of the main worked muscle, on the side towards the readout (left)
}

export interface Athlete {
  group: THREE.Group;
  update: (frame: AthleteFrame) => AthleteState;
}

type Track = <T extends { dispose: () => void }>(item: T) => T;
type Vec = [number, number, number];

// --- Skeleton -----------------------------------------------------------------

const REST_ARM = THREE.MathUtils.degToRad(28); // A-pose: arms 28° from the body
const UPPER_ARM = 0.55;
const FOREARM = 0.49;
const SHOULDER_X = 0.38;

const BONE_NAMES = [
  "hips",
  "spine",
  "chest",
  "neck",
  "head",
  "upperArmL",
  "foreArmL",
  "handL",
  "upperArmR",
  "foreArmR",
  "handR",
  "upperLegL",
  "lowerLegL",
  "footL",
  "upperLegR",
  "lowerLegR",
  "footR"
] as const;
type BoneName = (typeof BONE_NAMES)[number];
const B = Object.fromEntries(BONE_NAMES.map((name, index) => [name, index])) as Record<BoneName, number>;

const PARENTS: Record<BoneName, BoneName | null> = {
  hips: null,
  spine: "hips",
  chest: "spine",
  neck: "chest",
  head: "neck",
  upperArmL: "chest",
  foreArmL: "upperArmL",
  handL: "foreArmL",
  upperArmR: "chest",
  foreArmR: "upperArmR",
  handR: "foreArmR",
  upperLegL: "hips",
  lowerLegL: "upperLegL",
  footL: "lowerLegL",
  upperLegR: "hips",
  lowerLegR: "upperLegR",
  footR: "lowerLegR"
};

// Muscle regions: A = biceps, deltoids, quads, glutes; B = pecs, triceps, calves, hamstrings.
const REGIONS = ["biceps", "deltoids", "quads", "glutes", "pecs", "triceps", "calves", "hamstrings"] as const;
type Region = (typeof REGIONS)[number];
const M = Object.fromEntries(REGIONS.map((name, index) => [name, index])) as Record<Region, number>;

const add = (a: Vec, b: Vec): Vec => [a[0] + b[0], a[1] + b[1], a[2] + b[2]];
const sub = (a: Vec, b: Vec): Vec => [a[0] - b[0], a[1] - b[1], a[2] - b[2]];
const scale = (a: Vec, s: number): Vec => [a[0] * s, a[1] * s, a[2] * s];
const length = (a: Vec) => Math.hypot(a[0], a[1], a[2]);

const SIDES = [
  [1, "L"],
  [-1, "R"]
] as const;

// The arm's direction at rest, and the elbow's hinge axis (across the arm, in
// the body's plane), per side.
const armDir = (side: number): Vec => [side * Math.sin(REST_ARM), -Math.cos(REST_ARM), 0];
const hingeAxis = (side: number): Vec => [Math.cos(REST_ARM), side * Math.sin(REST_ARM), 0];

const JOINTS = (() => {
  const joints = {
    hips: [0, 1.72, 0],
    spine: [0, 1.98, 0],
    chest: [0, 2.36, 0],
    neck: [0, 2.82, 0],
    head: [0, 3.0, 0.01]
  } as Record<BoneName, Vec>;
  for (const [side, s] of SIDES) {
    const shoulder: Vec = [side * SHOULDER_X, 2.69, -0.01];
    const elbow = add(shoulder, scale(armDir(side), UPPER_ARM));
    joints[`upperArm${s}`] = shoulder;
    joints[`foreArm${s}`] = elbow;
    joints[`hand${s}`] = add(elbow, scale(armDir(side), FOREARM));
    joints[`upperLeg${s}`] = [side * 0.16, 1.72, 0];
    joints[`lowerLeg${s}`] = [side * 0.155, 0.96, 0.02];
    joints[`foot${s}`] = [side * 0.15, 0.12, -0.01];
  }
  return joints;
})();

// --- Distance fields --------------------------------------------------------

interface Primitive {
  kind: 0 | 1; // 0 ellipsoid, 1 round cone
  c: Vec; // ellipsoid centre
  r: Vec; // ellipsoid radii
  inv: number[]; // inverse of the ellipsoid's rotation (row-major 3x3)
  a: Vec; // round cone from a (radius r1) to b (radius r2)
  b: Vec;
  r1: number;
  r2: number;
  bone: number;
  region: number;
  k: number; // smooth-union radius
  bc: Vec; // bounding sphere
  br: number;
  lo: Vec; // bounding box
  hi: Vec;
}

const ellipsoid = (c: Vec, r: Vec, bone: number, k: number, region = -1, rotZ = 0, rotX = 0): Primitive => {
  const e = new THREE.Matrix4().makeRotationFromEuler(new THREE.Euler(rotX, 0, rotZ)).invert().elements;
  return {
    kind: 0,
    c,
    r,
    inv: [e[0], e[4], e[8], e[1], e[5], e[9], e[2], e[6], e[10]],
    a: c,
    b: c,
    r1: 0,
    r2: 0,
    bone,
    region,
    k,
    bc: c,
    br: Math.max(...r),
    lo: sub(c, [Math.max(...r), Math.max(...r), Math.max(...r)]),
    hi: add(c, [Math.max(...r), Math.max(...r), Math.max(...r)])
  };
};

const cone = (a: Vec, b: Vec, r1: number, r2: number, bone: number, k: number, region = -1): Primitive => {
  const centre = scale(add(a, b), 0.5);
  return {
    kind: 1,
    c: centre,
    r: [0, 0, 0],
    inv: [],
    a,
    b,
    r1,
    r2,
    bone,
    region,
    k,
    bc: centre,
    br: length(sub(b, a)) / 2 + Math.max(r1, r2),
    lo: [0, 1, 2].map((axis) => Math.min(a[axis], b[axis]) - Math.max(r1, r2)) as Vec,
    hi: [0, 1, 2].map((axis) => Math.max(a[axis], b[axis]) + Math.max(r1, r2)) as Vec
  };
};

// Inigo Quilez's ellipsoid bound and round cone.
const sdEllipsoid = (p: Primitive, x: number, y: number, z: number) => {
  const dx = x - p.c[0];
  const dy = y - p.c[1];
  const dz = z - p.c[2];
  const m = p.inv;
  const qx = (m[0] * dx + m[1] * dy + m[2] * dz) / p.r[0];
  const qy = (m[3] * dx + m[4] * dy + m[5] * dz) / p.r[1];
  const qz = (m[6] * dx + m[7] * dy + m[8] * dz) / p.r[2];
  const k0 = Math.sqrt(qx * qx + qy * qy + qz * qz);
  const k1 = Math.sqrt((qx * qx) / (p.r[0] * p.r[0]) + (qy * qy) / (p.r[1] * p.r[1]) + (qz * qz) / (p.r[2] * p.r[2]));
  return k1 < 1e-9 ? -Math.min(...p.r) : (k0 * (k0 - 1)) / k1;
};

const sdRoundCone = (p: Primitive, x: number, y: number, z: number) => {
  const bax = p.b[0] - p.a[0];
  const bay = p.b[1] - p.a[1];
  const baz = p.b[2] - p.a[2];
  const l2 = bax * bax + bay * bay + baz * baz;
  const rr = p.r1 - p.r2;
  const a2 = l2 - rr * rr;
  const il2 = 1 / l2;
  const pax = x - p.a[0];
  const pay = y - p.a[1];
  const paz = z - p.a[2];
  const yy = pax * bax + pay * bay + paz * baz;
  const zz = yy - l2;
  const wx = pax * l2 - bax * yy;
  const wy = pay * l2 - bay * yy;
  const wz = paz * l2 - baz * yy;
  const x2 = wx * wx + wy * wy + wz * wz;
  const y2 = yy * yy * l2;
  const z2 = zz * zz * l2;
  const k = Math.sign(rr) * rr * rr * x2;
  if (Math.sign(zz) * a2 * z2 > k) return Math.sqrt(x2 + z2) * il2 - p.r2;
  if (Math.sign(yy) * a2 * y2 < k) return Math.sqrt(x2 + y2) * il2 - p.r1;
  return (Math.sqrt(x2 * a2 * il2) + yy * rr) * il2 - p.r1;
};

const distance = (p: Primitive, x: number, y: number, z: number) => (p.kind === 0 ? sdEllipsoid(p, x, y, z) : sdRoundCone(p, x, y, z));

const smin = (a: number, b: number, k: number) => {
  const h = Math.max(k - Math.abs(a - b), 0) / k;
  return Math.min(a, b) - h * h * k * 0.25;
};

const buildPrimitives = (): Primitive[] => {
  const j = JOINTS;
  const list: Primitive[] = [
    // Pelvis, glutes, waist, ribs, chest and pecs, the shoulder line, lats.
    ellipsoid([0, 1.8, -0.01], [0.255, 0.17, 0.165], B.hips, 0.09),
    ellipsoid([0.105, 1.71, -0.085], [0.14, 0.15, 0.125], B.hips, 0.06, M.glutes),
    ellipsoid([-0.105, 1.71, -0.085], [0.14, 0.15, 0.125], B.hips, 0.06, M.glutes),
    ellipsoid([0, 2.03, 0], [0.215, 0.22, 0.145], B.spine, 0.1),
    ellipsoid([0, 2.25, 0], [0.25, 0.2, 0.16], B.chest, 0.1),
    ellipsoid([0, 2.46, 0], [0.32, 0.25, 0.185], B.chest, 0.1),
    ellipsoid([0.125, 2.49, 0.105], [0.145, 0.098, 0.082], B.chest, 0.05, M.pecs, 0.18),
    ellipsoid([-0.125, 2.49, 0.105], [0.145, 0.098, 0.082], B.chest, 0.05, M.pecs, -0.18),
    cone([-0.28, 2.69, -0.035], [0.28, 2.69, -0.035], 0.1, 0.1, B.chest, 0.08),
    ellipsoid([0.225, 2.35, -0.04], [0.105, 0.21, 0.115], B.chest, 0.07, -1, 0.12),
    ellipsoid([-0.225, 2.35, -0.04], [0.105, 0.21, 0.115], B.chest, 0.07, -1, -0.12),
    // Neck and a faceless head: a cranium and a soft jaw, no features.
    cone([0, 2.76, -0.005], [0, 3.0, 0.01], 0.08, 0.068, B.neck, 0.06),
    ellipsoid([0, 3.17, -0.005], [0.148, 0.18, 0.168], B.head, 0.05),
    ellipsoid([0, 3.07, 0.045], [0.105, 0.1, 0.11], B.head, 0.07)
  ];
  for (const [side, s] of SIDES) {
    const dir = armDir(side);
    const shoulder = j[`upperArm${s}`];
    const elbow = j[`foreArm${s}`];
    const wrist = j[`hand${s}`];
    const tilt = side * REST_ARM;
    list.push(
      ellipsoid(add(shoulder, [side * 0.03, 0.005, 0]), [0.125, 0.13, 0.12], B[`upperArm${s}`], 0.06, M.deltoids, tilt),
      cone(shoulder, elbow, 0.084, 0.06, B[`upperArm${s}`], 0.04),
      ellipsoid(add(add(shoulder, scale(dir, 0.3)), [0, 0, 0.032]), [0.07, 0.135, 0.068], B[`upperArm${s}`], 0.035, M.biceps, tilt),
      ellipsoid(add(add(shoulder, scale(dir, 0.27)), [0, 0, -0.038]), [0.068, 0.15, 0.066], B[`upperArm${s}`], 0.035, M.triceps, tilt),
      cone(elbow, wrist, 0.064, 0.042, B[`foreArm${s}`], 0.035),
      ellipsoid(add(elbow, scale(dir, 0.13)), [0.066, 0.125, 0.061], B[`foreArm${s}`], 0.035, -1, tilt),
      // A mitten hand: palm and closed fingers in one volume.
      ellipsoid(add(wrist, scale(dir, 0.095)), [0.042, 0.09, 0.056], B[`hand${s}`], 0.03, -1, tilt)
    );
    const hip = j[`upperLeg${s}`];
    const knee = j[`lowerLeg${s}`];
    const ankle = j[`foot${s}`];
    const thighMid = scale(add(hip, knee), 0.5);
    list.push(
      cone(hip, knee, 0.135, 0.08, B[`upperLeg${s}`], 0.05),
      ellipsoid(add(thighMid, [side * 0.01, 0.05, 0.045]), [0.105, 0.24, 0.09], B[`upperLeg${s}`], 0.04, M.quads),
      ellipsoid(add(thighMid, [0, 0.06, -0.048]), [0.095, 0.22, 0.083], B[`upperLeg${s}`], 0.04, M.hamstrings),
      ellipsoid(add(knee, [0, 0, 0.015]), [0.07, 0.066, 0.068], B[`lowerLeg${s}`], 0.04),
      cone(knee, ankle, 0.072, 0.045, B[`lowerLeg${s}`], 0.035),
      ellipsoid(add(knee, [0, -0.24, -0.043]), [0.078, 0.16, 0.073], B[`lowerLeg${s}`], 0.035, M.calves),
      cone(add(ankle, [0, -0.06, -0.05]), [ankle[0], 0.045, 0.23], 0.05, 0.039, B[`foot${s}`], 0.04)
    );
  }
  return list;
};

const fieldAt = (prims: Primitive[], x: number, y: number, z: number) => {
  let d = 1e9;
  for (const p of prims) {
    const dx = x - p.bc[0];
    const dy = y - p.bc[1];
    const dz = z - p.bc[2];
    // Farther than the blend reach from this primitive's bounding sphere.
    if (Math.sqrt(dx * dx + dy * dy + dz * dz) - p.br > d + p.k) continue;
    d = smin(d, distance(p, x, y, z), p.k);
  }
  return d;
};

const gradientAt = (prims: Primitive[], x: number, y: number, z: number, out: Vec) => {
  const h = 0.003;
  out[0] = fieldAt(prims, x + h, y, z) - fieldAt(prims, x - h, y, z);
  out[1] = fieldAt(prims, x, y + h, z) - fieldAt(prims, x, y - h, z);
  out[2] = fieldAt(prims, x, y, z + h) - fieldAt(prims, x, y, z - h);
  const l = length(out) || 1;
  out[0] /= l;
  out[1] /= l;
  out[2] /= l;
  return out;
};

// Lets the page breathe while the body is built.
const breather = () => {
  let last = performance.now();
  return async () => {
    if (performance.now() - last < 12) return;
    await new Promise<void>((resolve) => setTimeout(resolve, 0));
    last = performance.now();
  };
};

// Surface nets: one vertex per grid cell the surface crosses (projected onto
// the exact surface), one quad per grid edge it crosses.
const buildSurface = async (prims: Primitive[], cell: number) => {
  const breathe = breather();
  const min: Vec = [Infinity, Infinity, Infinity];
  const max: Vec = [-Infinity, -Infinity, -Infinity];
  for (const p of prims) {
    for (let axis = 0; axis < 3; axis += 1) {
      min[axis] = Math.min(min[axis], p.lo[axis] - cell * 2);
      max[axis] = Math.max(max[axis], p.hi[axis] + cell * 2);
    }
  }
  const nx = Math.ceil((max[0] - min[0]) / cell) + 1;
  const ny = Math.ceil((max[1] - min[1]) / cell) + 1;
  const nz = Math.ceil((max[2] - min[2]) / cell) + 1;
  const field = new Float32Array(nx * ny * nz);
  const at = (i: number, j: number, k: number) => i + nx * (j + ny * k);
  for (let k = 0; k < nz; k += 1) {
    for (let j = 0; j < ny; j += 1) {
      for (let i = 0; i < nx; i += 1) {
        field[at(i, j, k)] = fieldAt(prims, min[0] + i * cell, min[1] + j * cell, min[2] + k * cell);
      }
    }
    await breathe();
  }

  const cx = nx - 1;
  const cy = ny - 1;
  const cellIndex = new Int32Array(cx * cy * (nz - 1)).fill(-1);
  const cellAt = (i: number, j: number, k: number) => i + cx * (j + cy * k);
  const positions: number[] = [];
  const corner = new Float32Array(8);
  // The 12 cube edges as pairs of corner ids (bit 0: +x, bit 1: +y, bit 2: +z).
  const EDGES = [0, 1, 2, 3, 4, 5, 6, 7, 0, 2, 1, 3, 4, 6, 5, 7, 0, 4, 1, 5, 2, 6, 3, 7];
  for (let k = 0; k < nz - 1; k += 1) {
    for (let j = 0; j < ny - 1; j += 1) {
      for (let i = 0; i < nx - 1; i += 1) {
        let inside = 0;
        for (let c = 0; c < 8; c += 1) {
          corner[c] = field[at(i + (c & 1), j + ((c >> 1) & 1), k + ((c >> 2) & 1))];
          if (corner[c] < 0) inside += 1;
        }
        if (inside === 0 || inside === 8) continue;
        let sx = 0;
        let sy = 0;
        let sz = 0;
        let crossings = 0;
        for (let e = 0; e < 24; e += 2) {
          const a = EDGES[e];
          const b = EDGES[e + 1];
          if (corner[a] < 0 === corner[b] < 0) continue;
          const t = corner[a] / (corner[a] - corner[b]);
          sx += (a & 1) + t * ((b & 1) - (a & 1));
          sy += ((a >> 1) & 1) + t * (((b >> 1) & 1) - ((a >> 1) & 1));
          sz += ((a >> 2) & 1) + t * (((b >> 2) & 1) - ((a >> 2) & 1));
          crossings += 1;
        }
        cellIndex[cellAt(i, j, k)] = positions.length / 3;
        positions.push(min[0] + (i + sx / crossings) * cell, min[1] + (j + sy / crossings) * cell, min[2] + (k + sz / crossings) * cell);
      }
    }
    await breathe();
  }

  const indices: number[] = [];
  const quad = (a: number, b: number, c: number, d: number, flip: boolean) => {
    if (flip) [b, d] = [d, b];
    // Split along the shorter diagonal.
    const ac = (positions[a * 3] - positions[c * 3]) ** 2 + (positions[a * 3 + 1] - positions[c * 3 + 1]) ** 2 + (positions[a * 3 + 2] - positions[c * 3 + 2]) ** 2;
    const bd = (positions[b * 3] - positions[d * 3]) ** 2 + (positions[b * 3 + 1] - positions[d * 3 + 1]) ** 2 + (positions[b * 3 + 2] - positions[d * 3 + 2]) ** 2;
    if (ac <= bd) indices.push(a, b, c, a, c, d);
    else indices.push(a, b, d, b, c, d);
  };
  for (let k = 0; k < nz - 1; k += 1) {
    for (let j = 0; j < ny - 1; j += 1) {
      for (let i = 0; i < nx - 1; i += 1) {
        const here = field[at(i, j, k)] < 0;
        // An edge along x is shared by the cells around it in y and z, and so on.
        if (j > 0 && k > 0 && here !== field[at(i + 1, j, k)] < 0) {
          quad(cellIndex[cellAt(i, j - 1, k - 1)], cellIndex[cellAt(i, j, k - 1)], cellIndex[cellAt(i, j, k)], cellIndex[cellAt(i, j - 1, k)], !here);
        }
        if (i > 0 && k > 0 && here !== field[at(i, j + 1, k)] < 0) {
          quad(cellIndex[cellAt(i - 1, j, k - 1)], cellIndex[cellAt(i - 1, j, k)], cellIndex[cellAt(i, j, k)], cellIndex[cellAt(i, j, k - 1)], !here);
        }
        if (i > 0 && j > 0 && here !== field[at(i, j, k + 1)] < 0) {
          quad(cellIndex[cellAt(i - 1, j - 1, k)], cellIndex[cellAt(i, j - 1, k)], cellIndex[cellAt(i, j, k)], cellIndex[cellAt(i - 1, j, k)], !here);
        }
      }
    }
    await breathe();
  }

  // Project every vertex onto the exact surface and take its normal there.
  const count = positions.length / 3;
  const position = new Float32Array(positions);
  const normal = new Float32Array(count * 3);
  const g: Vec = [0, 0, 0];
  for (let index = 0; index < count; index += 1) {
    let x = position[index * 3];
    let y = position[index * 3 + 1];
    let z = position[index * 3 + 2];
    for (let step = 0; step < 2; step += 1) {
      const d = THREE.MathUtils.clamp(fieldAt(prims, x, y, z), -cell * 0.5, cell * 0.5);
      gradientAt(prims, x, y, z, g);
      x -= g[0] * d;
      y -= g[1] * d;
      z -= g[2] * d;
    }
    gradientAt(prims, x, y, z, g);
    position.set([x, y, z], index * 3);
    normal.set(g, index * 3);
    if (index % 2000 === 0) await breathe();
  }

  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute("position", new THREE.BufferAttribute(position, 3));
  geometry.setAttribute("normal", new THREE.BufferAttribute(normal, 3));
  geometry.setIndex(indices);
  return geometry;
};

// Two bones per point, blended where two bones' volumes meet, and how close
// the point is to each muscle region.
const weigh = (prims: Primitive[], x: number, y: number, z: number) => {
  const BLEND = 0.09;
  let best = Infinity;
  let bestBone = 0;
  let second = Infinity;
  let secondBone = 0;
  const regions = new Array<number>(8).fill(Infinity);
  for (const p of prims) {
    const d = distance(p, x, y, z);
    if (p.region >= 0) regions[p.region] = Math.min(regions[p.region], d);
    if (p.bone === bestBone) {
      best = Math.min(best, d);
    } else if (d < best) {
      second = best;
      secondBone = bestBone;
      best = d;
      bestBone = p.bone;
    } else if (d < second) {
      second = d;
      secondBone = p.bone;
    }
  }
  const blend = second === Infinity ? 0 : Math.max(0, 0.5 - (second - best) / (2 * BLEND));
  const muscles = regions.map((d) => 1 - THREE.MathUtils.smoothstep(d, -0.01, 0.07));
  return { bones: [bestBone, secondBone], weights: [1 - blend, blend], muscles };
};

const random = (seed: number) => () => {
  seed = (seed + 0x6d2b79f5) | 0;
  let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
  t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
};

// Evenly spaced nodes on the skin (Poisson disc over the surface's vertices),
// each joined to its nearest neighbours along the skin.
const buildNetwork = async (surface: THREE.BufferGeometry, prims: Primitive[], spacing: number, lift: number) => {
  const breathe = breather();
  const position = surface.getAttribute("position");
  const normal = surface.getAttribute("normal");
  const rand = random(7);
  const order = Array.from({ length: position.count }, (_, index) => index);
  for (let index = order.length - 1; index > 0; index -= 1) {
    const swap = Math.floor(rand() * (index + 1));
    [order[index], order[swap]] = [order[swap], order[index]];
  }
  const grid = new Map<string, number[]>();
  const keyOf = (x: number, y: number, z: number) => `${Math.floor(x / spacing)},${Math.floor(y / spacing)},${Math.floor(z / spacing)}`;
  const nearby = (x: number, y: number, z: number, reach: number) => {
    const found: number[] = [];
    const cx = Math.floor(x / spacing);
    const cy = Math.floor(y / spacing);
    const cz = Math.floor(z / spacing);
    for (let dx = -reach; dx <= reach; dx += 1) {
      for (let dy = -reach; dy <= reach; dy += 1) {
        for (let dz = -reach; dz <= reach; dz += 1) {
          const bucket = grid.get(`${cx + dx},${cy + dy},${cz + dz}`);
          if (bucket) found.push(...bucket);
        }
      }
    }
    return found;
  };
  const points: Vec[] = [];
  const normals: Vec[] = [];
  for (const vertex of order) {
    const p: Vec = [position.getX(vertex), position.getY(vertex), position.getZ(vertex)];
    if (nearby(p[0], p[1], p[2], 1).some((other) => length(sub(points[other], p)) < spacing)) continue;
    const key = keyOf(...p);
    grid.set(key, [...(grid.get(key) ?? []), points.length]);
    points.push(p);
    normals.push([normal.getX(vertex), normal.getY(vertex), normal.getZ(vertex)]);
  }

  await breathe();
  const edges: number[] = [];
  const seen = new Set<number>();
  for (let a = 0; a < points.length; a += 1) {
    const p = points[a];
    const candidates = nearby(p[0], p[1], p[2], 2)
      .filter((b) => b !== a)
      .map((b) => ({ b, d: length(sub(points[b], p)) }))
      .filter(({ d }) => d < spacing * 1.8)
      .sort((left, right) => left.d - right.d);
    let linked = 0;
    for (const { b } of candidates) {
      if (linked >= 5) break;
      const n = normals[a];
      const m = normals[b];
      if (n[0] * m[0] + n[1] * m[1] + n[2] * m[2] < 0.35) continue;
      // Only along the skin: the link's midpoint must lie on (or just under)
      // the body, never out in a gap between two limbs.
      const mid = scale(add(p, points[b]), 0.5);
      const depth = fieldAt(prims, mid[0], mid[1], mid[2]);
      if (depth > 0.01 || depth < -0.05) continue;
      linked += 1;
      const id = Math.min(a, b) * 100000 + Math.max(a, b);
      if (seen.has(id)) continue;
      seen.add(id);
      edges.push(a, b);
    }
    if (a % 100 === 0) await breathe();
  }

  // Raised a little off the skin, so the body never hides its own network.
  const rest = new Float32Array(points.length * 3);
  points.forEach((p, index) => {
    rest.set(add(p, scale(normals[index], lift)), index * 3);
  });
  return { rest, edges, points };
};

// --- Shaders ------------------------------------------------------------------

const skinnedVertex = /* glsl */ `
  #include <common>
  #include <skinning_pars_vertex>
  attribute vec4 aMuscleA;
  attribute vec4 aMuscleB;
  uniform vec4 uStimA;
  uniform vec4 uStimB;
  uniform float uGlitch;
  uniform float uGlitchY;
  varying vec3 vNormal;
  varying vec3 vViewDir;
  varying float vWorldY;
  varying float vStim;
  #ifdef MIRROR
    uniform mat4 uMirror;
    uniform mat4 uFloorInverse;
    varying vec3 vFloor;
  #endif

  void main() {
    #include <skinbase_vertex>
    #include <beginnormal_vertex>
    #include <skinnormal_vertex>
    #include <defaultnormal_vertex>
    #include <begin_vertex>
    #include <skinning_vertex>
    vec4 world = modelMatrix * vec4(transformed, 1.0);
    // A rare glitch shifts a thin slice sideways.
    world.x += (1.0 - smoothstep(0.0, 0.05, abs(world.y - uGlitchY))) * uGlitch * 0.06;
    vWorldY = world.y;
    #ifdef MIRROR
      // The reflection in the platform: the same skinned body, mirrored across its top.
      vFloor = (uFloorInverse * world).xyz;
      world = uMirror * world;
    #endif
    vStim = dot(aMuscleA, uStimA) + dot(aMuscleB, uStimB);
    vec4 mvPosition = viewMatrix * world;
    vViewDir = -mvPosition.xyz;
    vNormal = normalize(transformedNormal);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

// The scanner's line: a crisp contour where it crosses the body, and a short
// afterglow on the side it has already read.
const scanChunk = /* glsl */ `
  uniform float uScan;
  uniform float uScanDir;

  float scanLine(float y) {
    return exp(-pow((y - uScan) * 60.0, 2.0));
  }

  float scanTrail(float y) {
    float passed = (uScan - y) * uScanDir;
    return passed > 0.0 ? exp(-passed * 7.0) : 0.0;
  }
`;

// A translucent body: a bright fresnel rim, a soft inner volume, a key light
// that models the anatomy, faint scan slices, and the scanner's line.
const bodyFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform vec3 uHot;
  uniform vec3 uGold;
  uniform vec3 uRed;
  uniform float uTime;
  uniform float uReveal;
  uniform float uOpacity;
  varying vec3 vNormal;
  varying vec3 vViewDir;
  varying float vWorldY;
  varying float vStim;
  ${scanChunk}

  void main() {
    // Materialises from the floor up.
    if (vWorldY > uReveal) discard;
    vec3 n = normalize(vNormal);
    float facing = abs(dot(n, normalize(vViewDir)));
    float rim = pow(1.0 - facing, 2.4);
    float volume = pow(facing, 3.0);
    float form = pow(max(dot(n, normalize(vec3(-0.35, 0.6, 0.72))), 0.0), 1.6);
    float slices = smoothstep(0.94, 1.0, abs(fract(vWorldY * 26.0 - uTime * 0.25) * 2.0 - 1.0));
    float line = scanLine(vWorldY);
    float trail = scanTrail(vWorldY);
    float revealEdge = exp(-pow((vWorldY - uReveal) * 12.0, 2.0));
    float flicker = 0.96 + 0.04 * sin(uTime * 29.0 + vWorldY * 7.0);
    float stim = clamp(vStim, 0.0, 1.0);
    vec3 stimColor = mix(uGold, uRed, smoothstep(0.4, 1.0, stim));
    vec3 color = mix(uColor, uHot, clamp(line + trail * 0.3 + revealEdge + form * 0.22, 0.0, 1.0));
    color = mix(color, stimColor, smoothstep(0.03, 0.45, stim));
    float light = 0.12 + rim * 1.2 + volume * 0.14 + form * 0.45 + slices * 0.07 + line * 2.0 + trail * 0.3 + stim * 1.1 + revealEdge * 1.5;
    float alpha = clamp((0.13 + rim * 0.76 + form * 0.2 + slices * 0.05 + line * 0.85 + trail * 0.16 + stim * 0.5) * flicker, 0.0, 1.0) * uOpacity;
    gl_FragColor = vec4(color * light, alpha);
    #include <colorspace_fragment>
  }
`;

// The reflection on the platform: faint, fading with the height above the top
// and towards the platform's edge (the stencil keeps it on the platform).
const reflectionFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform vec3 uGold;
  uniform vec3 uRed;
  uniform float uOpacity;
  uniform float uFloorRadius;
  uniform float uReflection;
  varying vec3 vNormal;
  varying vec3 vViewDir;
  varying float vStim;
  varying vec3 vFloor;

  void main() {
    float r = length(vFloor.xz);
    float fade = exp(-max(vFloor.y, 0.0) * 1.6) * (1.0 - smoothstep(uFloorRadius * 0.55, uFloorRadius, r));
    float rim = pow(1.0 - abs(dot(normalize(vNormal), normalize(vViewDir))), 2.0);
    float stim = clamp(vStim, 0.0, 1.0);
    vec3 color = mix(uColor, mix(uGold, uRed, smoothstep(0.4, 1.0, stim)), smoothstep(0.03, 0.45, stim));
    gl_FragColor = vec4(color * (0.5 + rim + stim), (0.1 + rim * 0.5 + stim * 0.4) * fade * uReflection * uOpacity);
    #include <colorspace_fragment>
  }
`;

const netVertex = /* glsl */ `
  attribute float aSeed;
  attribute vec4 aMuscleA;
  attribute vec4 aMuscleB;
  uniform vec4 uStimA;
  uniform vec4 uStimB;
  uniform float uTime;
  uniform float uPixelRatio;
  uniform float uPointScale;
  uniform float uGlitch;
  uniform float uGlitchY;
  varying float vStim;
  varying float vWorldY;
  varying float vTwinkle;

  void main() {
    vec4 world = modelMatrix * vec4(position, 1.0);
    world.x += (1.0 - smoothstep(0.0, 0.05, abs(world.y - uGlitchY))) * uGlitch * 0.06;
    vWorldY = world.y;
    vStim = clamp(dot(aMuscleA, uStimA) + dot(aMuscleB, uStimB), 0.0, 1.0);
    vec4 mvPosition = viewMatrix * world;
    vTwinkle = 0.6 + 0.4 * sin(uTime * (1.2 + aSeed * 2.6) + aSeed * 40.0);
    gl_PointSize = (1.6 + pow(aSeed, 6.0) * 4.5 + vStim * 2.0) * uPixelRatio * uPointScale / -mvPosition.z;
    gl_Position = projectionMatrix * mvPosition;
  }
`;

// The network is a quiet overlay: it shows where the scanner's line is
// reading and over the worked muscles, and stays faint elsewhere.
const lineFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform vec3 uGold;
  uniform vec3 uRed;
  uniform float uReveal;
  uniform float uOpacity;
  varying float vStim;
  varying float vWorldY;
  varying float vTwinkle;
  ${scanChunk}

  void main() {
    if (vWorldY > uReveal) discard;
    float read = clamp(scanLine(vWorldY) * 4.0, 0.0, 1.0) + scanTrail(vWorldY) * 0.55;
    vec3 color = mix(uColor, mix(uGold, uRed, smoothstep(0.4, 1.0, vStim)), smoothstep(0.05, 0.5, vStim));
    float alpha = (0.07 + read * 0.6 + vStim * 0.5) * uOpacity;
    gl_FragColor = vec4(color * (1.0 + vStim * 0.6), alpha);
    #include <colorspace_fragment>
  }
`;

const nodeFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform vec3 uGold;
  uniform vec3 uRed;
  uniform float uReveal;
  uniform float uOpacity;
  varying float vStim;
  varying float vWorldY;
  varying float vTwinkle;
  ${scanChunk}

  void main() {
    if (vWorldY > uReveal) discard;
    float d = length(gl_PointCoord - 0.5);
    float core = smoothstep(0.5, 0.0, d);
    float read = clamp(scanLine(vWorldY) * 4.0, 0.0, 1.0) + scanTrail(vWorldY) * 0.6;
    vec3 color = mix(uColor, mix(uGold, uRed, smoothstep(0.4, 1.0, vStim)), smoothstep(0.05, 0.5, vStim));
    float alpha = core * core * (vTwinkle * 0.28 + read * 1.1 + vStim * 0.7) * uOpacity;
    gl_FragColor = vec4(mix(color, vec3(1.0), core * 0.35) * 1.5, alpha);
    #include <colorspace_fragment>
  }
`;

const gearVertex = /* glsl */ `
  varying vec3 vNormal;
  varying vec3 vViewDir;
  varying float vWorldY;

  void main() {
    vec4 world = modelMatrix * vec4(position, 1.0);
    vWorldY = world.y;
    vec4 mvPosition = viewMatrix * world;
    vViewDir = -mvPosition.xyz;
    vNormal = normalize(normalMatrix * normal);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

const gearFragment = /* glsl */ `
  uniform vec3 uColor;
  uniform float uReveal;
  uniform float uOpacity;
  varying vec3 vNormal;
  varying vec3 vViewDir;
  varying float vWorldY;

  void main() {
    if (vWorldY > uReveal) discard;
    float facing = abs(dot(normalize(vNormal), normalize(vViewDir)));
    float rim = pow(1.0 - facing, 1.8);
    gl_FragColor = vec4(uColor * (0.3 + rim * 1.4), (0.32 + rim * 0.68) * uOpacity);
    #include <colorspace_fragment>
  }
`;

// --- Exercises ----------------------------------------------------------------

interface Pose {
  lean: number; // hips hinge forward (rad)
  armAbd: number; // upper arm away from the body, from hanging (rad)
  armFlex: number; // upper arm forward (rad, negative = forward while it hangs)
  armForward: number; // raised arms brought forward from the side (rad)
  elbowFlex: number; // curl (rad, negative = forward and up)
  elbowUp: number; // forearm turned up in the arm's plane (rad), for presses
  handTurn: number; // a quarter turn = neutral grip (rad)
  hip: number; // thigh forward (rad, negative)
  knee: number; // shin back (rad, positive)
}

const STANDING: Pose = { lean: 0, armAbd: 0.1, armFlex: 0, armForward: 0, elbowFlex: -0.12, elbowUp: 0, handTurn: 0, hip: 0, knee: 0 };

interface Exercise {
  yaw: number; // the angle that shows the movement best
  pose: (lift: number) => Pose;
  targets: Partial<Record<Region, number>>;
  focus: Region; // the muscle the readout points at
  via?: Pose; // passed through on the way in and out (the press racks the dumbbells first)
}

// Dumbbells racked at the shoulders, in front.
const RACKED: Pose = { ...STANDING, armFlex: -0.25, elbowFlex: -2.35 };

// lift: 0 at the start of a rep, 1 at its hardest point.
const EXERCISES: Exercise[] = [
  {
    // Biceps curl.
    yaw: -1.05,
    pose: (lift) => ({ ...STANDING, armFlex: -0.1 * lift, elbowFlex: -0.12 - 2.28 * lift }),
    targets: { biceps: 1 },
    focus: "biceps"
  },
  {
    // Dumbbell shoulder press.
    yaw: -0.6,
    pose: (lift) => {
      const armAbd = Math.PI / 2 + 0.1 + lift * 1.1;
      // The forearm stays vertical, tipping slightly inwards at the top.
      return { ...STANDING, armAbd, armForward: 0.3, elbowFlex: 0, elbowUp: Math.PI + lift * 0.1 - armAbd };
    },
    targets: { deltoids: 1, triceps: 0.7 },
    focus: "deltoids",
    via: RACKED
  },
  {
    // Dumbbell squat.
    yaw: -1.3,
    pose: (lift) => ({
      ...STANDING,
      lean: 0.55 * lift,
      armAbd: 0.13,
      armFlex: -0.55 * lift,
      handTurn: Math.PI / 2,
      hip: -1.3 * lift,
      knee: 1.75 * lift
    }),
    targets: { quads: 1, glutes: 0.85 },
    focus: "quads"
  },
  {
    // Romanian deadlift.
    yaw: -1.6,
    pose: (lift) => ({
      ...STANDING,
      lean: 1.15 * lift,
      armAbd: 0.05,
      armFlex: -1.15 * lift,
      elbowFlex: -0.06,
      hip: -0.34 * lift,
      knee: 0.44 * lift
    }),
    targets: { hamstrings: 1, glutes: 0.8 },
    focus: "hamstrings"
  }
];

const REP_SECONDS = 2.5;
const REPS = 3;
// After the last rep: a short rest (the last set reaches the scanner), then the
// change to the next exercise.
const HOLD = 1.8;
const SWITCH = 1.4;
const WORK = REP_SECONDS * REPS;
const SEGMENT = WORK + HOLD + SWITCH;

// The routine's shape, for the readout: the exercises in order and the sets each.
export const ROUTINE_EXERCISES = EXERCISES.length;
export const ROUTINE_SETS = REPS;
export const ROUTINE_REP_SECONDS = REP_SECONDS;

const smooth = (t: number) => t * t * (3 - 2 * t);
const clamp01 = (t: number) => Math.min(Math.max(t, 0), 1);
// One rep: lift (0..0.42), squeeze (..0.55), lower (..1).
const repCurve = (u: number) => (u < 0.42 ? smooth(u / 0.42) : u < 0.55 ? 1 : 1 - smooth((u - 0.55) / 0.45));

const mixPose = (a: Pose, b: Pose, t: number) => {
  const out = { ...a };
  for (const key of Object.keys(a) as Array<keyof Pose>) out[key] = a[key] + (b[key] - a[key]) * t;
  return out;
};

// From one exercise's start to the next one's, through a racked position when either needs it.
const switchPose = (from: Exercise, to: Exercise, t: number) => {
  const via = to.via ?? from.via;
  if (!via) return mixPose(from.pose(0), to.pose(0), smooth(t));
  return t < 0.5 ? mixPose(from.pose(0), via, smooth(t * 2)) : mixPose(via, to.pose(0), smooth(t * 2 - 1));
};

const targetVectors = (targets: Partial<Record<Region, number>>) => {
  const values = REGIONS.map((region) => targets[region] ?? 0);
  return [new THREE.Vector4(values[0], values[1], values[2], values[3]), new THREE.Vector4(values[4], values[5], values[6], values[7])] as const;
};

// Where each muscle's glow sits, per bone, at rest.
const glowAnchors = (): Array<{ region: number; bone: BoneName; at: Vec; side: number }> => {
  const anchors: Array<{ region: number; bone: BoneName; at: Vec; side: number }> = [];
  const j = JOINTS;
  for (const [side, s] of SIDES) {
    const dir = armDir(side);
    const thighMid = scale(add(j[`upperLeg${s}`], j[`lowerLeg${s}`]), 0.5);
    anchors.push(
      { region: M.biceps, bone: `upperArm${s}`, at: add(add(j[`upperArm${s}`], scale(dir, 0.3)), [0, 0, 0.07]), side },
      { region: M.deltoids, bone: `upperArm${s}`, at: add(j[`upperArm${s}`], [side * 0.05, 0.01, 0]), side },
      { region: M.triceps, bone: `upperArm${s}`, at: add(add(j[`upperArm${s}`], scale(dir, 0.28)), [0, 0, -0.07]), side },
      { region: M.quads, bone: `upperLeg${s}`, at: add(thighMid, [0, 0.05, 0.09]), side },
      { region: M.hamstrings, bone: `upperLeg${s}`, at: add(thighMid, [0, 0.06, -0.09]), side },
      { region: M.glutes, bone: "hips", at: [side * 0.11, 1.71, -0.15], side },
      { region: M.pecs, bone: "chest", at: [side * 0.13, 2.5, 0.15], side },
      { region: M.calves, bone: `lowerLeg${s}`, at: add(j[`lowerLeg${s}`], [0, -0.24, -0.08]), side }
    );
  }
  return anchors;
};

// --- Builder ------------------------------------------------------------------

// floorRadius: the platform's radius in the installation's units (the
// reflection fades out before its edge).
export const createAthlete = async (track: Track, pixelRatio: number, glow: THREE.Texture, floorRadius: number): Promise<Athlete> => {
  const prims = buildPrimitives();
  const breathe = breather();
  const surface = track(await buildSurface(prims, 0.021));

  // Skin and muscle weights for the body.
  const surfacePosition = surface.getAttribute("position");
  const vertexCount = surfacePosition.count;
  const skinIndex = new Uint16Array(vertexCount * 4);
  const skinWeight = new Float32Array(vertexCount * 4);
  const muscleA = new Float32Array(vertexCount * 4);
  const muscleB = new Float32Array(vertexCount * 4);
  for (let index = 0; index < vertexCount; index += 1) {
    const w = weigh(prims, surfacePosition.getX(index), surfacePosition.getY(index), surfacePosition.getZ(index));
    skinIndex.set([w.bones[0], w.bones[1], 0, 0], index * 4);
    skinWeight.set([w.weights[0], w.weights[1], 0, 0], index * 4);
    muscleA.set(w.muscles.slice(0, 4), index * 4);
    muscleB.set(w.muscles.slice(4), index * 4);
    if (index % 4000 === 0) await breathe();
  }
  surface.setAttribute("skinIndex", new THREE.Uint16BufferAttribute(skinIndex, 4));
  surface.setAttribute("skinWeight", new THREE.Float32BufferAttribute(skinWeight, 4));
  surface.setAttribute("aMuscleA", new THREE.Float32BufferAttribute(muscleA, 4));
  surface.setAttribute("aMuscleB", new THREE.Float32BufferAttribute(muscleB, 4));

  // The light network over the skin.
  const network = await buildNetwork(surface, prims, 0.06, 0.016);
  const nodeCount = network.points.length;
  const nodeBones = new Uint8Array(nodeCount * 2);
  const nodeWeights = new Float32Array(nodeCount);
  const nodeMuscleA = new Float32Array(nodeCount * 4);
  const nodeMuscleB = new Float32Array(nodeCount * 4);
  const nodeSeeds = new Float32Array(nodeCount);
  const rand = random(11);
  network.points.forEach((p, index) => {
    const w = weigh(prims, p[0], p[1], p[2]);
    nodeBones.set(w.bones, index * 2);
    nodeWeights[index] = w.weights[1];
    nodeMuscleA.set(w.muscles.slice(0, 4), index * 4);
    nodeMuscleB.set(w.muscles.slice(4), index * 4);
    nodeSeeds[index] = rand();
  });
  const nodePosition = new THREE.BufferAttribute(new Float32Array(network.rest), 3);
  nodePosition.setUsage(THREE.DynamicDrawUsage);
  const nodeAttributes = (geometry: THREE.BufferGeometry) => {
    geometry.setAttribute("position", nodePosition);
    geometry.setAttribute("aSeed", new THREE.BufferAttribute(nodeSeeds, 1));
    geometry.setAttribute("aMuscleA", new THREE.BufferAttribute(nodeMuscleA, 4));
    geometry.setAttribute("aMuscleB", new THREE.BufferAttribute(nodeMuscleB, 4));
    return geometry;
  };
  const lineGeometry = track(nodeAttributes(new THREE.BufferGeometry()));
  lineGeometry.setIndex(network.edges);
  const pointGeometry = track(nodeAttributes(new THREE.BufferGeometry()));

  // Skeleton at rest (A-pose), every bone unrotated, so local axes are body axes.
  const bones = BONE_NAMES.map(() => new THREE.Bone());
  BONE_NAMES.forEach((name, index) => {
    const parent = PARENTS[name];
    const offset = sub(JOINTS[name], parent ? JOINTS[parent] : [0, 0, 0]);
    bones[index].position.set(...offset);
    if (parent) bones[B[parent]].add(bones[index]);
  });
  const hipsRest = bones[B.hips].position.clone();

  const uniforms = {
    uColor: { value: new THREE.Color(0x36d6ff) },
    uHot: { value: new THREE.Color(0xf2fdff) },
    uGold: { value: new THREE.Color(0xffc23d) },
    uRed: { value: new THREE.Color(0xff3456) },
    uTime: { value: 0 },
    uScan: { value: -10 },
    uScanDir: { value: 1 },
    uReveal: { value: -10 },
    uOpacity: { value: 1 },
    uMirror: { value: new THREE.Matrix4() },
    uFloorInverse: { value: new THREE.Matrix4() },
    uFloorRadius: { value: floorRadius },
    uReflection: { value: 0.32 },
    uGlitch: { value: 0 },
    uGlitchY: { value: -10 },
    uStimA: { value: new THREE.Vector4() },
    uStimB: { value: new THREE.Vector4() },
    uPixelRatio: { value: pixelRatio },
    uPointScale: { value: 9 }
  };
  const additive = { transparent: true, depthWrite: false, blending: THREE.AdditiveBlending } as const;
  const shader = (vertexShader: string, fragmentShader: string) =>
    track(new THREE.ShaderMaterial({ uniforms, vertexShader, fragmentShader, ...additive }));

  const group = new THREE.Group();
  const turn = new THREE.Group();
  group.add(turn);

  const body = new THREE.SkinnedMesh(surface, shader(skinnedVertex, bodyFragment));
  body.add(bones[B.hips]);
  body.updateMatrixWorld(true);
  const skeleton = track(new THREE.Skeleton(bones));
  body.bind(skeleton);
  // Depth only: keeps the nearest skin, so the hologram reads as one clean shell.
  const prepass = new THREE.SkinnedMesh(surface, track(new THREE.MeshBasicMaterial({ colorWrite: false })));
  prepass.bind(skeleton);
  const lines = new THREE.LineSegments(lineGeometry, shader(netVertex, lineFragment));
  const nodes = new THREE.Points(pointGeometry, shader(netVertex, nodeFragment));
  // The reflection: drawn only where the platform's top wrote the stencil.
  const reflection = new THREE.SkinnedMesh(
    surface,
    track(
      new THREE.ShaderMaterial({
        uniforms,
        vertexShader: skinnedVertex,
        fragmentShader: reflectionFragment,
        defines: { MIRROR: "" },
        transparent: true,
        depthWrite: false,
        depthTest: false,
        side: THREE.DoubleSide,
        blending: THREE.AdditiveBlending,
        stencilWrite: true,
        stencilRef: 1,
        stencilFunc: THREE.EqualStencilFunc
      })
    )
  );
  reflection.bind(skeleton);
  reflection.renderOrder = -1;
  for (const object of [prepass, body, lines, nodes, reflection]) {
    object.frustumCulled = false;
    turn.add(object);
  }

  // Dumbbells (they follow the hand bones).
  const gearMaterial = track(
    new THREE.ShaderMaterial({
      uniforms: { uColor: { value: new THREE.Color(0x9af3ff) }, uReveal: uniforms.uReveal, uOpacity: uniforms.uOpacity },
      vertexShader: gearVertex,
      fragmentShader: gearFragment,
      ...additive
    })
  );
  const handle = track(new THREE.CylinderGeometry(0.021, 0.021, 0.32, 12));
  const plate = track(new THREE.CylinderGeometry(0.095, 0.095, 0.07, 6));
  const outer = track(new THREE.CylinderGeometry(0.075, 0.075, 0.05, 6));
  for (const [side, s] of SIDES) {
    // The handle runs across the fist, along the elbow's hinge.
    const grip = new THREE.Group();
    grip.position.set(...scale(armDir(side), 0.095));
    grip.rotation.z = side * REST_ARM;
    const bar = new THREE.Group();
    bar.rotation.z = Math.PI / 2;
    bar.add(new THREE.Mesh(handle, gearMaterial));
    for (const end of [-1, 1]) {
      const inner = new THREE.Mesh(plate, gearMaterial);
      inner.position.y = end * 0.125;
      const cap = new THREE.Mesh(outer, gearMaterial);
      cap.position.y = end * 0.185;
      bar.add(inner, cap);
    }
    grip.add(bar);
    bones[B[`hand${s}`]].add(grip);
  }

  // A soft glow over each worked muscle.
  const glowMaterials = REGIONS.map(() =>
    track(
      new THREE.SpriteMaterial({
        map: glow,
        color: 0xffc23d,
        transparent: true,
        opacity: 0,
        depthWrite: false,
        depthTest: false,
        blending: THREE.AdditiveBlending,
        toneMapped: false
      })
    )
  );
  // The readout points at the worked muscle on the side nearer to it (screen left).
  const focusPoints: Partial<Record<number, THREE.Object3D[]>> = {};
  for (const anchor of glowAnchors()) {
    const sprite = new THREE.Sprite(glowMaterials[anchor.region]);
    sprite.position.set(...sub(anchor.at, JOINTS[anchor.bone]));
    sprite.scale.setScalar(anchor.region === M.glutes || anchor.region === M.quads || anchor.region === M.hamstrings ? 0.5 : 0.4);
    bones[B[anchor.bone]].add(sprite);
    (focusPoints[anchor.region] ??= []).push(sprite);
  }

  // --- Posing -----------------------------------------------------------------
  const qa = new THREE.Quaternion();
  const qb = new THREE.Quaternion();
  const qc = new THREE.Quaternion();
  const X = new THREE.Vector3(1, 0, 0);
  const Y = new THREE.Vector3(0, 1, 0);
  const Z = new THREE.Vector3(0, 0, 1);
  const thighRest = new THREE.Vector3(...sub(JOINTS.lowerLegL, JOINTS.upperLegL));
  const shinRest = new THREE.Vector3(...sub(JOINTS.footL, JOINTS.lowerLegL));
  const restReach = thighRest.clone().add(shinRest);
  const reach = new THREE.Vector3();
  const shinReach = new THREE.Vector3();
  const hinge = SIDES.map(([side]) => new THREE.Vector3(...hingeAxis(side)));
  const forearmAxis = SIDES.map(([side]) => new THREE.Vector3(...armDir(side)));

  const applyPose = (pose: Pose) => {
    // The hips hinge forward; the legs are counter-rotated so their angles are
    // world angles, and the feet stay planted: the hips move by exactly what
    // the folded legs no longer reach.
    bones[B.hips].quaternion.setFromAxisAngle(X, pose.lean);
    reach.copy(thighRest).applyAxisAngle(X, pose.hip).add(shinReach.copy(shinRest).applyAxisAngle(X, pose.hip + pose.knee));
    bones[B.hips].position.copy(hipsRest).add(restReach).sub(reach);
    SIDES.forEach(([side, s], index) => {
      // Upper arm: away from the body (z), forward while hanging (x), forward while raised (y).
      qa.setFromAxisAngle(Z, side * (pose.armAbd - REST_ARM));
      qb.setFromAxisAngle(X, pose.armFlex);
      qc.setFromAxisAngle(Y, -side * pose.armForward);
      bones[B[`upperArm${s}`]].quaternion.copy(qc).multiply(qb).multiply(qa);
      // Forearm: the elbow's hinge, then turned up in the arm's plane.
      qa.setFromAxisAngle(hinge[index], pose.elbowFlex);
      qb.setFromAxisAngle(Z, side * pose.elbowUp);
      bones[B[`foreArm${s}`]].quaternion.copy(qb).multiply(qa);
      bones[B[`hand${s}`]].quaternion.setFromAxisAngle(forearmAxis[index], side * pose.handTurn);
      bones[B[`upperLeg${s}`]].quaternion.setFromAxisAngle(X, pose.hip - pose.lean);
      bones[B[`lowerLeg${s}`]].quaternion.setFromAxisAngle(X, pose.knee);
      bones[B[`foot${s}`]].quaternion.setFromAxisAngle(X, -(pose.hip + pose.knee));
    });
  };

  // The network is skinned on the CPU (two bones per node).
  const boneMatrices = bones.map(() => new THREE.Matrix4());
  const toLocal = new THREE.Matrix4();
  const restPoint = new THREE.Vector3();
  const moved = new THREE.Vector3();
  const blended = new THREE.Vector3();
  const skinNetwork = () => {
    turn.updateMatrixWorld(true);
    toLocal.copy(body.matrixWorld).invert();
    bones.forEach((bone, index) => {
      boneMatrices[index].multiplyMatrices(toLocal, bone.matrixWorld).multiply(skeleton.boneInverses[index]);
    });
    const out = nodePosition.array as Float32Array;
    for (let index = 0; index < nodeCount; index += 1) {
      restPoint.fromArray(network.rest, index * 3);
      const w = nodeWeights[index];
      blended.copy(restPoint).applyMatrix4(boneMatrices[nodeBones[index * 2]]);
      if (w > 0) blended.multiplyScalar(1 - w).addScaledVector(moved.copy(restPoint).applyMatrix4(boneMatrices[nodeBones[index * 2 + 1]]), w);
      blended.toArray(out, index * 3);
    }
    nodePosition.needsUpdate = true;
  };

  const targets = EXERCISES.map((exercise) => targetVectors(exercise.targets));
  const gold = new THREE.Color(0xffc23d);
  const red = new THREE.Color(0xff3456);
  const stimColor = new THREE.Color();
  const muscle = new THREE.Vector3();
  const candidate = new THREE.Vector3();

  const update = (frame: AthleteFrame) => {
    const { elapsed } = frame;
    const cycle = elapsed % (SEGMENT * EXERCISES.length);
    const index = Math.floor(cycle / SEGMENT);
    const local = cycle - index * SEGMENT;
    const exercise = EXERCISES[index];
    const next = EXERCISES[(index + 1) % EXERCISES.length];
    const working = local < WORK;
    // Between exercises the hologram turns to the next one's best angle.
    const progress = working ? 0 : clamp01((local - WORK - HOLD) / SWITCH);
    const lift = working ? repCurve((local % REP_SECONDS) / REP_SECONDS) : 0;
    applyPose(working ? exercise.pose(lift) : switchPose(exercise, next, progress));
    turn.rotation.y = exercise.yaw + (next.yaw - exercise.yaw) * smooth(progress);
    // Idle life: a slow breath through the trunk and a small turn of the head.
    const breath = Math.sin(elapsed * 1.5);
    bones[B.spine].rotation.set(0.01 * breath, 0, 0);
    bones[B.chest].rotation.set(-0.018 * breath, 0, 0);
    bones[B.neck].rotation.set(0.012 * breath, 0.07 * Math.sin(elapsed * 0.31), 0);

    // The worked muscles pulse gold to red with the effort.
    const throb = 0.82 + 0.18 * Math.sin(elapsed * 11);
    const fade = working ? smooth(clamp01(local / 0.6)) : 1 - clamp01((local - WORK) / 0.7);
    const stimulus = (0.2 + 0.8 * lift) * throb * fade * frame.emphasis;
    const [a, b] = targets[index];
    uniforms.uStimA.value.copy(a).multiplyScalar(stimulus);
    uniforms.uStimB.value.copy(b).multiplyScalar(stimulus);
    stimColor.copy(gold).lerp(red, smooth(clamp01((stimulus - 0.35) / 0.6)));
    REGIONS.forEach((region, regionIndex) => {
      glowMaterials[regionIndex].opacity = (exercise.targets[region] ?? 0) * stimulus * 0.55 * frame.presence;
      glowMaterials[regionIndex].color.copy(stimColor);
    });

    uniforms.uTime.value = elapsed;
    uniforms.uScan.value = frame.scanY;
    uniforms.uScanDir.value = frame.scanDir;
    uniforms.uMirror.value.copy(frame.mirror);
    uniforms.uFloorInverse.value.copy(frame.floorInverse);
    uniforms.uReveal.value = frame.reveal;
    uniforms.uGlitch.value = frame.glitch;
    uniforms.uGlitchY.value = frame.glitchY;
    // The depth pre-pass joins once the body is complete, so it never cuts an
    // invisible silhouette out of the beam while the hologram builds up.
    prepass.visible = frame.presence >= 1;
    reflection.visible = frame.presence >= 1;

    skinNetwork();
    let nearest = Infinity;
    for (const point of focusPoints[M[exercise.focus]] ?? []) {
      point.getWorldPosition(candidate);
      if (candidate.x < nearest) {
        nearest = candidate.x;
        muscle.copy(candidate);
      }
    }
    return {
      stimulus,
      exercise: index,
      sets: working ? Math.min(REPS, Math.floor(local / REP_SECONDS)) : REPS,
      rest: working ? 0 : clamp01((local - WORK) / (HOLD + SWITCH)),
      time: local,
      round: Math.floor(elapsed / (SEGMENT * EXERCISES.length)),
      muscle
    };
  };

  return { group, update };
};
