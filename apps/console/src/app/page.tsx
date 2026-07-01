"use client";

import useSWR from "swr";
import { Stat, Skeleton } from "@qeetrix/ui";
import { apiFetcher } from "@/lib/api";

type DeliveryStats = {
  channel: string;
  event_type: string;
  count: number;
  date: string;
};

type WorkflowsData = { workflows: { id: string; is_active: boolean }[]; total: number };
type SubscribersData = { subscribers: { id: string }[]; total: number };
type NotificationsData = { notifications: { id: string; status: string; created_at: string }[] };
type AnalyticsData = { stats: DeliveryStats[] };

function sumCounts(stats: DeliveryStats[], eventType: string): number {
  return stats.filter((s) => s.event_type === eventType).reduce((acc, s) => acc + s.count, 0);
}

function deliveryRate(stats: DeliveryStats[]): string {
  const sent = sumCounts(stats, "sent") + sumCounts(stats, "delivered");
  const failed = sumCounts(stats, "failed");
  const total = sent + failed;
  if (total === 0) return "—";
  return ((sent / total) * 100).toFixed(1) + "%";
}

export default function DashboardPage() {
  const { data: analytics, isLoading: loadingAnalytics } = useSWR<AnalyticsData>(
    "/api/v1/analytics/delivery",
    apiFetcher
  );
  const { data: workflows, isLoading: loadingWorkflows } = useSWR<WorkflowsData>(
    "/api/v1/workflows?limit=200",
    apiFetcher
  );
  const { data: subscribers, isLoading: loadingSubscribers } = useSWR<SubscribersData>(
    "/api/v1/subscribers?limit=1",
    apiFetcher
  );
  const { data: notifications, isLoading: loadingNotifs } = useSWR<NotificationsData>(
    "/api/v1/notifications?limit=5",
    apiFetcher
  );

  const stats = analytics?.stats ?? [];
  const sentToday = stats
    .filter((s) => s.date === new Date().toISOString().slice(0, 10))
    .reduce((acc, s) => acc + s.count, 0);
  const activeWorkflows = (workflows?.workflows ?? []).filter((w) => w.is_active).length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="text-sm text-muted-foreground mt-1">Overview of your notification platform</p>
      </div>

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {loadingAnalytics ? (
          Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-24 rounded-lg" />)
        ) : (
          <>
            <Stat
              label="Sent Today"
              value={sentToday.toLocaleString()}
              trend={undefined}
            />
            <Stat
              label="Delivery Rate (30d)"
              value={deliveryRate(stats)}
              trend={undefined}
            />
            <Stat
              label="Active Workflows"
              value={loadingWorkflows ? "…" : String(activeWorkflows)}
              trend={undefined}
            />
            <Stat
              label="Subscribers"
              value={loadingSubscribers ? "…" : String(subscribers?.total ?? 0)}
              trend={undefined}
            />
          </>
        )}
      </div>

      <div>
        <h2 className="text-lg font-medium mb-3">Recent Notifications</h2>
        {loadingNotifs ? (
          <Skeleton className="h-40 rounded-lg" />
        ) : (
          <div className="rounded-lg border divide-y">
            {(notifications?.notifications ?? []).length === 0 ? (
              <p className="p-4 text-sm text-muted-foreground">No notifications yet.</p>
            ) : (
              (notifications?.notifications ?? []).map((n) => (
                <div key={n.id} className="flex items-center justify-between px-4 py-3 text-sm">
                  <span className="font-mono text-xs text-muted-foreground">{n.id.slice(0, 8)}…</span>
                  <span>{n.status}</span>
                  <span className="text-muted-foreground">
                    {new Date(n.created_at).toLocaleString()}
                  </span>
                </div>
              ))
            )}
          </div>
        )}
      </div>
    </div>
  );
}
