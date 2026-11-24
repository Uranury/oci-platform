// The API client, shared by every screen so there is one token store and
// one refresh in flight (refresh tokens rotate: two concurrent refreshes
// would revoke each other).

const ACCESS = "access_token";
const REFRESH = "refresh_token";

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export const store = {
  get access(): string | null {
    return localStorage.getItem(ACCESS);
  },
  get refresh(): string | null {
    return localStorage.getItem(REFRESH);
  },
  set(pair: TokenPair) {
    localStorage.setItem(ACCESS, pair.access_token);
    localStorage.setItem(REFRESH, pair.refresh_token);
  },
  clear() {
    localStorage.removeItem(ACCESS);
    localStorage.removeItem(REFRESH);
  },
};

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

// fetch, with a network failure (offline, server unreachable) turned into an
// ApiError (status 0) that a screen can show as is, instead of the browser's
// "Failed to fetch" / "Load failed".
async function send(path: string, init: RequestInit): Promise<Response> {
  try {
    return await fetch(path, init);
  } catch {
    throw new ApiError(
      0,
      navigator.onLine
        ? "Can't reach Overlift right now. Try again in a moment."
        : "You're offline. Reconnect to load and save your training.",
    );
  }
}

let refreshing: Promise<boolean> | null = null; // parallel 401s refresh once

function refreshTokens(): Promise<boolean> {
  refreshing ??= (async () => {
    const rt = store.refresh;
    if (!rt) return false;
    const res = await send("/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: rt }),
    });
    if (!res.ok) return false;
    store.set((await res.json()) as TokenPair);
    return true;
  })().finally(() => {
    refreshing = null;
  });
  return refreshing;
}

// Ends this device's session: the local tokens go at once; revoking the
// refresh token on the server is best effort.
export function logout(): void {
  const rt = store.refresh;
  store.clear();
  if (rt) {
    fetch("/auth/logout", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: rt }),
    }).catch(() => {});
  }
  location.hash = "#/login";
}

interface Opts {
  auth?: boolean;
  retry?: boolean;
}

// Sends a request with the access token; a 401 is retried once after a
// token refresh, and if that fails the session is cleared and the login
// route shown.
async function authed(method: string, path: string, body: unknown, { auth = true, retry = true }: Opts): Promise<Response> {
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (auth && store.access) headers["Authorization"] = "Bearer " + store.access;

  const res = await send(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (res.status === 401 && auth && retry) {
    if (await refreshTokens()) return authed(method, path, body, { auth, retry: false });
    store.clear();
    location.hash = "#/login";
    throw new ApiError(401, "Session expired, please log in again");
  }
  return res;
}

async function failure(res: Response): Promise<ApiError> {
  const data = (await res.json().catch(() => null)) as { error?: string } | null;
  return new ApiError(res.status, data?.error || res.statusText);
}

// Sends a JSON request; resolves to the parsed body (null for 204) or throws
// ApiError with the server's error message.
export async function api<T = unknown>(method: string, path: string, body?: unknown, opts: Opts = {}): Promise<T> {
  const res = await authed(method, path, body, opts);
  if (res.status === 204) return null as T;
  if (!res.ok) throw await failure(res);
  return (await res.json().catch(() => null)) as T;
}

// Fetches a file the API sends as an attachment (the CSV export), with the
// name from its Content-Disposition. A plain link can't carry the bearer
// token, hence the fetch.
export async function download(path: string): Promise<{ blob: Blob; filename: string }> {
  const res = await authed("GET", path, undefined, {});
  if (!res.ok) throw await failure(res);
  const named = /filename="([^"]+)"/.exec(res.headers.get("Content-Disposition") ?? "");
  return { blob: await res.blob(), filename: named?.[1] ?? "overlift.csv" };
}
