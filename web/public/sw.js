// Overlift service worker: keeps the app shell on the device so the app
// opens without a connection. It never touches the API; training data always
// comes from the server, and screens say so when it can't be reached.
//
//   page loads  network first, the cached shell when offline
//   /assets/    cache first (Vite hashes every name, so a cached file is
//               never stale); entries the current shell no longer uses
//               are dropped whenever a fresh shell is stored
//   icons etc.  network first, cache as fallback
//   anything else (the API) is left alone.

const CACHE = "overlift-shell-v1";
const STATIC = ["/favicon.svg", "/favicon-32.png", "/manifest.webmanifest", "/icon-192.png"];

// Every /assets/ URL a text file refers to (scripts and CSS in index.html,
// fonts in the CSS).
function assetURLs(text) {
  return [...new Set(text.match(/\/assets\/[^"'()\s]+/g) || [])];
}

// Stores index.html (from `res`, or fetched) and everything it needs, then
// drops assets the new shell no longer references.
async function storeShell(res) {
  const cache = await caches.open(CACHE);
  res ??= await fetch("/", { cache: "no-cache" });
  if (!res.ok) return;
  const html = await res.clone().text();
  await cache.put("/", res);

  const wanted = new Set();
  for (const url of assetURLs(html)) {
    wanted.add(url);
    let hit = await cache.match(url);
    if (!hit) {
      const r = await fetch(url);
      if (!r.ok) continue;
      await cache.put(url, r.clone());
      hit = r;
    }
    if (url.endsWith(".css")) for (const inner of assetURLs(await hit.text())) wanted.add(inner);
  }
  await Promise.all(
    [...wanted].map(async (url) => {
      if (!(await cache.match(url))) {
        const r = await fetch(url);
        if (r.ok) await cache.put(url, r);
      }
    }),
  );
  for (const req of await cache.keys()) {
    const path = new URL(req.url).pathname;
    if (path.startsWith("/assets/") && !wanted.has(path)) await cache.delete(req);
  }
}

self.addEventListener("install", (event) => {
  event.waitUntil(
    (async () => {
      await storeShell();
      const cache = await caches.open(CACHE);
      await Promise.all(STATIC.map((url) => cache.add(url).catch(() => {})));
      await self.skipWaiting();
    })(),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      for (const name of await caches.keys()) if (name !== CACHE) await caches.delete(name);
      await self.clients.claim();
    })(),
  );
});

self.addEventListener("fetch", (event) => {
  const req = event.request;
  const url = new URL(req.url);
  if (req.method !== "GET" || url.origin !== self.location.origin) return;

  if (req.mode === "navigate") {
    event.respondWith(
      (async () => {
        try {
          const res = await fetch(req);
          const type = res.headers.get("Content-Type") || "";
          if (res.ok && url.pathname === "/" && type.startsWith("text/html")) {
            event.waitUntil(storeShell(res.clone()).catch(() => {}));
          }
          return res;
        } catch {
          return (await caches.match("/")) || Response.error();
        }
      })(),
    );
    return;
  }

  if (url.pathname.startsWith("/assets/")) {
    event.respondWith(
      (async () => {
        const hit = await caches.match(req);
        if (hit) return hit;
        const res = await fetch(req);
        if (res.ok) {
          const copy = res.clone();
          event.waitUntil(caches.open(CACHE).then((c) => c.put(req, copy)));
        }
        return res;
      })(),
    );
    return;
  }

  if (STATIC.includes(url.pathname)) {
    event.respondWith(
      (async () => {
        try {
          const res = await fetch(req);
          if (res.ok) {
            const copy = res.clone();
            event.waitUntil(caches.open(CACHE).then((c) => c.put(req, copy)));
          }
          return res;
        } catch {
          return (await caches.match(req)) || Response.error();
        }
      })(),
    );
  }
});
