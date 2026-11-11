import { Fragment } from "react";
import type { ReactNode } from "react";

import { Reveal } from "@/components/reveal/reveal";
import ui from "@/components/public_ui/public_ui.module.css";
import { BrandGlobe } from "@/components/brand_globe/brand_globe";
import { LaptopMockup, PhoneMockup } from "@/components/showcase/showcase";
import { joinClassNames } from "@utils/class_names";

import { CALLOUTS, SAMPLE_DATA_NOTE, SCREENS, STORY_CHAPTERS, type StoryChapter } from "./story_content";
import { ChapterCopy, StartActions, TONE_CLASS } from "./story_parts";
import styles from "./story_stacked.module.css";

const ChapterVisual = ({ chapter }: { chapter: StoryChapter }) => {
  if (chapter.id === "identity") {
    return <BrandGlobe className={styles.orb} />;
  }
  if (chapter.id === "platform") {
    return (
      <Reveal className={styles.platformVisual} y={24}>
        <LaptopMockup src={SCREENS.platform.src} alt={SCREENS.platform.alt} />
        <PhoneMockup className={styles.platformPhone} src={SCREENS.training.src} alt="" />
        <img className={styles.syncBadge} src={CALLOUTS.sync} alt="" aria-hidden="true" loading="lazy" decoding="async" />
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
        <img className={joinClassNames(styles.callout, styles.calloutTop)} src={first} alt="" aria-hidden="true" loading="lazy" decoding="async" />
        <img className={joinClassNames(styles.callout, styles.calloutBottom)} src={second} alt="" aria-hidden="true" loading="lazy" decoding="async" />
      </Reveal>
    );
  }
  return null;
};

// StoryStacked tells the same five-chapter story as the desktop 3D experience,
// as ordinary sections: the RYZE globe, the programs, then the app screens in
// device frames.
export const StoryStacked = ({ programs }: { programs: ReactNode }) => (
  <>
    {STORY_CHAPTERS.map((chapter, index) => (
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
          <div className={styles.visual}>
            <ChapterVisual chapter={chapter} />
          </div>
          <div className={styles.text}>
            <ChapterCopy chapter={chapter} headingId={`story-${chapter.id}`} as={index === 0 ? "h1" : "h2"} />
            {chapter.id === "identity" || chapter.id === "start" ? (
              <StartActions withEntryPoints={chapter.id === "identity"} />
            ) : null}
            {chapter.screen ? <p className={ui.caption}>{SAMPLE_DATA_NOTE}</p> : null}
          </div>
        </div>
      </section>
      {index === 0 ? programs : null}
      </Fragment>
    ))}
  </>
);
