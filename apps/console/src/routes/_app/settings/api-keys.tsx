import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
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
import { ClipboardIcon, KeyRoundIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/settings/api-keys")({
  component: ApiKeysPage,
});

type ApiKey = {
  id: string;
  name: string;
  prefix: string;
  scope: string;
  created_at: string;
  revoked_at?: string;
};

type ApiKeysResp = { api_keys: ApiKey[] };

type CreateKeyResp = {
  id: string;
  name: string;
  key: string;
  scope: string;
  created_at: string;
};

const SCOPES = ["full", "read", "send"];

function ApiKeysPage() {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ name: "", scope: "full" });
  const [newKey, setNewKey] = useState<CreateKeyResp | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["api-keys"],
    queryFn: () => api<ApiKeysResp>("/v1/api-keys"),
  });

  const createMutation = useMutation({
    mutationFn: (body: typeof form) =>
      api<CreateKeyResp>("/v1/api-keys", { method: "POST", body }),
    meta: { silent: true },
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ["api-keys"] });
      setOpen(false);
      setNewKey(res);
      setForm({ name: "", scope: "full" });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: (id: string) => api(`/v1/api-keys/${id}`, { method: "DELETE" }),
    meta: { successMessage: "API key revoked" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["api-keys"] }),
  });

  const apiKeys = (data?.api_keys ?? []).filter((k) => !k.revoked_at);

  async function copyKey(key: string) {
    try {
      await navigator.clipboard.writeText(key);
      toast.success("Copied to clipboard");
    } catch {
      toast.error("Could not copy");
    }
  }

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">API Keys</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Manage API keys for SDK and programmatic access
          </p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>
            <PlusIcon /> Create Key
          </SheetTrigger>
          <SheetContent className="w-[440px] sm:max-w-[440px]">
            <SheetHeader>
              <SheetTitle>Create API Key</SheetTitle>
            </SheetHeader>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                createMutation.mutate(form);
              }}
              className="mt-6 space-y-4"
            >
              <div className="space-y-1">
                <label className="text-sm font-medium">Name</label>
                <Input
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="Production SDK"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Scope</label>
                <Select
                  value={form.scope}
                  onValueChange={(v) => setForm({ ...form, scope: v ?? "full" })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {SCOPES.map((s) => (
                      <SelectItem key={s} value={s}>
                        {s}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  full = read + write + send · read = GET only · send = dispatch only
                </p>
              </div>
              <Button type="submit" disabled={createMutation.isPending} className="w-full">
                {createMutation.isPending ? "Creating…" : "Create API Key"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      {isLoading ? (
        <Skeleton className="h-40 rounded-lg" />
      ) : apiKeys.length === 0 ? (
        <EmptyState
          icon={KeyRoundIcon}
          title="No API keys"
          description="Create an API key to start integrating your applications."
        />
      ) : (
        <div className="rounded-lg border divide-y">
          {apiKeys.map((k) => (
            <div key={k.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex items-center gap-3">
                <div className="grid size-8 shrink-0 place-items-center rounded bg-muted">
                  <KeyRoundIcon className="size-3.5 text-muted-foreground" />
                </div>
                <div>
                  <p className="text-sm font-medium">{k.name}</p>
                  <p className="text-xs text-muted-foreground font-mono">
                    {k.prefix}••••••••••••
                  </p>
                </div>
                <Badge variant="outline">{k.scope}</Badge>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted-foreground">
                  {new Date(k.created_at).toLocaleDateString()}
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    if (confirm(`Revoke key "${k.name}"? This cannot be undone.`))
                      revokeMutation.mutate(k.id);
                  }}
                >
                  Revoke
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Dialog open={!!newKey} onOpenChange={(o) => !o && setNewKey(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>API Key Created</DialogTitle>
            <DialogDescription>
              Copy this key now — it will not be shown again.
            </DialogDescription>
          </DialogHeader>
          {newKey && (
            <div className="space-y-4">
              <Card className="bg-muted/50">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm">{newKey.name}</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 break-all rounded bg-background p-2 text-xs font-mono">
                      {newKey.key}
                    </code>
                    <Button
                      variant="outline"
                      size="icon"
                      className="shrink-0"
                      onClick={() => copyKey(newKey.key)}
                    >
                      <ClipboardIcon className="size-4" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
              <p className="text-xs text-muted-foreground">
                Scope: <strong>{newKey.scope}</strong> · Created:{" "}
                {new Date(newKey.created_at).toLocaleString()}
              </p>
              <Button className="w-full" onClick={() => setNewKey(null)}>
                Done
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
