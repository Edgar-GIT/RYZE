import { apiDelete, apiGet, apiPost, encodePathSegment } from "@utils/http_client";

/** The muscle groups of the recovery estimate, in display order. */
export const MUSCLES = [
  "chest",
  "shoulders",
  "biceps",
  "triceps",
  "forearms",
  "abs",
  "obliques",
  "traps",
  "lats",
  "upper_back",
  "lower_back",
  "glutes",
  "quads",
  "hamstrings",
  "adductors",
  "calves"
] as const;

export type MuscleName = (typeof MUSCLES)[number];
export type RecoveryStatus = "ready" | "recovering" | "fatigued";

/** The server's estimate for one muscle group (see api.MD, muscle recovery). */
export interface MuscleRecovery {
  muscle: MuscleName;
  /** 0 to 100. */
  recovery: number;
  status: RecoveryStatus;
  /** Sets of the last seven days; secondary work counts half. */
  weekly_sets: number;
  last_trained_at: string | null;
  /** When the muscle is estimated fully recovered; null when it already is. */
  recovered_at: string | null;
}

export interface MuscleRecoveryReport {
  window_days: number;
  generated_at: string;
  muscles: MuscleRecovery[];
}

export const fetchMuscleRecovery = (): Promise<MuscleRecoveryReport> => apiGet<MuscleRecoveryReport>("/me/statistics/muscles");

/** Weight in kg, body fat in %, the rest are circumferences in cm. */
export const MEASUREMENT_KINDS = [
  "weight",
  "body_fat",
  "neck",
  "shoulders",
  "chest",
  "biceps_left",
  "biceps_right",
  "forearm_left",
  "forearm_right",
  "waist",
  "hips",
  "thigh_left",
  "thigh_right",
  "calf_left",
  "calf_right"
] as const;

export type MeasurementKind = (typeof MEASUREMENT_KINDS)[number];

export const measurementUnit = (kind: MeasurementKind): string => (kind === "weight" ? "kg" : kind === "body_fat" ? "%" : "cm");

export interface BodyMeasurement {
  id: string;
  kind: MeasurementKind;
  value: number;
  /** YYYY-MM-DD */
  measured_on: string;
}

/** The member's measurements, oldest first. */
export const fetchMeasurements = async (): Promise<BodyMeasurement[]> =>
  (await apiGet<{ measurements: BodyMeasurement[] }>("/me/measurements")).measurements;

/** Logs a value; the same kind on the same day replaces the earlier value. */
export const saveMeasurement = (kind: MeasurementKind, value: number, measuredOn: string): Promise<BodyMeasurement> =>
  apiPost<BodyMeasurement>("/me/measurements", { kind, value, measured_on: measuredOn });

export const deleteMeasurement = (id: string): Promise<null> => apiDelete<null>(`/me/measurements/${encodePathSegment(id)}`);
