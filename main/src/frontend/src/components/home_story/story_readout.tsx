import { Check, Medal } from "lucide-react";
import { type RefObject, useEffect, useRef, useState } from "react";

import { useI18n } from "@/i18n/i18n";
import { joinClassNames } from "@utils/class_names";

import { READOUT_LOADS, useStory } from "./story_content";
import styles from "./story_readout.module.css";

/*
 * The hologram scanner's readout on the hero (owner request, 2026-11-15: the
 * drone takes the session's information while the athlete trains, tying the
 * scene to the app). It is a sped-up picture of a RYZE session in the workout
 * player's own terms — the set being logged (kg × reps), the exercise's
 * muscles, the rest before the next exercise — and, when the routine ends, the
 * saved workout's summary with the EXP the app awards (workout 100, 5 per set,
 * 50 per record), the lifter level and the record medal. Sample data, no
 * measurement: every rep of the hologram stands for one logged set.
 *
 * The scene calls the handler every drawn frame; positions are written to the
 * DOM directly and React re-renders only when a set or the exercise changes.
 */

// What the hologram's scanner is reading, placed on the stage (CSS pixels).
export interface ReadoutFrame {
  visible: number; // 0..1: shown on the hero once the hologram has materialised
  exercise: number;
  arrived: number; // sets of the current exercise logged (each rep stands for one)
  rest: number; // 0 while working; 0..1 through the change to the next exercise
  logged: number; // seconds since the last set's data reached the scanner (-1: none yet)
  time: number; // seconds into the current exercise
  round: number; // how many times the routine has already run
  lens: [number, number]; // the scanner's sensor
  muscle: [number, number]; // the worked muscle
  base: [number, number]; // the platform's centre
}

// The hologram's routine (story_athlete.ts): four exercises of three sets.
const EXERCISES = 4;
const SETS = 3;
const LOADS = READOUT_LOADS;
// The saved workout's EXP (backend achievements rules) and the sample lifter.
const WORKOUT_XP = 100;
const SET_XP = 5;
const RECORD_XP = 50;
const XP_BEFORE = 2340;
const LEVEL_FROM = 1500;
const LEVEL_TO = 4000;
// The summary stays into the first seconds of the next round.
const SUMMARY_INTO_NEXT = 3.2;
const LOGGED_SECONDS = 1.2;

interface View {
  exercise: number;
  shown: number; // sets logged in this exercise
  done: number; // sets logged in this session
  resting: boolean;
  justLogged: boolean;
  summary: boolean;
}

interface StoryReadoutProps {
  connect: (handler: ((frame: ReadoutFrame) => void) | null) => void;
  // What the cards must never cover: the hero copy and the story steps.
  avoid: ReadonlyArray<RefObject<HTMLElement | null>>;
}

interface Box {
  x: number;
  y: number;
  w: number;
  h: number;
}

const overlaps = (a: Box, b: Box) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;
// The floating navigation bar over the top of the stage.
const TOP_CLEARANCE = 92;
// How far (in hologram heights) the cards may slide down from chest height.
const SLIDES = [0, 0.06, 0.12, 0.18, 0.24, 0.3, 0.36];

const sameView = (a: View, b: View) =>
  a.exercise === b.exercise && a.shown === b.shown && a.done === b.done && a.resting === b.resting && a.justLogged === b.justLogged && a.summary === b.summary;

export const StoryReadout = ({ connect, avoid }: StoryReadoutProps) => {
  const { text } = useStory();
  const { locale } = useI18n();
  const copy = text.readout;
  const rootRef = useRef<HTMLDivElement>(null);
  const sessionRef = useRef<HTMLDivElement>(null);
  const overviewRef = useRef<HTMLDivElement>(null);
  const muscleLinkRef = useRef<SVGPathElement>(null);
  const muscleDotRef = useRef<SVGCircleElement>(null);
  const dataLinkRef = useRef<SVGPathElement>(null);
  const [view, setView] = useState<View>({ exercise: 0, shown: 0, done: 0, resting: false, justLogged: false, summary: false });

  useEffect(() => {
    let current = view;
    const sizes = { session: [248, 150], overview: [248, 120] };
    const observer = new ResizeObserver(() => {
      if (sessionRef.current) sizes.session = [sessionRef.current.offsetWidth, sessionRef.current.offsetHeight];
      if (overviewRef.current) sizes.overview = [overviewRef.current.offsetWidth, overviewRef.current.offsetHeight];
    });
    for (const element of [sessionRef.current, overviewRef.current]) {
      if (element) observer.observe(element);
    }

    // Where the cards may not go: every line of the hero copy (its real
    // length, not the column's width), its controls and the story steps.
    // Measured again after a resize, once the fonts are in and if the page
    // has scrolled since (the copy scrolls; the stage stays).
    let keepOut: Box[] = [];
    let stage = { w: 1, h: 1 };
    let measuredScroll = Number.NaN;
    let dirty = true;
    const markDirty = () => {
      dirty = true;
    };
    window.addEventListener("resize", markDirty);
    void document.fonts?.ready.then(markDirty);
    const measure = (root: HTMLElement) => {
      const origin = root.getBoundingClientRect();
      stage = { w: root.clientWidth, h: root.clientHeight };
      const boxes: Box[] = [];
      const add = (rect: DOMRect, margin: number) => {
        if (rect.width > 0 && rect.height > 0) {
          boxes.push({ x: rect.left - origin.left - margin, y: rect.top - origin.top - margin, w: rect.width + margin * 2, h: rect.height + margin * 2 });
        }
      };
      const range = document.createRange();
      for (const ref of avoid) {
        const element = ref.current;
        if (!element) continue;
        if (element.tagName === "NAV") {
          add(element.getBoundingClientRect(), 16);
          continue;
        }
        const walker = document.createTreeWalker(element, NodeFilter.SHOW_TEXT);
        for (let node = walker.nextNode(); node; node = walker.nextNode()) {
          if (!node.textContent?.trim()) continue;
          range.selectNodeContents(node);
          for (const rect of range.getClientRects()) add(rect, 18);
        }
        element.querySelectorAll("a, button").forEach((control) => add(control.getBoundingClientRect(), 18));
      }
      keepOut = boxes;
      measuredScroll = window.scrollY;
      dirty = false;
    };
    const fits = (box: Box) =>
      box.x >= 12 && box.y >= TOP_CLEARANCE && box.x + box.w <= stage.w - 12 && box.y + box.h <= stage.h - 12 && !keepOut.some((zone) => overlaps(box, zone));
    let chosen = 0;

    connect((frame) => {
      const root = rootRef.current;
      if (!root) return;
      root.style.opacity = String(frame.visible);
      root.style.visibility = frame.visible < 0.01 ? "hidden" : "visible";
      // Everything shown follows from the scene's time: a set counts once its
      // data has reached the scanner.
      const finished = frame.exercise === EXERCISES - 1 && frame.arrived === SETS;
      const next: View = {
        exercise: frame.exercise,
        shown: frame.arrived,
        done: frame.exercise * SETS + frame.arrived,
        resting: frame.arrived === SETS && frame.rest > 0.6 && !finished,
        justLogged: frame.logged >= 0 && frame.logged < LOGGED_SECONDS,
        summary: finished || (frame.round > 0 && frame.exercise === 0 && frame.time < SUMMARY_INTO_NEXT)
      };
      if (!sameView(next, current)) {
        current = next;
        setView(next);
      }
      if (frame.visible < 0.01) return;

      if (dirty || Math.abs(window.scrollY - measuredScroll) > 24) measure(root);

      // Placement, first that fits: a column between the copy and the beam
      // (the session card, the workout card under it), from chest height
      // sliding down while the copy is in the way; then the session card
      // alone the same way; then beside the scanner. Both follow the
      // hologram's size. Nothing fits: the readout steps aside.
      const height = Math.max(frame.base[1] - frame.lens[1], 1);
      const [sessionWidth, sessionHeight] = sizes.session;
      const [overviewWidth, overviewHeight] = sizes.overview;
      const right = frame.lens[0] - 0.28 * height;
      const candidates: Array<[Box, Box | null]> = [];
      for (const withOverview of [true, false]) {
        for (const drop of SLIDES) {
          const y = frame.lens[1] + (0.3 + drop) * height;
          const session = { x: right - sessionWidth, y, w: sessionWidth, h: sessionHeight };
          const overview = { x: right - overviewWidth, y: y + sessionHeight + 14, w: overviewWidth, h: overviewHeight };
          candidates.push([session, withOverview ? overview : null]);
        }
      }
      candidates.push([{ x: frame.lens[0] - 0.42 * height - sessionWidth, y: frame.lens[1] - 0.05 * height, w: sessionWidth, h: sessionHeight }, null]);
      const valid = ([session, overview]: [Box, Box | null]) => fits(session) && (!overview || fits(overview));
      // Keep the current place while it still fits, so the cards never hop.
      if (!(chosen >= 0 && valid(candidates[chosen]))) chosen = candidates.findIndex(valid);
      if (chosen < 0) {
        root.style.visibility = "hidden";
        return;
      }
      const [session, overview] = candidates[chosen];
      const sx = session.x;
      const sy = session.y;
      sessionRef.current?.style.setProperty("transform", `translate3d(${sx}px, ${sy}px, 0)`);
      if (overviewRef.current) {
        overviewRef.current.style.visibility = overview ? "" : "hidden";
        if (overview) overviewRef.current.style.transform = `translate3d(${overview.x}px, ${overview.y}px, 0)`;
      }

      const [mx, my] = frame.muscle;
      const edge = sx + sessionWidth;
      const rowY = sy + sessionHeight * 0.5;
      muscleLinkRef.current?.setAttribute("d", `M${mx} ${my}L${edge + 26} ${rowY}L${edge} ${rowY}`);
      muscleDotRef.current?.setAttribute("cx", String(mx));
      muscleDotRef.current?.setAttribute("cy", String(my));
      const [lx, ly] = frame.lens;
      const join = edge - 34;
      dataLinkRef.current?.setAttribute("d", `M${lx} ${ly + 4}C${lx - 0.12 * height} ${ly + 0.08 * height} ${join} ${sy - 0.16 * height} ${join} ${sy}`);
    });
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", markDirty);
      connect(null);
    };
    // The handler keeps the last view it set; it is connected once.
  }, [connect, avoid]);

  const number = new Intl.NumberFormat(locale === "pt" ? "pt-PT" : "en-GB");
  const exercise = copy.exercises[view.exercise] ?? copy.exercises[0];
  const upcoming = copy.exercises[(view.exercise + 1) % EXERCISES];
  const current = Math.min(view.shown + 1, SETS);
  const earned = WORKOUT_XP + SET_XP * SETS * EXERCISES + RECORD_XP;
  const levelShare = (xp: number) => `${(((xp - LEVEL_FROM) / (LEVEL_TO - LEVEL_FROM)) * 100).toFixed(1)}%`;

  return (
    <div ref={rootRef} className={styles.readout} aria-hidden="true" style={{ opacity: 0, visibility: "hidden" }}>
      <svg className={styles.links}>
        <path ref={dataLinkRef} className={joinClassNames(styles.dataLink, view.justLogged && styles.dataLinkActive)} />
        <path ref={muscleLinkRef} className={styles.muscleLink} />
        <circle ref={muscleDotRef} className={styles.muscleDot} r="4" />
      </svg>

      <div ref={sessionRef} className={styles.card}>
        {view.resting ? (
          <div className={styles.rest} key={`rest-${view.exercise}`}>
            <svg className={styles.restRing} viewBox="0 0 44 44">
              <circle cx="22" cy="22" r="19" className={styles.restTrack} />
              <circle cx="22" cy="22" r="19" className={styles.restArc} />
            </svg>
            <span className={styles.restTime}>{copy.restTime}</span>
            <div>
              <p className={styles.eyebrow}>{copy.rest}</p>
              <p className={styles.next}>
                {copy.next}: <strong>{upcoming.name}</strong>
              </p>
            </div>
          </div>
        ) : (
          <>
            <p className={styles.eyebrow}>
              {view.shown < SETS ? <span className={styles.live} /> : <Check className={styles.eyebrowIcon} aria-hidden="true" />}
              {copy.set(current, SETS)} · {view.shown < SETS ? copy.status : copy.logged}
            </p>
            <p className={styles.title}>{exercise.name}</p>
            <div className={styles.loadRow}>
              <strong className={styles.load}>{LOADS[view.exercise]}</strong>
              <span className={joinClassNames(styles.pill, view.justLogged && styles.pillOn)}>
                <Check aria-hidden="true" />
                {copy.logged}
              </span>
            </div>
            <ol className={styles.sets}>
              {Array.from({ length: SETS }, (_, index) => (
                <li key={index} data-state={index < view.shown ? "done" : index === view.shown ? "current" : "next"} />
              ))}
            </ol>
            <p className={styles.meta}>
              {copy.muscles} · {exercise.muscles}
            </p>
          </>
        )}
        <p className={styles.sample}>{copy.sample}</p>
      </div>

      <div ref={overviewRef} className={styles.card}>
        {view.summary ? (
          <div key="summary" className={styles.summary}>
            <p className={styles.eyebrow}>{copy.saved}</p>
            <p className={styles.xp}>+{number.format(earned)} EXP</p>
            <p className={styles.meta}>
              {copy.workout} +{WORKOUT_XP} · {copy.sets} +{SET_XP * SETS * EXERCISES} · {copy.records} +{RECORD_XP}
            </p>
            <div className={styles.level}>
              <span>{copy.level}</span>
              <span>
                {number.format(XP_BEFORE + earned)} / {number.format(LEVEL_TO)}
              </span>
            </div>
            <div className={styles.bar} style={{ ["--from" as string]: levelShare(XP_BEFORE), ["--to" as string]: levelShare(XP_BEFORE + earned) }}>
              <i />
            </div>
            <p className={styles.medal}>
              <Medal aria-hidden="true" />
              {copy.newMedal} · {copy.exercises[EXERCISES - 1].name} {LOADS[EXERCISES - 1].split(" × ")[0]}
            </p>
          </div>
        ) : (
          <div key="today">
            <p className={styles.eyebrow}>{copy.today}</p>
            <p className={styles.title}>{copy.todayTitle}</p>
            <ol className={styles.segments}>
              {Array.from({ length: EXERCISES }, (_, index) => (
                <li
                  key={index}
                  data-state={index < view.exercise ? "done" : index === view.exercise ? "current" : "next"}
                  style={{ ["--fill" as string]: index < view.exercise ? 1 : index === view.exercise ? view.shown / SETS : 0 }}
                >
                  <i />
                </li>
              ))}
            </ol>
            <p className={styles.meta}>{copy.progress(view.done, SETS * EXERCISES)}</p>
          </div>
        )}
      </div>
    </div>
  );
};
