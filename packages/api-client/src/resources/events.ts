import type { TriggerEventRequest, TriggerEventResponse } from "@qeet-notify/shared-types";
import type { QeetNotifyApiClient } from "../client";

export class EventsResource {
  constructor(private readonly client: QeetNotifyApiClient) {}

  trigger(req: TriggerEventRequest, idempotencyKey?: string): Promise<TriggerEventResponse> {
    return this.client.trigger(req, idempotencyKey);
  }
}
