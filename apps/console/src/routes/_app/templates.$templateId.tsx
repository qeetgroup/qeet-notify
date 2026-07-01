import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  Textarea,
} from "@qeetrix/ui";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, createFileRoute, useNavigate } from "@tanstack/react-router";
import { ArrowLeftIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/templates/$templateId")({
  component: TemplateDetailPage,
});

type Template = {
  id: string;
  name: string;
  channel: string;
  status: string;
  subject?: string;
  body: string;
  content_type: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

const CHANNELS = ["email", "sms", "whatsapp", "inapp", "webhook"];

function TemplateDetailPage() {
  const { templateId } = Route.useParams();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [form, setForm] = useState({ name: "", channel: "", subject: "", body: "", content_type: "" });

  const { data: template, isLoading } = useQuery({
    queryKey: ["template", templateId],
    queryFn: () => api<Template>(`/v1/templates/${templateId}`),
  });

  useEffect(() => {
    if (template) {
      setForm({
        name: template.name,
        channel: template.channel,
        subject: template.subject ?? "",
        body: template.body,
        content_type: template.content_type,
      });
    }
  }, [template]);

  const updateMutation = useMutation({
    mutationFn: (body: typeof form) =>
      api(`/v1/templates/${templateId}`, { method: "PUT", body }),
    meta: { successMessage: "Template saved" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["template", templateId] }),
  });

  const publishMutation = useMutation({
    mutationFn: () => api(`/v1/templates/${templateId}/publish`, { method: "POST" }),
    meta: { successMessage: "Template published" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["template", templateId] }),
  });

  const deleteMutation = useMutation({
    mutationFn: () => api(`/v1/templates/${templateId}`, { method: "DELETE" }),
    meta: { successMessage: "Template deleted" },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["templates"] });
      navigate({ to: "/templates" });
    },
  });

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!template) return null;

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" render={<Link to="/templates" />}>
          <ArrowLeftIcon />
        </Button>
        <div className="flex-1">
          <h1 className="text-xl font-semibold">{template.name}</h1>
          <p className="text-xs text-muted-foreground font-mono">{template.id}</p>
        </div>
        <Badge variant={template.status === "published" ? "default" : "secondary"}>
          {template.status}
        </Badge>
        <div className="flex items-center gap-2">
          {template.status !== "published" && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => publishMutation.mutate()}
              disabled={publishMutation.isPending}
            >
              Publish
            </Button>
          )}
          <Button
            variant="destructive"
            size="sm"
            onClick={() => {
              if (confirm("Delete this template permanently?")) deleteMutation.mutate();
            }}
          >
            Delete
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Edit Template</CardTitle>
        </CardHeader>
        <CardContent>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              updateMutation.mutate(form);
            }}
            className="space-y-4"
          >
            <div className="space-y-1">
              <label className="text-sm font-medium">Name</label>
              <Input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
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
                  onValueChange={(v) => setForm({ ...form, content_type: v ?? "text/html" })}
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
                rows={16}
                className="font-mono text-xs"
              />
            </div>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? "Saving…" : "Save Changes"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
