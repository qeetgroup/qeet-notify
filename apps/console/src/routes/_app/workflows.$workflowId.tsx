import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Skeleton,
} from "@qeetrix/ui";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, createFileRoute, useNavigate } from "@tanstack/react-router";
import { ArrowLeftIcon, ClockIcon } from "lucide-react";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/workflows/$workflowId")({
  component: WorkflowDetailPage,
});

type Workflow = {
  id: string;
  name: string;
  trigger_event: string;
  steps: unknown[];
  is_active: boolean;
  created_at: string;
};

type WorkflowRun = {
  id: string;
  status: string;
  started_at: string;
  completed_at?: string;
};

type RunsResp = { runs: WorkflowRun[] };

const RUN_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  completed: "default",
  running: "secondary",
  failed: "destructive",
};

function WorkflowDetailPage() {
  const { workflowId } = Route.useParams();
  const qc = useQueryClient();
  const navigate = useNavigate();

  const { data: workflow, isLoading } = useQuery({
    queryKey: ["workflow", workflowId],
    queryFn: () => api<Workflow>(`/v1/workflows/${workflowId}`),
  });

  const { data: runsData } = useQuery({
    queryKey: ["workflow-runs", workflowId],
    queryFn: () => api<RunsResp>(`/v1/workflows/${workflowId}/runs`),
    enabled: !!workflow,
  });

  const toggleMutation = useMutation({
    mutationFn: () =>
      api(`/v1/workflows/${workflowId}/${workflow?.is_active ? "pause" : "activate"}`, {
        method: "POST",
      }),
    meta: { successMessage: "Workflow updated" },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workflow", workflowId] }),
  });

  const deleteMutation = useMutation({
    mutationFn: () => api(`/v1/workflows/${workflowId}`, { method: "DELETE" }),
    meta: { successMessage: "Workflow archived" },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["workflows"] });
      navigate({ to: "/workflows" });
    },
  });

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (!workflow) return null;

  const runs = runsData?.runs ?? [];

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" render={<Link to="/workflows" />}>
          <ArrowLeftIcon />
        </Button>
        <div className="flex-1">
          <h1 className="text-xl font-semibold">{workflow.name}</h1>
          <p className="text-xs text-muted-foreground font-mono">{workflow.trigger_event}</p>
        </div>
        <Badge variant={workflow.is_active ? "default" : "secondary"}>
          {workflow.is_active ? "active" : "paused"}
        </Badge>
        <Button
          variant="outline"
          size="sm"
          onClick={() => toggleMutation.mutate()}
          disabled={toggleMutation.isPending}
        >
          {workflow.is_active ? "Pause" : "Activate"}
        </Button>
        <Button
          variant="destructive"
          size="sm"
          onClick={() => {
            if (confirm("Archive this workflow?")) deleteMutation.mutate();
          }}
        >
          Archive
        </Button>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Steps</CardTitle>
            <CardDescription>
              {workflow.steps.length} step{workflow.steps.length !== 1 ? "s" : ""}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {workflow.steps.length === 0 ? (
              <p className="text-sm text-muted-foreground">No steps defined.</p>
            ) : (
              <pre className="rounded-md bg-muted p-3 text-xs overflow-auto max-h-64">
                {JSON.stringify(workflow.steps, null, 2)}
              </pre>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Recent Runs</CardTitle>
            <CardDescription>Last 20 executions</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            {runs.length === 0 ? (
              <div className="px-6 pb-6 pt-2">
                <EmptyState
                  icon={ClockIcon}
                  title="No runs yet"
                  description="Runs appear here once the workflow is triggered."
                />
              </div>
            ) : (
              <ul className="divide-y">
                {runs.map((run) => (
                  <li key={run.id} className="flex items-center justify-between px-4 py-2.5">
                    <div>
                      <p className="text-xs font-mono text-muted-foreground truncate max-w-40">
                        {run.id}
                      </p>
                      <p className="text-[11px] text-muted-foreground">
                        {new Date(run.started_at).toLocaleString()}
                      </p>
                    </div>
                    <Badge variant={RUN_VARIANT[run.status] ?? "outline"}>{run.status}</Badge>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
