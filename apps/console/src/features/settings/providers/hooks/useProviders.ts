import { useQuery } from "@tanstack/react-query";
import { providersApi } from "../api/providers";

export function useProviders() {
  return useQuery({
    queryKey: ["providers"],
    queryFn: () => providersApi.list(),
  });
}
