import { useEffect, useState } from "react";

// useMediaQuery mirrors a CSS media query so behaviour (for example whether a
// region is a scrollable tab stop) can follow the same breakpoint as the layout.
export const useMediaQuery = (query: string): boolean => {
  const [matches, setMatches] = useState(false);

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
