// Where a visitor wanted to go before being asked to sign in (for example the
// middle of a premium journey, or a public profile to follow). /app reads it
// once after sign-in. Only same-site service and profile paths are accepted,
// so the value can never redirect away.
const KEY = "ryze.afterLogin";
const ALLOWED = /^\/(services\/[A-Za-z0-9\-/?=&_%.]*|u\/[a-z0-9._]{3,30})$/;

export const rememberAfterLogin = (path: string) => {
  try {
    if (ALLOWED.test(path)) sessionStorage.setItem(KEY, path);
  } catch {
    // Storage may be unavailable (private mode); the visitor lands on home.
  }
};

export const takeAfterLogin = (): string | null => {
  try {
    const value = sessionStorage.getItem(KEY);
    sessionStorage.removeItem(KEY);
    return value && ALLOWED.test(value) ? value : null;
  } catch {
    return null;
  }
};
