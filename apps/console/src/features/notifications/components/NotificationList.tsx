import {
  Badge,
  DataState,
  Sheet,
  SheetContent,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TimeSince,
} from "@qeetrix/ui";
import { BellIcon } from "lucide-react";
import { useState } from "react";

import { useNotifications } from "../hooks/useNotifications";
import { NotificationDetail } from "./NotificationDetail";

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  sent:       "default",
  delivered:  "default",
  failed:     "destructive",
  suppressed: "secondary",
  pending:    "outline",
};

const CHANNEL_LABELS: Record<string, string> = {
  email: "Email", sms: "SMS", whatsapp: "WhatsApp",
  inapp: "In-app", webhook: "Webhook", push: "Push",
};

export function NotificationList() {
  const { data, isLoading, isError, error } = useNotifications();
  const items = (data as any)?.data ?? data ?? [];
  const [selected, setSelected] = useState<any>(null);

  return (
    <>
      <DataState
        isLoading={isLoading}
        isError={isError}
        error={error instanceof Error ? error : undefined}
        isEmpty={!isLoading && items.length === 0}
        emptyIcon={BellIcon}
        emptyTitle="No notifications yet"
        emptyDescription="Trigger an event to see notifications here."
        skeletonRows={6}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Channel</TableHead>
              <TableHead>Subscriber</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Sent</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((n: any) => (
              <TableRow
                key={n.id}
                className="cursor-pointer"
                onClick={() => setSelected(n)}
              >
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {n.id.slice(0, 8)}…
                </TableCell>
                <TableCell>
                  <Badge variant="outline">{CHANNEL_LABELS[n.channel] ?? n.channel}</Badge>
                </TableCell>
                <TableCell className="text-sm">{n.subscriber_id}</TableCell>
                <TableCell>
                  <Badge variant={STATUS_VARIANT[n.status] ?? "outline"}>{n.status}</Badge>
                </TableCell>
                <TableCell>
                  <TimeSince value={n.created_at} className="text-sm text-muted-foreground" />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataState>

      <Sheet open={!!selected} onOpenChange={(o) => !o && setSelected(null)}>
        <SheetContent side="right" className="w-full sm:max-w-lg">
          {selected && <NotificationDetail notification={selected} />}
        </SheetContent>
      </Sheet>
    </>
  );
}
