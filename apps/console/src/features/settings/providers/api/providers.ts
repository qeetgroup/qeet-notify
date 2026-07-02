import { apiFetcher, apiCall } from "@/lib/api";

export interface ProviderConfig {
  id: string;
  channel: string;
  provider_name: string;
  priority: number;
  is_active: boolean;
  created_at: string;
}

export const providersApi = {
  list: () => apiFetcher<{ providers: ProviderConfig[] }>("/v1/providers"),
  create: (data: Partial<ProviderConfig> & { config: Record<string, string> }) =>
    apiCall("/v1/providers", { method: "POST", body: data }),
  update: (id: string, data: Partial<ProviderConfig>) =>
    apiCall(`/v1/providers/${id}`, { method: "PUT", body: data }),
  remove: (id: string) => apiCall(`/v1/providers/${id}`, { method: "DELETE" }),
};
