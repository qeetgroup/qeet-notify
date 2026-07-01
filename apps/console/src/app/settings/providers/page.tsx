"use client";

import { useState } from "react";
import useSWR, { mutate } from "swr";
import {
  Button,
  Badge,
  Input,
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
  Skeleton,
  EmptyState,
} from "@qeetrix/ui";
import { apiFetcher, apiCall } from "@/lib/api";

type Provider = {
  id: string;
  channel: string;
  provider: string;
  priority: number;
  is_active: boolean;
  created_at: string;
};

const PROVIDERS_BY_CHANNEL: Record<string, string[]> = {
  email: ["ses", "resend"],
  sms: ["msg91", "2factor"],
  whatsapp: ["meta"],
  push: ["fcm", "apns"],
  webhook: ["custom"],
};

export default function ProvidersPage() {
  const { data, isLoading } = useSWR<{ providers: Provider[] }>(
    "/api/v1/providers",
    apiFetcher
  );
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<{
    channel: string;
    provider: string;
    priority: number;
    config: string;
  }>({ channel: "email", provider: "ses", priority: 1, config: "{}" });
  const [saving, setSaving] = useState(false);

  const providers = data?.providers ?? [];

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      let config: Record<string, string> = {};
      try { config = JSON.parse(form.config); } catch { /* use empty */ }
      const res = await apiCall("/api/v1/providers", {
        method: "POST",
        body: JSON.stringify({ ...form, config }),
      });
      if (res.ok) {
        await mutate("/api/v1/providers");
        setOpen(false);
        setForm({ channel: "email", provider: "ses", priority: 1, config: "{}" });
      }
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Remove this provider config?")) return;
    await apiCall(`/api/v1/providers/${id}`, { method: "DELETE" });
    await mutate("/api/v1/providers");
  }

  const availableProviders = PROVIDERS_BY_CHANNEL[form.channel] ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Providers</h1>
          <p className="text-sm text-muted-foreground mt-1">Configure email, SMS, and WhatsApp provider credentials</p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>Add Provider</SheetTrigger>
          <SheetContent className="w-[520px] sm:max-w-[520px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Add Provider Config</SheetTitle>
            </SheetHeader>
            <form onSubmit={handleCreate} className="mt-6 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-sm font-medium">Channel</label>
                  <Select
                    value={form.channel}
                    onValueChange={(v) => {
                      const ch = v ?? "email";
                      const firstProvider = (PROVIDERS_BY_CHANNEL[ch] ?? [])[0] ?? "";
                      setForm({ ...form, channel: ch, provider: firstProvider });
                    }}
                  >
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {Object.keys(PROVIDERS_BY_CHANNEL).map((c) => (
                        <SelectItem key={c} value={c}>{c}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium">Provider</label>
                  <Select value={form.provider} onValueChange={(v) => setForm({ ...form, provider: v ?? "" })}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {availableProviders.map((p) => (
                        <SelectItem key={p} value={p}>{p}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Priority (1 = primary)</label>
                <Input
                  type="number"
                  min={1}
                  max={5}
                  value={form.priority}
                  onChange={(e) => setForm({ ...form, priority: parseInt(e.target.value) || 1 })}
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Config (JSON)</label>
                <textarea
                  value={form.config}
                  onChange={(e) => setForm({ ...form, config: e.target.value })}
                  rows={8}
                  className="w-full rounded-md border bg-background px-3 py-2 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder='{"api_key": "...", "region": "ap-south-1"}'
                />
                <p className="text-xs text-muted-foreground">Credentials are stored encrypted at rest.</p>
              </div>
              <Button type="submit" disabled={saving} className="w-full">
                {saving ? "Saving…" : "Add Provider"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      {isLoading ? (
        <Skeleton className="h-40 rounded-lg" />
      ) : providers.length === 0 ? (
        <EmptyState
          title="No providers configured"
          description="Add provider credentials to start sending notifications."
        />
      ) : (
        <div className="rounded-lg border divide-y">
          {providers.map((p) => (
            <div key={p.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex items-center gap-3">
                <div>
                  <p className="text-sm font-medium">{p.provider}</p>
                  <p className="text-xs text-muted-foreground">Priority {p.priority}</p>
                </div>
                <Badge variant="outline">{p.channel}</Badge>
                <Badge variant={p.is_active ? "default" : "secondary"}>
                  {p.is_active ? "active" : "inactive"}
                </Badge>
              </div>
              <Button variant="ghost" size="sm" onClick={() => handleDelete(p.id)}>
                Remove
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
