const DEFAULT_BASE_URL = "https://notify.api.qeet.in";

export interface QeetNotifyOptions {
  baseUrl?: string;
}

export interface TriggerParams {
  subscriberId: string;
  payload?: Record<string, unknown>;
}

export class QeetNotify {
  private readonly apiKey: string;
  private readonly baseUrl: string;

  constructor(apiKey: string, options: QeetNotifyOptions = {}) {
    this.apiKey = apiKey;
    this.baseUrl = options.baseUrl ?? DEFAULT_BASE_URL;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const resp = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers: {
        "Content-Type": "application/json",
        "X-Qeet-Api-Key": this.apiKey,
      },
      body: body ? JSON.stringify(body) : undefined,
    });

    const raw = await resp.text();
    if (!resp.ok) {
      throw new Error(`Qeet Notify API ${resp.status}: ${raw}`);
    }
    return raw ? (JSON.parse(raw) as T) : ({} as T);
  }

  /** Trigger a notification event for a subscriber. */
  async trigger(event: string, params: TriggerParams): Promise<void> {
    await this.request("POST", "/v1/events", {
      event,
      subscriber_id: params.subscriberId,
      payload: params.payload ?? {},
    });
  }
}

export default QeetNotify;
