import { ArrowRight } from "lucide-react";

import { PublicAction } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { joinClassNames } from "@utils/class_names";

import { type StoryChapter, type StoryTone, useStory } from "./story_content";
import styles from "./story_parts.module.css";

export const TONE_CLASS: Record<StoryTone, string> = {
  navy: ui.toneNavy
};

interface ChapterCopyProps {
  chapter: StoryChapter;
  headingId: string;
  as?: "h1" | "h2";
}

// ChapterCopy renders a chapter's label, headline (with the brand-coloured
// continuation), lead and feature points. Shared by the 3D and stacked layouts.
export const ChapterCopy = ({ chapter, headingId, as: Heading = "h2" }: ChapterCopyProps) => (
  <div className={styles.copy}>
    <p className={ui.label}>{chapter.label}</p>
    <Heading id={headingId} className={Heading === "h1" ? ui.display : ui.heading}>
      {chapter.headline.map((segment) =>
        segment.highlight ? (
          <span key={segment.text} className={ui.hl}>
            {segment.text}
          </span>
        ) : (
          <span key={segment.text}>{segment.text}</span>
        )
      )}
    </Heading>
    <p className={ui.lead}>{chapter.lead}</p>
    {chapter.points ? (
      <ul className={styles.points}>
        {chapter.points.map((point) => {
          const Icon = point.icon;

          return (
            <li key={point.text}>
              <span className={ui.iconTile} aria-hidden="true">
                <Icon strokeWidth={1.9} />
              </span>
              <span>{point.text}</span>
            </li>
          );
        })}
      </ul>
    ) : null}
  </div>
);

interface StartActionsProps {
  withEntryPoints?: boolean;
}

// The two existing hero CTAs and, on the opening chapter, the three entry points.
export const StartActions = ({ withEntryPoints = false }: StartActionsProps) => {
  const { text, entryPoints } = useStory();
  return (
  <div className={styles.start}>
    <div className={joinClassNames(ui.actions, styles.actions)}>
      <PublicAction to="/services/generic-program" icon={<ArrowRight />}>
        {text.startFree}
      </PublicAction>
      <PublicAction to="/services" variant="ghostOnDark">
        {text.explorePlans}
      </PublicAction>
    </div>
    {withEntryPoints ? (
      <ul className={ui.chips} aria-label={text.waysToTrain}>
        {entryPoints.map((entry) => {
          const Icon = entry.icon;

          return (
            <li key={entry.label} className={ui.chip}>
              <Icon aria-hidden="true" strokeWidth={2} />
              {entry.label}
            </li>
          );
        })}
      </ul>
    ) : null}
  </div>
);
};
