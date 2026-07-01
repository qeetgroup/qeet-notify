import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@qeetrix/ui";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/analytics")({ component: AnalyticsPage });

type DeliveryStats = {
  queued: number;
  sent: number;
  delivered: number;
  failed: number;
  opened: number;
};

const FUNNEL_COLORS = [
  "var(--chart-2)",
  "var(--chart-1)",
  "var(--chart-3)",
  "var(--chart-5)",
];

function AnalyticsPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["analytics-delivery"],
    queryFn: () => api<DeliveryStats>("/v1/analytics/delivery"),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const funnel = data
    ? [
        { name: "Queued", value: data.queued },
        { name: "Sent", value: data.sent },
        { name: "Delivered", value: data.delivered },
        { name: "Opened", value: data.opened },
        { name: "Failed", value: data.failed },
      ]
    : [];

  const sent = data?.sent ?? 0;
  const delivered = data?.delivered ?? 0;
  const failed = data?.failed ?? 0;
  const deliveryRate = sent > 0 ? ((delivered / sent) * 100).toFixed(1) : "—";
  const failRate = sent > 0 ? ((failed / sent) * 100).toFixed(1) : "—";

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Analytics</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Delivery pipeline metrics across all channels
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        {[
          { label: "Total Sent", value: (data?.sent ?? 0).toLocaleString() },
          { label: "Delivery Rate", value: `${deliveryRate}%` },
          { label: "Fail Rate", value: `${failRate}%` },
        ].map(({ label, value }) => (
          <Card key={label}>
            <CardContent className="pt-6">
              <p className="text-2xl font-semibold tabular-nums">{isLoading ? "—" : value}</p>
              <p className="text-sm text-muted-foreground">{label}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Delivery Funnel</CardTitle>
          <CardDescription>
            All-time notification pipeline from queue to open/fail
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="h-64 animate-pulse rounded bg-muted/40" />
          ) : (
            <ResponsiveContainer width="100%" height={280}>
              <BarChart data={funnel} margin={{ left: 8, right: 8 }}>
                <CartesianGrid vertical={false} strokeDasharray="3 3" stroke="var(--border)" />
                <XAxis
                  dataKey="name"
                  axisLine={false}
                  tickLine={false}
                  tick={{ fontSize: 12, fill: "var(--muted-foreground)" }}
                />
                <YAxis
                  axisLine={false}
                  tickLine={false}
                  tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                  width={48}
                />
                <Tooltip
                  cursor={{ fill: "var(--muted)", opacity: 0.3 }}
                  contentStyle={{
                    background: "var(--card)",
                    border: "1px solid var(--border)",
                    borderRadius: "8px",
                    fontSize: 12,
                  }}
                />
                <Bar dataKey="value" radius={[4, 4, 0, 0]}>
                  {funnel.map((_, i) => (
                    <Cell key={i} fill={FUNNEL_COLORS[i % FUNNEL_COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
