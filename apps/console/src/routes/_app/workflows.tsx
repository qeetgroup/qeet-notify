import {
  Badge,
  Button,
  EmptyState,
  Input,
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
import { PlusIcon, WorkflowIcon } from "lucide-react";
import { useState } from "react";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/workflows")({ component: WorkflowsPage });

type Workflow = {
  id: string;
  name: string;
  trigger_event: string;
  steps: unknown[];
  is_active: boolean;
  created_at: string;
};

type WorkflowsResp = { workflows: Workflow[] };

function WorkflowsPage() {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ name: "", trigger_event: "", steps: "[]" });

  const { data, isLoading } = useQuery({
    queryKey: ["workflows"],
    queryFn: () => api<WorkflowsResp>("/v1/workflows"),
  });

  const createMutation = useMutation({
    mutationFn: (f: typeof form) => {
      let steps: unknown[] = [];
      try {
        steps = JSON.parse(f.steps);
      } catch {
        /* ignore */
      }
      return api("/v1/workflows", {
        method: "POST",
        body: { name: f.name, trigger_event: f.trigger_event, steps },
      });
    },
    meta: { successMessage: "Workflow created" },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["workflows"] });
      setOpen(false);
      setForm({ name: "", trigger_event: "", steps: "[]" });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) =>
      api(`/v1/workflows/${id}/${isActive ? "pause" : "activate"}`, { method: "POST" }),
    meta: { successMessage: "Workflow updated" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workflows"] }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api(`/v1/workflows/${id}`, { method: "DELETE" }),
    meta: { successMessage: "Workflow archived" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workflows"] }),
  });

  const workflows = data?.workflows ?? [];

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Workflows</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Multi-step notification journeys triggered by events
          </p>
        </div>
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger render={<Button />}>
            <PlusIcon /> New Workflow
          </SheetTrigger>
          <SheetContent className="w-[560px] sm:max-w-[560px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Create Workflow</SheetTitle>
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
                  placeholder="User Onboarding"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Trigger Event</label>
                <Input
                  value={form.trigger_event}
                  onChange={(e) => setForm({ ...form, trigger_event: e.target.value })}
                  placeholder="user.signup"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-sm font-medium">Steps (JSON)</label>
                <Textarea
                  value={form.steps}
                  onChange={(e) => setForm({ ...form, steps: e.target.value })}
                  rows={8}
                  className="font-mono text-xs"
                  placeholder={'[{"type":"send","channel":"email","template_id":"..."}]'}
                />
                <p className="text-xs text-muted-foreground">
                  Array of step objects: type, channel, template_id, delay_seconds, condition
                </p>
              </div>
              <Button type="submit" disabled={createMutation.isPending} className="w-full">
                {createMutation.isPending ? "Creating…" : "Create Workflow"}
              </Button>
            </form>
          </SheetContent>
        </Sheet>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-14 rounded-lg" />
          ))}
        </div>
      ) : workflows.length === 0 ? (
        <EmptyState
          icon={WorkflowIcon}
          title="No workflows yet"
          description="Create a workflow to orchestrate multi-channel notification journeys."
        />
      ) : (
        <div className="rounded-lg border divide-y">
          {workflows.map((wf) => (
            <div key={wf.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex items-center gap-3">
                <div>
                  <Link
                    to="/workflows/$workflowId"
                    params={{ workflowId: wf.id }}
                    className="text-sm font-medium hover:underline"
                  >
                    {wf.name}
                  </Link>
                  <p className="text-xs text-muted-foreground font-mono">{wf.trigger_event}</p>
                </div>
                <Badge variant="outline">
                  {wf.steps.length} step{wf.steps.length !== 1 ? "s" : ""}
                </Badge>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant={wf.is_active ? "default" : "secondary"}>
                  {wf.is_active ? "active" : "paused"}
                </Badge>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => toggleMutation.mutate({ id: wf.id, isActive: wf.is_active })}
                  disabled={toggleMutation.isPending}
                >
                  {wf.is_active ? "Pause" : "Activate"}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    if (confirm(`Archive workflow "${wf.name}"?`)) deleteMutation.mutate(wf.id);
                  }}
                >
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
