import {
  Badge,
  Button,
  EmptyState,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
} from "@qeetrix/ui";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { ScrollTextIcon } from "lucide-react";
import { useState } from "react";
import { api } from "@/lib/api";

export const Route = createFileRoute("/_app/logs")({ component: LogsPage });

type Notification = {
  id: string;
  channel: string;
  status: string;
  subscriber_id: string;
  template_id: string;
  created_at: string;
};

type NotificationsResp = { notifications: Notification[]; total: number };

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  delivered: "default",
  sent: "default",
  pending: "secondary",
  queued: "secondary",
  failed: "destructive",
};

const STATUSES = ["", "pending", "queued", "sent", "delivered", "failed"];
const CHANNELS = ["", "email", "sms", "whatsapp", "inapp", "webhook"];

function LogsPage() {
  const [status, setStatus] = useState("");
  const [channel, setChannel] = useState("");
  const [page, setPage] = useState(1);
  const limit = 20;

  const { data, isLoading } = useQuery({
    queryKey: ["notifications", { status, channel, page }],
    queryFn: () =>
      api<NotificationsResp>("/v1/notifications", {
        query: {
          limit,
          offset: (page - 1) * limit,
          status: status || undefined,
          channel: channel || undefined,
        },
      }),
  });

  const notifications = data?.notifications ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / limit));

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Notification Logs</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {total.toLocaleString()} total notifications
        </p>
      </div>

      <div className="flex items-center gap-3">
        <Select value={status} onValueChange={(v) => { setStatus(v ?? ""); setPage(1); }}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="All statuses" />
          </SelectTrigger>
          <SelectContent>
            {STATUSES.map((s) => (
              <SelectItem key={s} value={s}>
                {s || "All statuses"}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={channel} onValueChange={(v) => { setChannel(v ?? ""); setPage(1); }}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder="All channels" />
          </SelectTrigger>
          <SelectContent>
            {CHANNELS.map((c) => (
              <SelectItem key={c} value={c}>
                {c || "All channels"}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-11 rounded-lg" />
          ))}
        </div>
      ) : notifications.length === 0 ? (
        <EmptyState
          icon={ScrollTextIcon}
          title="No notifications found"
          description="Notifications will appear here once dispatched."
        />
      ) : (
        <>
          <div className="rounded-lg border divide-y">
            {notifications.map((n) => (
              <div key={n.id} className="flex items-center justify-between px-4 py-2.5">
                <div className="flex items-center gap-3 min-w-0">
                  <div className="min-w-0">
                    <p className="text-xs font-medium font-mono truncate max-w-50">
                      {n.subscriber_id}
                    </p>
                    <p className="text-[11px] text-muted-foreground">
                      {new Date(n.created_at).toLocaleString()}
                    </p>
                  </div>
                  <Badge variant="outline" className="shrink-0">{n.channel}</Badge>
                </div>
                <Badge variant={STATUS_VARIANT[n.status] ?? "outline"}>{n.status}</Badge>
              </div>
            ))}
          </div>
          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span>
              Page {page} of {totalPages} · {total.toLocaleString()} total
            </span>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                Previous
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
