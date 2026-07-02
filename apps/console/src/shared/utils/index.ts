// Shared utility functions. Add helpers here as the codebase grows.

/** Format a date string to a human-readable relative time. */
export function formatRelativeTime(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return new Date(dateStr).toLocaleDateString();
}

/** Truncate a UUID to a short display prefix: "abcd1234..." */
export function shortID(id: string): string {
  return id.slice(0, 8) + "...";
}
