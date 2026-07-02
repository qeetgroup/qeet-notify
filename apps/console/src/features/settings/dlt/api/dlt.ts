import { apiFetcher, apiCall } from "@/lib/api";

export interface DLTTemplate {
  id: string;
  template_id: string;
  sender_id: string;
  category: "transactional" | "promotional" | "otp";
  body_regex: string;
  status: string;
  carrier?: string;
}

export const dltApi = {
  list: () => apiFetcher<{ templates: DLTTemplate[] }>("/v1/dlt/templates"),
  register: (data: Partial<DLTTemplate>) =>
    apiCall("/v1/dlt/templates", { method: "POST", body: data }),
  update: (id: string, data: Partial<DLTTemplate>) =>
    apiCall(`/v1/dlt/templates/${id}`, { method: "PUT", body: data }),
  remove: (id: string) => apiCall(`/v1/dlt/templates/${id}`, { method: "DELETE" }),
};
