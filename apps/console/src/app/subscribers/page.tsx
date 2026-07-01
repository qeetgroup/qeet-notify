"use client";

import { useState } from "react";
import useSWR, { mutate } from "swr";
import {
  Button,
  Badge,
  Input,
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
  EmptyState,
  Skeleton,
} from "@qeetrix/ui";
import { apiFetcher, apiCall } from "@/lib/api";

type Subscriber = {
  id: string;
  external_id: string;
  locale: string;
  timezone: string;
  created_at: string;
};

export default function SubscribersPage() {
  const [search, setSearch] = useState("");
  const { data, isLoading } = useSWR<{ subscribers: Subscriber[]; total: number }>(
    "/api/v1/subscribers?limit=100",
    apiFetcher
  );
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ external_id: "", email: "", phone: "", locale: "en", timezone: "Asia/Kolkata" });
  const [saving, setSaving] = useState(false);

  const subscribers = (data?.subscribers ?? []).filter(
    (s) => !search || s.external_id.toLowerCase().includes(search.toLowerCase())
  );

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const res = await apiCall("/api/v1/subscribers", {
        method: "POST",
        body: JSON.stringify(form),
      });
      if (res.ok) {
        await mutate("/api/v1/subscribers?limit=100");
        setOpen(false);
        setForm({ external_id: "", email: "", phone: "", locale: "en", timezone: "Asia/Kolkata" });
      }
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Permanently erase subscriber? This is irreversible (DPDP right to erasure).")) return;
    await apiCall(`/api/v1/subscribers/${id}`, { method: "DELETE" });
    await mutate("/api/v1/subscribers?limit=100");
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Subscribers</h1>
          <p className="text-sm text-muted-foreground mt-1">{data?.total ?? 0} total subscribers</p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>Add Subscriber</SheetTrigger>
          <SheetContent className="w-[480px] sm:max-w-[480px]">
            <SheetHeader>
              <SheetTitle>Add Subscriber</SheetTitle>
            </SheetHeader>
            <form onSubmit={handleCreate} className="mt-6 space-y-4">
              <div className="space-y-1">
                <label className="text-sm font-medium">External ID</label>
                <Input
                  value={form.external_id}
                  onChange={(e) => setForm({ ...form, external_id: e.target.value })}
                  placeholder="your-internal-user-id"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Email</label>
                <Input
                  type="email"
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                  placeholder="user@example.com"
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Phone (E.164)</label>
                <Input
                  value={form.phone}
                  onChange={(e) => setForm({ ...form, phone: e.target.value })}
                  placeholder="+919876543210"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-sm font-medium">Locale</label>
                  <Input
                    value={form.locale}
                    onChange={(e) => setForm({ ...form, locale: e.target.value })}
                    placeholder="en"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium">Timezone</label>
                  <Input
                    value={form.timezone}
                    onChange={(e) => setForm({ ...form, timezone: e.target.value })}
                    placeholder="Asia/Kolkata"
                  />
                </div>
              </div>
              <Button type="submit" disabled={saving} className="w-full">
                {saving ? "Adding…" : "Add Subscriber"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      <Input
        placeholder="Search by external ID…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="max-w-sm"
      />

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-12 rounded-lg" />)}
        </div>
      ) : subscribers.length === 0 ? (
        <EmptyState
          title="No subscribers found"
          description="Add subscribers via the API or the button above."
        />
      ) : (
        <div className="rounded-lg border divide-y">
          {subscribers.map((s) => (
            <div key={s.id} className="flex items-center justify-between px-4 py-3">
              <div>
                <p className="text-sm font-medium font-mono">{s.external_id}</p>
                <p className="text-xs text-muted-foreground">{s.locale} · {s.timezone}</p>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="outline">{new Date(s.created_at).toLocaleDateString()}</Badge>
                <Button variant="ghost" size="sm" onClick={() => handleDelete(s.id)}>
                  Erase
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
