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

type DLTTemplate = {
  id: string;
  carrier: string;
  channel: string;
  template_id_ext: string;
  template_name: string;
  pe_id?: string;
  sender_id?: string;
  category: string;
  status: string;
  created_at: string;
};

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  approved: "default",
  pending: "secondary",
  rejected: "destructive",
};

const CARRIERS = ["airtel", "jio", "vodafone", "bsnl", "all"];
const CATEGORIES = ["transactional", "promotional", "service_explicit", "service_implicit"];

export default function DLTPage() {
  const { data, isLoading } = useSWR<{ dlt_templates: DLTTemplate[] }>(
    "/api/v1/dlt/templates",
    apiFetcher
  );
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({
    carrier: "all",
    channel: "sms",
    template_id_ext: "",
    template_name: "",
    pe_id: "",
    sender_id: "",
    category: "transactional",
    body_regex: "",
  });
  const [saving, setSaving] = useState(false);

  const templates = data?.dlt_templates ?? [];

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload = { ...form };
      const res = await apiCall("/api/v1/dlt/templates", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      if (res.ok) {
        await mutate("/api/v1/dlt/templates");
        setOpen(false);
        setForm({ carrier: "all", channel: "sms", template_id_ext: "", template_name: "", pe_id: "", sender_id: "", category: "transactional", body_regex: "" });
      }
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Delete this DLT template?")) return;
    await apiCall(`/api/v1/dlt/templates/${id}`, { method: "DELETE" });
    await mutate("/api/v1/dlt/templates");
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">India DLT Templates</h1>
          <p className="text-sm text-muted-foreground mt-1">TRAI DLT and WhatsApp BSP template registrations</p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>Register Template</SheetTrigger>
          <SheetContent className="w-[560px] sm:max-w-[560px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Register DLT Template</SheetTitle>
            </SheetHeader>
            <form onSubmit={handleCreate} className="mt-6 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-sm font-medium">Carrier</label>
                  <Select value={form.carrier} onValueChange={(v) => setForm({ ...form, carrier: v })}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {CARRIERS.map((c) => <SelectItem key={c} value={c}>{c}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium">Category</label>
                  <Select value={form.category} onValueChange={(v) => setForm({ ...form, category: v })}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {CATEGORIES.map((c) => <SelectItem key={c} value={c}>{c}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">DLT Template ID (TRAI / Meta)</label>
                <Input
                  value={form.template_id_ext}
                  onChange={(e) => setForm({ ...form, template_id_ext: e.target.value })}
                  placeholder="1207170046950"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Template Name</label>
                <Input
                  value={form.template_name}
                  onChange={(e) => setForm({ ...form, template_name: e.target.value })}
                  placeholder="OTP Verification"
                  required
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-sm font-medium">PE ID</label>
                  <Input
                    value={form.pe_id}
                    onChange={(e) => setForm({ ...form, pe_id: e.target.value })}
                    placeholder="1201160042760"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium">Sender ID</label>
                  <Input
                    value={form.sender_id}
                    onChange={(e) => setForm({ ...form, sender_id: e.target.value })}
                    placeholder="QEETID"
                  />
                </div>
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Body Regex</label>
                <Input
                  value={form.body_regex}
                  onChange={(e) => setForm({ ...form, body_regex: e.target.value })}
                  placeholder="Your OTP is \d{6}\. Valid for 10 minutes\."
                  required
                />
                <p className="text-xs text-muted-foreground">Regex that matches allowed message content.</p>
              </div>
              <Button type="submit" disabled={saving} className="w-full">
                {saving ? "Registering…" : "Register Template"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      {isLoading ? (
        <Skeleton className="h-40 rounded-lg" />
      ) : templates.length === 0 ? (
        <EmptyState
          title="No DLT templates registered"
          description="Register TRAI DLT or WhatsApp BSP templates to enable compliant messaging."
        />
      ) : (
        <div className="rounded-lg border divide-y">
          {templates.map((t) => (
            <div key={t.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex items-center gap-3">
                <div>
                  <p className="text-sm font-medium">{t.template_name}</p>
                  <p className="text-xs text-muted-foreground font-mono">{t.template_id_ext}</p>
                </div>
                <Badge variant="outline">{t.carrier}</Badge>
                <Badge variant="outline">{t.category}</Badge>
                <Badge variant={STATUS_VARIANTS[t.status] ?? "outline"}>{t.status}</Badge>
              </div>
              <Button variant="ghost" size="sm" onClick={() => handleDelete(t.id)}>
                Delete
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
