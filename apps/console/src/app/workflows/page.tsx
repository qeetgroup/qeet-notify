"use client";

import { useState } from "react";
import Link from "next/link";
import useSWR, { mutate } from "swr";
import {
  Button,
  Badge,
  Input,
  Textarea,
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
  EmptyState,
  Skeleton,
} from "@qeetrix/ui";
import { apiFetcher, apiCall } from "@/lib/api";

type Workflow = {
  id: string;
  name: string;
  trigger_event: string;
  steps: unknown[];
  is_active: boolean;
  created_at: string;
};

export default function WorkflowsPage() {
  const { data, isLoading } = useSWR<{ workflows: Workflow[] }>(
    "/api/v1/workflows",
    apiFetcher
  );
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ name: "", trigger_event: "", steps: "[]" });
  const [saving, setSaving] = useState(false);

  const workflows = data?.workflows ?? [];

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      let steps: unknown[] = [];
      try { steps = JSON.parse(form.steps); } catch { /* use empty array */ }
      const res = await apiCall("/api/v1/workflows", {
        method: "POST",
        body: JSON.stringify({ name: form.name, trigger_event: form.trigger_event, steps }),
      });
      if (res.ok) {
        await mutate("/api/v1/workflows");
        setOpen(false);
        setForm({ name: "", trigger_event: "", steps: "[]" });
      }
    } finally {
      setSaving(false);
    }
  }

  async function handleToggle(id: string, isActive: boolean) {
    const action = isActive ? "pause" : "activate";
    await apiCall(`/api/v1/workflows/${id}/${action}`, { method: "POST" });
    await mutate("/api/v1/workflows");
  }

  async function handleDelete(id: string) {
    if (!confirm("Archive this workflow?")) return;
    await apiCall(`/api/v1/workflows/${id}`, { method: "DELETE" });
    await mutate("/api/v1/workflows");
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Workflows</h1>
          <p className="text-sm text-muted-foreground mt-1">Multi-step notification journeys</p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>New Workflow</SheetTrigger>
          <SheetContent className="w-[560px] sm:max-w-[560px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Create Workflow</SheetTitle>
            </SheetHeader>
            <form onSubmit={handleCreate} className="mt-6 space-y-4">
              <div className="space-y-1">
                <label className="text-sm font-medium">Name</label>
                <Input
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="e.g. User Onboarding"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Trigger Event</label>
                <Input
                  value={form.trigger_event}
                  onChange={(e) => setForm({ ...form, trigger_event: e.target.value })}
                  placeholder="e.g. user.signup"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Steps (JSON)</label>
                <Textarea
                  value={form.steps}
                  onChange={(e) => setForm({ ...form, steps: e.target.value })}
                  rows={8}
                  placeholder='[{"type":"send","channel":"email","template_id":"..."}]'
                  className="font-mono text-xs"
                />
              </div>
              <Button type="submit" disabled={saving} className="w-full">
                {saving ? "Creating…" : "Create Workflow"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-14 rounded-lg" />)}
        </div>
      ) : workflows.length === 0 ? (
        <EmptyState
          title="No workflows yet"
          description="Create a workflow to orchestrate multi-channel notification journeys."
        />
      ) : (
        <div className="rounded-lg border divide-y">
          {workflows.map((wf) => (
            <div key={wf.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex items-center gap-3">
                <div>
                  <Link href={`/workflows/${wf.id}`} className="text-sm font-medium hover:underline">
                    {wf.name}
                  </Link>
                  <p className="text-xs text-muted-foreground font-mono">{wf.trigger_event}</p>
                </div>
                <Badge variant="outline">{wf.steps.length} step{wf.steps.length !== 1 ? "s" : ""}</Badge>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant={wf.is_active ? "default" : "secondary"}>
                  {wf.is_active ? "active" : "paused"}
                </Badge>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleToggle(wf.id, wf.is_active)}
                >
                  {wf.is_active ? "Pause" : "Activate"}
                </Button>
                <Button variant="ghost" size="sm" onClick={() => handleDelete(wf.id)}>
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
