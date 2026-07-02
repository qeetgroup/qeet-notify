export type NotificationStatus =
  | "pending"
  | "sent"
  | "delivered"
  | "failed"
  | "suppressed"
  | "bounced"
  | "opened"
  | "clicked";

export interface Notification {
  id: string;
  tenantId: string;
  subscriberId: string;
  channel: string;
  status: NotificationStatus;
  subject?: string;
  body?: string;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
}

export interface NotificationListResponse {
  data: Notification[];
  meta: {
    total: number;
    limit: number;
    offset: number;
  };
}
