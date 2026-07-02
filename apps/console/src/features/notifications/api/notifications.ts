import { apiFetcher } from "@/lib/api";

export interface Notification {
  id: string;
  tenant_id: string;
  subscriber_id: string;
  channel: string;
  status: string;
  is_read: boolean;
  created_at: string;
}

export const notificationsApi = {
  list: (params?: { subscriber_id?: string; channel?: string; status?: string }) =>
    apiFetcher<{ notifications: Notification[]; total: number }>(
      "/v1/notifications",
      { params }
    ),
  get: (id: string) => apiFetcher<Notification>(`/v1/notifications/${id}`),
};
