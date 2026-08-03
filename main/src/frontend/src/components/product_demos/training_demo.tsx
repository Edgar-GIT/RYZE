import { GlassPanel } from "@/components/glass_panel/glass_panel";
import { joinClassNames } from "@utils/class_names";

import styles from "./product_demos.module.css";

const sessions = [
  { day: "Mon", label: "Push", done: true },
  { day: "Tue", label: "Pull", done: true },
  { day: "Wed", label: "Rest", done: false, rest: true },
  { day: "Thu", label: "Legs", done: true },
  { day: "Fri", label: "Upper", done: false, active: true },
  { day: "Sat", label: "Cardio", done: false },
  { day: "Sun", label: "Rest", done: false, rest: true }
] as const;

const lifts = [
  { name: "Bench press", sets: "4 × 6", load: "80 kg" },
  { name: "Incline DB", sets: "3 × 10", load: "28 kg" },
  { name: "Cable fly", sets: "3 × 12", load: "18 kg" }
] as const;

export const TrainingDemo = () => (
  <GlassPanel className={styles.window} as="article">
    <div className={styles.windowChrome}>
      <span />
      <span />
      <span />
      <p>Training · This week</p>
    </div>

    <div className={styles.trainingLayout}>
      <div className={styles.weekStrip}>
        {sessions.map((session) => (
          <div
            key={session.day}
            className={joinClassNames(
              styles.dayCell,
              session.done && styles.dayDone,
              "active" in session && session.active && styles.dayActive,
              "rest" in session && session.rest && styles.dayRest
            )}
          >
            <span>{session.day}</span>
            <strong>{session.label}</strong>
          </div>
        ))}
      </div>

      <div className={styles.sessionPanel}>
        <div className={styles.sessionMeta}>
          <span>Live session</span>
          <strong>Friday · Upper strength</strong>
        </div>
        <ul className={styles.liftList}>
          {lifts.map((lift) => (
            <li key={lift.name}>
              <div>
                <strong>{lift.name}</strong>
                <span>{lift.sets}</span>
              </div>
              <em>{lift.load}</em>
            </li>
          ))}
        </ul>
      </div>
    </div>
  </GlassPanel>
);
