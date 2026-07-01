"use client";

import { useState } from "react";
import useSWR from "swr";
import {
  Badge,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  EmptyState,
  Skeleton,
} from "@qeetrix/ui";
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
  pending: "secondary",
  queued: "secondary",
  sent: "default",
  delivered: "outline",
  failed: "destructive",
  skipped: "secondary",
};

const STATUSES = ["", "pending", "queued", "sent", "delivered", "failed", "skipped"];
const CHANNELS = ["", "email", "sms", "whatsapp", "push", "inapp", "webhook"];

export default function LogsPage() {
  const [status, setStatus] = useState("");
  const [channel, setChannel] = useState("");

  const params = new URLSearchParams({ limit: "100" });
  if (status) params.set("status", status);
  if (channel) params.set("channel", channel);

  const { data, isLoading } = useSWR<{ notifications: Notification[] }>(
    `/api/v1/notifications?${params.toString()}`,
    apiFetcher
  );

  const notifications = data?.notifications ?? [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Notification Logs</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {notifications.length} notification{notifications.length !== 1 ? "s" : ""}
        </p>
      </div>

      <div className="flex gap-3">
        <Select value={status || "all"} onValueChange={(v) => setStatus(v === "all" ? "" : v)}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="All statuses" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            {STATUSES.filter(Boolean).map((s) => (
              <SelectItem key={s} value={s}>{s}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={channel || "all"} onValueChange={(v) => setChannel(v === "all" ? "" : v)}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="All channels" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All channels</SelectItem>
            {CHANNELS.filter(Boolean).map((c) => (
              <SelectItem key={c} value={c}>{c}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-12 rounded-lg" />)}
        </div>
      ) : notifications.length === 0 ? (
        <EmptyState title="No notifications found" description="Notifications appear here after events are triggered." />
      ) : (
        <div className="rounded-lg border divide-y">
          {notifications.map((n) => (
            <div key={n.id} className="flex items-center justify-between px-4 py-3 text-sm">
              <span className="font-mono text-xs text-muted-foreground w-24 truncate">{n.id.slice(0, 8)}…</span>
              <Badge variant="outline">{n.channel}</Badge>
              <Badge variant={STATUS_VARIANTS[n.status] ?? "outline"}>{n.status}</Badge>
              <span className="text-muted-foreground text-xs w-28 truncate">{n.provider ?? "—"}</span>
              <span className="text-muted-foreground text-xs">{new Date(n.created_at).toLocaleString()}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
