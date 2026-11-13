import { ChevronsDown } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent, ReactNode } from "react";

import { BrandGlobe } from "@/components/brand_globe/brand_globe";
import { BRAND_ASSETS } from "@/constants/brand_assets";
import earthLandMask from "@resources/img/globe/earth_land_mask.webp";
import { joinClassNames } from "@utils/class_names";

import { CALLOUTS, SCREENS, STORY_CHAPTER_IDS, type StoryChapterId, useStory } from "./story_content";
import { ChapterCopy, StartActions, TONE_CLASS } from "./story_parts";
import type { StoryScene } from "./story_scene";
import styles from "./story_immersive.module.css";

/*
 * Desktop Home: one 3D stage stays pinned behind the hero, the programs and
 * the device chapters, with the five story steps on its right edge.
 *   hero scrolling away          0    → 0.1   the Earth travels to the corner
 *   programs passing over        0.1  → 0.14  the Earth waits in the corner
 *   chapter track                0.14 → 1     the phone flies in, then the
 *                                              laptop; the Earth returns
 * The track starts while the programs are still leaving, so the phone and the
 * Train copy arrive right behind them. The stage is released (scrolls away)
 * when the track ends.
 */
const HERO_END = 0.1;
const TRACK_START = 0.14;
// The track starts when its top edge reaches this fraction of the viewport.
const TRACK_LEAD = 0.55;

// Every chapter after the opening one (ids only; the copy comes from useStory).
const CHAPTER_IDS = STORY_CHAPTER_IDS.slice(1);

// Progress windows in which each chapter's copy is visible: fade in, hold, fade out.
const TRAIN_IN = 0.19;
const WINDOWS: Partial<Record<StoryChapterId, [number, number, number, number]>> = {
  train: [TRAIN_IN, 0.24, 0.39, 0.44],
  fuel: [0.47, 0.52, 0.63, 0.68],
  platform: [0.74, 0.79, 0.87, 0.91],
  start: [0.93, 0.97, 2, 2]
};

// Where each step of the chapter navigation scrolls to: the middle of its hold.
const ANCHORS: Partial<Record<StoryChapterId, number>> = {
  train: 0.31,
  fuel: 0.58,
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
  [0.14, "hero"],
  [0.24, "train"],
  [0.44, "train"],
  [0.52, "fuel"],
  [0.66, "fuel"],
  [0.74, "platform"],
  [0.89, "platform"],
  [0.95, "hero"]
];

const clamp01 = (value: number) => Math.min(Math.max(value, 0), 1);

const smoothstep = (edge0: number, edge1: number, value: number) => {
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

interface Layout {
  root: HTMLElement;
  hero: HTMLElement;
  programs: HTMLElement;
  track: HTMLElement;
}

// Piecewise map from scroll to scene progress (see the comment at the top).
const progressFor = ({ root, hero, track }: Layout) => {
  const viewport = window.innerHeight;
  const y = -root.getBoundingClientRect().top;
  const heroLength = Math.max(hero.offsetHeight, 1);
  const trackFrom = track.offsetTop - viewport * TRACK_LEAD;
  const trackTo = track.offsetTop + track.offsetHeight - viewport;
  if (y >= trackFrom) {
    return TRACK_START + clamp01((y - trackFrom) / Math.max(trackTo - trackFrom, 1)) * (1 - TRACK_START);
  }
  if (y <= heroLength) {
    return clamp01(y / heroLength) * HERO_END;
  }
  return HERO_END + clamp01((y - heroLength) / Math.max(trackFrom - heroLength, 1)) * (TRACK_START - HERO_END);
};

// Inverse for the chapter track.
const scrollTopFor = ({ root, track }: Layout, progress: number) => {
  const viewport = window.innerHeight;
  const rootTop = root.getBoundingClientRect().top + window.scrollY;
  const trackFrom = track.offsetTop - viewport * TRACK_LEAD;
  const trackTo = track.offsetTop + track.offsetHeight - viewport;
  return rootTop + trackFrom + ((progress - TRACK_START) / (1 - TRACK_START)) * (trackTo - trackFrom);
};

interface StoryImmersiveProps {
  onFailure: () => void;
  programs: ReactNode;
}

export const StoryImmersive = ({ onFailure, programs }: StoryImmersiveProps) => {
  const { text, chapters } = useStory();
  const [identity, ...laterChapters] = chapters;
  const rootRef = useRef<HTMLDivElement>(null);
  const heroRef = useRef<HTMLElement>(null);
  const heroCopyRef = useRef<HTMLDivElement>(null);
  const programsRef = useRef<HTMLDivElement>(null);
  const trackRef = useRef<HTMLDivElement>(null);
  const stageRef = useRef<HTMLElement>(null);
  const canvasHostRef = useRef<HTMLDivElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);
  const chapterRefs = useRef<Partial<Record<StoryChapterId, HTMLDivElement | null>>>({});
  const sceneRef = useRef<StoryScene | null>(null);
  const visibleRef = useRef(false);
  const [ready, setReady] = useState(false);
  const [activeStep, setActiveStep] = useState<StoryChapterId>("identity");
  // The steps step aside while the programs fill the screen.
  const [stepsHidden, setStepsHidden] = useState(false);

  const layout = (): Layout | null => {
    const root = rootRef.current;
    const hero = heroRef.current;
    const programsElement = programsRef.current;
    const track = trackRef.current;
    return root && hero && programsElement && track ? { root, hero, programs: programsElement, track } : null;
  };

  // Scroll → progress → hero fade, chapter copy, steps, backdrop and scene.
  useEffect(() => {
    let frame = 0;

    const update = () => {
      frame = 0;
      const current = layout();
      if (!current) {
        return;
      }
      const progress = progressFor(current);
      sceneRef.current?.setProgress(progress);

      // The hero copy fades as it scrolls away, while the Earth heads for the corner.
      if (heroCopyRef.current) {
        const away = progress / HERO_END;
        heroCopyRef.current.style.opacity = (1 - smoothstep(0.2, 0.8, away)).toFixed(3);
      }

      const [a, b, glow] = paletteAt(progress);
      if (backdropRef.current) {
        backdropRef.current.style.background = `radial-gradient(70rem 46rem at 78% 30%, ${css(glow)}, transparent 66%), linear-gradient(160deg, ${css(a)} 0%, ${css(b)} 100%)`;
      }

      let strongest: StoryChapterId = "identity";
      let strongestOpacity = 0.5;
      for (const chapterId of CHAPTER_IDS) {
        const element = chapterRefs.current[chapterId];
        const { opacity, shift } = chapterVisibility(chapterId, progress);
        if (element) {
          // Opacity only: every chapter stays in the accessibility tree.
          element.style.opacity = opacity.toFixed(3);
          element.style.transform = `translate3d(0, ${shift.toFixed(1)}px, 0)`;
          element.style.pointerEvents = opacity > 0.6 ? "auto" : "none";
        }
        if (opacity > strongestOpacity) {
          strongestOpacity = opacity;
          strongest = chapterId;
        }
      }
      // Between chapters the last one reached stays current.
      setActiveStep((previous) => (strongestOpacity > 0.5 ? strongest : progress < TRAIN_IN ? "identity" : previous));

      const programsBox = current.programs.getBoundingClientRect();
      const middle = window.innerHeight / 2;
      setStepsHidden(programsBox.top < middle && programsBox.bottom > middle);
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

  // Scene lifecycle: dynamic import, render only while the story is on screen.
  useEffect(() => {
    const host = canvasHostRef.current;
    const stage = stageRef.current;
    const root = rootRef.current;
    if (!host || !stage || !root) {
      return;
    }
    // Each scene instance owns its canvas, so a disposed instance (for example
    // after a fast remount) can never release another instance's WebGL context.
    const canvas = document.createElement("canvas");
    canvas.className = styles.canvas;
    canvas.setAttribute("aria-hidden", "true");
    host.appendChild(canvas);
    let disposed = false;
    let scene: StoryScene | null = null;
    const handleContextLost = (event: Event) => {
      event.preventDefault();
      onFailure();
    };
    canvas.addEventListener("webglcontextlost", handleContextLost);

    const observer = new IntersectionObserver(([entry]) => {
      visibleRef.current = entry.isIntersecting;
      if (entry.isIntersecting) {
        sceneRef.current?.start();
      } else {
        sceneRef.current?.stop();
      }
    });
    observer.observe(stage);

    const resizeObserver = new ResizeObserver(([entry]) => {
      sceneRef.current?.resize(entry.contentRect.width, entry.contentRect.height);
    });
    resizeObserver.observe(stage);

    import("./story_scene")
      .then(({ createStoryScene }) => {
        if (disposed) {
          throw new Error("Story unmounted before the scene was created");
        }
        return createStoryScene(canvas, {
          logo: BRAND_ASSETS.icon,
          landMask: earthLandMask,
          screens: {
            training: SCREENS.training,
            nutrition: SCREENS.nutrition,
            synced: SCREENS.synced,
            platform: SCREENS.platform
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
          created.setProgress(progressFor(current), true);
        }
        if (visibleRef.current) {
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
      canvas.remove();
    };
  }, [onFailure]);

  // The stage fills the viewport, so the pointer is read against the window.
  const handlePointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    sceneRef.current?.setPointer((event.clientX / window.innerWidth) * 2 - 1, -((event.clientY / window.innerHeight) * 2 - 1));
  };

  const scrollToStep = (id: StoryChapterId, behavior: ScrollBehavior) => {
    const current = layout();
    if (!current) {
      return;
    }
    const anchor = ANCHORS[id];
    const top = anchor === undefined ? current.root.getBoundingClientRect().top + window.scrollY : scrollTopFor(current, anchor);
    window.scrollTo({ top, behavior });
  };

  // Keyboard focus inside a faded chapter brings that chapter on screen.
  const revealFocusedChapter = (id: StoryChapterId) => {
    const element = chapterRefs.current[id];
    if (element && Number(element.style.opacity || "1") < 0.6) {
      // Instant, so the page-wide smooth scrolling cannot leave the chapter half faded.
      scrollToStep(id, "instant");
    }
  };

  return (
    <div ref={rootRef} className={joinClassNames(styles.story, ready && styles.storyReady)} onPointerMove={handlePointerMove}>
      <section ref={stageRef} className={styles.stage} aria-label={text.discover}>
        <div ref={backdropRef} className={styles.backdrop} aria-hidden="true" />
        {ready ? null : (
          <div className={styles.placeholder} aria-hidden="true">
            <BrandGlobe />
          </div>
        )}
        <div ref={canvasHostRef} className={styles.canvasHost} />

        <div className={styles.chapters}>
          {laterChapters.map((chapter) => (
            <div
              key={chapter.id}
              ref={(element) => {
                chapterRefs.current[chapter.id] = element;
              }}
              className={joinClassNames(styles.chapter, TONE_CLASS[chapter.tone], chapter.id === "start" && styles.chapterStart)}
              style={{ opacity: 0 }}
              onFocus={() => revealFocusedChapter(chapter.id)}
            >
              <ChapterCopy chapter={chapter} headingId={`story-${chapter.id}`} />
              {chapter.id === "start" ? <StartActions /> : null}
              {chapter.screen ? <p className={styles.sampleNote}>{text.sampleNote}</p> : null}
              {/* The device screens are drawn in WebGL; describe them for assistive technology. */}
              {chapter.screen ? <p className={styles.srOnly}>{chapter.screen.alt}</p> : null}
            </div>
          ))}
        </div>

        <nav className={joinClassNames(styles.steps, stepsHidden && styles.stepsHidden)} aria-label={text.steps}>
          <ol>
            {chapters.map((chapter, index) => (
              <li key={chapter.id}>
                <button
                  type="button"
                  className={styles.step}
                  aria-current={activeStep === chapter.id ? "step" : undefined}
                  onClick={() => scrollToStep(chapter.id, "smooth")}
                >
                  <span className={styles.stepIndex}>{String(index + 1).padStart(2, "0")}</span>
                  <span className={styles.stepName}>{chapter.navLabel}</span>
                </button>
              </li>
            ))}
          </ol>
        </nav>
      </section>

      <section ref={heroRef} className={joinClassNames(styles.hero, TONE_CLASS[identity.tone])} aria-labelledby="story-identity">
        <div className={styles.heroInner}>
          <div ref={heroCopyRef} className={styles.heroCopy}>
            <ChapterCopy chapter={identity} headingId="story-identity" as="h1" />
            <StartActions withEntryPoints />
          </div>
        </div>
        <p className={styles.scrollCue} aria-hidden="true">
          <ChevronsDown />
          {text.scrollCue}
        </p>
      </section>

      <div ref={programsRef} className={styles.programs}>
        {programs}
      </div>

      <div ref={trackRef} className={styles.track} aria-hidden="true" />
    </div>
  );
};
