// Adaptive durations (owner request, 2026-11-14): 45 seconds reads "45 s",
// 61 seconds reads "1 min 1 s", 90 seconds "1 min 30 s", 120 seconds "2 min".

export const formatDuration = (totalSeconds: number | null | undefined): string => {
  if (totalSeconds === null || totalSeconds === undefined || !Number.isFinite(totalSeconds) || totalSeconds < 0) {
    return "";
  }
  const seconds = Math.round(totalSeconds);
  if (seconds < 60) {
    return `${seconds} s`;
  }
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const rest = seconds % 60;
  const parts: string[] = [];
  if (hours > 0) parts.push(`${hours} h`);
  if (minutes > 0) parts.push(`${minutes} min`);
  if (rest > 0) parts.push(`${rest} s`);
  return parts.join(" ");
};

// parseDuration reads what a coach types into a rest field: "90", "90s",
// "1:30", "1m30", "1 min 30 s" or "2 min". Returns seconds, or null when the
// text is empty or unreadable.
export const parseDuration = (raw: string): number | null => {
  const text = raw.trim().toLowerCase().replace(",", ".");
  if (!text) return null;

  const clock = /^(\d{1,3}):([0-5]?\d)$/.exec(text);
  if (clock) {
    return Number(clock[1]) * 60 + Number(clock[2]);
  }
  if (/^\d+(\.\d+)?$/.test(text)) {
    return Math.round(Number(text));
  }
  // "1m30" or "1 min 30" (no unit on the seconds).
  const compact = /^(\d{1,3})\s*m(?:in)?\s*(\d{1,2})$/.exec(text);
  if (compact) {
    return Number(compact[1]) * 60 + Number(compact[2]);
  }

  const units = /(\d+(?:\.\d+)?)\s*(h|hr|hrs|hora|horas|m|min|mins|minuto|minutos|minute|minutes|s|sec|secs|seg|segundo|segundos|second|seconds)\b/g;
  let total = 0;
  let matched = false;
  let match: RegExpExecArray | null;
  while ((match = units.exec(text)) !== null) {
    matched = true;
    const value = Number(match[1]);
    const unit = match[2];
    if (unit.startsWith("h")) total += value * 3600;
    else if (unit.startsWith("m")) total += value * 60;
    else total += value;
  }
  return matched ? Math.round(total) : null;
};
