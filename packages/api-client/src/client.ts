import type { TriggerEventRequest, TriggerEventResponse } from "@qeet-notify/shared-types";

export interface QeetNotifyClientConfig {
  apiKey: string;
  baseUrl?: string;
}

const DEFAULT_BASE_URL = "https://notify.api.qeet.in";

export class QeetNotifyApiClient {
  private readonly baseUrl: string;
  private readonly headers: Record<string, string>;

  constructor({ apiKey, baseUrl = DEFAULT_BASE_URL }: QeetNotifyClientConfig) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.headers = {
      "Content-Type": "application/json",
      "X-Qeet-Api-Key": apiKey,
      "User-Agent": "@qeet-notify/api-client/0.1.0",
    };
  }

  async trigger(
    req: TriggerEventRequest,
    idempotencyKey?: string
  ): Promise<TriggerEventResponse> {
    const res = await fetch(`${this.baseUrl}/v1/events`, {
      method: "POST",
      headers: {
        ...this.headers,
        "Idempotency-Key": idempotencyKey ?? crypto.randomUUID(),
      },
      body: JSON.stringify({
        name: req.name,
        subscriber_id: req.subscriberId,
        payload: req.payload ?? {},
      }),
    });
    if (!res.ok) {
      const body = await res.text().catch(() => "");
      throw new Error(`Qeet Notify API error ${res.status}: ${body}`);
    }
    return res.json();
  }
}
