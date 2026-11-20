import { Fragment } from "react";
import type { ReactNode } from "react";

import { Reveal } from "@/components/reveal/reveal";
import { useMediaQuery } from "@/hooks/use_media_query";
import ui from "@/components/public_ui/public_ui.module.css";
import { BrandGlobe } from "@/components/brand_globe/brand_globe";
import { LaptopMockup, PhoneMockup } from "@/components/showcase/showcase";
import { joinClassNames } from "@utils/class_names";

import { CALLOUTS, SCREENS, type StoryChapter, useStory } from "./story_content";
import { HeroShowcase } from "./hero_showcase";
import { ChapterCopy, StartActions, TONE_CLASS } from "./story_parts";
import styles from "./story_stacked.module.css";

// Phones and tablets get the plain version: copy and the real app screens, no
// stand-ins for the desktop 3D (the drone opening and planet RYZE render only
// as the desktop fallback, from 1024 px, when WebGL or full motion is off).
const DESKTOP_QUERY = "(min-width: 1024px)";
// The decorative cards lifted out of the screens join from tablet width.
const TABLET_QUERY = "(min-width: 768px)";

const hasVisual = (chapter: StoryChapter, desktop: boolean) =>
  chapter.id === "identity" || chapter.id === "start" ? desktop : Boolean(chapter.screen);

const ChapterVisual = ({ chapter, chapters, tablet }: { chapter: StoryChapter; chapters: StoryChapter[]; tablet: boolean }) => {
  if (chapter.id === "identity") {
    // The hero's opening, light: the drone projects the phone, a dumbbell and a meal.
    return <HeroShowcase className={styles.showcase} screen={SCREENS.training} />;
  }
  if (chapter.id === "start") {
    // Planet RYZE: "join planet RYZE".
    return <BrandGlobe className={styles.orb} />;
  }
  if (chapter.id === "platform") {
    const training = chapters.find((entry) => entry.id === "train")?.screen;
    return (
      <Reveal className={styles.platformVisual} y={24}>
        {chapter.screen ? <LaptopMockup src={chapter.screen.src} alt={chapter.screen.alt} /> : null}
        {tablet && training ? <PhoneMockup className={styles.platformPhone} src={training.src} alt="" /> : null}
        {tablet ? <img className={styles.syncBadge} src={CALLOUTS.sync} alt="" aria-hidden="true" loading="lazy" decoding="async" /> : null}
      </Reveal>
    );
  }
  if (chapter.screen) {
    // Two pieces of the screen lifted out beside the phone, as on desktop.
    // The last two cards of each chapter: the logged set and rest timer, the protein and meal.
    const [, first, second] = chapter.id === "fuel" ? CALLOUTS.fuel : CALLOUTS.train;
    return (
      <Reveal className={styles.phoneVisual} y={24}>
        <PhoneMockup src={chapter.screen.src} alt={chapter.screen.alt} />
        {tablet ? (
          <>
            <img className={joinClassNames(styles.callout, styles.calloutTop)} src={first} alt="" aria-hidden="true" loading="lazy" decoding="async" />
            <img className={joinClassNames(styles.callout, styles.calloutBottom)} src={second} alt="" aria-hidden="true" loading="lazy" decoding="async" />
          </>
        ) : null}
      </Reveal>
    );
  }
  return null;
};

// StoryStacked tells the same five-chapter story as the desktop 3D experience,
// as ordinary sections: the RYZE hologram, the programs, then the app screens in
// device frames.
export const StoryStacked = ({ programs }: { programs: ReactNode }) => {
  const { text, chapters } = useStory();
  const desktop = useMediaQuery(DESKTOP_QUERY);
  const tablet = useMediaQuery(TABLET_QUERY);
  return (
  <>
    {chapters.map((chapter, index) => (
      <Fragment key={chapter.id}>
      <section
        className={joinClassNames(
          ui.section,
          TONE_CLASS[chapter.tone],
          styles.chapter,
          styles[chapter.id]
        )}
        aria-labelledby={`story-${chapter.id}`}
      >
        <div className={joinClassNames(ui.container, styles.layout)}>
          {hasVisual(chapter, desktop) ? (
            <div className={styles.visual}>
              <ChapterVisual chapter={chapter} chapters={chapters} tablet={tablet} />
            </div>
          ) : null}
          <div className={styles.text}>
            <ChapterCopy chapter={chapter} headingId={`story-${chapter.id}`} as={index === 0 ? "h1" : "h2"} />
            {chapter.id === "identity" || chapter.id === "start" ? (
              <StartActions withEntryPoints={chapter.id === "identity"} />
            ) : null}
            {chapter.screen ? <p className={ui.caption}>{text.sampleNote}</p> : null}
          </div>
        </div>
      </section>
      {index === 0 ? programs : null}
      </Fragment>
    ))}
  </>
);
};
