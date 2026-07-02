import { apiFetcher, apiCall } from "@/lib/api";

export interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  scope: "full" | "read" | "send";
  created_at: string;
}

export const apiKeysApi = {
  list: () => apiFetcher<{ keys: ApiKey[] }>("/v1/api-keys"),
  create: (data: { name: string; scope: ApiKey["scope"] }) =>
    apiCall<{ id: string; key: string }>("/v1/api-keys", { method: "POST", body: data }),
  revoke: (id: string) => apiCall(`/v1/api-keys/${id}`, { method: "DELETE" }),
};
