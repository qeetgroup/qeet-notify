import {
  Badge,
  Button,
  EmptyState,
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
} from "@qeetrix/ui";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { PlusIcon, ShieldCheckIcon } from "lucide-react";
import { useState } from "react";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/settings/dlt")({ component: DLTPage });

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

type DLTResp = { dlt_templates: DLTTemplate[] };

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  approved: "default",
  pending: "secondary",
  rejected: "destructive",
};

const CARRIERS = ["airtel", "jio", "vodafone", "bsnl", "all"];
const CATEGORIES = [
  "transactional",
  "promotional",
  "service_explicit",
  "service_implicit",
];

function DLTPage() {
  const qc = useQueryClient();
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

  const { data, isLoading } = useQuery({
    queryKey: ["dlt-templates"],
    queryFn: () => api<DLTResp>("/v1/dlt/templates"),
  });

  const createMutation = useMutation({
    mutationFn: (body: typeof form) =>
      api("/v1/dlt/templates", { method: "POST", body }),
    meta: { successMessage: "DLT template registered" },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["dlt-templates"] });
      setOpen(false);
      setForm({
        carrier: "all",
        channel: "sms",
        template_id_ext: "",
        template_name: "",
        pe_id: "",
        sender_id: "",
        category: "transactional",
        body_regex: "",
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api(`/v1/dlt/templates/${id}`, { method: "DELETE" }),
    meta: { successMessage: "DLT template deleted" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["dlt-templates"] }),
  });

  const templates = data?.dlt_templates ?? [];

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">India DLT Templates</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            TRAI DLT and WhatsApp BSP template registrations
          </p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>
            <PlusIcon /> Register Template
          </SheetTrigger>
          <SheetContent className="w-[560px] sm:max-w-[560px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Register DLT Template</SheetTitle>
            </SheetHeader>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                createMutation.mutate(form);
              }}
              className="mt-6 space-y-4"
            >
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-sm font-medium">Carrier</label>
                  <Select
                    value={form.carrier}
                    onValueChange={(v) => setForm({ ...form, carrier: v ?? "all" })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {CARRIERS.map((c) => (
                        <SelectItem key={c} value={c}>
                          {c}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium">Category</label>
                  <Select
                    value={form.category}
                    onValueChange={(v) => setForm({ ...form, category: v ?? "transactional" })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {CATEGORIES.map((c) => (
                        <SelectItem key={c} value={c}>
                          {c}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">DLT Template ID (TRAI / Meta)</label>
                <Input
                  value={form.template_id_ext}
                  onChange={(e) =>
                    setForm({ ...form, template_id_ext: e.target.value })
                  }
                  placeholder="1207170046950"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Template Name</label>
                <Input
                  value={form.template_name}
                  onChange={(e) =>
                    setForm({ ...form, template_name: e.target.value })
                  }
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
                  placeholder={String.raw`Your OTP is \d{6}\. Valid for 10 minutes\.`}
                  required
                />
                <p className="text-xs text-muted-foreground">
                  Regex that matches allowed message content.
                </p>
              </div>
              <Button type="submit" disabled={createMutation.isPending} className="w-full">
                {createMutation.isPending ? "Registering…" : "Register Template"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      {isLoading ? (
        <Skeleton className="h-40 rounded-lg" />
      ) : templates.length === 0 ? (
        <EmptyState
          icon={ShieldCheckIcon}
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
                <Badge variant={STATUS_VARIANT[t.status] ?? "outline"}>{t.status}</Badge>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  if (confirm("Delete this DLT template?")) deleteMutation.mutate(t.id);
                }}
              >
                Delete
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
