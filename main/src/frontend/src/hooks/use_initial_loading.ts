import { useEffect, useState } from "react";

const INITIAL_LOADING_MINIMUM_MS = 900;

export const useInitialLoading = (): boolean => {
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let isMounted = true;

    const minimumDuration = new Promise<void>((resolve) => {
      window.setTimeout(resolve, INITIAL_LOADING_MINIMUM_MS);
    });

    const fontsReady =
      "fonts" in document
        ? document.fonts.ready.then(() => undefined)
        : Promise.resolve();

    Promise.all([minimumDuration, fontsReady]).then(() => {
      if (isMounted) {
        setIsLoading(false);
      }
    });

    return () => {
      isMounted = false;
    };
  }, []);

  return isLoading;
};
