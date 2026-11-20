import { useEffect, useState } from "react";

// useMediaQuery mirrors a CSS media query so behaviour (for example whether a
// region is a scrollable tab stop, or whether a visual renders at all) can
// follow the same breakpoint as the layout. It reads the query on the first
// render, so a phone never renders (and downloads) a desktop-only element.
export const useMediaQuery = (query: string): boolean => {
  const [matches, setMatches] = useState(() => typeof window !== "undefined" && window.matchMedia(query).matches);

  useEffect(() => {
    const mediaQuery = window.matchMedia(query);
    const update = () => setMatches(mediaQuery.matches);

    update();
    mediaQuery.addEventListener("change", update);

    return () => {
      mediaQuery.removeEventListener("change", update);
    };
  }, [query]);

  return matches;
};
