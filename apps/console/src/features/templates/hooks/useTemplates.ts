import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { templatesApi } from "../api/templates";

export function useTemplates() {
  return useQuery({
    queryKey: ["templates"],
    queryFn: () => templatesApi.list(),
  });
}

export function useTemplate(id: string) {
  return useQuery({
    queryKey: ["templates", id],
    queryFn: () => templatesApi.get(id),
    enabled: !!id,
  });
}

export function usePublishTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => templatesApi.publish(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["templates"] }),
  });
}
