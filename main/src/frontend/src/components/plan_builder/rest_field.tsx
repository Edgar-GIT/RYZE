import { useEffect, useState } from "react";

import { formatDuration, parseDuration } from "./duration";
import styles from "./plan_builder.module.css";

interface RestFieldProps {
  // Seconds as a string ("" when unset), like the rest of the draft.
  value: string;
  onChange: (seconds: string) => void;
  label: string;
  placeholder: string;
}

const STEP = 15;

// RestField edits a rest time. It shows the adaptive form ("45 s",
// "1 min 30 s") and accepts what a coach types ("90", "1:30", "2 min");
// the arrow keys add or remove 15 seconds.
export const RestField = ({ value, onChange, label, placeholder }: RestFieldProps) => {
  const display = value === "" ? "" : formatDuration(Number(value));
  const [text, setText] = useState(display);
  const [focused, setFocused] = useState(false);

  useEffect(() => {
    if (!focused) setText(display);
  }, [display, focused]);

  return (
    <input
      className={styles.cellInput}
      value={text}
      placeholder="—"
      title={placeholder}
      aria-label={label}
      onFocus={() => setFocused(true)}
      onBlur={() => {
        setFocused(false);
        setText(display);
      }}
      onKeyDown={(event) => {
        if (event.key !== "ArrowUp" && event.key !== "ArrowDown") return;
        event.preventDefault();
        const current = value === "" ? 0 : Number(value);
        const next = Math.max(0, Math.min(3600, current + (event.key === "ArrowUp" ? STEP : -STEP)));
        onChange(String(next));
        setText(formatDuration(next));
      }}
      onChange={(event) => {
        const raw = event.target.value;
        setText(raw);
        if (raw.trim() === "") {
          onChange("");
          return;
        }
        const parsed = parseDuration(raw);
        if (parsed !== null) {
          onChange(String(Math.min(parsed, 3600)));
        }
      }}
    />
  );
};
