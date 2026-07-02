export interface Workflow {
  id?: string;
  name: string;
  triggerEvent: string;
  status?: "active" | "paused" | "draft";
}

export class WorkflowsResource {
  constructor(
    private readonly baseUrl: string,
    private readonly headers: Record<string, string>
  ) {}

  async list(): Promise<Workflow[]> {
    const res = await fetch(`${this.baseUrl}/v1/workflows`, { headers: this.headers });
    if (!res.ok) throw new Error(`Qeet Notify API error ${res.status}`);
    return res.json();
  }

  async activate(id: string): Promise<Workflow> {
    const res = await fetch(`${this.baseUrl}/v1/workflows/${id}/activate`, {
      method: "POST",
      headers: this.headers,
    });
    if (!res.ok) throw new Error(`Qeet Notify API error ${res.status}`);
    return res.json();
  }

  async pause(id: string): Promise<Workflow> {
    const res = await fetch(`${this.baseUrl}/v1/workflows/${id}/pause`, {
      method: "POST",
      headers: this.headers,
    });
    if (!res.ok) throw new Error(`Qeet Notify API error ${res.status}`);
    return res.json();
  }
}
