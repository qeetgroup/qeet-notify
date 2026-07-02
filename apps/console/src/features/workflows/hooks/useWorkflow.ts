import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { workflowsApi } from "../api/workflows";

export function useWorkflows() {
  return useQuery({
    queryKey: ["workflows"],
    queryFn: () => workflowsApi.list(),
  });
}

export function useWorkflow(id: string) {
  return useQuery({
    queryKey: ["workflows", id],
    queryFn: () => workflowsApi.get(id),
    enabled: !!id,
  });
}

export function useWorkflowRuns(id: string) {
  return useQuery({
    queryKey: ["workflows", id, "runs"],
    queryFn: () => workflowsApi.runs(id),
    enabled: !!id,
  });
}

export function useToggleWorkflow() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, active }: { id: string; active: boolean }) =>
      active ? workflowsApi.activate(id) : workflowsApi.pause(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workflows"] }),
  });
}
