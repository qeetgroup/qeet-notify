import { useQuery } from "@tanstack/react-query";
import { dltApi } from "../api/dlt";

export function useDlt() {
  return useQuery({
    queryKey: ["dlt-templates"],
    queryFn: () => dltApi.list(),
  });
}
