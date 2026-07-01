"use client";

import { use, useState } from "react";
import useSWR, { mutate } from "swr";
import { Button, Badge, Input, Textarea, Skeleton } from "@qeetrix/ui";
import { apiFetcher, apiCall } from "@/lib/api";

type Template = {
  id: string;
  name: string;
  channel: string;
  locale: string;
  subject?: string;
  body: string;
  is_active: boolean;
  version: number;
  metadata: Record<string, unknown>;
};

export default function TemplateDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { data: tmpl, isLoading } = useSWR<Template>(`/api/v1/templates/${id}`, apiFetcher);
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState<Partial<Template>>({});
  const [saving, setSaving] = useState(false);

  function startEdit() {
    if (!tmpl) return;
    setForm({ name: tmpl.name, subject: tmpl.subject, body: tmpl.body });
    setEditing(true);
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const res = await apiCall(`/api/v1/templates/${id}`, {
        method: "PUT",
        body: JSON.stringify(form),
      });
      if (res.ok) {
        await mutate(`/api/v1/templates/${id}`);
        setEditing(false);
      }
    } finally {
      setSaving(false);
    }
  }

  async function handlePublish() {
    await apiCall(`/api/v1/templates/${id}/publish`, { method: "POST" });
    await mutate(`/api/v1/templates/${id}`);
  }

  if (isLoading) return <Skeleton className="h-64 rounded-lg" />;
  if (!tmpl) return <p className="text-sm text-muted-foreground">Template not found.</p>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{tmpl.name}</h1>
            <p className="text-sm text-muted-foreground">{tmpl.locale} · v{tmpl.version}</p>
          </div>
          <Badge variant="outline">{tmpl.channel}</Badge>
          <Badge variant={tmpl.is_active ? "default" : "secondary"}>
            {tmpl.is_active ? "active" : "archived"}
          </Badge>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handlePublish}>Publish v{tmpl.version}</Button>
          <Button onClick={startEdit}>Edit</Button>
        </div>
      </div>

      {editing ? (
        <form onSubmit={handleSave} className="space-y-4 max-w-2xl">
          <div className="space-y-1">
            <label className="text-sm font-medium">Name</label>
            <Input
              value={(form.name as string) ?? ""}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </div>
          {tmpl.channel === "email" && (
            <div className="space-y-1">
              <label className="text-sm font-medium">Subject</label>
              <Input
                value={(form.subject as string) ?? ""}
                onChange={(e) => setForm({ ...form, subject: e.target.value })}
              />
            </div>
          )}
          <div className="space-y-1">
            <label className="text-sm font-medium">Body</label>
            <Textarea
              value={(form.body as string) ?? ""}
              onChange={(e) => setForm({ ...form, body: e.target.value })}
              rows={14}
              className="font-mono text-xs"
            />
          </div>
          <div className="flex gap-2">
            <Button type="submit" disabled={saving}>{saving ? "Saving…" : "Save"}</Button>
            <Button type="button" variant="outline" onClick={() => setEditing(false)}>Cancel</Button>
          </div>
        </form>
      ) : (
        <div className="space-y-4 max-w-2xl">
          {tmpl.subject && (
            <div>
              <p className="text-xs font-medium text-muted-foreground mb-1">Subject</p>
              <p className="text-sm">{tmpl.subject}</p>
            </div>
          )}
          <div>
            <p className="text-xs font-medium text-muted-foreground mb-1">Body</p>
            <pre className="rounded-lg bg-muted p-4 text-xs overflow-auto max-h-96 whitespace-pre-wrap">
              {tmpl.body}
            </pre>
          </div>
          {tmpl.metadata.published_version != null && (
            <p className="text-xs text-muted-foreground">
              Published version: {String(tmpl.metadata.published_version)}
              {tmpl.metadata.published_at ? ` · ${new Date(String(tmpl.metadata.published_at)).toLocaleString()}` : ""}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
