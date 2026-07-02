import { apiFetcher } from "@/lib/api";

export interface DeliveryBucket {
  bucket: string;
  channel: string;
  sent: number;
  failed: number;
  suppressed: number;
}

export const analyticsApi = {
  delivery: (params?: { channel?: string; from?: string; to?: string }) =>
    apiFetcher<DeliveryBucket[]>("/v1/analytics/delivery", { params }),
};
