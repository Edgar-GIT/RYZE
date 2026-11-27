/**
 * Official RYZE plans (ADMIN_2, free plans; owner request 2026-11-23): their
 * cover is fixed. The API's cover URL is the wide picture (a colour
 * background with the word RYZE); portrait plan cards show the RYZE
 * "Base plan" picture instead.
 */
export const OFFICIAL_COVER_URL = "/uploads/programs/official/ryze_base_plan.png";
const OFFICIAL_PORTRAIT_URL = "/uploads/programs/official/ryze_base_plan_portrait.png";

// portraitCover swaps an official plan's wide cover for its portrait picture.
export const portraitCover = (url: string): string => (url.endsWith(OFFICIAL_COVER_URL) ? url.replace(OFFICIAL_COVER_URL, OFFICIAL_PORTRAIT_URL) : url);
