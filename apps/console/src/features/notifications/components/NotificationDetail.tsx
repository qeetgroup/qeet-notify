import {
  Badge,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  TimeSince,
} from "@qeetrix/ui";

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  sent: "default", delivered: "default", failed: "destructive",
  suppressed: "secondary", pending: "outline",
};

interface NotificationDetailProps {
  notification: Record<string, any>;
}

export function NotificationDetail({ notification: n }: NotificationDetailProps) {
  return (
    <>
      <SheetHeader className="mb-4">
        <SheetTitle className="font-mono text-sm">{n.id}</SheetTitle>
        <SheetDescription>Notification details</SheetDescription>
      </SheetHeader>

      <dl className="space-y-3 text-sm">
        {[
          { label: "Channel",     value: n.channel },
          { label: "Subscriber",  value: n.subscriber_id },
          { label: "Subject",     value: n.subject ?? "—" },
        ].map(({ label, value }) => (
          <div key={label} className="flex justify-between gap-4">
            <dt className="text-muted-foreground">{label}</dt>
            <dd className="font-medium">{value}</dd>
          </div>
        ))}

        <div className="flex justify-between gap-4">
          <dt className="text-muted-foreground">Status</dt>
          <dd><Badge variant={STATUS_VARIANT[n.status] ?? "outline"}>{n.status}</Badge></dd>
        </div>

        <div className="flex justify-between gap-4">
          <dt className="text-muted-foreground">Sent</dt>
          <dd><TimeSince value={n.created_at} /></dd>
        </div>

        {n.error_message && (
          <div className="rounded-md bg-destructive/10 p-3 text-xs text-destructive">
            {n.error_message}
          </div>
        )}
      </dl>
    </>
  );
}
