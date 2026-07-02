import { apiFetcher, apiCall } from "@/lib/api";

export interface Workflow {
  id: string;
  name: string;
  trigger_event: string;
  steps: Record<string, unknown>[];
  is_active: boolean;
  created_at: string;
}

export const workflowsApi = {
  list: () => apiFetcher<{ workflows: Workflow[] }>("/v1/workflows"),
  get: (id: string) => apiFetcher<Workflow>(`/v1/workflows/${id}`),
  runs: (id: string) => apiFetcher<{ runs: unknown[] }>(`/v1/workflows/${id}/runs`),
  create: (data: Partial<Workflow>) => apiCall("/v1/workflows", { method: "POST", body: data }),
  update: (id: string, data: Partial<Workflow>) =>
    apiCall(`/v1/workflows/${id}`, { method: "PUT", body: data }),
  activate: (id: string) => apiCall(`/v1/workflows/${id}/activate`, { method: "POST" }),
  pause: (id: string) => apiCall(`/v1/workflows/${id}/pause`, { method: "POST" }),
  archive: (id: string) => apiCall(`/v1/workflows/${id}`, { method: "DELETE" }),
};
