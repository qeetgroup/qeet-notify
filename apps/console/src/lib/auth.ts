import { keyStore } from "./api";

export function isAuthenticated(): boolean {
  return !!keyStore.get();
}

export function getApiKey(): string | null {
  return keyStore.get();
}

export function signOut(): void {
  keyStore.clear();
  if (typeof window !== "undefined") {
    window.location.assign("/sign-in");
  }
}
