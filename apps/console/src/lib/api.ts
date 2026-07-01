const API_KEY_STORAGE_KEY = "qn.api_key";

export const API_BASE =
  (import.meta.env?.VITE_API_URL as string | undefined) ?? "http://localhost:8080";

export class ApiError extends Error {
  status: number;
  code: string;
  details?: unknown;

  constructor(status: number, code: string, message: string, details?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

export const keyStore = {
  get: () =>
    typeof window !== "undefined" ? window.localStorage.getItem(API_KEY_STORAGE_KEY) : null,
  set: (k: string) => window.localStorage.setItem(API_KEY_STORAGE_KEY, k),
  clear: () => window.localStorage.removeItem(API_KEY_STORAGE_KEY),
};

type RequestOpts = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  query?: Record<string, string | number | boolean | undefined>;
  signal?: AbortSignal;
};

function onAuthLost() {
  keyStore.clear();
  if (typeof window !== "undefined" && window.location.pathname !== "/sign-in") {
    window.location.assign("/sign-in");
  }
}

export async function api<T = unknown>(path: string, opts: RequestOpts = {}): Promise<T> {
  const { method = "GET", body, query, signal } = opts;

  const url = new URL(path.startsWith("/") ? path.slice(1) : path, `${API_BASE}/`);
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null && v !== "") url.searchParams.set(k, String(v));
    }
  }

  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const key = keyStore.get();
  if (key) headers["X-Qeet-Api-Key"] = key;

  const res = await fetch(url, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    signal,
  });

  if (res.status === 401) {
    onAuthLost();
    throw new ApiError(401, "unauthorized", "Session expired. Please sign in again.");
  }

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  const data = text ? safeParse(text) : null;

  if (!res.ok) {
    const err = (
      data as { error?: { code?: string; message?: string; details?: unknown } } | null
    )?.error;
    throw new ApiError(
      res.status,
      err?.code ?? `http_${res.status}`,
      err?.message ?? res.statusText ?? "Request failed",
      err?.details,
    );
  }

  return data as T;
}

function safeParse(s: string): unknown {
  try {
    return JSON.parse(s);
  } catch {
    return s;
  }
}
