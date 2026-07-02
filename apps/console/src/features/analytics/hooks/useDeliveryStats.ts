import { useQuery } from "@tanstack/react-query";
import { analyticsApi } from "../api/analytics";

export function useDeliveryStats(params?: { channel?: string }) {
  return useQuery({
    queryKey: ["analytics", "delivery", params],
    queryFn: () => analyticsApi.delivery(params),
  });
}
