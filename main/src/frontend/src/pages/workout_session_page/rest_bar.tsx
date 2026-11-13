import { Timer, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { useI18n } from "@/i18n/i18n";
import { formatClock } from "@/services/training_insights";

import styles from "./workout_session_page.module.css";

interface RestBarProps {
  // When the server says the rest after the last set ends, and how long it
  // was planned to be.
  endsAt: string;
  plannedSeconds: number;
  onDismiss: () => void;
}

// RestBar counts down the rest between sets. The end comes from the server;
// +/-15 s and Skip only change this screen. When it reaches zero the phone
// vibrates (where supported) and the bar says so briefly.
export const RestBar = ({ endsAt, plannedSeconds, onDismiss }: RestBarProps) => {
  const { t } = useI18n();
  const [now, setNow] = useState(() => Date.now());
  const [offset, setOffset] = useState(0);
  const notified = useRef(false);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 250);
    return () => window.clearInterval(timer);
  }, []);

  const end = new Date(endsAt).getTime() + offset * 1000;
  const remaining = Math.max(0, Math.ceil((end - now) / 1000));
  const total = Math.max(1, plannedSeconds + offset);
  const progress = Math.min(1, Math.max(0, 1 - remaining / total));

  useEffect(() => {
    if (remaining > 0 || notified.current) {
      return;
    }
    notified.current = true;
    if (typeof navigator !== "undefined" && "vibrate" in navigator) {
      navigator.vibrate?.([180, 80, 180]);
    }
    const timer = window.setTimeout(onDismiss, 4000);
    return () => window.clearTimeout(timer);
  }, [remaining, onDismiss]);

  return (
    <div className={styles.restBar} role="timer" aria-live={remaining === 0 ? "assertive" : "off"}>
      <div className={styles.restTrack} aria-hidden="true">
        <span style={{ transform: `scaleX(${progress})` }} />
      </div>
      <div className={styles.restContent}>
        <Timer className={styles.restIcon} aria-hidden="true" />
        {remaining > 0 ? (
          <p className={styles.restTime}>
            <span>{t("training.player.restLabel")}</span>
            <strong>{formatClock(remaining)}</strong>
          </p>
        ) : (
          <p className={styles.restTime}>
            <strong>{t("training.player.restOver")}</strong>
          </p>
        )}
        {remaining > 0 ? (
          <div className={styles.restActions}>
            <button type="button" onClick={() => setOffset((value) => value - 15)}>
              {t("training.player.restSub")}
            </button>
            <button type="button" onClick={() => setOffset((value) => value + 15)}>
              {t("training.player.restAdd")}
            </button>
            <button type="button" className={styles.restSkip} onClick={onDismiss}>
              {t("training.player.restSkip")}
            </button>
          </div>
        ) : (
          <button type="button" className={styles.restClose} onClick={onDismiss} aria-label={t("common.close")}>
            <X aria-hidden="true" />
          </button>
        )}
      </div>
    </div>
  );
};
