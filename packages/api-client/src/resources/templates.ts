export interface Template {
  id?: string;
  name: string;
  channel: string;
  subject?: string;
  body: string;
  status?: "draft" | "published";
}

export class TemplatesResource {
  constructor(
    private readonly baseUrl: string,
    private readonly headers: Record<string, string>
  ) {}

  async list(): Promise<Template[]> {
    const res = await fetch(`${this.baseUrl}/v1/templates`, { headers: this.headers });
    if (!res.ok) throw new Error(`Qeet Notify API error ${res.status}`);
    return res.json();
  }

  async create(template: Template): Promise<Template> {
    const res = await fetch(`${this.baseUrl}/v1/templates`, {
      method: "POST",
      headers: { ...this.headers, "Content-Type": "application/json" },
      body: JSON.stringify(template),
    });
    if (!res.ok) throw new Error(`Qeet Notify API error ${res.status}`);
    return res.json();
  }

  async publish(id: string): Promise<Template> {
    const res = await fetch(`${this.baseUrl}/v1/templates/${id}/publish`, {
      method: "POST",
      headers: this.headers,
    });
    if (!res.ok) throw new Error(`Qeet Notify API error ${res.status}`);
    return res.json();
  }
}
