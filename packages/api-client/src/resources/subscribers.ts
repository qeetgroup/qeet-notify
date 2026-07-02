export interface Subscriber {
  id?: string;
  externalId: string;
  email?: string;
  phone?: string;
  firstName?: string;
  lastName?: string;
  data?: Record<string, unknown>;
}

export class SubscribersResource {
  constructor(
    private readonly baseUrl: string,
    private readonly headers: Record<string, string>
  ) {}

  async get(externalId: string): Promise<Subscriber> {
    const res = await fetch(`${this.baseUrl}/v1/subscribers/${externalId}`, { headers: this.headers });
    if (!res.ok) throw new Error(`Qeet Notify API error ${res.status}`);
    return res.json();
  }

  async upsert(subscriber: Subscriber): Promise<Subscriber> {
    const res = await fetch(`${this.baseUrl}/v1/subscribers`, {
      method: "POST",
      headers: { ...this.headers, "Content-Type": "application/json" },
      body: JSON.stringify(subscriber),
    });
    if (!res.ok) throw new Error(`Qeet Notify API error ${res.status}`);
    return res.json();
  }
}
