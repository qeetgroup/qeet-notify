import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
  DataState,
  Sheet,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TimeSince,
} from "@qeetrix/ui";
import { GitBranchIcon, PlusIcon } from "lucide-react";
import { useState } from "react";

import { useToggleWorkflow, useWorkflows } from "../hooks/useWorkflow";

const STATUS_VARIANT: Record<string, "default" | "secondary" | "outline"> = {
  active: "default",
  paused: "secondary",
  draft:  "outline",
};

export function WorkflowList() {
  const { data, isLoading, isError, error } = useWorkflows();
  const { mutate: toggle } = useToggleWorkflow();
  const items = (data as any)?.data ?? data ?? [];
  const [deleteTarget, setDeleteTarget] = useState<any>(null);
  const [runsTarget, setRunsTarget] = useState<any>(null);

  return (
    <>
      <div className="flex items-center justify-between pb-4">
        <h2 className="text-sm font-medium text-muted-foreground">All workflows</h2>
        <Button size="sm">
          <PlusIcon className="mr-1.5 size-4" /> New workflow
        </Button>
      </div>

      <DataState
        isLoading={isLoading}
        isError={isError}
        error={error instanceof Error ? error : undefined}
        isEmpty={!isLoading && items.length === 0}
        emptyIcon={GitBranchIcon}
        emptyTitle="No workflows yet"
        emptyDescription="Create a workflow to orchestrate multi-step notifications."
        skeletonRows={4}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Trigger event</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Active</TableHead>
              <TableHead>Created</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((w: any) => (
              <TableRow key={w.id}>
                <TableCell
                  className="cursor-pointer font-medium hover:underline"
                  onClick={() => setRunsTarget(w)}
                >
                  {w.name}
                </TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{w.trigger_event}</TableCell>
                <TableCell>
                  <Badge variant={STATUS_VARIANT[w.status] ?? "outline"}>{w.status}</Badge>
                </TableCell>
                <TableCell>
                  <Switch
                    checked={w.status === "active"}
                    onCheckedChange={() =>
                      toggle({ id: w.id, active: w.status !== "active" })
                    }
                  />
                </TableCell>
                <TableCell>
                  <TimeSince value={w.created_at} className="text-sm text-muted-foreground" />
                </TableCell>
                <TableCell>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    onClick={() => setDeleteTarget(w)}
                  >
                    Delete
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataState>

      {/* Runs sheet */}
      <Sheet open={!!runsTarget} onOpenChange={(o) => !o && setRunsTarget(null)}>
        {/* WorkflowRuns rendered lazily */}
      </Sheet>

      {/* Delete confirm */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(o) => !o && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete workflow?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete <strong>{deleteTarget?.name}</strong> and stop all active runs.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <Button
              variant="destructive"
              onClick={() => {
                // TODO: wire delete mutation
                setDeleteTarget(null);
              }}
            >
              Delete
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
