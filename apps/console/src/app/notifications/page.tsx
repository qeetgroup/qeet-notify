"use client";

import useSWR from "swr";
import { Badge, Skeleton, EmptyState } from "@qeetrix/ui";
import { apiFetcher } from "@/lib/api";

type Notification = {
  id: string;
  subscriber_id: string;
  channel: string;
  status: string;
  provider?: string;
  created_at: string;
};

const STATUS_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  sent: "default",
  delivered: "outline",
  failed: "destructive",
  pending: "secondary",
  queued: "secondary",
  skipped: "secondary",
};

export default function NotificationsPage() {
  const { data, isLoading } = useSWR<{ notifications: Notification[] }>(
    "/api/v1/notifications?limit=50",
    apiFetcher
  );

  const notifications = data?.notifications ?? [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Notifications</h1>
        <p className="text-sm text-muted-foreground mt-1">Recent outbound notifications</p>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-12 rounded-lg" />)}
        </div>
      ) : notifications.length === 0 ? (
        <EmptyState title="No notifications yet" description="Trigger an event via the API to see notifications here." />
      ) : (
        <div className="rounded-lg border divide-y">
          {notifications.map((n) => (
            <div key={n.id} className="flex items-center justify-between px-4 py-3 text-sm">
              <span className="font-mono text-xs text-muted-foreground">{n.id.slice(0, 8)}…</span>
              <Badge variant="outline">{n.channel}</Badge>
              <Badge variant={STATUS_VARIANTS[n.status] ?? "outline"}>{n.status}</Badge>
              <span className="text-xs text-muted-foreground">{n.provider ?? "—"}</span>
              <span className="text-xs text-muted-foreground">{new Date(n.created_at).toLocaleString()}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
