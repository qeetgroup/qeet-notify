"use client";

import useSWR from "swr";
import { Badge, Skeleton, EmptyState } from "@qeetrix/ui";
import { apiFetcher } from "@/lib/api";

type Stat = {
  channel: string;
  event_type: string;
  count: number;
  date: string;
};

const EVENT_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  sent: "default",
  delivered: "outline",
  opened: "outline",
  clicked: "outline",
  bounced: "destructive",
  complained: "destructive",
  failed: "destructive",
};

export default function AnalyticsPage() {
  const { data, isLoading } = useSWR<{ stats: Stat[] }>(
    "/api/v1/analytics/delivery",
    apiFetcher
  );

  const stats = data?.stats ?? [];

  const summary = stats.reduce<Record<string, number>>((acc, s) => {
    acc[s.event_type] = (acc[s.event_type] ?? 0) + s.count;
    return acc;
  }, {});

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Delivery Analytics</h1>
        <p className="text-sm text-muted-foreground mt-1">30-day delivery funnel across all channels</p>
      </div>

      {isLoading ? (
        <Skeleton className="h-40 rounded-lg" />
      ) : (
        <div className="grid grid-cols-3 gap-4 lg:grid-cols-6">
          {["sent", "delivered", "opened", "clicked", "bounced", "failed"].map((event) => (
            <div key={event} className="rounded-lg border p-4 text-center">
              <p className="text-2xl font-semibold">{(summary[event] ?? 0).toLocaleString()}</p>
              <Badge variant={EVENT_VARIANTS[event] ?? "outline"} className="mt-1">{event}</Badge>
            </div>
          ))}
        </div>
      )}

      <div>
        <h2 className="text-lg font-medium mb-3">Daily Breakdown</h2>
        {isLoading ? (
          <Skeleton className="h-64 rounded-lg" />
        ) : stats.length === 0 ? (
          <EmptyState title="No analytics data" description="Delivery events will appear here once notifications are sent." />
        ) : (
          <div className="rounded-lg border overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  {["Date", "Channel", "Event", "Count"].map((h) => (
                    <th key={h} className="text-left px-4 py-2 font-medium text-muted-foreground">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y">
                {stats.map((s, i) => (
                  <tr key={i}>
                    <td className="px-4 py-2 text-muted-foreground">{s.date}</td>
                    <td className="px-4 py-2">
                      <Badge variant="outline">{s.channel}</Badge>
                    </td>
                    <td className="px-4 py-2">
                      <Badge variant={EVENT_VARIANTS[s.event_type] ?? "outline"}>{s.event_type}</Badge>
                    </td>
                    <td className="px-4 py-2 font-mono">{s.count.toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
