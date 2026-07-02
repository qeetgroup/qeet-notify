export type WebhookEventType =
  | "notification.sent"
  | "notification.delivered"
  | "notification.failed"
  | "notification.bounced"
  | "notification.opened"
  | "notification.clicked";

export interface WebhookPayload {
  id: string;
  type: WebhookEventType;
  tenantId: string;
  notificationId: string;
  subscriberId: string;
  channel: string;
  timestamp: string;
  data: Record<string, unknown>;
}
