import { apiGet, apiPut } from "@utils/http_client";

// Training reminders (v2.131.0): an occasional nudge in the app and by email
// after a few days without training; the member can turn it off.
export const fetchTrainingReminders = async (): Promise<boolean> =>
  (await apiGet<{ enabled: boolean }>("/me/training-reminders")).enabled;

export const saveTrainingReminders = async (enabled: boolean): Promise<boolean> =>
  (await apiPut<{ enabled: boolean }>("/me/training-reminders", { enabled })).enabled;
