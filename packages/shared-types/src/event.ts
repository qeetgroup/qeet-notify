export interface TriggerEventRequest {
  name: string;
  subscriberId: string;
  payload?: Record<string, unknown>;
}

export interface TriggerEventResponse {
  id: string;
  status: "accepted";
}

export interface DeliveryEvent {
  id: string;
  notificationId: string;
  tenantId: string;
  channel: string;
  status: string;
  provider: string;
  latencyMs: number;
  occurredAt: string;
}
