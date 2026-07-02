import { apiFetcher, apiCall } from "@/lib/api";

export interface Template {
  id: string;
  name: string;
  channel: string;
  subject?: string;
  body: string;
  is_active: boolean;
  created_at: string;
}

export const templatesApi = {
  list: () => apiFetcher<{ templates: Template[] }>("/v1/templates"),
  get: (id: string) => apiFetcher<Template>(`/v1/templates/${id}`),
  create: (data: Partial<Template>) => apiCall("/v1/templates", { method: "POST", body: data }),
  update: (id: string, data: Partial<Template>) =>
    apiCall(`/v1/templates/${id}`, { method: "PUT", body: data }),
  remove: (id: string) => apiCall(`/v1/templates/${id}`, { method: "DELETE" }),
  publish: (id: string) => apiCall(`/v1/templates/${id}/publish`, { method: "POST" }),
};
