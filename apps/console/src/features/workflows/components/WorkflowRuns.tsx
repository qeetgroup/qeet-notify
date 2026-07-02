import {
  Badge,
  DataState,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TimeSince,
} from "@qeetrix/ui";
import { GitBranchIcon } from "lucide-react";

import { useWorkflowRuns } from "../hooks/useWorkflow";

const STATUS_VARIANT: Record<string, "default" | "destructive" | "secondary" | "outline"> = {
  completed: "default",
  failed:    "destructive",
  running:   "secondary",
  pending:   "outline",
};

interface WorkflowRunsProps {
  workflowId: string;
  workflowName: string;
}

export function WorkflowRuns({ workflowId, workflowName }: WorkflowRunsProps) {
  const { data, isLoading, isError, error } = useWorkflowRuns(workflowId);
  const runs = (data as any)?.data ?? data ?? [];

  return (
    <SheetContent side="right" className="w-full sm:max-w-lg">
      <SheetHeader className="mb-4">
        <SheetTitle>{workflowName}</SheetTitle>
        <SheetDescription>Run history</SheetDescription>
      </SheetHeader>

      <DataState
        isLoading={isLoading}
        isError={isError}
        error={error instanceof Error ? error : undefined}
        isEmpty={!isLoading && runs.length === 0}
        emptyIcon={GitBranchIcon}
        emptyTitle="No runs yet"
        emptyDescription="This workflow hasn't been triggered yet."
        skeletonRows={4}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Run ID</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Started</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {runs.map((r: any) => (
              <TableRow key={r.id}>
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {r.id.slice(0, 8)}…
                </TableCell>
                <TableCell>
                  <Badge variant={STATUS_VARIANT[r.status] ?? "outline"}>{r.status}</Badge>
                </TableCell>
                <TableCell>
                  <TimeSince value={r.started_at ?? r.created_at} className="text-sm text-muted-foreground" />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataState>
    </SheetContent>
  );
}
