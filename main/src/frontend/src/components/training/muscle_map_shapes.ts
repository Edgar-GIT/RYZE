// Path data of the body map (viewBox 200 × 440 per figure, the left half
// drawn and mirrored around x = 100). Generated from a reviewed drawing;
// edit the shapes here.
import type { MuscleName } from "@/services/body_stats_api";

export interface MuscleShape {
  muscle: MuscleName;
  d: string;
  mirror: boolean;
}

export interface DetailLine {
  d: string;
  mirror: boolean;
}

export const SILHOUETTE = "M100 60 L91 60 Q90 70 86 75 Q72 79 60 82 Q43 86 40 106 Q37 128 39 150 Q37 170 34 182 Q28 205 27 226 Q25 240 28 250 Q34 256 39 247 Q41 237 43 228 Q53 204 57 182 Q61 160 63 140 L65 130 Q65 160 67 196 Q64 206 66 216 Q60 252 64 300 Q66 312 66 322 Q60 352 64 390 Q62 404 60 412 Q66 419 80 417 Q82 404 80 392 Q86 360 86 324 Q86 312 88 304 Q94 270 98 232 L100 230 Z";

export const FRONT: MuscleShape[] = [
  { muscle: "traps", d: "M88 66 Q80 76 63 82 L86 84 Q90 76 91 66 Z", mirror: true },
  { muscle: "shoulders", d: "M62 84 Q46 86 42 104 Q40 120 44 131 Q52 120 62 112 Q68 98 66 86 Z", mirror: true },
  { muscle: "chest", d: "M98 88 L72 86 Q62 96 63 112 Q66 124 78 128 Q90 130 98 124 Z", mirror: true },
  { muscle: "biceps", d: "M46 132 Q39 148 42 166 Q47 174 54 168 Q60 154 59 136 Q53 127 46 132 Z", mirror: true },
  { muscle: "forearms", d: "M41 176 Q33 196 30 222 Q33 230 39 228 Q49 206 56 180 Q50 172 41 176 Z", mirror: true },
  { muscle: "abs", d: "M87 132 L113 132 Q115 160 113 196 Q100 214 87 196 Q85 160 87 132 Z", mirror: false },
  { muscle: "obliques", d: "M68 124 Q62 146 68 172 Q74 190 84 204 L85 194 Q83 160 85 134 Q76 128 68 124 Z", mirror: true },
  { muscle: "quads", d: "M70 214 Q60 248 66 294 Q72 312 84 306 Q90 296 92 280 Q96 250 93 226 Q84 210 70 214 Z", mirror: true },
  { muscle: "adductors", d: "M95 226 Q98 232 99 242 L98 268 Q93 280 88 294 Q90 262 95 226 Z", mirror: true },
  { muscle: "calves", d: "M69 324 Q63 352 67 388 L77 390 Q83 360 84 326 Q76 318 69 324 Z", mirror: true }
];

export const BACK: MuscleShape[] = [
  { muscle: "traps", d: "M100 62 L91 64 Q86 74 64 82 Q82 92 90 116 L100 134 Z", mirror: true },
  { muscle: "shoulders", d: "M62 84 Q46 86 42 104 Q40 120 44 130 Q54 118 64 108 Q70 96 66 86 Z", mirror: true },
  { muscle: "upper_back", d: "M66 88 Q82 95 90 116 L91 128 Q80 124 70 126 Q64 110 66 88 Z", mirror: true },
  { muscle: "lats", d: "M69 130 Q64 148 72 168 Q80 182 88 192 L89 138 Q80 132 69 130 Z", mirror: true },
  { muscle: "triceps", d: "M44 132 Q38 150 41 168 Q48 175 55 168 Q60 150 58 128 Q50 124 44 132 Z", mirror: true },
  { muscle: "forearms", d: "M41 176 Q33 196 30 222 Q33 230 39 228 Q49 206 56 180 Q50 172 41 176 Z", mirror: true },
  { muscle: "lower_back", d: "M90 142 Q95 140 99 142 L99 206 Q92 208 88 202 Q87 172 90 142 Z", mirror: true },
  { muscle: "glutes", d: "M70 210 Q64 236 76 250 Q90 256 99 246 L99 212 Q86 204 70 210 Z", mirror: true },
  { muscle: "hamstrings", d: "M70 256 Q63 286 69 314 Q78 320 86 312 Q92 286 95 258 Q84 250 70 256 Z", mirror: true },
  { muscle: "calves", d: "M68 324 Q59 350 68 384 Q77 388 83 378 Q89 350 84 326 Q76 318 68 324 Z", mirror: true }
];

// Detail lines (v2.130.0, owner request 2026-11-21: "a bit more detail"):
// muscle heads, tendons and separations drawn over the colour.
export const FRONT_LINES: DetailLine[] = [
  // Abdominals: linea alba and the tendinous bands of a six-pack.
  { d: "M100 134 L100 206", mirror: false },
  { d: "M88 150 Q100 154 112 150", mirror: false },
  { d: "M88 168 Q100 172 112 168", mirror: false },
  { d: "M88 186 Q100 190 112 186", mirror: false },
  // Chest: sternum and the fibres of the pectorals.
  { d: "M98 90 L98 124", mirror: false },
  { d: "M74 96 Q86 100 96 98", mirror: true },
  { d: "M68 108 Q82 114 96 110", mirror: true },
  // Neck (sternocleidomastoid) and collarbone.
  { d: "M92 62 Q95 74 98 86", mirror: true },
  { d: "M70 84 Q84 82 97 87", mirror: true },
  // Deltoid heads, biceps heads and the forearm split.
  { d: "M58 88 Q53 102 52 122", mirror: true },
  { d: "M52 136 Q50 152 51 168", mirror: true },
  { d: "M47 182 Q41 202 35 222", mirror: true },
  // Serratus and obliques.
  { d: "M68 136 L76 140", mirror: true },
  { d: "M67 148 L77 153", mirror: true },
  { d: "M68 160 L79 166", mirror: true },
  // V-line, quadriceps heads (rectus, vastus medialis teardrop), knee.
  { d: "M72 204 Q86 214 96 228", mirror: true },
  { d: "M81 222 Q75 262 81 302", mirror: true },
  { d: "M90 272 Q83 286 85 304", mirror: true },
  { d: "M70 312 Q77 306 84 312 Q77 320 70 312", mirror: true },
  // Shin (tibialis) and the inner calf.
  { d: "M75 330 Q73 358 73 386", mirror: true }
];

export const BACK_LINES: DetailLine[] = [
  // Spine and the erectors.
  { d: "M100 64 L100 134", mirror: false },
  { d: "M95 146 L95 202", mirror: true },
  // Trapezius (upper and middle) and the rear deltoid.
  { d: "M70 84 Q84 92 92 106", mirror: true },
  { d: "M57 89 Q51 104 49 120", mirror: true },
  // Teres and infraspinatus; the lat sweep.
  { d: "M69 116 Q79 114 89 122", mirror: true },
  { d: "M73 146 Q80 164 88 182", mirror: true },
  // Triceps horseshoe and the forearm.
  { d: "M51 134 Q48 150 51 164", mirror: true },
  { d: "M46 162 Q51 170 57 162", mirror: true },
  { d: "M47 182 Q41 202 35 222", mirror: true },
  // Glutes (medius and the fold) and the hamstrings.
  { d: "M72 214 Q84 208 96 214", mirror: true },
  { d: "M73 246 Q86 254 98 248", mirror: true },
  { d: "M82 262 Q79 290 80 314", mirror: true },
  // Back of the knee, the two calf heads and the Achilles tendon.
  { d: "M70 318 Q77 322 84 318", mirror: true },
  { d: "M76 330 Q75 356 76 380", mirror: true },
  { d: "M70 366 Q76 376 83 366", mirror: true },
  { d: "M76 384 L77 404", mirror: true }
];
