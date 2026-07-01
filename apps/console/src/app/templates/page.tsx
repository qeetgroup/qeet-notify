"use client";

import { useState } from "react";
import useSWR, { mutate } from "swr";
import {
  Button,
  Badge,
  Input,
  Textarea,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
  EmptyState,
  Skeleton,
} from "@qeetrix/ui";
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
  created_at: string;
};

const CHANNELS = ["email", "sms", "whatsapp", "push", "inapp", "webhook"];

export default function TemplatesPage() {
  const { data, isLoading } = useSWR<{ templates: Template[] }>(
    "/api/v1/templates",
    apiFetcher
  );
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ name: "", channel: "email", locale: "en", subject: "", body: "" });
  const [saving, setSaving] = useState(false);

  const templates = data?.templates ?? [];

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const res = await apiCall("/api/v1/templates", {
        method: "POST",
        body: JSON.stringify(form),
      });
      if (res.ok) {
        await mutate("/api/v1/templates");
        setOpen(false);
        setForm({ name: "", channel: "email", locale: "en", subject: "", body: "" });
      }
    } finally {
      setSaving(false);
    }
  }

  async function handlePublish(id: string) {
    await apiCall(`/api/v1/templates/${id}/publish`, { method: "POST" });
    await mutate("/api/v1/templates");
  }

  async function handleDelete(id: string) {
    if (!confirm("Archive this template?")) return;
    await apiCall(`/api/v1/templates/${id}`, { method: "DELETE" });
    await mutate("/api/v1/templates");
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Templates</h1>
          <p className="text-sm text-muted-foreground mt-1">Manage Handlebars notification templates</p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>New Template</SheetTrigger>
          <SheetContent className="w-[560px] sm:max-w-[560px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Create Template</SheetTitle>
            </SheetHeader>
            <form onSubmit={handleCreate} className="mt-6 space-y-4">
              <div className="space-y-1">
                <label className="text-sm font-medium">Name</label>
                <Input
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="e.g. welcome_email"
                  required
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-sm font-medium">Channel</label>
                  <Select value={form.channel} onValueChange={(v) => setForm({ ...form, channel: v ?? "email" })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {CHANNELS.map((c) => (
                        <SelectItem key={c} value={c}>{c}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium">Locale</label>
                  <Input
                    value={form.locale}
                    onChange={(e) => setForm({ ...form, locale: e.target.value })}
                    placeholder="en"
                  />
                </div>
              </div>
              {form.channel === "email" && (
                <div className="space-y-1">
                  <label className="text-sm font-medium">Subject</label>
                  <Input
                    value={form.subject}
                    onChange={(e) => setForm({ ...form, subject: e.target.value })}
                    placeholder="Welcome to {{company_name}}"
                  />
                </div>
              )}
              <div className="space-y-1">
                <label className="text-sm font-medium">Body (Handlebars)</label>
                <Textarea
                  value={form.body}
                  onChange={(e) => setForm({ ...form, body: e.target.value })}
                  rows={10}
                  placeholder="Hello {{subscriber.name}}, ..."
                  required
                  className="font-mono text-xs"
                />
              </div>
              <Button type="submit" disabled={saving} className="w-full">
                {saving ? "Creating…" : "Create Template"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-14 rounded-lg" />)}
        </div>
      ) : templates.length === 0 ? (
        <EmptyState
          title="No templates yet"
          description="Create your first notification template to get started."
        />
      ) : (
        <div className="rounded-lg border divide-y">
          {templates.map((t) => (
            <div key={t.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex items-center gap-3">
                <div>
                  <p className="text-sm font-medium">{t.name}</p>
                  <p className="text-xs text-muted-foreground">{t.locale} · v{t.version}</p>
                </div>
                <Badge variant={t.is_active ? "default" : "secondary"}>{t.channel}</Badge>
              </div>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm" onClick={() => handlePublish(t.id)}>
                  Publish
                </Button>
                <Button variant="ghost" size="sm" onClick={() => handleDelete(t.id)}>
                  Archive
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
