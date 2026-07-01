import {
  Badge,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
} from "@qeetrix/ui";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
  ActivityIcon,
  BellIcon,
  CheckCircleIcon,
  SendIcon,
  UsersIcon,
  WorkflowIcon,
} from "lucide-react";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/")({ component: DashboardPage });

type DeliveryStats = {
  queued: number;
  sent: number;
  delivered: number;
  failed: number;
  opened: number;
};

type NotificationRow = {
  id: string;
  channel: string;
  status: string;
  subscriber_id: string;
  created_at: string;
};

type WorkflowsResp = { workflows: { is_active: boolean }[] };
type SubscribersResp = { total: number };
type NotificationsResp = { notifications: NotificationRow[]; total: number };

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  delivered: "default",
  sent: "default",
  pending: "secondary",
  queued: "secondary",
  failed: "destructive",
};

function StatCard({
  icon,
  title,
  value,
  sub,
  iconClass = "bg-primary/10 text-primary",
}: {
  icon: React.ReactNode;
  title: string;
  value: string | number;
  sub?: string;
  iconClass?: string;
}) {
  return (
    <Card>
      <CardContent className="flex items-center gap-4 pt-6">
        <div
          className={`grid size-10 shrink-0 place-items-center rounded-lg [&_svg]:size-5 ${iconClass}`}
        >
          {icon}
        </div>
        <div>
          <p className="text-2xl font-semibold tabular-nums">{value}</p>
          <p className="text-sm text-muted-foreground">{title}</p>
          {sub && <p className="text-xs text-muted-foreground">{sub}</p>}
        </div>
      </CardContent>
    </Card>
  );
}

function DashboardPage() {
  const { data: delivery, isLoading: dlLoading } = useQuery({
    queryKey: ["analytics-delivery"],
    queryFn: () => api<DeliveryStats>("/v1/analytics/delivery"),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const { data: workflowsData } = useQuery({
    queryKey: ["workflows-dashboard"],
    queryFn: () => api<WorkflowsResp>("/v1/workflows"),
    staleTime: 60_000,
  });

  const { data: subscribersData } = useQuery({
    queryKey: ["subscribers-count"],
    queryFn: () => api<SubscribersResp>("/v1/subscribers?limit=1"),
    staleTime: 60_000,
  });

  const { data: recentData, isLoading: recentLoading } = useQuery({
    queryKey: ["notifications-recent"],
    queryFn: () => api<NotificationsResp>("/v1/notifications?limit=5"),
    staleTime: 30_000,
    refetchInterval: 30_000,
  });

  const activeWorkflows =
    workflowsData?.workflows?.filter((w) => w.is_active).length ?? 0;
  const totalSubscribers = subscribersData?.total ?? 0;
  const sent = delivery?.sent ?? 0;
  const delivered = delivery?.delivered ?? 0;
  const deliveryRate = sent > 0 ? Math.round((delivered / sent) * 100) : 0;

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Real-time overview of your notification platform
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {dlLoading ? (
          <>
            {Array.from({ length: 4 }).map((_, i) => (
              <Card key={i}>
                <CardContent className="pt-6">
                  <div className="h-8 w-20 animate-pulse rounded bg-muted" />
                  <div className="mt-2 h-3 w-28 animate-pulse rounded bg-muted" />
                </CardContent>
              </Card>
            ))}
          </>
        ) : (
          <>
            <StatCard
              icon={<SendIcon />}
              title="Sent today"
              value={(delivery?.sent ?? 0).toLocaleString()}
              iconClass="bg-primary/10 text-primary"
            />
            <StatCard
              icon={<CheckCircleIcon />}
              title="Delivery rate"
              value={`${deliveryRate}%`}
              sub={`${delivered.toLocaleString()} delivered`}
              iconClass="bg-green-500/10 text-green-600 dark:text-green-400"
            />
            <StatCard
              icon={<WorkflowIcon />}
              title="Active workflows"
              value={activeWorkflows}
              iconClass="bg-blue-500/10 text-blue-600 dark:text-blue-400"
            />
            <StatCard
              icon={<UsersIcon />}
              title="Total subscribers"
              value={totalSubscribers.toLocaleString()}
              iconClass="bg-violet-500/10 text-violet-600 dark:text-violet-400"
            />
          </>
        )}
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Delivery Funnel</CardTitle>
            <CardDescription>
              All-time notification pipeline metrics
            </CardDescription>
          </CardHeader>
          <CardContent>
            {dlLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 4 }).map((_, i) => (
                  <div key={i} className="h-8 animate-pulse rounded bg-muted" />
                ))}
              </div>
            ) : (
              <div className="space-y-3">
                {[
                  { label: "Queued", value: delivery?.queued ?? 0, color: "bg-muted" },
                  { label: "Sent", value: delivery?.sent ?? 0, color: "bg-primary" },
                  { label: "Delivered", value: delivery?.delivered ?? 0, color: "bg-green-500" },
                  { label: "Failed", value: delivery?.failed ?? 0, color: "bg-destructive" },
                ].map(({ label, value, color }) => {
                  const max = delivery?.queued ?? 1;
                  const pct = max > 0 ? Math.round((value / max) * 100) : 0;
                  return (
                    <div key={label} className="space-y-1">
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">{label}</span>
                        <span className="tabular-nums font-medium">
                          {value.toLocaleString()}
                          <span className="ml-1.5 text-xs text-muted-foreground">
                            ({pct}%)
                          </span>
                        </span>
                      </div>
                      <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                        <div
                          className={`h-full rounded-full transition-all ${color}`}
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Recent Notifications</CardTitle>
            <CardDescription>Last 5 dispatched</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            {recentLoading ? (
              <div className="space-y-2 px-6 pb-6 pt-2">
                {Array.from({ length: 5 }).map((_, i) => (
                  <div key={i} className="h-9 animate-pulse rounded bg-muted" />
                ))}
              </div>
            ) : !recentData?.notifications?.length ? (
              <div className="px-6 pb-6">
                <EmptyState
                  icon={ActivityIcon}
                  title="No notifications yet"
                  description="Notifications will appear here once sent."
                />
              </div>
            ) : (
              <ul className="divide-y">
                {recentData.notifications.map((n) => (
                  <li key={n.id} className="flex items-center gap-3 px-4 py-2.5">
                    <div className="grid size-7 shrink-0 place-items-center rounded bg-muted">
                      <BellIcon className="size-3.5 text-muted-foreground" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-medium font-mono">
                        {n.subscriber_id}
                      </p>
                      <p className="text-[11px] text-muted-foreground">{n.channel}</p>
                    </div>
                    <Badge
                      variant={STATUS_VARIANT[n.status] ?? "outline"}
                      className="shrink-0 text-[10px]"
                    >
                      {n.status}
                    </Badge>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
