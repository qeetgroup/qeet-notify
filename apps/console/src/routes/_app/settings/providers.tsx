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
import { PlusIcon, ServerIcon } from "lucide-react";
import { useState } from "react";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/settings/providers")({
  component: ProvidersPage,
});

type Provider = {
  id: string;
  channel: string;
  provider: string;
  priority: number;
  is_active: boolean;
  created_at: string;
};

type ProvidersResp = { providers: Provider[] };

const PROVIDERS_BY_CHANNEL: Record<string, string[]> = {
  email: ["ses", "resend"],
  sms: ["msg91", "2factor"],
  whatsapp: ["meta"],
  push: ["fcm", "apns"],
  webhook: ["custom"],
};

function ProvidersPage() {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<{
    channel: string;
    provider: string;
    priority: number;
    config: string;
  }>({ channel: "email", provider: "ses", priority: 1, config: "{}" });

  const { data, isLoading } = useQuery({
    queryKey: ["providers"],
    queryFn: () => api<ProvidersResp>("/v1/providers"),
  });

  const createMutation = useMutation({
    mutationFn: (f: typeof form) => {
      let config: Record<string, string> = {};
      try {
        config = JSON.parse(f.config);
      } catch {
        /* use empty */
      }
      return api("/v1/providers", { method: "POST", body: { ...f, config } });
    },
    meta: { successMessage: "Provider added" },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["providers"] });
      setOpen(false);
      setForm({ channel: "email", provider: "ses", priority: 1, config: "{}" });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api(`/v1/providers/${id}`, { method: "DELETE" }),
    meta: { successMessage: "Provider removed" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["providers"] }),
  });

  const providers = data?.providers ?? [];
  const availableProviders = PROVIDERS_BY_CHANNEL[form.channel] ?? [];

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Providers</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Configure email, SMS, and WhatsApp provider credentials
          </p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>
            <PlusIcon /> Add Provider
          </SheetTrigger>
          <SheetContent className="w-[520px] sm:max-w-[520px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Add Provider Config</SheetTitle>
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
                  <label className="text-sm font-medium">Channel</label>
                  <Select
                    value={form.channel}
                    onValueChange={(v) => {
                      const ch = v ?? "email";
                      const first = (PROVIDERS_BY_CHANNEL[ch] ?? [])[0] ?? "";
                      setForm({ ...form, channel: ch, provider: first });
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {Object.keys(PROVIDERS_BY_CHANNEL).map((c) => (
                        <SelectItem key={c} value={c}>
                          {c}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium">Provider</label>
                  <Select
                    value={form.provider}
                    onValueChange={(v) => setForm({ ...form, provider: v ?? "" })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {availableProviders.map((p) => (
                        <SelectItem key={p} value={p}>
                          {p}
                        </SelectItem>
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
                  onChange={(e) =>
                    setForm({ ...form, priority: parseInt(e.target.value) || 1 })
                  }
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
                <p className="text-xs text-muted-foreground">
                  Credentials are stored encrypted at rest (PGP).
                </p>
              </div>
              <Button type="submit" disabled={createMutation.isPending} className="w-full">
                {createMutation.isPending ? "Saving…" : "Add Provider"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      {isLoading ? (
        <Skeleton className="h-40 rounded-lg" />
      ) : providers.length === 0 ? (
        <EmptyState
          icon={ServerIcon}
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
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  if (confirm("Remove this provider config?")) deleteMutation.mutate(p.id);
                }}
              >
                Remove
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
