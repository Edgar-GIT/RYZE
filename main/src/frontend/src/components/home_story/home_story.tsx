import type { ReactNode } from "react";

import { useImmersiveStory } from "@/hooks/use_immersive_story";

import { StoryImmersive } from "./story_immersive";
import { StoryStacked } from "./story_stacked";

interface HomeStoryProps {
  // Rendered right after the opening chapter, so visitors reach the programs first.
  programs: ReactNode;
}

// HomeStory presents RYZE in five chapters (identity, training, nutrition,
// platform, get started) with the programs right after the first one. Desktop
// pointers with WebGL and full motion get the scroll-driven Three.js
// experience; everything else gets the stacked layout, which never downloads
// Three.js.
export const HomeStory = ({ programs }: HomeStoryProps) => {
  const { immersive, disable } = useImmersiveStory();

  return immersive ? <StoryImmersive onFailure={disable} programs={programs} /> : <StoryStacked programs={programs} />;
};
