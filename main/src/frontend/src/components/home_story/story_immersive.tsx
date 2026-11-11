import { ChevronsDown } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent, ReactNode } from "react";

import { BrandGlobe } from "@/components/brand_globe/brand_globe";
import { BRAND_ASSETS } from "@/constants/brand_assets";
import earthLandMask from "@resources/img/globe/earth_land_mask.webp";
import { joinClassNames } from "@utils/class_names";

import { CALLOUTS, SAMPLE_DATA_NOTE, SCREENS, STORY_CHAPTERS, type StoryChapterId } from "./story_content";
import { ChapterCopy, StartActions, TONE_CLASS } from "./story_parts";
import type { StoryScene } from "./story_scene";
import styles from "./story_immersive.module.css";

/*
 * Desktop Home, top to bottom:
 *   1. The opening: a pinned stage with the Earth. Scrolling fades the copy
 *      while the Earth turns to the centre, then the stage scrolls away.
 *   2. The programs, in normal page flow (nothing pinned behind them).
 *   3. The device chapters: a second pinned stage that scrolls into view with
 *      the phone already in place, and scrolls away after "Get started".
 * Both stages share one WebGL canvas, which moves to whichever stage is on
 * screen (the programs are at least a viewport tall, so both are never visible
 * together). Scene progress:
 *   opening   0    → 0.1
 *   chapters  0.14 → 1
 */
const HERO_END = 0.1;
const TRACK_START = 0.14;

const [IDENTITY, ...CHAPTERS] = STORY_CHAPTERS;

// Progress windows in which each chapter's copy is visible: fade in, hold, fade
// out. The first chapter is already visible when its stage arrives.
const WINDOWS: Partial<Record<StoryChapterId, [number, number, number, number]>> = {
  train: [TRACK_START, TRACK_START, 0.37, 0.42],
  fuel: [0.45, 0.5, 0.63, 0.68],
  platform: [0.74, 0.79, 0.87, 0.91],
  start: [0.93, 0.97, 2, 2]
};

// The middle of each chapter's hold (keyboard focus scrolls there).
const ANCHORS: Partial<Record<StoryChapterId, number>> = {
  train: 0.28,
  fuel: 0.57,
  platform: 0.83,
  start: 1
};

type Rgba = [number, number, number, number];

// The backdrop never leaves the navy family: each chapter only shifts where
// the blue light sits and how strong it is, so the story reads as one place.
type Light = "hero" | "train" | "fuel" | "platform";
const PALETTES: Record<Light, [Rgba, Rgba, Rgba]> = {
  hero: [[2, 4, 31, 1], [5, 14, 58, 1], [4, 129, 220, 0.34]],
  train: [[2, 5, 34, 1], [6, 22, 78, 1], [0, 160, 240, 0.3]],
  fuel: [[2, 6, 38, 1], [7, 30, 96, 1], [0, 203, 255, 0.28]],
  platform: [[3, 8, 44, 1], [8, 36, 112, 1], [4, 129, 220, 0.42]]
};

const STOPS: Array<[number, Light]> = [
  [TRACK_START, "train"],
  [0.4, "train"],
  [0.5, "fuel"],
  [0.66, "fuel"],
  [0.74, "platform"],
  [0.89, "platform"],
  [0.95, "hero"]
];

const clamp01 = (value: number) => Math.min(Math.max(value, 0), 1);

const smoothstep = (edge0: number, edge1: number, value: number) => {
  if (edge1 <= edge0) {
    return value >= edge1 ? 1 : 0;
  }
  const t = clamp01((value - edge0) / (edge1 - edge0));
  return t * t * (3 - 2 * t);
};

const mix = (a: Rgba, b: Rgba, t: number): Rgba => [
  a[0] + (b[0] - a[0]) * t,
  a[1] + (b[1] - a[1]) * t,
  a[2] + (b[2] - a[2]) * t,
  a[3] + (b[3] - a[3]) * t
];

const css = (color: Rgba) => `rgba(${color[0].toFixed(0)}, ${color[1].toFixed(0)}, ${color[2].toFixed(0)}, ${color[3].toFixed(3)})`;

const paletteAt = (progress: number): [Rgba, Rgba, Rgba] => {
  if (progress <= STOPS[0][0]) {
    return PALETTES[STOPS[0][1]];
  }
  for (let index = 0; index < STOPS.length - 1; index += 1) {
    const [fromAt, fromTone] = STOPS[index];
    const [toAt, toTone] = STOPS[index + 1];
    if (progress <= toAt) {
      const t = smoothstep(fromAt, toAt, progress);
      const from = PALETTES[fromTone];
      const to = PALETTES[toTone];
      return [mix(from[0], to[0], t), mix(from[1], to[1], t), mix(from[2], to[2], t)];
    }
  }
  return PALETTES[STOPS[STOPS.length - 1][1]];
};

const chapterVisibility = (id: StoryChapterId, progress: number) => {
  const [inStart, inEnd, outStart, outEnd] = WINDOWS[id] ?? [2, 2, 2, 2];
  const rise = smoothstep(inStart, inEnd, progress);
  const fall = 1 - smoothstep(outStart, outEnd, progress);
  return { opacity: Math.min(rise, fall), shift: (1 - rise) * 28 - (1 - fall) * 28 };
};

type Slot = "hero" | "track";

interface Layout {
  intro: HTMLElement;
  track: HTMLElement;
}

// Which stage is on screen, and the scene progress there.
const positionFor = ({ intro, track }: Layout): { slot: Slot; progress: number } => {
  const viewport = window.innerHeight;
  const trackTop = track.getBoundingClientRect().top;
  if (trackTop < viewport) {
    const travel = Math.max(track.offsetHeight - viewport, 1);
    return { slot: "track", progress: TRACK_START + clamp01(-trackTop / travel) * (1 - TRACK_START) };
  }
  const pin = Math.max(intro.offsetHeight - viewport, 1);
  return { slot: "hero", progress: clamp01(-intro.getBoundingClientRect().top / pin) * HERO_END };
};

// Inverse for the chapter track.
const scrollTopFor = ({ track }: Layout, progress: number) => {
  const travel = Math.max(track.offsetHeight - window.innerHeight, 1);
  const trackTop = track.getBoundingClientRect().top + window.scrollY;
  return trackTop + ((progress - TRACK_START) / (1 - TRACK_START)) * travel;
};

interface StoryImmersiveProps {
  onFailure: () => void;
  programs: ReactNode;
}

export const StoryImmersive = ({ onFailure, programs }: StoryImmersiveProps) => {
  const introRef = useRef<HTMLElement>(null);
  const heroStageRef = useRef<HTMLDivElement>(null);
  const heroHostRef = useRef<HTMLDivElement>(null);
  const heroCopyRef = useRef<HTMLDivElement>(null);
  const cueRef = useRef<HTMLParagraphElement>(null);
  const trackRef = useRef<HTMLElement>(null);
  const trackStageRef = useRef<HTMLDivElement>(null);
  const trackHostRef = useRef<HTMLDivElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);
  const chapterRefs = useRef<Partial<Record<StoryChapterId, HTMLDivElement | null>>>({});
  const sceneRef = useRef<StoryScene | null>(null);
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const slotRef = useRef<Slot>("hero");
  const [ready, setReady] = useState(false);

  const layout = (): Layout | null => {
    const intro = introRef.current;
    const track = trackRef.current;
    return intro && track ? { intro, track } : null;
  };

  const hostFor = (slot: Slot) => (slot === "hero" ? heroHostRef.current : trackHostRef.current);

  // Scroll → stage, progress, copy and backdrop.
  useEffect(() => {
    let frame = 0;

    const update = () => {
      frame = 0;
      const current = layout();
      if (!current) {
        return;
      }
      const { slot, progress } = positionFor(current);

      // Move the canvas to the stage on screen and jump the scene there.
      const moved = slot !== slotRef.current;
      slotRef.current = slot;
      const canvas = canvasRef.current;
      const host = hostFor(slot);
      if (canvas && host && canvas.parentElement !== host) {
        host.appendChild(canvas);
      }
      sceneRef.current?.setProgress(progress, moved);

      // Opening: the copy fades and lifts while the Earth comes to the centre.
      const opening = slot === "hero" ? progress / HERO_END : 1;
      const fade = smoothstep(0.08, 0.75, opening);
      if (heroCopyRef.current) {
        heroCopyRef.current.style.opacity = (1 - fade).toFixed(3);
        heroCopyRef.current.style.transform = `translate3d(0, ${(-56 * smoothstep(0.08, 0.9, opening)).toFixed(1)}px, 0)`;
        heroCopyRef.current.style.pointerEvents = fade > 0.4 ? "none" : "auto";
      }
      if (cueRef.current) {
        cueRef.current.style.opacity = (1 - smoothstep(0, 0.2, opening)).toFixed(3);
      }

      const [a, b, glow] = paletteAt(progress);
      if (backdropRef.current) {
        backdropRef.current.style.background = `radial-gradient(70rem 46rem at 78% 30%, ${css(glow)}, transparent 66%), linear-gradient(160deg, ${css(a)} 0%, ${css(b)} 100%)`;
      }

      for (const chapter of CHAPTERS) {
        const element = chapterRefs.current[chapter.id];
        const { opacity, shift } = chapterVisibility(chapter.id, slot === "track" ? progress : 0);
        if (element) {
          // Opacity only: every chapter stays in the accessibility tree.
          element.style.opacity = opacity.toFixed(3);
          element.style.transform = `translate3d(0, ${shift.toFixed(1)}px, 0)`;
          element.style.pointerEvents = opacity > 0.6 ? "auto" : "none";
        }
      }
    };

    const schedule = () => {
      if (!frame) {
        frame = window.requestAnimationFrame(update);
      }
    };

    update();
    window.addEventListener("scroll", schedule, { passive: true });
    window.addEventListener("resize", schedule);
    return () => {
      window.removeEventListener("scroll", schedule);
      window.removeEventListener("resize", schedule);
      if (frame) {
        window.cancelAnimationFrame(frame);
      }
    };
  }, []);

  // Scene lifecycle: dynamic import, render only while a stage is on screen.
  useEffect(() => {
    const heroStage = heroStageRef.current;
    const trackStage = trackStageRef.current;
    const host = hostFor(slotRef.current);
    if (!heroStage || !trackStage || !host) {
      return;
    }
    // Each scene instance owns its canvas, so a disposed instance (for example
    // after a fast remount) can never release another instance's WebGL context.
    const canvas = document.createElement("canvas");
    canvas.className = styles.canvas;
    canvas.setAttribute("aria-hidden", "true");
    host.appendChild(canvas);
    canvasRef.current = canvas;
    let disposed = false;
    let scene: StoryScene | null = null;
    const handleContextLost = (event: Event) => {
      event.preventDefault();
      onFailure();
    };
    canvas.addEventListener("webglcontextlost", handleContextLost);

    const onScreen = new Map<Element, boolean>();
    const isVisible = () => [...onScreen.values()].some(Boolean);
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        onScreen.set(entry.target, entry.isIntersecting);
      }
      if (isVisible()) {
        sceneRef.current?.start();
      } else {
        sceneRef.current?.stop();
      }
    });
    observer.observe(heroStage);
    observer.observe(trackStage);

    // Both stages fill the viewport, so either one gives the canvas size.
    const resizeObserver = new ResizeObserver(([entry]) => {
      sceneRef.current?.resize(entry.contentRect.width, entry.contentRect.height);
    });
    resizeObserver.observe(heroStage);

    import("./story_scene")
      .then(({ createStoryScene }) => {
        if (disposed) {
          throw new Error("Story unmounted before the scene was created");
        }
        return createStoryScene(canvas, {
          logo: BRAND_ASSETS.icon,
          landMask: earthLandMask,
          screens: {
            training: SCREENS.training.src,
            nutrition: SCREENS.nutrition.src,
            synced: SCREENS.synced.src,
            platform: SCREENS.platform.src
          },
          callouts: CALLOUTS
        });
      })
      .then((created) => {
        scene = created;
        if (disposed) {
          created.dispose();
          return;
        }
        sceneRef.current = created;
        const current = layout();
        if (current) {
          const { slot, progress } = positionFor(current);
          slotRef.current = slot;
          const target = hostFor(slot);
          if (target && canvas.parentElement !== target) {
            target.appendChild(canvas);
          }
          created.setProgress(progress, true);
        }
        if (isVisible()) {
          created.start();
        }
        setReady(true);
      })
      .catch(() => {
        if (!disposed) {
          onFailure();
        }
      });

    return () => {
      disposed = true;
      observer.disconnect();
      resizeObserver.disconnect();
      canvas.removeEventListener("webglcontextlost", handleContextLost);
      scene?.dispose();
      if (sceneRef.current === scene) {
        sceneRef.current = null;
      }
      if (canvasRef.current === canvas) {
        canvasRef.current = null;
      }
      canvas.remove();
    };
  }, [onFailure]);

  // The stages fill the viewport, so the pointer is read against the window.
  const handlePointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    sceneRef.current?.setPointer((event.clientX / window.innerWidth) * 2 - 1, -((event.clientY / window.innerHeight) * 2 - 1));
  };

  // Keyboard focus inside faded copy brings that copy back on screen.
  const revealHero = () => {
    const intro = introRef.current;
    if (intro && Number(heroCopyRef.current?.style.opacity || "1") < 0.6) {
      window.scrollTo({ top: intro.getBoundingClientRect().top + window.scrollY, behavior: "auto" });
    }
  };

  const revealFocusedChapter = (id: StoryChapterId) => {
    const element = chapterRefs.current[id];
    const current = layout();
    const anchor = ANCHORS[id];
    if (element && current && anchor !== undefined && Number(element.style.opacity || "1") < 0.6) {
      window.scrollTo({ top: scrollTopFor(current, anchor), behavior: "auto" });
    }
  };

  return (
    <div className={joinClassNames(styles.story, ready && styles.storyReady)} onPointerMove={handlePointerMove}>
      <section ref={introRef} className={joinClassNames(styles.intro, TONE_CLASS[IDENTITY.tone])} aria-labelledby="story-identity">
        <div ref={heroStageRef} className={joinClassNames(styles.stage, styles.heroStage)}>
          <div className={joinClassNames(styles.backdrop, styles.heroBackdrop)} aria-hidden="true" />
          {ready ? null : (
            <div className={styles.placeholder} aria-hidden="true">
              <BrandGlobe />
            </div>
          )}
          <div ref={heroHostRef} className={joinClassNames(styles.canvasHost, styles.heroHost)} />

          <div className={styles.heroInner}>
            <div ref={heroCopyRef} className={styles.heroCopy} onFocus={revealHero}>
              <ChapterCopy chapter={IDENTITY} headingId="story-identity" as="h1" />
              <StartActions withEntryPoints />
            </div>
          </div>
          <p ref={cueRef} className={styles.scrollCue} aria-hidden="true">
            <ChevronsDown />
            Scroll to explore
          </p>
        </div>
      </section>

      <div className={styles.programs}>{programs}</div>

      <section ref={trackRef} className={styles.track} aria-label="Discover RYZE">
        <div ref={trackStageRef} className={joinClassNames(styles.stage, styles.trackStage)}>
          <div ref={backdropRef} className={joinClassNames(styles.backdrop, styles.trackBackdrop)} aria-hidden="true" />
          <div ref={trackHostRef} className={joinClassNames(styles.canvasHost, styles.trackHost)} />

          <div className={styles.chapters}>
            {CHAPTERS.map((chapter) => (
              <div
                key={chapter.id}
                ref={(element) => {
                  chapterRefs.current[chapter.id] = element;
                }}
                className={joinClassNames(styles.chapter, TONE_CLASS[chapter.tone], chapter.id === "start" && styles.chapterStart)}
                style={{ opacity: chapter.id === "train" ? 1 : 0 }}
                onFocus={() => revealFocusedChapter(chapter.id)}
              >
                <ChapterCopy chapter={chapter} headingId={`story-${chapter.id}`} />
                {chapter.id === "start" ? <StartActions /> : null}
                {chapter.screen ? <p className={styles.sampleNote}>{SAMPLE_DATA_NOTE}</p> : null}
                {/* The device screens are drawn in WebGL; describe them for assistive technology. */}
                {chapter.screen ? <p className={styles.srOnly}>{chapter.screen.alt}</p> : null}
              </div>
            ))}
          </div>
        </div>
      </section>
    </div>
  );
};
