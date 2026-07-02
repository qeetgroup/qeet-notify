import type { Notification, NotificationListResponse } from "@qeet-notify/shared-types";

export interface ListNotificationsParams {
  limit?: number;
  offset?: number;
  channel?: string;
  status?: string;
}

export class NotificationsResource {
  constructor(
    private readonly baseUrl: string,
    private readonly headers: Record<string, string>
  ) {}

  async list(params: ListNotificationsParams = {}): Promise<NotificationListResponse> {
    const qs = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)])
    );
    const res = await fetch(`${this.baseUrl}/v1/notifications?${qs}`, { headers: this.headers });
    if (!res.ok) throw new Error(`Qeet Notify API error ${res.status}`);
    return res.json();
  }

  async get(id: string): Promise<Notification> {
    const res = await fetch(`${this.baseUrl}/v1/notifications/${id}`, { headers: this.headers });
    if (!res.ok) throw new Error(`Qeet Notify API error ${res.status}`);
    return res.json();
  }
}
