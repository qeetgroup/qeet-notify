import { keyStore } from "./api";

const SESSION_KEY = "qn.identity";

export const sessionStore = {
  get: (): { userId: string } | null => {
    try {
      const raw = typeof window !== "undefined" ? window.localStorage.getItem(SESSION_KEY) : null;
      return raw ? JSON.parse(raw) : null;
    } catch {
      return null;
    }
  },
  set: (userId: string) =>
    window.localStorage.setItem(SESSION_KEY, JSON.stringify({ userId })),
  clear: () => window.localStorage.removeItem(SESSION_KEY),
};

/** Returns true if the user has either a qeet-id session OR a stored API key. */
export function isAuthenticated(): boolean {
  return !!sessionStore.get() || !!keyStore.get();
}

export function getApiKey(): string | null {
  return keyStore.get();
}

export function signOut(): void {
  sessionStore.clear();
  keyStore.clear();
  if (typeof window !== "undefined") {
    window.location.assign("/sign-in");
  }
}
