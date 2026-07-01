"use client";

import { use } from "react";
import useSWR from "swr";
import { Badge, Skeleton } from "@qeetrix/ui";
import { apiFetcher } from "@/lib/api";

type Workflow = {
  id: string;
  name: string;
  trigger_event: string;
  steps: unknown[];
  is_active: boolean;
  created_at: string;
};
type Run = {
  id: string;
  status: string;
  current_step_index: number;
  error?: string;
  created_at: string;
};

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  running: "default",
  completed: "outline",
  failed: "destructive",
  cancelled: "secondary",
};

export default function WorkflowDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { data: wf, isLoading } = useSWR<Workflow>(`/api/v1/workflows/${id}`, apiFetcher);
  const { data: runsData } = useSWR<{ runs: Run[] }>(`/api/v1/workflows/${id}/runs`, apiFetcher);

  if (isLoading) return <Skeleton className="h-64 rounded-lg" />;
  if (!wf) return <p className="text-sm text-muted-foreground">Workflow not found.</p>;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{wf.name}</h1>
          <p className="text-sm text-muted-foreground font-mono">{wf.trigger_event}</p>
        </div>
        <Badge variant={wf.is_active ? "default" : "secondary"}>
          {wf.is_active ? "active" : "paused"}
        </Badge>
      </div>

      <div>
        <h2 className="text-lg font-medium mb-3">Steps ({wf.steps.length})</h2>
        <pre className="rounded-lg bg-muted p-4 text-xs overflow-auto max-h-80">
          {JSON.stringify(wf.steps, null, 2)}
        </pre>
      </div>

      <div>
        <h2 className="text-lg font-medium mb-3">Recent Runs</h2>
        {(runsData?.runs ?? []).length === 0 ? (
          <p className="text-sm text-muted-foreground">No runs yet.</p>
        ) : (
          <div className="rounded-lg border divide-y">
            {(runsData?.runs ?? []).map((run) => (
              <div key={run.id} className="flex items-center justify-between px-4 py-3 text-sm">
                <span className="font-mono text-xs text-muted-foreground">{run.id.slice(0, 8)}…</span>
                <Badge variant={STATUS_VARIANTS[run.status] ?? "outline"}>{run.status}</Badge>
                <span className="text-muted-foreground">step {run.current_step_index}</span>
                <span className="text-muted-foreground">{new Date(run.created_at).toLocaleString()}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
