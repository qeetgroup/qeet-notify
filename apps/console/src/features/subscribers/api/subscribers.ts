import { apiFetcher, apiCall } from "@/lib/api";

export interface Subscriber {
  id: string;
  external_id: string;
  email?: string;
  phone?: string;
  created_at: string;
}

export const subscribersApi = {
  list: () => apiFetcher<{ subscribers: Subscriber[] }>("/v1/subscribers"),
  get: (id: string) => apiFetcher<Subscriber>(`/v1/subscribers/${id}`),
  create: (data: Partial<Subscriber>) =>
    apiCall("/v1/subscribers", { method: "POST", body: data }),
  update: (id: string, data: Partial<Subscriber>) =>
    apiCall(`/v1/subscribers/${id}`, { method: "PUT", body: data }),
  remove: (id: string) => apiCall(`/v1/subscribers/${id}`, { method: "DELETE" }),
  getPreferences: (id: string) =>
    apiFetcher<Record<string, boolean>>(`/v1/subscribers/${id}/preferences`),
  updatePreferences: (id: string, prefs: Record<string, boolean>) =>
    apiCall(`/v1/subscribers/${id}/preferences`, { method: "PUT", body: prefs }),
};
