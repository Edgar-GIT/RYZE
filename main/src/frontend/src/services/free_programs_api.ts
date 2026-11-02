import { apiGet, apiPost, encodePathSegment } from "@utils/http_client";

/** Safe summary of a Free programme returned by the activation endpoints. */
export interface FreeProgramSummary {
  id: string;
  name: string;
  description: string;
  type: string;
  level: string | null;
  duration_weeks: number | null;
  frequency_per_week: number | null;
  training_type: string | null;
  price_minor_units: number;
  currency: string;
}

export interface FreeProgramActivation {
  id: string;
  activated_at: string;
  expires_at: string | null;
  program: FreeProgramSummary;
}

export const fetchFreeActivations = (): Promise<FreeProgramActivation[]> =>
  apiGet<FreeProgramActivation[]>("/me/free-programs");

export const activateFreeProgram = (programId: string): Promise<FreeProgramActivation> =>
  apiPost<FreeProgramActivation>(
    `/me/programs/${encodePathSegment(programId)}/free-activation`,
    {}
  );
