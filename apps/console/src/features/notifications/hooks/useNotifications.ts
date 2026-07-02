import { useQuery } from "@tanstack/react-query";
import { notificationsApi } from "../api/notifications";

export function useNotifications(params?: { subscriber_id?: string; channel?: string }) {
  return useQuery({
    queryKey: ["notifications", params],
    queryFn: () => notificationsApi.list(params),
  });
}
