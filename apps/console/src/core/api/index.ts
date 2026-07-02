// Canonical API client re-export.
// Active implementation lives in src/lib/api.ts for backward compat.
// Migrate to @/core/api imports incrementally.
export { apiFetcher, apiCall, setApiKey, getApiKey, ApiError } from "../../lib/api";
