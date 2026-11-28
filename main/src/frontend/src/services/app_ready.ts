// Whether the first loading screen has gone (owner request, 2026-11-28: the
// Home hero's entrance must play in view, not behind the loading screen).
let ready = false;

export const markAppReady = () => {
  ready = true;
};

export const isAppReady = () => ready;
