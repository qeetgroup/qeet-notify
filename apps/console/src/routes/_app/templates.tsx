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
  Textarea,
} from "@qeetrix/ui";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";
import { FileTextIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/templates")({ component: TemplatesPage });

type Template = {
  id: string;
  name: string;
  channel: string;
  status: string;
  created_at: string;
};

type TemplatesResp = { templates: Template[] };

const CHANNELS = ["email", "sms", "whatsapp", "inapp", "webhook"];

const STATUS_VARIANT: Record<string, "default" | "secondary" | "outline"> = {
  published: "default",
  draft: "secondary",
};

function TemplatesPage() {
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({
    name: "",
    channel: "email",
    subject: "",
    body: "",
    content_type: "text/html",
  });

  const { data, isLoading } = useQuery({
    queryKey: ["templates"],
    queryFn: () => api<TemplatesResp>("/v1/templates"),
  });

  const createMutation = useMutation({
    mutationFn: (body: typeof form) => api("/v1/templates", { method: "POST", body }),
    meta: { successMessage: "Template created" },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["templates"] });
      setOpen(false);
      setForm({ name: "", channel: "email", subject: "", body: "", content_type: "text/html" });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api(`/v1/templates/${id}`, { method: "DELETE" }),
    meta: { successMessage: "Template deleted" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["templates"] }),
  });

  const publishMutation = useMutation({
    mutationFn: (id: string) => api(`/v1/templates/${id}/publish`, { method: "POST" }),
    meta: { successMessage: "Template published" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["templates"] }),
  });

  const templates = (data?.templates ?? []).filter(
    (t) => !search || t.name.toLowerCase().includes(search.toLowerCase()),
  );

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Templates</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Manage notification templates across channels
          </p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>
            <PlusIcon /> New Template
          </SheetTrigger>
          <SheetContent className="w-[560px] sm:max-w-[560px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Create Template</SheetTitle>
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
                  placeholder="Welcome Email"
                  required
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-sm font-medium">Channel</label>
                  <Select
                    value={form.channel}
                    onValueChange={(v) => setForm({ ...form, channel: v ?? "email" })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {CHANNELS.map((c) => (
                        <SelectItem key={c} value={c}>
                          {c}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1">
                  <label className="text-sm font-medium">Content Type</label>
                  <Select
                    value={form.content_type}
                    onValueChange={(v) =>
                      setForm({ ...form, content_type: v ?? "text/html" })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="text/html">HTML</SelectItem>
                      <SelectItem value="text/plain">Plain Text</SelectItem>
                      <SelectItem value="application/json">JSON</SelectItem>
                    </SelectContent>
                  </Select>
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
                <label className="text-sm font-medium">Body</label>
                <Textarea
                  value={form.body}
                  onChange={(e) => setForm({ ...form, body: e.target.value })}
                  rows={10}
                  className="font-mono text-xs"
                  placeholder="<p>Hello {{first_name}},</p>"
                />
                <p className="text-xs text-muted-foreground">
                  Supports Handlebars: {"{{variable}}"}, {"{{#if cond}}"}…{"{{/if}}"}
                </p>
              </div>
              <Button type="submit" disabled={createMutation.isPending} className="w-full">
                {createMutation.isPending ? "Creating…" : "Create Template"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      <Input
        placeholder="Search templates…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="max-w-sm"
      />

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-14 rounded-lg" />
          ))}
        </div>
      ) : templates.length === 0 ? (
        <EmptyState
          icon={FileTextIcon}
          title="No templates found"
          description="Create a template to start sending notifications."
        />
      ) : (
        <div className="rounded-lg border divide-y">
          {templates.map((t) => (
            <div key={t.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex items-center gap-3">
                <div>
                  <Link
                    to="/templates/$templateId"
                    params={{ templateId: t.id }}
                    className="text-sm font-medium hover:underline"
                  >
                    {t.name}
                  </Link>
                  <p className="text-xs text-muted-foreground font-mono truncate max-w-[160px]">
                    {t.id}
                  </p>
                </div>
                <Badge variant="outline">{t.channel}</Badge>
                <Badge variant={STATUS_VARIANT[t.status] ?? "outline"}>{t.status}</Badge>
              </div>
              <div className="flex items-center gap-2">
                {t.status !== "published" && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => publishMutation.mutate(t.id)}
                    disabled={publishMutation.isPending}
                  >
                    Publish
                  </Button>
                )}
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    if (confirm(`Delete template "${t.name}"?`)) deleteMutation.mutate(t.id);
                  }}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
