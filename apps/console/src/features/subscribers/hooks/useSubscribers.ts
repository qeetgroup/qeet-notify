import { useQuery } from "@tanstack/react-query";
import { subscribersApi } from "../api/subscribers";

export function useSubscribers() {
  return useQuery({
    queryKey: ["subscribers"],
    queryFn: () => subscribersApi.list(),
  });
}

export function useSubscriber(id: string) {
  return useQuery({
    queryKey: ["subscribers", id],
    queryFn: () => subscribersApi.get(id),
    enabled: !!id,
  });
}
