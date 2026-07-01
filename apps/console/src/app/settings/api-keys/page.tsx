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
  CopyableSecret,
} from "@qeetrix/ui";
import { apiFetcher, apiCall, setApiKey } from "@/lib/api";

type APIKey = {
  id: string;
  name: string;
  prefix: string;
  scope: string;
  created_at: string;
  revoked_at?: string;
};

const SCOPES = ["full", "read", "send"];

export default function APIKeysPage() {
  const { data, isLoading } = useSWR<{ api_keys: APIKey[] }>(
    "/api/v1/api-keys",
    apiFetcher
  );
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ name: "", scope: "full" });
  const [saving, setSaving] = useState(false);
  const [newKey, setNewKey] = useState<string | null>(null);

  const keys = data?.api_keys ?? [];

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const res = await apiCall("/api/v1/api-keys", {
        method: "POST",
        body: JSON.stringify(form),
      });
      if (res.ok) {
        const body = await res.json() as { key: string };
        setNewKey(body.key);
        setApiKey(body.key);
        await mutate("/api/v1/api-keys");
        setForm({ name: "", scope: "full" });
      }
    } finally {
      setSaving(false);
    }
  }

  async function handleRevoke(id: string) {
    if (!confirm("Revoke this API key? This cannot be undone.")) return;
    await apiCall(`/api/v1/api-keys/${id}`, { method: "DELETE" });
    await mutate("/api/v1/api-keys");
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">API Keys</h1>
          <p className="text-sm text-muted-foreground mt-1">Scoped API keys for SDK and CI integrations</p>
        </div>
        <Sheet open={open} onOpenChange={(v) => { setOpen(v); if (!v) setNewKey(null); }}>
          <SheetTrigger asChild>
            <Button>Create Key</Button>
          </SheetTrigger>
          <SheetContent className="w-[480px] sm:max-w-[480px]">
            <SheetHeader>
              <SheetTitle>Create API Key</SheetTitle>
            </SheetHeader>
            {newKey ? (
              <div className="mt-6 space-y-4">
                <div className="rounded-lg bg-amber-50 border border-amber-200 p-4 text-sm text-amber-900">
                  Store this key securely — it will <strong>not</strong> be shown again.
                </div>
                <CopyableSecret value={newKey} label="API Key" />
                <Button className="w-full" onClick={() => { setNewKey(null); setOpen(false); }}>
                  Done
                </Button>
              </div>
            ) : (
              <form onSubmit={handleCreate} className="mt-6 space-y-4">
                <div className="space-y-1">
                  <label className="text-sm font-medium">Name</label>
                  <Input
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    placeholder="e.g. Production SDK"
                    required
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium">Scope</label>
                  <Select value={form.scope} onValueChange={(v) => setForm({ ...form, scope: v })}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {SCOPES.map((s) => <SelectItem key={s} value={s}>{s}</SelectItem>)}
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">
                    full = all operations · read = read-only · send = events only
                  </p>
                </div>
                <Button type="submit" disabled={saving} className="w-full">
                  {saving ? "Creating…" : "Create API Key"}
                </Button>
              </form>
            )}
          </SheetContent>
        </Sheet>
      </div>

      {isLoading ? (
        <Skeleton className="h-40 rounded-lg" />
      ) : keys.length === 0 ? (
        <EmptyState
          title="No API keys"
          description="Create an API key to connect your application to Qeet Notify."
        />
      ) : (
        <div className="rounded-lg border divide-y">
          {keys.map((k) => (
            <div key={k.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex items-center gap-3">
                <div>
                  <p className="text-sm font-medium">{k.name}</p>
                  <p className="text-xs text-muted-foreground font-mono">{k.prefix}…</p>
                </div>
                <Badge variant="outline">{k.scope}</Badge>
                {k.revoked_at && <Badge variant="destructive">revoked</Badge>}
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted-foreground">{new Date(k.created_at).toLocaleDateString()}</span>
                {!k.revoked_at && (
                  <Button variant="ghost" size="sm" onClick={() => handleRevoke(k.id)}>
                    Revoke
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
