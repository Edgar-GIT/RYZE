/* RYZE service worker.
 *
 * What it caches, and why it is safe:
 * - Built assets under /assets/ (file names carry a content hash, so a cached
 *   file can never be stale) and the app icons: cache first.
 * - Page navigations: network first, so an online visit always gets the
 *   current app; the last page shell is kept only to open the app offline.
 *
 * What it never touches: the API (/api/), any other origin, and any request
 * that is not a GET. Account data, sessions and payment pages always go to
 * the network and are never stored on the device by this worker.
 *
 * Updates: a new worker waits until the page asks it to take over (the app
 * shows "A new version is available"), then old caches are deleted.
 */
// Bump VERSION whenever these rules change: the activating worker deletes
// every cache of an older version.
const VERSION = "ryze-sw-v2";
const STATIC_CACHE = `${VERSION}-static`;
const SHELL_CACHE = `${VERSION}-shell`;
const MAX_STATIC_ENTRIES = 120;

self.addEventListener("install", () => {
  // Nothing is precached: the first online visit fills the caches.
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      const keys = await caches.keys();
      await Promise.all(keys.filter((key) => !key.startsWith(VERSION)).map((key) => caches.delete(key)));
      await self.clients.claim();
    })()
  );
});

self.addEventListener("message", (event) => {
  if (event.data === "SKIP_WAITING") {
    self.skipWaiting();
  }
});

// Only content-hashed build output and the app icons are cached; any other
// file always comes from the network, so it can never be served stale.
const isStaticPath = (pathname) => pathname.startsWith("/assets/") || pathname.startsWith("/icons/");

const trim = async (cacheName, max) => {
  const cache = await caches.open(cacheName);
  const keys = await cache.keys();
  for (let i = 0; i < keys.length - max; i += 1) {
    await cache.delete(keys[i]);
  }
};

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") {
    return;
  }
  const url = new URL(request.url);
  if (url.origin !== self.location.origin || url.pathname.startsWith("/api/") || url.pathname === "/sw.js") {
    return;
  }

  if (request.mode === "navigate") {
    event.respondWith(
      (async () => {
        try {
          const response = await fetch(request);
          if (response.ok && response.type === "basic") {
            const cache = await caches.open(SHELL_CACHE);
            await cache.put("/", response.clone());
          }
          return response;
        } catch {
          const cached = await caches.match("/", { cacheName: SHELL_CACHE });
          return cached || Response.error();
        }
      })()
    );
    return;
  }

  if (isStaticPath(url.pathname)) {
    event.respondWith(
      (async () => {
        const cached = await caches.match(request, { cacheName: STATIC_CACHE });
        if (cached) {
          return cached;
        }
        const response = await fetch(request);
        if (response.ok && response.type === "basic") {
          const cache = await caches.open(STATIC_CACHE);
          await cache.put(request, response.clone());
          event.waitUntil(trim(STATIC_CACHE, MAX_STATIC_ENTRIES));
        }
        return response;
      })()
    );
  }
});
